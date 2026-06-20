package providers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/scrypt"
)

// APIKeyManager manages API keys securely
type APIKeyManager struct {
	keys     map[ProviderType]string
	filePath string
	mu       sync.RWMutex
}

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager(filePath string) (*APIKeyManager, error) {
	manager := &APIKeyManager{
		keys:     make(map[ProviderType]string),
		filePath: filePath,
	}

	if err := manager.load(); err != nil {
		return nil, fmt.Errorf("failed to load API keys: %w", err)
	}

	return manager, nil
}

// SetKey sets an API key for a provider
func (m *APIKeyManager) SetKey(provider ProviderType, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.keys[provider] = key

	return m.save()
}

// GetKey gets an API key for a provider
func (m *APIKeyManager) GetKey(provider ProviderType) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key, exists := m.keys[provider]
	return key, exists
}

// DeleteKey deletes an API key for a provider
func (m *APIKeyManager) DeleteKey(provider ProviderType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.keys, provider)

	return m.save()
}

// ListProviders returns all providers that have keys
func (m *APIKeyManager) ListProviders() []ProviderType {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providers := make([]ProviderType, 0, len(m.keys))
	for provider := range m.keys {
		providers = append(providers, provider)
	}
	return providers
}

// HasKey checks if a provider has an API key
func (m *APIKeyManager) HasKey(provider ProviderType) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.keys[provider]
	return exists
}

// ClearAll clears all API keys
func (m *APIKeyManager) ClearAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.keys = make(map[ProviderType]string)

	return m.save()
}

// load loads API keys from file
func (m *APIKeyManager) load() error {
	if m.filePath == "" {
		return nil
	}

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// [SAFETY] Get passphrase from environment variable
	passphrase := os.Getenv("MUSKETEERS_VAULT_PASSPHRASE")
	if passphrase == "" {
		// [FALLBACK] If no passphrase, use insecure base64 method (backward compatibility)
		decoded, err := base64.StdEncoding.DecodeString(string(data))
		if err != nil {
			return fmt.Errorf("failed to decode API keys: %w", err)
		}

		var keys map[string]string
		if err := json.Unmarshal(decoded, &keys); err != nil {
			return fmt.Errorf("failed to unmarshal API keys: %w", err)
		}

		m.keys = make(map[ProviderType]string)
		for provider, key := range keys {
			m.keys[ProviderType(provider)] = key
		}
		return nil
	}

	// [SAFETY] Decrypt using AES-256-GCM
	decrypted, err := m.decryptAPIKeys(data, passphrase)
	if err != nil {
		return fmt.Errorf("failed to decrypt API keys: %w", err)
	}

	var keys map[string]string
	if err := json.Unmarshal(decrypted, &keys); err != nil {
		return fmt.Errorf("failed to unmarshal API keys: %w", err)
	}

	m.keys = make(map[ProviderType]string)
	for provider, key := range keys {
		m.keys[ProviderType(provider)] = key
	}

	return nil
}

// save saves API keys to file
func (m *APIKeyManager) save() error {
	if m.filePath == "" {
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Convert to string map
	keys := make(map[string]string)
	for provider, key := range m.keys {
		keys[string(provider)] = key
	}

	// Marshal JSON
	data, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal API keys: %w", err)
	}

	// [SAFETY] Get passphrase from environment variable
	passphrase := os.Getenv("MUSKETEERS_VAULT_PASSPHRASE")
	if passphrase == "" {
		// [FALLBACK] If no passphrase, use insecure base64 method (backward compatibility)
		encoded := base64.StdEncoding.EncodeToString(data)
		if err := os.WriteFile(m.filePath, []byte(encoded), 0600); err != nil {
			return fmt.Errorf("failed to write API keys: %w", err)
		}
		return nil
	}

	// [SAFETY] Encrypt using AES-256-GCM
	encrypted, err := m.encryptAPIKeys(data, passphrase)
	if err != nil {
		return fmt.Errorf("failed to encrypt API keys: %w", err)
	}

	// Write to file
	if err := os.WriteFile(m.filePath, encrypted, 0600); err != nil {
		return fmt.Errorf("failed to write API keys: %w", err)
	}

	return nil
}

// GetDefaultKeyFilePath returns the default file path for API keys
func GetDefaultKeyFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".musketeers", "api_keys.json"), nil
}

