package services

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type KVStore interface {
	Set(ctx context.Context, key, value string) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
	Count(ctx context.Context) (int, error)
	Mode() string
}

type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[string]string)}
}

func (m *MemoryStore) Set(_ context.Context, key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return nil
}

func (m *MemoryStore) Get(_ context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.data[key]
	if !ok {
		return "", redis.Nil
	}
	return value, nil
}

func (m *MemoryStore) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *MemoryStore) Count(_ context.Context) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data), nil
}

func (m *MemoryStore) Mode() string {
	return "In-memory demo mode"
}

type DiceDBStore struct {
	client *redis.Client
	label  string
}

func NewDiceDBStore(addr, password string, db int) (*DiceDBStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &DiceDBStore{
		client: client,
		label:  fmt.Sprintf("DiceDB connected: %s", addr),
	}, nil
}

func (d *DiceDBStore) Set(ctx context.Context, key, value string) error {
	return d.client.Set(ctx, key, value, 0).Err()
}

func (d *DiceDBStore) Get(ctx context.Context, key string) (string, error) {
	return d.client.Get(ctx, key).Result()
}

func (d *DiceDBStore) Delete(ctx context.Context, key string) error {
	return d.client.Del(ctx, key).Err()
}

func (d *DiceDBStore) Count(ctx context.Context) (int, error) {
	iter := d.client.Scan(ctx, 0, "*", 0).Iterator()
	keys := make([]string, 0)
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return 0, err
	}
	sort.Strings(keys)
	return len(keys), nil
}

func (d *DiceDBStore) Mode() string {
	return d.label
}

func NewKVStoreFromEnv() KVStore {
	addr := os.Getenv("DICEDB_ADDR")
	if addr == "" {
		return NewMemoryStore()
	}

	store, err := NewDiceDBStore(addr, os.Getenv("DICEDB_PASSWORD"), 0)
	if err != nil {
		return NewMemoryStore()
	}
	return store
}
