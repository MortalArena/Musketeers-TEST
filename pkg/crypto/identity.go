package crypto

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"os"
	"strconv"

	"github.com/mr-tron/base58"
)

const (
	// ❌ حذف: DefaultPowDifficulty = 18
	// ✅ استخدام: DefaultPowDifficulty من pow.go

	DefaultIdentityTTL = 365 * 24 * 3600 // سنة واحدة بالثواني
)

// KeyPair زوج مفاتيح Ed25519 مع DID
type KeyPair struct {
	Private ed25519.PrivateKey
	Public  ed25519.PublicKey
	DID     string
}

// GenerateKeyPair يولّد زوج مفاتيح جديد مع DID
func GenerateKeyPair() (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("فشل توليد المفاتيح: %w", err)
	}
	return &KeyPair{
		Private: priv,
		Public:  pub,
		DID:     DIDFromPublicKey(pub),
	}, nil
}

// KeyPairFromPrivate ينشئ KeyPair من مفتاح خاص
func KeyPairFromPrivate(priv ed25519.PrivateKey) *KeyPair {
	pub := priv.Public().(ed25519.PublicKey)
	return &KeyPair{
		Private: priv,
		Public:  pub,
		DID:     DIDFromPublicKey(pub),
	}
}

// DIDFromPublicKey يحسب DID من المفتاح العام: did:mskt:<base58(sha256(pub)[:16])>
func DIDFromPublicKey(pub ed25519.PublicKey) string {
	h := sha256.Sum256(pub)
	return "did:mskt:" + base58.Encode(h[:16])
}

// PowDifficulty يقرأ صعوبة PoW من البيئة
func PowDifficulty() int {
	if v := os.Getenv("NR_POW_DIFFICULTY"); v != "" {
		if d, err := strconv.Atoi(v); err == nil && d >= MinPowDifficulty && d <= MaxPowDifficulty {
			return d
		}
	}
	return DefaultPowDifficulty
}

// MinePow يعدّن PoW باستخدام scrypt — أول بايت من الناتج == 0x00
// ✅ استخدام الدالة المحسّنة من pow.go
func MinePow(ctx context.Context, did string, difficulty int) ([]byte, error) {
	result, err := MineIdentity(ctx, did, difficulty)
	if err != nil {
		return nil, err
	}
	return []byte(result.Nonce), nil
}

// VerifyPow يتحقق من صحة PoW
// ✅ استخدام الدالة المحسّنة من pow.go
func VerifyPow(did string, nonce []byte, difficulty int) bool {
	valid, err := VerifyPoW(did, string(nonce), difficulty)
	if err != nil {
		return false
	}
	return valid
}

// PublicKeyHex يرجع المفتاح العام كـ hex
func PublicKeyHex(pub ed25519.PublicKey) string {
	return fmt.Sprintf("%x", pub)
}
