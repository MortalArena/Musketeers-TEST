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
	// Removed: DefaultPowDifficulty = 18
	// Using: DefaultPowDifficulty from pow.go

	DefaultIdentityTTL = 365 * 24 * 3600 // One year in seconds
)

// KeyPair Ed25519 key pair with DID
type KeyPair struct {
	Private ed25519.PrivateKey
	Public  ed25519.PublicKey
	DID     string
}

// GenerateKeyPair generates a new key pair with DID
func GenerateKeyPair() (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate keys: %w", err)
	}
	return &KeyPair{
		Private: priv,
		Public:  pub,
		DID:     DIDFromPublicKey(pub),
	}, nil
}

// KeyPairFromPrivate creates KeyPair from private key
func KeyPairFromPrivate(priv ed25519.PrivateKey) *KeyPair {
	pub := priv.Public().(ed25519.PublicKey)
	return &KeyPair{
		Private: priv,
		Public:  pub,
		DID:     DIDFromPublicKey(pub),
	}
}

// DIDFromPublicKey calculates DID from public key: did:mskt:<base58(sha256(pub)[:16])>
func DIDFromPublicKey(pub ed25519.PublicKey) string {
	h := sha256.Sum256(pub)
	return "did:mskt:" + base58.Encode(h[:16])
}

// PowDifficulty reads PoW difficulty from environment
func PowDifficulty() int {
	if v := os.Getenv("NR_POW_DIFFICULTY"); v != "" {
		if d, err := strconv.Atoi(v); err == nil && d >= MinPowDifficulty && d <= MaxPowDifficulty {
			return d
		}
	}
	return DefaultPowDifficulty
}

// MinePow mines PoW using scrypt — first byte of output == 0x00
// Using optimized function from pow.go
func MinePow(ctx context.Context, did string, difficulty int) ([]byte, error) {
	result, err := MineIdentity(ctx, did, difficulty)
	if err != nil {
		return nil, err
	}
	return []byte(result.Nonce), nil
}

// VerifyPow verifies PoW validity
// Using optimized function from pow.go
func VerifyPow(did string, nonce []byte, difficulty int) bool {
	valid, err := VerifyPoW(did, string(nonce), difficulty)
	if err != nil {
		return false
	}
	return valid
}

// PublicKeyHex returns public key as hex
func PublicKeyHex(pub ed25519.PublicKey) string {
	return fmt.Sprintf("%x", pub)
}
