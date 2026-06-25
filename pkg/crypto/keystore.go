package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/argon2"
)

const (
	keystoreVersion = 1
	// [SAFETY] Argon2id parameters (RFC 9106 recommended)
	argon2Time    = 3
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Threads = 4
	argon2KeyLen  = 32
	saltLen       = 32
)

// KeystoreFile encrypted key file format
type KeystoreFile struct {
	Version    int    `json:"version"`
	DID        string `json:"did"`
	Salt       string `json:"salt"`               // hex
	Nonce      string `json:"nonce"`              // hex — AES-GCM nonce
	Ciphertext string `json:"ciphertext"`         // hex — encrypted private key
	Mnemonic   string `json:"mnemonic,omitempty"` // encrypted inside ciphertext if present
}

// keystorePlaintext data before encryption
type keystorePlaintext struct {
	PrivateKey string `json:"private_key"` // hex
	Mnemonic   string `json:"mnemonic,omitempty"`
}

// SaveKeystore saves private key encrypted to disk
func SaveKeystore(path, passphrase string, kp *KeyPair, mnemonic string) error {
	// [SAFETY] Validate passphrase strength
	if len(passphrase) < 8 {
		return fmt.Errorf("passphrase must be at least 8 characters")
	}

	plain := keystorePlaintext{
		PrivateKey: hex.EncodeToString(kp.Private),
		Mnemonic:   mnemonic,
	}
	plainJSON, err := json.Marshal(plain)
	if err != nil {
		return err
	}

	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return err
	}

	passphrase = NormalizePassphrase(passphrase)
	// [SAFETY] Use argon2id instead of scrypt for better security
	derived := argon2.IDKey([]byte(passphrase), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	block, err := aes.NewCipher(derived)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}

	ciphertext := gcm.Seal(nil, nonce, plainJSON, []byte("Musketeers-keystore-v1"))

	ks := KeystoreFile{
		Version:    keystoreVersion,
		DID:        kp.DID,
		Salt:       hex.EncodeToString(salt),
		Nonce:      hex.EncodeToString(nonce),
		Ciphertext: hex.EncodeToString(ciphertext),
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(ks, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// LoadKeystore loads private key from encrypted file
func LoadKeystore(path, passphrase string) (*KeyPair, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read keystore: %w", err)
	}

	var ks KeystoreFile
	if err := json.Unmarshal(data, &ks); err != nil {
		return nil, "", err
	}
	if ks.Version != keystoreVersion {
		return nil, "", fmt.Errorf("unsupported keystore version: %d", ks.Version)
	}

	salt, err := hex.DecodeString(ks.Salt)
	if err != nil {
		return nil, "", err
	}
	nonce, err := hex.DecodeString(ks.Nonce)
	if err != nil {
		return nil, "", err
	}
	ciphertext, err := hex.DecodeString(ks.Ciphertext)
	if err != nil {
		return nil, "", err
	}

	passphrase = NormalizePassphrase(passphrase)
	// [SAFETY] Use argon2id instead of scrypt for better security
	derived := argon2.IDKey([]byte(passphrase), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	block, err := aes.NewCipher(derived)
	if err != nil {
		return nil, "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, "", err
	}

	plainJSON, err := gcm.Open(nil, nonce, ciphertext, []byte("Musketeers-keystore-v1"))
	if err != nil {
		return nil, "", fmt.Errorf("incorrect passphrase or corrupted file")
	}

	var plain keystorePlaintext
	if err := json.Unmarshal(plainJSON, &plain); err != nil {
		return nil, "", err
	}

	privBytes, err := hex.DecodeString(plain.PrivateKey)
	if err != nil || len(privBytes) != ed25519.PrivateKeySize {
		return nil, "", fmt.Errorf("corrupted private key in keystore")
	}

	kp := KeyPairFromPrivate(ed25519.PrivateKey(privBytes))
	if ks.DID != "" && kp.DID != ks.DID {
		return nil, "", fmt.Errorf("DID does not match file")
	}
	return kp, plain.Mnemonic, nil
}

// KeystorePath builds default keystore path
func KeystorePath(dataDir string) string {
	return filepath.Join(dataDir, "identity.key")
}

// KeystoreExists checks if keystore exists
func KeystoreExists(dataDir string) bool {
	_, err := os.Stat(KeystorePath(dataDir))
	return err == nil
}
