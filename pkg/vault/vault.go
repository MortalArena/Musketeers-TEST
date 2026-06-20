package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/vault/encryption"
	"github.com/MortalArena/Musketeers/pkg/vault/keyprovider"
	"golang.org/x/crypto/scrypt"
)

type Vault struct {
	mu       sync.RWMutex
	provider keyprovider.KeyProvider
}

type Secret struct {
	Name      string            `json:"name"`
	Value     []byte            `json:"value"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt int64             `json:"created_at"`
	UpdatedAt int64             `json:"updated_at"`
}

func New(provider keyprovider.KeyProvider) *Vault {
	return &Vault{provider: provider}
}

func (v *Vault) Store(name string, value []byte, metadata map[string]string) error {
	if err := validateSecretName(name); err != nil {
		return err
	}
	if v.provider == nil {
		return fmt.Errorf("key provider is nil")
	}
	masterKey, err := v.ensureMasterKey()
	if err != nil {
		return err
	}
	ciphertext, err := encryption.Encrypt(value, masterKey)
	if err != nil {
		return err
	}
	now := timeNow()
	secret := Secret{Name: name, Value: ciphertext, Metadata: metadata, CreatedAt: now, UpdatedAt: now}
	data, err := json.Marshal(secret)
	if err != nil {
		return err
	}
	return v.provider.Store(secretStoreName(name), data)
}

func (v *Vault) Retrieve(name string) ([]byte, error) {
	if err := validateSecretName(name); err != nil {
		return nil, err
	}
	if v.provider == nil {
		return nil, fmt.Errorf("key provider is nil")
	}
	masterKey, err := v.provider.Load("master")
	if err != nil {
		return nil, err
	}
	data, err := v.provider.Load(secretStoreName(name))
	if err != nil {
		return nil, err
	}
	var secret Secret
	if err := json.Unmarshal(data, &secret); err != nil {
		return nil, err
	}
	return encryption.Decrypt(secret.Value, masterKey)
}

func (v *Vault) Delete(name string) error {
	if err := validateSecretName(name); err != nil {
		return err
	}
	if v.provider == nil {
		return fmt.Errorf("key provider is nil")
	}
	return v.provider.Delete(secretStoreName(name))
}

func (v *Vault) List() ([]string, error) {
	if v.provider == nil {
		return nil, fmt.Errorf("key provider is nil")
	}
	names, err := v.provider.List()
	if err != nil {
		return nil, err
	}
	secrets := make([]string, 0)
	for _, encoded := range names {
		name, ok := secretName(encoded)
		if ok {
			secrets = append(secrets, name)
		}
	}
	sort.Strings(secrets)
	return secrets, nil
}

func (v *Vault) ensureMasterKey() ([]byte, error) {
	// [SAFETY] Use mutex to prevent race conditions
	v.mu.Lock()
	defer v.mu.Unlock()

	// [SAFETY] Get passphrase from environment variable
	passphrase := os.Getenv("MUSKETEERS_VAULT_PASSPHRASE")
	if passphrase == "" {
		// [FALLBACK] If no passphrase, use insecure method (backward compatibility)
		// This is NOT recommended for production
		masterKey, err := v.provider.Load("master")
		if err == nil {
			return masterKey, nil
		}
		masterKey = make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, masterKey); err != nil {
			return nil, err
		}
		if err := v.provider.Store("master", masterKey); err != nil {
			return nil, err
		}
		return masterKey, nil
	}

	// [SAFETY] Try to load encrypted master key
	encryptedMasterKey, err := v.provider.Load("master_encrypted")
	if err == nil {
		// Decrypt the master key
		masterKey, err := v.decryptMasterKey(encryptedMasterKey, passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt master key: %w", err)
		}
		return masterKey, nil
	}

	// [SAFETY] Generate new master key
	masterKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, masterKey); err != nil {
		return nil, err
	}

	// [SAFETY] Encrypt master key with passphrase
	encrypted, err := v.encryptMasterKey(masterKey, passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt master key: %w", err)
	}

	// [SAFETY] Store encrypted master key
	if err := v.provider.Store("master_encrypted", encrypted); err != nil {
		return nil, err
	}

	return masterKey, nil
}

// [SAFETY] encryptMasterKey encrypts the master key using scrypt + AES-256-GCM
func (v *Vault) encryptMasterKey(masterKey []byte, passphrase string) ([]byte, error) {
	// Generate salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	// Derive key using scrypt (N=131072, r=8, p=1, keylen=32)
	derivedKey, err := scrypt.Key([]byte(passphrase), salt, 131072, 8, 1, 32)
	if err != nil {
		return nil, err
	}

	// Create AES cipher
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, err
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// Encrypt
	ciphertext := gcm.Seal(nil, nonce, masterKey, nil)

	// Return salt + nonce + ciphertext
	result := append(salt, nonce...)
	result = append(result, ciphertext...)
	return result, nil
}

// [SAFETY] decryptMasterKey decrypts the master key using scrypt + AES-256-GCM
func (v *Vault) decryptMasterKey(encrypted []byte, passphrase string) ([]byte, error) {
	if len(encrypted) < 16+12 { // salt(16) + nonce(12) minimum
		return nil, fmt.Errorf("invalid encrypted data length")
	}

	// Extract salt
	salt := encrypted[:16]

	// Extract nonce
	nonceStart := 16
	nonceEnd := 16 + 12
	nonce := encrypted[nonceStart:nonceEnd]

	// Extract ciphertext
	ciphertext := encrypted[nonceEnd:]

	// Derive key using scrypt
	derivedKey, err := scrypt.Key([]byte(passphrase), salt, 131072, 8, 1, 32)
	if err != nil {
		return nil, err
	}

	// Create AES cipher
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, err
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Decrypt
	masterKey, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return masterKey, nil
}

func secretStoreName(name string) string {
	return "secret_" + hex.EncodeToString([]byte(name))
}

func secretName(encoded string) (string, bool) {
	if !strings.HasPrefix(encoded, "secret_") {
		return "", false
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(encoded, "secret_"))
	if err != nil {
		return "", false
	}
	return string(decoded), true
}

func validateSecretName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("secret name is required")
	}
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return fmt.Errorf("invalid secret name: %s", name)
	}
	return nil
}
