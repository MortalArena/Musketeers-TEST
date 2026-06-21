package content

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/protocol"
	"github.com/MortalArena/Musketeers/pkg/storage"
	"github.com/dgraph-io/badger/v4"
)

// BlockStore block storage interface
type BlockStore interface {
	Get(cid string) ([]byte, error)
	Put(cid string, data []byte, did string) error
	Size() int64
	ListKeys(prefix string) ([]string, error) // New method for mailbox support
}

// CIDFromData calculates CID = hex(sha256(data))
func CIDFromData(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// VerifyCID verifies that data matches CID
func VerifyCID(cid string, data []byte) error {
	computed := CIDFromData(data)
	if computed != cid {
		return fmt.Errorf("CID mismatch: expected %s, got %s", cid, computed)
	}
	return nil
}

// BadgerBlockStore BlockStore implementation using BadgerDB
type BadgerBlockStore struct {
	db       *badger.DB
	mu       sync.RWMutex
	size     int64
	quotaMgr *storage.QuotaManager // Connected to unified QuotaManager
	prefix   []byte
}

// NewBadgerBlockStore creates a block store
func NewBadgerBlockStore(db *badger.DB, qm *storage.QuotaManager) *BadgerBlockStore {
	return &BadgerBlockStore{
		db:       db,
		quotaMgr: qm,
		prefix:   []byte("block:"),
	}
}

func (s *BadgerBlockStore) blockKey(cid string) []byte {
	key := make([]byte, len(s.prefix)+len(cid))
	copy(key, s.prefix)
	copy(key[len(s.prefix):], cid)
	return key
}

// Get retrieves a block by CID
func (s *BadgerBlockStore) Get(cid string) ([]byte, error) {
	var data []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(s.blockKey(cid))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			data = append([]byte(nil), val...)
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("block not found: %s", cid)
	}
	return data, nil
}

// Put stores a block
func (s *BadgerBlockStore) Put(cid string, data []byte, did string) error {
	if len(data) > protocol.MaxBlockSize {
		return fmt.Errorf("block size exceeds limit (%d)", protocol.MaxBlockSize)
	}
	if err := VerifyCID(cid, data); err != nil {
		return err
	}

	// Use unified QuotaManager for quota check
	if err := s.quotaMgr.CheckAndAdd(did, int64(len(data))); err != nil {
		return fmt.Errorf("quota check failed: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(s.blockKey(cid), data)
	})
	if err != nil {
		// On failure, release reserved space
		s.quotaMgr.Release(did, int64(len(data)))
		return fmt.Errorf("failed to store block: %w", err)
	}
	s.size += int64(len(data))
	return nil
}

// Size returns total used size
func (s *BadgerBlockStore) Size() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.size
}

// ListKeys lists all keys with a given prefix
func (s *BadgerBlockStore) ListKeys(prefix string) ([]string, error) {
	var keys []string
	searchPrefix := s.blockKey(prefix)

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(searchPrefix); it.ValidForPrefix(searchPrefix); it.Next() {
			item := it.Item()
			key := item.KeyCopy(nil)
			// Remove the "block:" prefix to return just the CID
			if len(key) > len(s.prefix) {
				keys = append(keys, string(key[len(s.prefix):]))
			}
		}
		return nil
	})

	return keys, err
}

// MemoryBlockStore in-memory store for tests
type MemoryBlockStore struct {
	mu       sync.RWMutex
	blocks   map[string][]byte
	size     int64
	quotaMgr *storage.QuotaManager // Connected to unified QuotaManager
}

// NewMemoryBlockStore creates a memory store
func NewMemoryBlockStore(qm *storage.QuotaManager) *MemoryBlockStore {
	return &MemoryBlockStore{
		blocks:   make(map[string][]byte),
		quotaMgr: qm,
	}
}

func (s *MemoryBlockStore) Get(cid string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, ok := s.blocks[cid]
	if !ok {
		return nil, fmt.Errorf("block not found: %s", cid)
	}
	return append([]byte(nil), data...), nil
}

func (s *MemoryBlockStore) Put(cid string, data []byte, did string) error {
	if len(data) > protocol.MaxBlockSize {
		return fmt.Errorf("block size exceeds limit")
	}
	if err := VerifyCID(cid, data); err != nil {
		return err
	}

	// Use unified QuotaManager for quota check
	if err := s.quotaMgr.CheckAndAdd(did, int64(len(data))); err != nil {
		return fmt.Errorf("quota check failed: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.blocks[cid] = append([]byte(nil), data...)
	s.size += int64(len(data))
	return nil
}

func (s *MemoryBlockStore) Size() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.size
}

// ListKeys lists all keys with a given prefix
func (s *MemoryBlockStore) ListKeys(prefix string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var keys []string
	for key := range s.blocks {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			keys = append(keys, key[len(prefix):])
		}
	}
	return keys, nil
}
