package db

import (
	"context"
	"fmt"

	"github.com/derbylock/async-integration-testing/pkg/asit"
)

const (
	KEY_ALL_CLIENTS        = "all_clients"
	KEY_CLIENT_PREFIX      = "client:"
	KEY_CLIENT_KEY_PREFIX  = "client_key:"
	KEY_CLIENT_KEYS_PREFIX = "client_keys:"
)

type ClientsRepository interface {
	getAllClients(ctx context.Context) ([]*asit.Client, error)
	getClientById(ctx context.Context, id string) (*asit.Client, error)
	getClientByKey(ctx context.Context, key string) (*asit.Client, error)
	addClientKey(ctx context.Context, client *asit.Client, key string) error
	removeClientKey(ctx context.Context, key string) error
	setClient(ctx context.Context, client *asit.Client) error
	removeClient(ctx context.Context, clientId string) error
}

type KVClientsRepository struct {
	storage Storage
}

func NewKVClientsRepository(store Storage) *KVClientsRepository {
	return &KVClientsRepository{
		storage: store,
	}
}

func (r *KVClientsRepository) getAllClients(ctx context.Context) ([]*asit.Client, error) {
	clients := &asit.ClientList{}
	ok, err := r.storage.Get(ctx, KEY_ALL_CLIENTS, clients)
	if !ok {
		return nil, fmt.Errorf("db key not found %s", KEY_ALL_CLIENTS)
	}
	if err != nil {
		return nil, fmt.Errorf("can't retrieve db key %s, %w", KEY_ALL_CLIENTS, err)
	}

	return clients.Clients, nil
}

func (r *KVClientsRepository) getClientById(ctx context.Context, id string) (*asit.Client, error) {
	client := &asit.Client{}
	ok, err := r.storage.Get(ctx, KEY_CLIENT_PREFIX+id, client)
	if !ok {
		return nil, fmt.Errorf("db key not found %s", KEY_CLIENT_PREFIX+id)
	}
	if err != nil {
		return nil, fmt.Errorf("can't retrieve db key %s, %w", KEY_CLIENT_PREFIX+id, err)
	}

	return client, nil
}

func (r *KVClientsRepository) getClientByKey(ctx context.Context, key string) (*asit.Client, error) {
	client := &asit.Client{}
	ok, err := r.storage.Get(ctx, KEY_CLIENT_KEY_PREFIX+key, client)
	if !ok {
		return nil, fmt.Errorf("db key not found %s", KEY_CLIENT_KEY_PREFIX+key)
	}
	if err != nil {
		return nil, fmt.Errorf("can't retrieve db key %s, %w", KEY_CLIENT_KEY_PREFIX+key, err)
	}

	return client, nil
}

func (r *KVClientsRepository) addClientKey(ctx context.Context, client *asit.Client, key string) error {
	err := r.storage.Set(ctx, KEY_CLIENT_KEY_PREFIX+key, client)
	if err != nil {
		return fmt.Errorf("can't set db key %s, %w", KEY_CLIENT_KEY_PREFIX+key, err)
	}
	return nil
}

func (r *KVClientsRepository) removeClientKey(ctx context.Context, key string) error {
	err := r.storage.Delete(ctx, KEY_CLIENT_KEY_PREFIX+key)
	if err != nil {
		return fmt.Errorf("can't delete db key %s, %w", KEY_CLIENT_KEY_PREFIX+key, err)
	}
	return nil
}

func (r *KVClientsRepository) setClient(ctx context.Context, client *asit.Client) error {
	err := r.storage.Set(ctx, KEY_CLIENT_PREFIX+client.Id, client)
	if err != nil {
		return fmt.Errorf("can't set db key %s, %w", KEY_CLIENT_PREFIX+client.Id, err)
	}
	return nil
}

func (r *KVClientsRepository) removeClient(ctx context.Context, clientId string) error {
	var outdatedKeys *[]string
	err := r.storage.SetAndDeleteAtomically(ctx, []SetValueCommand{
		{
			key: KEY_ALL_CLIENTS,
			updater: func(oldValue interface{}) (interface{}, error) {
				clientList, ok := oldValue.(*asit.ClientList)
				if !ok {
					return nil, fmt.Errorf("can't cast db value with key %s", KEY_ALL_CLIENTS)
				}
				// filter slice, remove client with the specified Id if it exists
				newClients := make([]*asit.Client, len(clientList.Clients))
				i := 0
				for _, client := range clientList.Clients {
					newClients[i] = client
					i++
				}
				return asit.ClientList{Clients: newClients[:i]}, nil
			},
		},
		{
			key: KEY_CLIENT_KEYS_PREFIX + clientId,
			updater: func(oldValue interface{}) (interface{}, error) {
				clientKeys, ok := oldValue.(*asit.ClientKeys)
				if !ok {
					return nil, fmt.Errorf("can't cast db value with key %s", KEY_ALL_CLIENTS)
				}
				outdatedKeys = &clientKeys.Keys
				return nil, nil
			},
		},
	},
		[]string{
			KEY_CLIENT_PREFIX + clientId,
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
