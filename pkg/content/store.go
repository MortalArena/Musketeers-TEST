package content

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/dgraph-io/badger/v4"
	"github.com/MortalArena/Musketeers/pkg/protocol"
)

// BlockStore واجهة تخزين الكتل
type BlockStore interface {
	Get(cid string) ([]byte, error)
	Put(cid string, data []byte) error
	Size() int64
}

// CIDFromData يحسب CID = hex(sha256(data))
func CIDFromData(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// VerifyCID يتحقق أن البيانات تطابق CID
func VerifyCID(cid string, data []byte) error {
	computed := CIDFromData(data)
	if computed != cid {
		return fmt.Errorf("CID mismatch: expected %s, got %s", cid, computed)
	}
	return nil
}

// BadgerBlockStore تطبيق BlockStore باستخدام BadgerDB
type BadgerBlockStore struct {
	db       *badger.DB
	mu       sync.RWMutex
	size     int64
	quota    int64 // بالبايت
	prefix   []byte
}

// NewBadgerBlockStore ينشئ مخزن كتل
func NewBadgerBlockStore(db *badger.DB, quotaMB int64) *BadgerBlockStore {
	return &BadgerBlockStore{
		db:     db,
		quota:  quotaMB * 1024 * 1024,
		prefix: []byte("block:"),
	}
}

func (s *BadgerBlockStore) blockKey(cid string) []byte {
	key := make([]byte, len(s.prefix)+len(cid))
	copy(key, s.prefix)
	copy(key[len(s.prefix):], cid)
	return key
}

// Get يجلب كتلة بالـ CID
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
		return nil, fmt.Errorf("كتلة غير موجودة: %s", cid)
	}
	return data, nil
}

// Put يخزّن كتلة
func (s *BadgerBlockStore) Put(cid string, data []byte) error {
	if len(data) > protocol.MaxBlockSize {
		return fmt.Errorf("حجم الكتلة يتجاوز الحد (%d)", protocol.MaxBlockSize)
	}
	if err := VerifyCID(cid, data); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.quota > 0 && s.size+int64(len(data)) > s.quota {
		return fmt.Errorf("تجاوز حصة التخزين")
	}

	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(s.blockKey(cid), data)
	})
	if err != nil {
		return err
	}
	s.size += int64(len(data))
	return nil
}

// Size يرجع الحجم الإجمالي المستخدم
func (s *BadgerBlockStore) Size() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.size
}

// MemoryBlockStore مخزن في الذاكرة للاختبارات
type MemoryBlockStore struct {
	mu    sync.RWMutex
	blocks map[string][]byte
	size   int64
	quota  int64
}

// NewMemoryBlockStore ينشئ مخزن ذاكرة
func NewMemoryBlockStore(quotaMB int64) *MemoryBlockStore {
	return &MemoryBlockStore{
		blocks: make(map[string][]byte),
		quota:  quotaMB * 1024 * 1024,
	}
}

func (s *MemoryBlockStore) Get(cid string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, ok := s.blocks[cid]
	if !ok {
		return nil, fmt.Errorf("كتلة غير موجودة: %s", cid)
	}
	return append([]byte(nil), data...), nil
}

func (s *MemoryBlockStore) Put(cid string, data []byte) error {
	if len(data) > protocol.MaxBlockSize {
		return fmt.Errorf("حجم الكتلة يتجاوز الحد")
	}
	if err := VerifyCID(cid, data); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.quota > 0 && s.size+int64(len(data)) > s.quota {
		return fmt.Errorf("تجاوز حصة التخزين")
	}
	s.blocks[cid] = append([]byte(nil), data...)
	s.size += int64(len(data))
	return nil
}

func (s *MemoryBlockStore) Size() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.size
}
