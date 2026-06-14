package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
)

var ErrKeyNotFound = errors.New("key not found")

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
	mode string
}

func NewMemoryStore(mode string) *MemoryStore {
	if mode == "" {
		mode = "In-memory demo mode"
	}
	return &MemoryStore{
		data: make(map[string]string),
		mode: mode,
	}
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
		return "", ErrKeyNotFound
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
	return m.mode
}

func NewKVStoreFromEnv() KVStore {
	addr := os.Getenv("DICEDB_ADDR")
	if addr == "" {
		return NewMemoryStore("In-memory demo mode")
	}

	return NewMemoryStore(fmt.Sprintf("In-memory demo mode (DiceDB target configured: %s)", addr))
}
