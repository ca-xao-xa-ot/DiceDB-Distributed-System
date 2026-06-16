package services

import (
	"errors"
	"os"
	"sort"
	"sync"
)

var ErrKeyNotFound = errors.New("key not found")

type KVStore interface {
	Set(key, value string) error
	Get(key string) (string, error)
	Delete(key string) error
	Count() int
	Keys() []string
}

type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewKVStoreFromEnv() KVStore {
	store := &MemoryStore{data: map[string]string{}}
	if os.Getenv("DASHBOARD_SEED") == "false" {
		return store
	}
	_ = store.Set("project", "DiceDB")
	_ = store.Set("course", "Ung dung Phan tan")
	_ = store.Set("mode", "demo")
	return store
}

func (m *MemoryStore) Set(key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return nil
}

func (m *MemoryStore) Get(key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.data[key]
	if !ok {
		return "", ErrKeyNotFound
	}
	return value, nil
}

func (m *MemoryStore) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[key]; !ok {
		return ErrKeyNotFound
	}
	delete(m.data, key)
	return nil
}

func (m *MemoryStore) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data)
}

func (m *MemoryStore) Keys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]string, 0, len(m.data))
	for key := range m.data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
