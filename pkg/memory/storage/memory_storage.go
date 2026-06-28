package storage

import (
	"context"
	"fmt"
	"sync"
)

// MemoryStorage تخزين في الذاكرة
type MemoryStorage struct {
	data map[string][]byte
	mu   sync.RWMutex
}

// NewMemoryStorage ينشئ تخزين في الذاكرة جديد
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string][]byte),
	}
}

// Save يحفظ البيانات
func (ms *MemoryStorage) Save(ctx context.Context, key string, data []byte) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.data[key] = data
	return nil
}

// Load يحمل البيانات
func (ms *MemoryStorage) Load(ctx context.Context, key string) ([]byte, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	data, exists := ms.data[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	return data, nil
}

// Delete يحذف البيانات
func (ms *MemoryStorage) Delete(ctx context.Context, key string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	delete(ms.data, key)
	return nil
}
