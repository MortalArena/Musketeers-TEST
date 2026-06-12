package vault

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/vault/encryption"
	"github.com/MortalArena/Musketeers/pkg/vault/keyprovider"
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
