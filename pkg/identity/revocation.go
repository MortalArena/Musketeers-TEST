package identity

import (
	"crypto/ed25519"
	"fmt"
	"strconv"
	"sync"
	"time"

	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
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
	return "/nr/revoke/" + rec.DID
}

// CRLCache ذاكرة تخزين مؤقت لسجلات الإلغاء
type CRLCache struct {
	mu      sync.RWMutex
	revoked map[string]int64 // DID -> RevokedAt
	ttl     time.Duration
}

// NewCRLCache ينشئ CRL cache
func NewCRLCache(ttl time.Duration) *CRLCache {
	return &CRLCache{
		revoked: make(map[string]int64),
		ttl:     ttl,
	}
}

// MarkRevoked يسجّل DID كملغى
func (c *CRLCache) MarkRevoked(did string, revokedAt int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.revoked) >= 5000 {
		for k := range c.revoked {
			delete(c.revoked, k)
			if len(c.revoked) < 4500 {
				break
			}
		}
	}
	c.revoked[did] = revokedAt
}

// IsRevoked يتحقق محلياً من الإلغاء
func (c *CRLCache) IsRevoked(did string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.revoked[did]
	return ok
}

// Clear يمسح الذاكرة
func (c *CRLCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.revoked = make(map[string]int64)
}
