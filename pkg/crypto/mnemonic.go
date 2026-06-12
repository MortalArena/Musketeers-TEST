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

// NormalizePassphrase يطبّق NFKD على عبارة المرور (معيار BIP39)
func NormalizePassphrase(passphrase string) string {
	return norm.NFKD.String(passphrase)
}

// GenerateMnemonic يولّد عبارة تذكيرية BIP39 من 24 كلمة (256 بت إنتروبيا)
func GenerateMnemonic() (string, error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", fmt.Errorf("فشل توليد الإنتروبيا: %w", err)
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("فشل توليد العبارة التذكيرية: %w", err)
	}
	return mnemonic, nil
}

// SeedFromMnemonic يشتق seed من العبارة التذكيرية مع passphrase اختيارية
func SeedFromMnemonic(mnemonic, passphrase string) ([]byte, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, fmt.Errorf("عبارة تذكيرية غير صالحة")
	}
	passphrase = NormalizePassphrase(passphrase)
	seed := bip39.NewSeed(mnemonic, passphrase)
	return seed, nil
}

// DeriveEd25519Key يشتق مفتاح Ed25519 من seed عبر HKDF-SHA3-256
func DeriveEd25519Key(seed []byte) (ed25519.PrivateKey, error) {
	if len(seed) < 16 {
		return nil, fmt.Errorf("seed قصير جداً")
	}
	// standard HKDF using SHA-256
	kdf := hkdf.New(sha256.New, seed, nil, []byte(hkdfInfo))
	key := make([]byte, ed25519.SeedSize)
	if _, err := io.ReadFull(kdf, key); err != nil {
		return nil, err
	}
	return ed25519.NewKeyFromSeed(key), nil
}

// IdentityFromMnemonic يستعيد الهوية من العبارة التذكيرية
func IdentityFromMnemonic(mnemonic, passphrase string) (ed25519.PrivateKey, error) {
	seed, err := SeedFromMnemonic(mnemonic, passphrase)
	if err != nil {
		return nil, err
	}
	return DeriveEd25519Key(seed)
}

// ValidateMnemonicWords يتحقق من صحة العبارة التذكيرية
func ValidateMnemonicWords(mnemonic string) bool {
	if !utf8.ValidString(mnemonic) {
		return false
	}
	return bip39.IsMnemonicValid(mnemonic)
}
