package state

import (
	"fmt"
	"sync"

	"github.com/dgraph-io/badger/v4"
)

type StateStore interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	Delete(key string) error
	List(prefix string) ([]string, error)
	Close() error
}

type MemoryStateStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func NewMemoryStateStore() *MemoryStateStore {
	return &MemoryStateStore{data: make(map[string][]byte)}
}

func (s *MemoryStateStore) Get(key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, exists := s.data[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	return append([]byte(nil), val...), nil
}

func (s *MemoryStateStore) Set(key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = append([]byte(nil), value...)
	return nil
}

func (s *MemoryStateStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

func (s *MemoryStateStore) List(prefix string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0)
	for k := range s.data {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

func (s *MemoryStateStore) Close() error { return nil }

type BadgerStateStore struct {
	db *badger.DB
}

func NewBadgerStateStore(dbPath string) (*BadgerStateStore, error) {
	opts := badger.DefaultOptions(dbPath)
	opts.Logger = nil
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &BadgerStateStore{db: db}, nil
}

func (s *BadgerStateStore) Get(key string) ([]byte, error) {
	var val []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err == badger.ErrKeyNotFound {
			return nil
		}
		if err != nil {
			return err
		}
		return item.Value(func(v []byte) error {
			val = append([]byte(nil), v...)
			return nil
		})
	})
	return val, err
}

func (s *BadgerStateStore) Set(key string, value []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), value)
	})
}

func (s *BadgerStateStore) Delete(key string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

func (s *BadgerStateStore) List(prefix string) ([]string, error) {
	var keys []string
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		prefixBytes := []byte(prefix)
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			keys = append(keys, string(it.Item().Key()))
		}
		return nil
	})
	return keys, err
}

func (s *BadgerStateStore) Close() error { return s.db.Close() }
