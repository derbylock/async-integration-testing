package db

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v9"
	"google.golang.org/protobuf/proto"
)

const maxRetries = 10

// Storage is an abstraction for different key-value store implementations.
// A store must be able to store, retrieve and delete key-value pairs,
// with the key being a string and the value being any Go interface{}.
type SetValueCommand struct {
	key     string
	updater func(oldValue func(proto.Message) (bool, error)) (bool, proto.Message, error)
}

type SetValueUnlockedCommand struct {
	key      string
	newValue proto.Message
}

type Storage interface {
	// Set stores the given value for the given key.
	// The implementation automatically marshalls the value.
	// The marshalling format depends on the implementation. It can be JSON, gob etc.
	// The key must not be "" and the value must not be nil.
	Set(ctx context.Context, k string, v proto.Message) error

	// Updates multiple values specified by SetValueCommand, deleteKeys and afterRemovalKeysSupplier atomically
	// Deletes keys sepcified in deleteKeys
	// If keys used lockedSets and lockedDeleteKeys not changed during update, rocesses all commands provided by the unlockedSets func
	// Not that unlockedSets will be processed even if values with the same keys changed/set/removed during the lockedSets, lockedDeleteKeys processing
	SetAndDeleteAtomically(ctx context.Context, lockedSets []SetValueCommand, lockedDeleteKeys []string, unlockedSets func() []SetValueUnlockedCommand, unlockedDeleteKeys func() []string) error

	// Get retrieves the value for the given key.
	// The implementation automatically unmarshalls the value.
	// The unmarshalling source depends on the implementation. It can be JSON, gob etc.
	// The automatic unmarshalling requires a pointer to an object of the correct type
	// being passed as parameter.
	// In case of a struct the Get method will populate the fields of the object
	// that the passed pointer points to with the values of the retrieved object's values.
	// If no value is found it returns (false, nil).
	// The key must not be "" and the pointer must not be nil.
	Get(ctx context.Context, k string, v proto.Message) (found bool, err error)

	// Delete deletes the stored value for the given key.
	// Deleting a non-existing key-value pair does NOT lead to an error.
	// The key must not be "".
	Delete(ctx context.Context, k ...string) error

	// Close must be called when the work with the key-value store is done.
	// Most (if not all) implementations are meant to be used long-lived,
	// so only call Close() at the very end.
	// Depending on the store implementation it might do one or more of the following:
	// Make sure all pending updates make their way to disk,
	// finish open transactions,
	// close the file handle to an embedded DB,
	// close the connection to the DB server,
	// release any open resources,
	// etc.
	// Some implementUniversalation might not need the store to be closed,
	// but as long as you work with the gokv.Store interface you never know which implementation
	// is passed to your method, so you should always call it.
	Close() error
}

type StorageCodec interface {
	Marshal(v proto.Message) ([]byte, error)
	Unmarshal(data []byte, v proto.Message) error
}

type RedisStorage struct {
	client redis.UniversalClient
	codec  StorageCodec
}

func NewRedisStorage(client redis.UniversalClient, codec StorageCodec) *RedisStorage {
	return &RedisStorage{client: client, codec: codec}
}

func (s *RedisStorage) Set(ctx context.Context, k string, v proto.Message) error {
	bytes, err := s.codec.Marshal(v)
	if err != nil {
		return err
	}
	s.client.Set(ctx, k, bytes, 0)
	return nil
}

func (s *RedisStorage) Get(ctx context.Context, k string, v proto.Message) (found bool, err error) {
	bytes, err := s.client.Get(ctx, k).Bytes()
	if err == redis.Nil {
		err = nil
		bytes = nil
	}
	if err != nil {
		return false, err
	}
	if bytes == nil {
		return false, nil
	}

	err = s.codec.Unmarshal(bytes, v)
	return true, err
}

func (s *RedisStorage) Delete(ctx context.Context, k ...string) error {
	return s.client.Del(ctx, k...).Err()
}

func (s *RedisStorage) Close() error {
	return s.client.Close()
}

func (s *RedisStorage) SetAndDeleteAtomically(ctx context.Context, lockedSets []SetValueCommand, lockedDeleteKeys []string, unlockedSets func() []SetValueUnlockedCommand, unlockedDeleteKeys func() []string) error {
	deleteKeysCount := len(lockedDeleteKeys)
	setsKeysCount := len(lockedSets)
	allKeys := make([]string, deleteKeysCount+setsKeysCount)
	for i := 0; i < deleteKeysCount; i++ {
		allKeys[i] = lockedDeleteKeys[i]
	}

	for i := 0; i < setsKeysCount; i++ {
		allKeys[deleteKeysCount+i] = lockedSets[i].key
	}

	txf := func(tx *redis.Tx) error {
		oldBytes := make([][]byte, len(lockedSets))
		for i, set := range lockedSets {
			bytes, err := tx.Get(ctx, set.key).Bytes()
			if err == redis.Nil {
				err = nil
				bytes = nil
			}
			if err != nil {
				return err
			}
			oldBytes[i] = bytes
		}
		// Operation is commited only if the watched keys remain unchanged.
		_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			for i, set := range lockedSets {
				requiresUpdate, newVal, errx := set.updater(func(m proto.Message) (bool, error) {
					oldBytesCurrent := oldBytes[i]
					if oldBytesCurrent == nil {
						return false, nil
					}
					return true, proto.Unmarshal(oldBytesCurrent, m)
				})
				if errx != nil {
					return errx
				}
				if requiresUpdate {
					if newVal == nil {
						pipe.Del(ctx, set.key)
						continue
					}
					newValBytes, err := s.codec.Marshal(newVal)
					if err != nil {
						return err
					}
					pipe.Set(ctx, set.key, newValBytes, 0)
				}
			}
			for _, deleteKey := range lockedDeleteKeys {
				pipe.Del(ctx, deleteKey)
			}
			sets := unlockedSets()
			for _, set := range sets {
				newValBytes, err := s.codec.Marshal(set.newValue)
				if err != nil {
					return err
				}
				pipe.Set(ctx, set.key, newValBytes, 0)
			}

			afterRemovalKeys := unlockedDeleteKeys()
			for _, deleteKey := range afterRemovalKeys {
				pipe.Del(ctx, deleteKey)
			}
			return nil
		})
		return err
	}

	for i := 0; i < maxRetries; i++ {
		err := s.client.Watch(ctx, txf, allKeys...)
		if err == nil {
			// Success.
			return nil
		}
		if err == redis.TxFailedErr {
			// Optimistic lock lost. Retry.
			continue
		}
		// Return any other error.
		return err
	}

	return fmt.Errorf("concurrent updates of the keys %v", allKeys)
}
