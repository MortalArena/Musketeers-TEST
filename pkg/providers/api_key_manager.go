package providers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
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
	
	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return fmt.Errorf("failed to decode API keys: %w", err)
	}
	
	// Unmarshal JSON
	var keys map[string]string
	if err := json.Unmarshal(decoded, &keys); err != nil {
		return fmt.Errorf("failed to unmarshal API keys: %w", err)
	}
	
	// Convert to ProviderType map
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
	
	// Encode base64
	encoded := base64.StdEncoding.EncodeToString(data)
	
	// Write to file
	if err := os.WriteFile(m.filePath, []byte(encoded), 0600); err != nil {
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
	EnvOpenAI      = "OPENAI_API_KEY"
	EnvAnthropic   = "ANTHROPIC_API_KEY"
	EnvGoogle      = "GOOGLE_API_KEY"
	EnvDeepSeek    = "DEEPSEEK_API_KEY"
	EnvXAI         = "XAI_API_KEY"
	EnvMistral     = "MISTRAL_API_KEY"
	EnvQwen        = "QWEN_API_KEY"
	EnvMoonshot    = "MOONSHOT_API_KEY"
	EnvNVIDIA      = "NVIDIA_API_KEY"
	EnvXiaomi      = "XIAOMI_API_KEY"
	EnvZAI         = "ZAI_API_KEY"
	EnvTencent     = "TENCENT_API_KEY"
	EnvStepFun     = "STEPFUN_API_KEY"
	EnvPoolside    = "POOLSIDE_API_KEY"
	EnvRecraft     = "RECRAFT_API_KEY"
	EnvSourceful   = "SOURCEFUL_API_KEY"
	EnvOpenRouter  = "OPENROUTER_API_KEY"
	EnvCohere      = "COHERE_API_KEY"
	EnvGroq        = "GROQ_API_KEY"
	EnvTogetherAI  = "TOGETHERAI_API_KEY"
	EnvPerplexity  = "PERPLEXITY_API_KEY"
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
