package db

import (
	"context"
	"fmt"
	"time"

	"github.com/derbylock/async-integration-testing/pkg/asit"
	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	KEY_ALL_CLIENTS        = "all_clients"
	KEY_CLIENT_PREFIX      = "client:"
	KEY_CLIENT_KEY_PREFIX  = "client_key:"
	KEY_CLIENT_KEYS_PREFIX = "client_keys:"
)

type ClientsRepository interface {
	GetAllClients(ctx context.Context) ([]*asit.Client, error)
	GetClientById(ctx context.Context, id string) (*asit.Client, error)
	SetClient(ctx context.Context, client *asit.Client) error
	RemoveClient(ctx context.Context, clientId string) error

	GetClientKeys(ctx context.Context, clientId string) (*asit.ClientKeys, error)
	AddClientKey(ctx context.Context, clientId string, key string) error
	GetClientByKey(ctx context.Context, key string) (*asit.Client, error)
	RemoveClientKey(ctx context.Context, key string) error
}

type KVClientsRepository struct {
	storage Storage
}

func NewKVClientsRepository(store Storage) *KVClientsRepository {
	return &KVClientsRepository{
		storage: store,
	}
}

func (r *KVClientsRepository) GetAllClients(ctx context.Context) ([]*asit.Client, error) {
	clients := &asit.ClientList{}
	ok, err := r.storage.Get(ctx, KEY_ALL_CLIENTS, clients)
	if err != nil {
		return nil, fmt.Errorf("can't retrieve db key %s, %w", KEY_ALL_CLIENTS, err)
	}
	if !ok {
		return []*asit.Client{}, nil
	}

	return clients.Clients, nil
}

func (r *KVClientsRepository) GetClientById(ctx context.Context, id string) (*asit.Client, error) {
	client := &asit.Client{}
	ok, err := r.storage.Get(ctx, KEY_CLIENT_PREFIX+id, client)
	if err != nil {
		return nil, fmt.Errorf("can't retrieve db key %s, %w", KEY_CLIENT_PREFIX+id, err)
	}
	if !ok {
		return nil, nil
	}

	return client, nil
}

func (r *KVClientsRepository) GetClientKeys(ctx context.Context, clientId string) (*asit.ClientKeys, error) {
	clientKeys := &asit.ClientKeys{}
	ok, err := r.storage.Get(ctx, KEY_CLIENT_KEYS_PREFIX+clientId, clientKeys)
	if err != nil {
		return nil, fmt.Errorf("can't retrieve db key %s, %w", KEY_CLIENT_KEYS_PREFIX+clientId, err)
	}
	if !ok {
		return nil, nil
	}

	return clientKeys, nil
}

func (r *KVClientsRepository) GetClientByKey(ctx context.Context, key string) (*asit.Client, error) {
	client := &asit.Client{}
	ok, err := r.storage.Get(ctx, KEY_CLIENT_KEY_PREFIX+key, client)
	if err != nil {
		return nil, fmt.Errorf("can't retrieve db key %s, %w", KEY_CLIENT_KEY_PREFIX+key, err)
	}
	if !ok {
		return nil, nil
	}

	return client, nil
}

type NonUniqueClientKeyError struct {
	key string
}

func (e *NonUniqueClientKeyError) Error() string {
	return fmt.Sprintf("non-unique client key %s", e.key)
}

func (r *KVClientsRepository) AddClientKey(ctx context.Context, clientId string, key string) error {
	client := &asit.Client{}
	err := r.storage.SetAndDeleteAtomically(ctx, []SetValueCommand{
		{
			key: KEY_CLIENT_PREFIX + clientId,
			updater: func(oldValue func(proto.Message) (bool, error)) (bool, proto.Message, error) {
				found, err := oldValue(client)
				if err != nil || !found {
					return false, nil, err
				}
				// keep old value
				return false, client, nil
			},
		},
		{
			key: KEY_CLIENT_KEYS_PREFIX + clientId,
			updater: func(oldValue func(proto.Message) (bool, error)) (bool, proto.Message, error) {
				clientKeys := &asit.ClientKeys{}
				found, err := oldValue(clientKeys)
				if err != nil {
					return false, nil, err
				}
				if found {
					newKeys := clientKeys.Keys
					if !slices.Contains(newKeys, key) {
						newKeys = append(newKeys, key)
					}
					return true, &asit.ClientKeys{Keys: newKeys}, nil
				}
				return true, &asit.ClientKeys{Keys: []string{key}}, nil
			},
		},
		{
			key: KEY_CLIENT_KEY_PREFIX + key,
			updater: func(oldValue func(proto.Message) (bool, error)) (bool, proto.Message, error) {
				oldClient := &asit.Client{}
				found, err := oldValue(oldClient)
				if err != nil {
					return false, nil, err
				}
				if found && oldClient.Id != client.Id {
					return false, nil, &NonUniqueClientKeyError{key: key}
				}
				return true, client, nil
			},
		},
	}, []string{}, func() []SetValueUnlockedCommand { return nil }, func() []string { return nil })

	if err != nil {
		return fmt.Errorf("can't add client key %s, %w", key, err)
	}
	return nil
}

