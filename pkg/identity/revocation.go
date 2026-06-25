package identity

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
	"github.com/golang/groupcache/lru"
)

// RevocationRecord سجل إلغاء الهوية
type RevocationRecord struct {
	DID          string `json:"did"`
	RevokedAt    int64  `json:"revoked_at"`
	PublicKeyHex string `json:"public_key_hex,omitempty"`
	Signature    string `json:"signature"`
}

// RevocationPayload payload ثابت للتوقيع
func RevocationPayload(rec *RevocationRecord) string {
	return rec.DID + "|" + strconv.FormatInt(rec.RevokedAt, 10)
}

// NewRevocationRecord ينشئ سجل إلغاء
func NewRevocationRecord(did string, priv ed25519.PrivateKey) (*RevocationRecord, error) {
	rec := &RevocationRecord{
		DID:          did,
		RevokedAt:    time.Now().Unix(),
		PublicKeyHex: nrcrypto.PublicKeyHex(priv.Public().(ed25519.PublicKey)),
	}
	payload := RevocationPayload(rec)
	sig, err := nrcrypto.SignPayloadHex(priv, nrcrypto.DomainRevocation, payload)
	if err != nil {
		return nil, err
	}
	rec.Signature = sig
	return rec, nil
}

// Verify يتحقق من سجل الإلغاء
func (rec *RevocationRecord) Verify(pub ed25519.PublicKey) error {
	if rec.DID == "" || rec.Signature == "" {
		return fmt.Errorf("حقول مطلوبة ناقصة")
	}
	if nrcrypto.DIDFromPublicKey(pub) != rec.DID {
		return fmt.Errorf("DID لا يطابق المفتاح العام")
	}
	payload := RevocationPayload(rec)
	return nrcrypto.VerifyPayloadHex(pub, nrcrypto.DomainRevocation, payload, rec.Signature)
}

// DHTKey مفتاح DHT للإلغاء
func (rec *RevocationRecord) DHTKey() string {
	return "/mskt/revoke/" + rec.DID
}

// CRLCache ذاكرة تخزين مؤقت لسجلات الإلغاء
type CRLCache struct {
	mu      sync.RWMutex
	revoked map[string]int64 // DID -> RevokedAt
	ttl     time.Duration
	// [SAFETY] LRU cache for better eviction policy
	lru *lru.Cache
	// [SAFETY] Disk persistence path
	diskPath string
}

// NewCRLCache ينشئ CRL cache
func NewCRLCache(ttl time.Duration) *CRLCache {
	// [SAFETY] Create LRU cache with 5000 entries
	cache := lru.New(5000)
	return &CRLCache{
		revoked:  make(map[string]int64),
		ttl:      ttl,
		lru:      cache,
		diskPath: "",
	}
}

// NewCRLCacheWithDisk ينشئ CRL cache مع disk persistence
func NewCRLCacheWithDisk(ttl time.Duration, diskPath string) *CRLCache {
	cache := lru.New(5000)
	c := &CRLCache{
		revoked:  make(map[string]int64),
		ttl:      ttl,
		lru:      cache,
		diskPath: diskPath,
	}
	// [SAFETY] Load from disk if path is provided
	if diskPath != "" {
		c.loadFromDisk()
	}
	return c
}

// MarkRevoked يسجّل DID كملغى
func (c *CRLCache) MarkRevoked(did string, revokedAt int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// [SAFETY] Use LRU cache for better eviction
	c.lru.Add(did, revokedAt)
	c.revoked[did] = revokedAt
	// [SAFETY] Save to disk if path is configured
	if c.diskPath != "" {
		c.saveToDisk()
	}
}

// IsRevoked يتحقق محلياً من الإلغاء
func (c *CRLCache) IsRevoked(did string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// [SAFETY] Check LRU cache first
	if _, ok := c.lru.Get(did); ok {
		return true
	}
	_, ok := c.revoked[did]
	return ok
}

// Clear يمسح الذاكرة
func (c *CRLCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.revoked = make(map[string]int64)
	// [SAFETY] Clear LRU cache by creating new instance
	c.lru = lru.New(5000)
	// [SAFETY] Clear disk persistence
	if c.diskPath != "" {
		c.saveToDisk()
	}
}

// [SAFETY] saveToDisk يحفظ CRL إلى القرص
func (c *CRLCache) saveToDisk() error {
	if c.diskPath == "" {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := json.Marshal(c.revoked)
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(c.diskPath), 0700); err != nil {
		return err
	}

	// [SAFETY] Use atomic write to prevent corruption
	tmpPath := c.diskPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmpPath, c.diskPath)
}

// [SAFETY] loadFromDisk يحمّل CRL من القرص
func (c *CRLCache) loadFromDisk() error {
	if c.diskPath == "" {
		return nil
	}

	data, err := os.ReadFile(c.diskPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, that's okay
		}
		return err
	}

	var revoked map[string]int64
	if err := json.Unmarshal(data, &revoked); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.revoked = revoked

	// Populate LRU cache
	for did, revokedAt := range revoked {
		c.lru.Add(did, revokedAt)
	}

	return nil
}
