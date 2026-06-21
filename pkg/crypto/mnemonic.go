package crypto

import (
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/text/unicode/norm"
)

const hkdfInfo = "Musketeers-ed25519-seed"

// NormalizePassphrase applies NFKD to passphrase (BIP39 standard)
func NormalizePassphrase(passphrase string) string {
	return norm.NFKD.String(passphrase)
}

// GenerateMnemonic generates BIP39 mnemonic phrase of 24 words (256 bits entropy)
func GenerateMnemonic() (string, error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", fmt.Errorf("failed to generate entropy: %w", err)
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("failed to generate mnemonic: %w", err)
	}
	return mnemonic, nil
}

// SeedFromMnemonic derives seed from mnemonic with optional passphrase
func SeedFromMnemonic(mnemonic, passphrase string) ([]byte, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, fmt.Errorf("invalid mnemonic")
	}
	passphrase = NormalizePassphrase(passphrase)
	seed := bip39.NewSeed(mnemonic, passphrase)
	return seed, nil
}

// DeriveEd25519Key derives Ed25519 key from seed via HKDF-SHA3-256
func DeriveEd25519Key(seed []byte) (ed25519.PrivateKey, error) {
	if len(seed) < 16 {
		return nil, fmt.Errorf("seed too short")
	}
	// standard HKDF using SHA-256
	kdf := hkdf.New(sha256.New, seed, nil, []byte(hkdfInfo))
	key := make([]byte, ed25519.SeedSize)
	if _, err := io.ReadFull(kdf, key); err != nil {
		return nil, err
	}
	return ed25519.NewKeyFromSeed(key), nil
}

// IdentityFromMnemonic recovers identity from mnemonic
func IdentityFromMnemonic(mnemonic, passphrase string) (ed25519.PrivateKey, error) {
	seed, err := SeedFromMnemonic(mnemonic, passphrase)
	if err != nil {
		return nil, err
	}
	return DeriveEd25519Key(seed)
}

// ValidateMnemonicWords validates mnemonic phrase
func ValidateMnemonicWords(mnemonic string) bool {
	if !utf8.ValidString(mnemonic) {
		return false
	}
	return bip39.IsMnemonicValid(mnemonic)
}
