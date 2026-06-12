package observability

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/badger/v4"
)

type AuditLog interface {
	Log(entry AuditEntry) error
	Query(filter AuditFilter) ([]AuditEntry, error)
	Close() error
}

type AuditEntry struct {
	ID        string         `json:"id"`
	Timestamp time.Time      `json:"timestamp"`
	Actor     string         `json:"actor"`
	Action    string         `json:"action"`
	Resource  string         `json:"resource"`
	Result    string         `json:"result"`
	Details   map[string]any `json:"details"`
	Signature string         `json:"signature,omitempty"`
}

type AuditFilter struct {
	Actor     string
	Action    string
	Resource  string
	Result    string
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Offset    int
}

type BadgerAuditLog struct {
	mu sync.RWMutex
	db *badger.DB
}

func NewBadgerAuditLog(dbPath string) (*BadgerAuditLog, error) {
	opts := badger.DefaultOptions(dbPath)
	opts.Logger = nil
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &BadgerAuditLog{db: db}, nil
}

func (a *BadgerAuditLog) Log(entry AuditEntry) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if entry.ID == "" {
		entry.ID = generateAuditID()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return a.db.Update(func(txn *badger.Txn) error {
		key := []byte(fmt.Sprintf("audit:%s:%s", entry.Timestamp.Format(time.RFC3339Nano), entry.ID))
		return txn.Set(key, data)
	})
}

func (a *BadgerAuditLog) Query(filter AuditFilter) ([]AuditEntry, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	var entries []AuditEntry
	err := a.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()
		prefix := []byte("audit:")
		skipped := 0
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var entry AuditEntry
				if err := json.Unmarshal(val, &entry); err != nil {
					return err
				}
				if filter.Actor != "" && entry.Actor != filter.Actor {
					return nil
				}
				if filter.Action != "" && entry.Action != filter.Action {
					return nil
				}
				if filter.Resource != "" && entry.Resource != filter.Resource {
					return nil
				}
				if filter.Result != "" && entry.Result != filter.Result {
					return nil
				}
				if filter.StartTime != nil && entry.Timestamp.Before(*filter.StartTime) {
					return nil
				}
				if filter.EndTime != nil && entry.Timestamp.After(*filter.EndTime) {
					return nil
				}
				if filter.Offset > 0 && skipped < filter.Offset {
					skipped++
					return nil
				}
				entries = append(entries, entry)
				if filter.Limit > 0 && len(entries) >= filter.Limit {
					return errLimitReached
				}
				return nil
			})
			if err == errLimitReached {
				return nil
			}
			if err != nil {
				return err
			}
		}
		return nil
	})
	return entries, err
}

func (a *BadgerAuditLog) Close() error {
	return a.db.Close()
}

type MemoryAuditLog struct {
	mu      sync.RWMutex
	entries []AuditEntry
}

func NewMemoryAuditLog() *MemoryAuditLog {
	return &MemoryAuditLog{}
}

func (a *MemoryAuditLog) Log(entry AuditEntry) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if entry.ID == "" {
		entry.ID = generateAuditID()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	a.entries = append(a.entries, entry)
	return nil
}

func (a *MemoryAuditLog) Query(filter AuditFilter) ([]AuditEntry, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	var entries []AuditEntry
	for _, entry := range a.entries {
		if filter.Actor != "" && entry.Actor != filter.Actor {
			continue
		}
		if filter.Action != "" && entry.Action != filter.Action {
			continue
		}
		if filter.Resource != "" && entry.Resource != filter.Resource {
			continue
		}
		if filter.Result != "" && entry.Result != filter.Result {
			continue
		}
		if filter.StartTime != nil && entry.Timestamp.Before(*filter.StartTime) {
			continue
		}
		if filter.EndTime != nil && entry.Timestamp.After(*filter.EndTime) {
			continue
		}
		entries = append(entries, entry)
		if filter.Limit > 0 && len(entries) >= filter.Limit {
			break
		}
	}
	return entries, nil
}

func (a *MemoryAuditLog) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = nil
	return nil
}

var auditSeq uint64

func generateAuditID() string {
	return fmt.Sprintf("audit-%d", atomic.AddUint64(&auditSeq, 1))
}

var errLimitReached = fmt.Errorf("limit reached")
