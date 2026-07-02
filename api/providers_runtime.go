package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/MortalArena/Musketeers/pkg/providers"
)

func mapDashboardProviderType(providerType string) providers.ProviderType {
	switch strings.ToLower(strings.TrimSpace(providerType)) {
	case "openai":
		return providers.ProviderOpenAI
	case "anthropic":
		return providers.ProviderAnthropic
	case "google":
		return providers.ProviderGoogle
	case "mistral", "mistral.ai":
		return providers.ProviderMistral
	case "openrouter", "openrouter.ai":
		return providers.ProviderOpenRouter
	case "ollama":
		return providers.ProviderOllama
	case "custom":
		return providers.ProviderCustom
	case "deepseek":
		return providers.ProviderDeepSeek
	case "groq":
		return providers.ProviderGroq
	case "cohere":
		return providers.ProviderCohere
	case "togetherai":
		return providers.ProviderTogetherAI
	case "perplexity":
		return providers.ProviderPerplexity
	default:
		return providers.ProviderType(strings.ToLower(providerType))
	}
}

func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 4 {
		return "****"
	}
	return "****" + key[len(key)-4:]
}

func (s *Server) listProviderConfigs() []ProviderConfig {
	s.providersMu.RLock()
	if len(s.providers) > 0 {
		out := make([]ProviderConfig, 0, len(s.providers))
		for _, provider := range s.providers {
			safe := provider
			safe.APIKey = ""
			out = append(out, safe)
		}
		s.providersMu.RUnlock()
		return out
	}
	s.providersMu.RUnlock()

	if s.providerRegistry == nil {
		return []ProviderConfig{}
	}

	out := make([]ProviderConfig, 0)
	for _, provider := range s.providerRegistry.List() {
		pt := provider.Type()
		status := "disconnected"
		health := "unknown"

		if s.apiKeyManager != nil {
			if _, ok := s.apiKeyManager.GetKey(pt); ok {
				status = "connected"
				health = "ok"
			}
		}

		pingCtx, pingCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		if provider.Ping(pingCtx) == nil {
			status = "connected"
			health = "ok"
		}
		pingCancel()

		out = append(out, ProviderConfig{
			ID:     string(pt),
			Name:   provider.Name(),
			Type:   string(pt),
			Status: status,
			Health: health,
		})
	}
	return out
}

func (s *Server) connectProvider(ctx context.Context, config ProviderConfig) ProviderConfig {
	pt := mapDashboardProviderType(config.Type)
	if config.Type == "" {
		config.Type = string(pt)
	}
	if config.ID == "" {
		config.ID = string(pt)
	}
	if config.Name == "" {
		config.Name = config.ID
	}

	apiKey := config.APIKey
	if apiKey == "" && s.apiKeyManager != nil {
		if stored, ok := s.apiKeyManager.GetKey(pt); ok {
			apiKey = stored
		}
	}

	if apiKey != "" && s.apiKeyManager != nil {
		_ = s.apiKeyManager.SetKey(pt, apiKey)
	}

	if s.providerRegistry != nil {
		if provider, ok := s.providerRegistry.Get(pt); ok {
			initCfg := providers.ProviderConfig{
				APIKey:  apiKey,
				BaseURL: config.Endpoint,
				Timeout: 30 * time.Second,
			}
			if err := provider.Initialize(ctx, initCfg); err != nil {
				config.Status = "error"
				config.Health = "error"
				config.UpdatedAt = time.Now()
				return config
			}
			if err := provider.Ping(ctx); err != nil {
				config.Status = "disconnected"
				config.Health = "error"
			} else {
				config.Status = "connected"
				config.Health = "ok"
			}
		}
	}

	if config.Status == "" {
		if apiKey != "" || pt == providers.ProviderOllama {
			config.Status = "connected"
			config.Health = "ok"
		} else {
			config.Status = "disconnected"
			config.Health = "unknown"
		}
	}

	if config.CreatedAt.IsZero() {
		config.CreatedAt = time.Now()
	}
	config.UpdatedAt = time.Now()
	return config
}