// Environment variable names for API keys
const (
	EnvOpenAI     = "OPENAI_API_KEY"
	EnvAnthropic  = "ANTHROPIC_API_KEY"
	EnvGoogle     = "GOOGLE_API_KEY"
	EnvDeepSeek   = "DEEPSEEK_API_KEY"
	EnvXAI        = "XAI_API_KEY"
	EnvMistral    = "MISTRAL_API_KEY"
	EnvQwen       = "QWEN_API_KEY"
	EnvMoonshot   = "MOONSHOT_API_KEY"
	EnvNVIDIA     = "NVIDIA_API_KEY"
	EnvXiaomi     = "XIAOMI_API_KEY"
	EnvZAI        = "ZAI_API_KEY"
	EnvTencent    = "TENCENT_API_KEY"
	EnvStepFun    = "STEPFUN_API_KEY"
	EnvPoolside   = "POOLSIDE_API_KEY"
	EnvRecraft    = "RECRAFT_API_KEY"
	EnvSourceful  = "SOURCEFUL_API_KEY"
	EnvOpenRouter = "OPENROUTER_API_KEY"
	EnvCohere     = "COHERE_API_KEY"
	EnvGroq       = "GROQ_API_KEY"
	EnvTogetherAI = "TOGETHERAI_API_KEY"
	EnvPerplexity = "PERPLEXITY_API_KEY"
)

// GetEnvVarName returns the environment variable name for a provider
func GetEnvVarName(provider ProviderType) string {
	switch provider {
	case ProviderOpenAI:
		return EnvOpenAI
	case ProviderAnthropic:
		return EnvAnthropic
	case ProviderGoogle:
		return EnvGoogle
	case ProviderDeepSeek:
		return EnvDeepSeek
	case ProviderXAI:
		return EnvXAI
	case ProviderMistral:
		return EnvMistral
	case ProviderQwen:
		return EnvQwen
	case ProviderMoonshot:
		return EnvMoonshot
	case ProviderNVIDIA:
		return EnvNVIDIA
	case ProviderXiaomi:
		return EnvXiaomi
	case ProviderZAI:
		return EnvZAI
	case ProviderTencent:
		return EnvTencent
	case ProviderStepFun:
		return EnvStepFun
	case ProviderPoolside:
		return EnvPoolside
	case ProviderRecraft:
		return EnvRecraft
	case ProviderSourceful:
		return EnvSourceful
	case ProviderOpenRouter:
		return EnvOpenRouter
	case ProviderCohere:
		return EnvCohere
	case ProviderGroq:
		return EnvGroq
	case ProviderTogetherAI:
		return EnvTogetherAI
	case ProviderPerplexity:
		return EnvPerplexity
	default:
		return ""
	}
}

// LoadFromEnv loads API keys from environment variables
func (m *APIKeyManager) LoadFromEnv() {
	providers := []ProviderType{
		ProviderOpenAI, ProviderAnthropic, ProviderGoogle, ProviderDeepSeek,
		ProviderXAI, ProviderMistral, ProviderQwen, ProviderMoonshot,
		ProviderNVIDIA, ProviderXiaomi, ProviderZAI, ProviderTencent,
		ProviderStepFun, ProviderPoolside, ProviderRecraft, ProviderSourceful,
		ProviderOpenRouter, ProviderCohere, ProviderGroq, ProviderTogetherAI,
		ProviderPerplexity,
	}

	for _, provider := range providers {
		envVar := GetEnvVarName(provider)
		if key := os.Getenv(envVar); key != "" {
			m.SetKey(provider, key)
		}
	}
}

// [SAFETY] encryptAPIKeys encrypts API keys using scrypt + AES-256-GCM
func (m *APIKeyManager) encryptAPIKeys(data []byte, passphrase string) ([]byte, error) {
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
	ciphertext := gcm.Seal(nil, nonce, data, nil)

	// Return salt + nonce + ciphertext
	result := append(salt, nonce...)
	result = append(result, ciphertext...)
	return result, nil
}

// [SAFETY] decryptAPIKeys decrypts API keys using scrypt + AES-256-GCM
func (m *APIKeyManager) decryptAPIKeys(encrypted []byte, passphrase string) ([]byte, error) {
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
	data, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return data, nil
}