func (r *KVClientsRepository) RemoveClientKey(ctx context.Context, key string) error {
	client, err := r.GetClientByKey(ctx, key)
	if err != nil {
		return err
	}

	if client == nil {
		return nil
	}

	err = r.storage.SetAndDeleteAtomically(ctx, []SetValueCommand{
		{
			key: KEY_CLIENT_KEYS_PREFIX + client.Id,
			updater: func(oldValue func(proto.Message) (bool, error)) (bool, proto.Message, error) {
				clientKeys := &asit.ClientKeys{}
				if found, err := oldValue(clientKeys); err != nil || !found {
					return false, nil, err
				}
				// filter slice, remove client with the specified Id if it exists
				newKeys := make([]string, len(clientKeys.Keys))
				i := 0
				for _, clientKey := range clientKeys.Keys {
					if key == clientKey {
						continue
					}
					newKeys[i] = clientKey
					i++
				}
				return true, &asit.ClientKeys{Keys: newKeys[:i]}, nil
			},
		},
	}, []string{
		KEY_CLIENT_KEY_PREFIX + key,
	}, func() []SetValueUnlockedCommand { return nil },
		func() []string { return nil })

	if err != nil {
		return fmt.Errorf("can't delete client key %s, %w", key, err)
	}
	return nil
}

func (r *KVClientsRepository) SetClient(ctx context.Context, client *asit.Client) error {
	var outdatedKeys *[]string
	err := r.storage.SetAndDeleteAtomically(ctx, []SetValueCommand{
		{
			key: KEY_ALL_CLIENTS,
			updater: func(oldValue func(proto.Message) (bool, error)) (bool, proto.Message, error) {
				clientList := &asit.ClientList{}
				found, err := oldValue(clientList)
				if err != nil {
					return false, nil, err
				}
				if !found {
					return true, &asit.ClientList{Clients: []*asit.Client{client}}, nil
				}
				// filter slice, remove client with the specified Id if it exists
				newClients := clientList.Clients
				indexFound := -1
				for i, c := range newClients {
					if c.Id == client.Id {
						indexFound = i
						break
					}
				}

				if indexFound < 0 {
					client.LastUpdated = timestamppb.New(time.Now())
					newClients = append(newClients, client)
				} else {
					newClients[indexFound].ClientProperties = client.ClientProperties
					newClients[indexFound].LastUpdated = timestamppb.New(time.Now())
				}
				return true, &asit.ClientList{Clients: newClients}, nil
			},
		},
		{
			// just to prevent any changes in client's key during update and to update allKeys
			key: KEY_CLIENT_KEYS_PREFIX + client.Id,
			updater: func(oldValue func(proto.Message) (bool, error)) (bool, proto.Message, error) {
				clientKeys := &asit.ClientKeys{}
				if found, err := oldValue(clientKeys); err != nil || !found {
					return false, nil, err
				}

				outdatedKeys = &clientKeys.Keys
				return true, clientKeys, nil
			},
		},
		{
			key: KEY_CLIENT_PREFIX + client.Id,
			updater: func(oldValue func(proto.Message) (bool, error)) (bool, proto.Message, error) {
				return true, client, nil
			},
		},
	},
		[]string{},
		func() []SetValueUnlockedCommand {
			if outdatedKeys == nil {
				return nil
			}
			cmds := make([]SetValueUnlockedCommand, len(*outdatedKeys))
			for i, key := range *outdatedKeys {
				cmds[i] = SetValueUnlockedCommand{key: KEY_CLIENT_KEY_PREFIX + key, newValue: client}
			}
			return cmds
		}, func() []string {
			return nil
		})
	if err != nil {
		return fmt.Errorf("can't set client with Id %s, %w", client.Id, err)
	}

	return nil
}

func (r *KVClientsRepository) RemoveClient(ctx context.Context, clientId string) error {
	var outdatedKeys *[]string
	err := r.storage.SetAndDeleteAtomically(ctx, []SetValueCommand{
		{
			key: KEY_ALL_CLIENTS,
			updater: func(oldValue func(proto.Message) (bool, error)) (bool, proto.Message, error) {
				clientList := &asit.ClientList{}
				found, err := oldValue(clientList)
				if err != nil {
					return false, nil, err
				}
				if !found {
					return true, &asit.ClientList{Clients: []*asit.Client{}}, nil
				}
				// filter slice, remove client with the specified Id if it exists
				newClients := make([]*asit.Client, len(clientList.Clients))
				i := 0
				for _, client := range clientList.Clients {
					if clientId == client.Id {
						continue
					}
					newClients[i] = client
					i++
				}
				return true, &asit.ClientList{Clients: newClients[:i]}, nil
			},
		},
		{
			key: KEY_CLIENT_KEYS_PREFIX + clientId,
			updater: func(oldValue func(proto.Message) (bool, error)) (bool, proto.Message, error) {
				clientKeys := &asit.ClientKeys{}
				if found, err := oldValue(clientKeys); err != nil || !found {
					return false, nil, err
				}
				outdatedKeys = &clientKeys.Keys
				return true, nil, nil
			},
		},
	},
		[]string{
			KEY_CLIENT_PREFIX + clientId,
		},
		func() []SetValueUnlockedCommand {
			return []SetValueUnlockedCommand{}
		}, func() []string {
			if outdatedKeys != nil {
				return *outdatedKeys
			}
			return nil
		})
	if err != nil {
		return fmt.Errorf("can't delete db key %s, %w", KEY_CLIENT_PREFIX+clientId, err)
	}

	return nil
}
