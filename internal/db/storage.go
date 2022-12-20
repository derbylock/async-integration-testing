package db

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v9"
)

const maxRetries = 10

// Storage is an abstraction for different key-value store implementations.
// A store must be able to store, retrieve and delete key-value pairs,
// with the key being a string and the value being any Go interface{}.
type SetValueCommand struct {
	key     string
	updater func(oldValue interface{}) (interface{}, error)
}

type Storage interface {
	// Set stores the given value for the given key.
	// The implementation automatically marshalls the value.
	// The marshalling format depends on the implementation. It can be JSON, gob etc.
	// The key must not be "" and the value must not be nil.
	Set(ctx context.Context, k string, v interface{}) error

	// Updates multiple values specified by SetValueCommand, deleteKeys and afterRemovalKeysSupplier atomically
	// Deletes keys sepcified in deleteKeys
	// If keys used SetValueCommand and deleteKeys not changed during update, removes all keys provided by the afterRemovalKeys func
	// Not that afterRemovalKeys will be removed even if values with the same keys changed/set/removed during the SetValueCommand, deleteKeys processing
	SetAndDeleteAtomically(ctx context.Context, sets []SetValueCommand, deleteKeys []string, afterRemovalKeysSupplier func() []string) error

	// Get retrieves the value for the given key.
	// The implementation automatically unmarshalls the value.
	// The unmarshalling source depends on the implementation. It can be JSON, gob etc.
	// The automatic unmarshalling requires a pointer to an object of the correct type
	// being passed as parameter.
	// In case of a struct the Get method will populate the fields of the object
	// that the passed pointer points to with the values of the retrieved object's values.
	// If no value is found it returns (false, nil).
	// The key must not be "" and the pointer must not be nil.
	Get(ctx context.Context, k string, v interface{}) (found bool, err error)

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
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
}

type RedisStorage struct {
	client redis.UniversalClient
	codec  StorageCodec
}

func NewRedisStorage(client redis.UniversalClient, codec StorageCodec) *RedisStorage {
	return &RedisStorage{client: client, codec: codec}
}

func (s *RedisStorage) Set(ctx context.Context, k string, v interface{}) error {
	bytes, err := s.codec.Marshal(v)
	if err != nil {
		return err
	}
	s.client.Set(ctx, k, bytes, 0)
	return nil
}

func (s *RedisStorage) Get(ctx context.Context, k string, v interface{}) (found bool, err error) {
	bytes, err := s.client.Get(ctx, k).Bytes()
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

func (s *RedisStorage) SetAndDeleteAtomically(ctx context.Context, sets []SetValueCommand, deleteKeys []string, afterRemovalKeysSupplier func() []string) error {
	deleteKeysCount := len(deleteKeys)
	setsKeysCount := len(deleteKeys)
	allKeys := make([]string, deleteKeysCount+setsKeysCount)
	for i := 0; i < deleteKeysCount; i++ {
		allKeys[i] = deleteKeys[i]
	}

	for i := 0; i < setsKeysCount; i++ {
		allKeys[deleteKeysCount+i] = sets[i].key
	}

	txf := func(tx *redis.Tx) error {
		oldVals := make([]interface{}, len(sets))
		for i, set := range sets {
			bytes, err := tx.Get(ctx, set.key).Bytes()
			if err != nil {
				return err
			}
			if bytes == nil {
				oldVals[i] = nil
				continue
			}

			err = s.codec.Unmarshal(bytes, oldVals[i])
			if err != nil {
				return nil
			}
		}
		// Operation is commited only if the watched keys remain unchanged.
		_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			for i, set := range sets {
				newVal, errx := set.updater(oldVals[i])
				if errx != nil {
					return errx
				}
				if newVal == nil {
					pipe.Del(ctx, set.key)
					continue
				}
				pipe.Set(ctx, set.key, newVal, 0)
			}
			for _, deleteKey := range deleteKeys {
				pipe.Del(ctx, deleteKey)
			}
			afterRemovalKeys := afterRemovalKeysSupplier()
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