func (s *Server) listModelsFromRuntime(ctx context.Context) []map[string]interface{} {
	s.log.Info("listModelsFromRuntime called")
	models := make([]map[string]interface{}, 0)

	if s.providerRegistry == nil {
		s.log.Warn("Provider registry is nil, using fallback models")
		s.providersMu.RLock()
		defer s.providersMu.RUnlock()
		for _, provider := range s.providers {
			if provider.Status != "connected" {
				continue
			}
			models = append(models, s.fallbackModelsForType(provider)...)
		}
		s.log.WithField("total_models", len(models)).Info("Total models returned (fallback)")
		return models
	}

	s.log.Info("Provider registry is not nil, fetching models from provider registry")

	// Fetch models from provider registry
	providers := s.providerRegistry.List()
	for _, provider := range providers {
		providerModels, err := provider.ListModels(ctx)
		if err != nil {
			s.log.WithError(err).WithField("provider", provider.Name()).Warn("Failed to list models from provider")
			continue
		}
		for _, model := range providerModels {
			// Convert capabilities to strings
			capabilities := make([]string, len(model.Capabilities))
			for i, cap := range model.Capabilities {
				capabilities[i] = string(cap)
			}
			models = append(models, map[string]interface{}{
				"id":           model.ID,
				"name":         model.Name,
				"provider":     string(provider.Type()),
				"max_context":  model.ContextLength,
				"capabilities": capabilities,
			})
		}
	}

	// If no models found, use fallback
	if len(models) == 0 {
		s.log.Warn("No models found from providers, using fallback")
		fallbackModels := []map[string]interface{}{
			{"id": "mistral-large-2512", "name": "Mistral Large 2512", "provider": "mistral", "max_context": 32768, "capabilities": []string{"chat", "completion"}},
			{"id": "openrouter/owl-alpha", "name": "OpenRouter Owl Alpha", "provider": "openrouter", "max_context": 16384, "capabilities": []string{"chat", "completion"}},
			{"id": "qwen3.7-plus", "name": "Qwen 3.7 Plus", "provider": "qwen", "max_context": 8192, "capabilities": []string{"chat", "completion"}},
		}
		models = fallbackModels
	}

	s.log.WithField("total_models", len(models)).Info("Total models returned")
	return models
}

func (s *Server) fallbackModelsForType(provider ProviderConfig) []map[string]interface{} {
	switch provider.Type {
	case "openai":
		return []map[string]interface{}{
			{"id": "gpt-4", "name": "GPT-4", "provider": provider.ID, "max_context": 8192, "capabilities": []string{"chat", "completion"}},
			{"id": "gpt-3.5-turbo", "name": "GPT-3.5 Turbo", "provider": provider.ID, "max_context": 4096, "capabilities": []string{"chat", "completion"}},
		}
	case "anthropic":
		return []map[string]interface{}{
			{"id": "claude-3-opus", "name": "Claude 3 Opus", "provider": provider.ID, "max_context": 200000, "capabilities": []string{"chat", "completion", "vision"}},
		}
	case "ollama":
		return []map[string]interface{}{
			{"id": "llama2", "name": "Llama 2", "provider": provider.ID, "max_context": 4096, "capabilities": []string{"chat", "completion"}},
		}
	default:
		return []map[string]interface{}{
			{"id": fmt.Sprintf("%s-default", provider.Type), "name": provider.Name, "provider": provider.ID, "max_context": 4096, "capabilities": []string{"chat"}},
		}
	}
}

func (s *Server) deleteProviderConfig(providerID string) {
	s.providersMu.Lock()
	delete(s.providers, providerID)
	s.providersMu.Unlock()

	if s.apiKeyManager == nil {
		return
	}
	pt := mapDashboardProviderType(providerID)
	_ = s.apiKeyManager.DeleteKey(pt)
}
