package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ModelCatalog manages model information from all providers
type ModelCatalog struct {
	models     map[string]ModelInfo
	byProvider map[ProviderType][]ModelInfo
	mu         sync.RWMutex
	filePath   string
}

// NewModelCatalog creates a new model catalog
func NewModelCatalog(filePath string) (*ModelCatalog, error) {
	catalog := &ModelCatalog{
		models:     make(map[string]ModelInfo),
		byProvider: make(map[ProviderType][]ModelInfo),
		filePath:   filePath,
	}

	if err := catalog.load(); err != nil {
		return nil, fmt.Errorf("failed to load model catalog: %w", err)
	}

	return catalog, nil
}

// GetModel gets a model by ID
func (c *ModelCatalog) GetModel(modelID string) (*ModelInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	model, exists := c.models[modelID]
	return &model, exists
}

// GetModelsByProvider gets all models for a provider
func (c *ModelCatalog) GetModelsByProvider(provider ProviderType) []ModelInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	models, exists := c.byProvider[provider]
	if !exists {
		return []ModelInfo{}
	}

	// Return a copy to avoid race conditions
	result := make([]ModelInfo, len(models))
	copy(result, models)
	return result
}

// GetModelsByCapability gets all models with a specific capability
func (c *ModelCatalog) GetModelsByCapability(capability ModelCapability) []ModelInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []ModelInfo
	for _, model := range c.models {
		for _, cap := range model.Capabilities {
			if cap == capability {
				result = append(result, model)
				break
			}
		}
	}
	return result
}

// GetModelsByCategory gets all models in a category
func (c *ModelCatalog) GetModelsByCategory(category string) []ModelInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []ModelInfo
	for _, model := range c.models {
		for _, cat := range model.Categories {
			if cat == category {
				result = append(result, model)
				break
			}
		}
	}
	return result
}

// ListAllModels lists all models
func (c *ModelCatalog) ListAllModels() []ModelInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]ModelInfo, 0, len(c.models))
	for _, model := range c.models {
		result = append(result, model)
	}
	return result
}

// SearchModels searches models by name or description
func (c *ModelCatalog) SearchModels(query string) []ModelInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []ModelInfo
	for _, model := range c.models {
		if contains(model.Name, query) || contains(model.Description, query) {
			result = append(result, model)
		}
	}
	return result
}

// GetFreeModels gets all free models
func (c *ModelCatalog) GetFreeModels() []ModelInfo {
	return c.GetModelsByCategory("free")
}

// GetLocalModels gets all local models
func (c *ModelCatalog) GetLocalModels() []ModelInfo {
	return c.GetModelsByCategory("local")
}

// AddModel adds a model to the catalog
func (c *ModelCatalog) AddModel(model ModelInfo) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.models[model.ID] = model
	c.byProvider[model.Provider] = append(c.byProvider[model.Provider], model)

	return c.save()
}

// RemoveModel removes a model from the catalog
func (c *ModelCatalog) RemoveModel(modelID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	model, exists := c.models[modelID]
	if !exists {
		return nil
	}

	delete(c.models, modelID)

	// Remove from provider list
	providerModels := c.byProvider[model.Provider]
	for i, m := range providerModels {
		if m.ID == modelID {
			c.byProvider[model.Provider] = append(providerModels[:i], providerModels[i+1:]...)
			break
		}
	}

	return c.save()
}

// Refresh reloads the catalog from file
func (c *ModelCatalog) Refresh() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.load()
}

// load loads the catalog from file
func (c *ModelCatalog) load() error {
	if c.filePath == "" {
		return nil
	}

	data, err := os.ReadFile(c.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var catalogData struct {
		Providers map[string]struct {
			Name    string `json:"name"`
			BaseURL string `json:"base_url"`
		} `json:"providers"`
		Models []ModelInfo `json:"models"`
	}

	if err := json.Unmarshal(data, &catalogData); err != nil {
		return fmt.Errorf("failed to unmarshal catalog: %w", err)
	}

	// Clear existing data
	c.models = make(map[string]ModelInfo)
	c.byProvider = make(map[ProviderType][]ModelInfo)

	// Load models
	for _, model := range catalogData.Models {
		c.models[model.ID] = model
		c.byProvider[model.Provider] = append(c.byProvider[model.Provider], model)
	}

	return nil
}

// save saves the catalog to file
func (c *ModelCatalog) save() error {
	if c.filePath == "" {
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(c.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Build catalog data
	catalogData := struct {
		Providers map[string]struct {
			Name    string `json:"name"`
			BaseURL string `json:"base_url"`
		} `json:"providers"`
		Models []ModelInfo `json:"models"`
	}{
		Providers: make(map[string]struct {
			Name    string `json:"name"`
			BaseURL string `json:"base_url"`
		}),
		Models: make([]ModelInfo, 0, len(c.models)),
	}

	// Add provider info
	for provider, models := range c.byProvider {
		if len(models) > 0 {
			catalogData.Providers[string(provider)] = struct {
				Name    string `json:"name"`
				BaseURL string `json:"base_url"`
			}{
				Name:    models[0].Owner,
				BaseURL: getProviderBaseURL(provider),
			}
		}
	}

	// Add models
	for _, model := range c.models {
		catalogData.Models = append(catalogData.Models, model)
	}

	// Marshal JSON
	data, err := json.MarshalIndent(catalogData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal catalog: %w", err)
	}

	// Write to file
	if err := os.WriteFile(c.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write catalog: %w", err)
	}

	return nil
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr))
}

// getProviderBaseURL returns the base URL for a provider
func getProviderBaseURL(provider ProviderType) string {
	switch provider {
	case ProviderOpenAI:
		return "https://api.openai.com/v1"
	case ProviderAnthropic:
		return "https://api.anthropic.com/v1"
	case ProviderGoogle:
		return "https://generativelanguage.googleapis.com/v1beta"
	case ProviderDeepSeek:
		return "https://api.deepseek.com/v1"
	case ProviderXAI:
		return "https://api.x.ai/v1"
	case ProviderMistral:
		return "https://api.mistral.ai/v1"
	case ProviderQwen:
		return "https://dashscope.aliyuncs.com/compatible-mode/v1"
	case ProviderMoonshot:
		return "https://api.moonshot.cn/v1"
	case ProviderNVIDIA:
		return "https://integrate.api.nvidia.com/v1"
	case ProviderXiaomi:
		return "https://api.mimo.chat/v1"
	case ProviderZAI:
		return "https://api.z.ai/v1"
	case ProviderTencent:
		return "https://hunyuan.cloud.tencent.com/hyllm/v1"
	case ProviderStepFun:
		return "https://api.stepfun.com/v1"
	case ProviderPoolside:
		return "https://api.poolside.ai/v1"
	case ProviderRecraft:
		return "https://api.recraft.ai/v1"
	case ProviderSourceful:
		return "https://api.sourceful.ai/v1"
	case ProviderOpenRouter:
		return "https://openrouter.ai/api/v1"
	case ProviderCohere:
		return "https://api.cohere.ai/v1"
	case ProviderGroq:
		return "https://api.groq.com/openai/v1"
	case ProviderTogetherAI:
		return "https://api.together.xyz/v1"
	case ProviderPerplexity:
		return "https://api.perplexity.ai"
	case ProviderOllama:
		return "http://localhost:11434"
	default:
		return ""
	}
}

// GetDefaultCatalogPath returns the default path for the model catalog
func GetDefaultCatalogPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".musketeers", "models.json"), nil
}
