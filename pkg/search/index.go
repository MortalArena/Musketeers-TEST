package search

import (
	"crypto/ed25519"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
	"github.com/MortalArena/Musketeers/pkg/protocol"
)

// IndexEntry إعلان بحث موزع
type IndexEntry struct {
	DID          string `json:"did"`
	PeerID       string `json:"peer_id"`
	Keyword      string `json:"keyword"`
	Meta         string `json:"meta"`
	Expires      int64  `json:"expires"`
	PublicKeyHex string `json:"public_key_hex,omitempty"`
	Sig          string `json:"sig"`
}

// IndexPayload payload ثابت للتوقيع
func IndexPayload(entry *IndexEntry) string {
	return strings.Join([]string{
		entry.DID,
		entry.PeerID,
		entry.Keyword,
		strconv.FormatInt(entry.Expires, 10),
	}, "|")
}

// NewIndexEntry ينشئ إعلان بحث موقّع
func NewIndexEntry(did, peerID, keyword, meta string, ttlSeconds int64, priv ed25519.PrivateKey) (*IndexEntry, error) {
	if len(meta) > protocol.MaxMetaSize {
		return nil, fmt.Errorf("Meta يتجاوز الحد الأقصى (%d)", protocol.MaxMetaSize)
	}
	entry := &IndexEntry{
		DID:          did,
		PeerID:       peerID,
		Keyword:      keyword,
		Meta:         meta,
		Expires:      time.Now().Unix() + ttlSeconds,
		PublicKeyHex: nrcrypto.PublicKeyHex(priv.Public().(ed25519.PublicKey)),
	}
	payload := IndexPayload(entry)
	sig, err := nrcrypto.SignPayloadHex(priv, nrcrypto.DomainSearch, payload)
	if err != nil {
		return nil, err
	}
	entry.Sig = sig
	return entry, nil
}

// Verify يتحقق من إعلان البحث
func (e *IndexEntry) Verify(pub ed25519.PublicKey) error {
	if e.DID == "" || e.PeerID == "" || e.Keyword == "" || e.Sig == "" {
		return fmt.Errorf("حقول مطلوبة ناقصة")
	}
	if nrcrypto.DIDFromPublicKey(pub) != e.DID {
		return fmt.Errorf("DID لا يطابق المفتاح العام")
	}
	if time.Now().Unix() > e.Expires {
		return fmt.Errorf("الإعلان منتهي الصلاحية")
	}
	payload := IndexPayload(e)
	return nrcrypto.VerifyPayloadHex(pub, nrcrypto.DomainSearch, payload, e.Sig)
}

// DHTKey مفتاح DHT
func (e *IndexEntry) DHTKey() string {
	return "/nr/search/" + e.Keyword + "/" + e.DID
}

// TokenBucket rate limiter لكل PeerID
type TokenBucket struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64 // tokens per second
	capacity float64
}

type bucket struct {
	tokens   float64
	lastTime time.Time
}

// NewTokenBucket ينشئ token bucket limiter
func NewTokenBucket(ratePerMinute, capacity int) *TokenBucket {
	return &TokenBucket{
		buckets:  make(map[string]*bucket),
		rate:     float64(ratePerMinute) / 60.0,
		capacity: float64(capacity),
	}
}

// Allow يتحقق إن كان مسموحاً للـ peer
func (tb *TokenBucket) Allow(peerID string) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	b, ok := tb.buckets[peerID]
	if !ok {
		b = &bucket{tokens: tb.capacity, lastTime: now}
		tb.buckets[peerID] = b
	}

	elapsed := now.Sub(b.lastTime).Seconds()
	b.tokens += elapsed * tb.rate
	if b.tokens > tb.capacity {
		b.tokens = tb.capacity
	}
	b.lastTime = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Cleanup يزيل buckets قديمة
func (tb *TokenBucket) Cleanup(maxAge time.Duration) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	cutoff := time.Now().Add(-maxAge)
	for id, b := range tb.buckets {
		if b.lastTime.Before(cutoff) {
			delete(tb.buckets, id)
		}
	}
}
