package identity

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
)

// IdentityRecord سجل الهوية على DHT
type IdentityRecord struct {
	DID          string   `json:"did"`
	PublicKeyHex string   `json:"public_key_hex"`
	Capabilities []string `json:"capabilities"`
	Sequence     uint64   `json:"sequence"`
	CreatedAt    int64    `json:"created_at"`
	ExpiresAt    int64    `json:"expires_at"`
	Nonce        string   `json:"nonce"` // hex — PoW nonce
	Signature    string   `json:"signature"`
}

// IdentityPayload يبني payload ثابت للتوقيع (canonical form)
func IdentityPayload(rec *IdentityRecord) string {
	caps := strings.Join(rec.Capabilities, ",")
	return strings.Join([]string{
		rec.DID,
		rec.PublicKeyHex,
		caps,
		strconv.FormatUint(rec.Sequence, 10),
		strconv.FormatInt(rec.CreatedAt, 10),
		strconv.FormatInt(rec.ExpiresAt, 10),
		rec.Nonce,
	}, "|")
}

// NewIdentityRecord ينشئ سجل هوية جديد مع PoW
func NewIdentityRecord(ctx context.Context, kp *nrcrypto.KeyPair, capabilities []string, ttlSeconds int64) (*IdentityRecord, error) {
	now := time.Now().Unix()
	nonce, err := nrcrypto.MinePow(ctx, kp.DID, nrcrypto.PowDifficulty())
	if err != nil {
		return nil, fmt.Errorf("فشل PoW: %w", err)
	}

	rec := &IdentityRecord{
		DID:          kp.DID,
		PublicKeyHex: nrcrypto.PublicKeyHex(kp.Public),
		Capabilities: capabilities,
		Sequence:     1,
		CreatedAt:    now,
		ExpiresAt:    now + ttlSeconds,
		Nonce:        hex.EncodeToString(nonce),
	}

	payload := IdentityPayload(rec)
	sig, err := nrcrypto.SignPayloadHex(kp.Private, nrcrypto.DomainIdentity, payload)
	if err != nil {
		return nil, err
	}
	rec.Signature = sig
	return rec, nil
}

// Verify يتحقق من صحة سجل الهوية
func (rec *IdentityRecord) Verify() error {
	if rec.DID == "" || rec.PublicKeyHex == "" || rec.Signature == "" {
		return fmt.Errorf("حقول مطلوبة ناقصة")
	}

	pub, err := nrcrypto.PubKeyFromHex(rec.PublicKeyHex)
	if err != nil {
		return err
	}

	// التحقق من تطابق DID
	if nrcrypto.DIDFromPublicKey(pub) != rec.DID {
		return fmt.Errorf("DID لا يطابق المفتاح العام")
	}

	// التحقق من PoW
	nonceBytes, err := hex.DecodeString(rec.Nonce)
	if err != nil {
		return fmt.Errorf("nonce غير صالح: %w", err)
	}
	if !nrcrypto.VerifyPow(rec.DID, nonceBytes, nrcrypto.PowDifficulty()) {
		return fmt.Errorf("PoW غير صالح")
	}

	// التحقق من التوقيع
	payload := IdentityPayload(rec)
	if err := nrcrypto.VerifyPayloadHex(pub, nrcrypto.DomainIdentity, payload, rec.Signature); err != nil {
		return err
	}

	// التحقق من الصلاحية
	if time.Now().Unix() > rec.ExpiresAt {
		return fmt.Errorf("الهوية منتهية الصلاحية")
	}

	return nil
}

// UpdateSequence يحدّث السجل بزيادة Sequence
func (rec *IdentityRecord) UpdateSequence(ctx context.Context, priv ed25519.PrivateKey, capabilities []string, ttlSeconds int64) (*IdentityRecord, error) {
	now := time.Now().Unix()
	nonce, err := nrcrypto.MinePow(ctx, rec.DID, nrcrypto.PowDifficulty())
	if err != nil {
		return nil, err
	}

	updated := &IdentityRecord{
		DID:          rec.DID,
		PublicKeyHex: rec.PublicKeyHex,
		Capabilities: capabilities,
		Sequence:     rec.Sequence + 1,
		CreatedAt:    now,
		ExpiresAt:    now + ttlSeconds,
		Nonce:        hex.EncodeToString(nonce),
	}

	payload := IdentityPayload(updated)
	sig, err := nrcrypto.SignPayloadHex(priv, nrcrypto.DomainIdentity, payload)
	if err != nil {
		return nil, err
	}
	updated.Signature = sig
	return updated, nil
}

// PublicKey يستخرج المفتاح العام
func (rec *IdentityRecord) PublicKey() (ed25519.PublicKey, error) {
	return nrcrypto.PubKeyFromHex(rec.PublicKeyHex)
}

// DHTKey مفتاح DHT للهوية
func (rec *IdentityRecord) DHTKey() string {
	return "/nr/identity/" + rec.DID
}
