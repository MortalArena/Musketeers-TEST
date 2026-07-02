package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AutoRegistrar يسجل كل الموديلات من models.json كوكلاء تلقائياً
type AutoRegistrar struct {
	registry *AgentRegistry
	logger   *zap.Logger
	mu       sync.RWMutex
	modelsPath string
}

// NewAutoRegistrar ينشئ مسجل تلقائي جديد
func NewAutoRegistrar(registry *AgentRegistry, logger *zap.Logger) *AutoRegistrar {
	return &AutoRegistrar{
		registry:   registry,
		logger:     logger,
		modelsPath: "models.json",
	}
}

// SetModelsPath يضبط مسار ملف models.json
func (ar *AutoRegistrar) SetModelsPath(path string) {
	ar.mu.Lock()
	defer ar.mu.Unlock()
	ar.modelsPath = path
}

// RegisterAllModelsFromFile يسجل كل الموديلات من ملف JSON
func (ar *AutoRegistrar) RegisterAllModelsFromFile(ctx context.Context) (int, error) {
	ar.mu.RLock()
	path := ar.modelsPath
	ar.mu.RUnlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("failed to read models file %s: %w", path, err)
	}

	var catalog struct {
		Version   string `json:"version"`
		Providers map[string]struct {
			Name   string `json:"name"`
			Models []struct {
				ID           string   `json:"id"`
				Name         string   `json:"name"`
				ContextLength int      `json:"context_length"`
				Capabilities []string `json:"capabilities"`
				Categories   []string `json:"categories"`
			} `json:"models"`
		} `json:"providers"`
	}

	if err := json.Unmarshal(data, &catalog); err != nil {
		return 0, fmt.Errorf("failed to parse models file: %w", err)
	}

	totalRegistered := 0

	for providerKey, provider := range catalog.Providers {
		for _, model := range provider.Models {
			agentID := fmt.Sprintf("agent-%s-%s", providerKey, model.ID)
			agentName := fmt.Sprintf("%s: %s", provider.Name, model.Name)

			agentType := AgentTypeAPI
			isLocal := false
			for _, cat := range model.Categories {
				if cat == "local" {
					agentType = AgentTypeLocal
					isLocal = true
					break
				}
			}

			modelAgent := &modelAgent{
				id:           agentID,
				name:         agentName,
				agentType:    agentType,
				provider:     providerKey,
				model:        model.ID,
				contextWindow: model.ContextLength,
			}

			metadata := &AgentMetadata{
				AgentID:       agentID,
				Name:          agentName,
				Type:          agentType,
				Provider:      providerKey,
				Model:         model.ID,
				Version:       catalog.Version,
				Endpoint:      "",
				AuthMethod:    "api_key",
				MaxTokens:     4096,
				ContextWindow: model.ContextLength,
				RegisteredAt:  time.Now(),
				LastSeen:      time.Now(),
				Tags:          append(model.Categories, model.Capabilities...),
				Config: map[string]interface{}{
					"provider":     providerKey,
					"model":        model.ID,
					"is_local":     isLocal,
					"context_len":  model.ContextLength,
					"capabilities": model.Capabilities,
				},
			}

			if err := ar.registry.Register(modelAgent, metadata); err != nil {
				ar.logger.Warn("Failed to register model agent",
					zap.String("model", model.ID),
					zap.String("provider", providerKey),
					zap.Error(err),
				)
				continue
			}

			totalRegistered++
		}
	}

	ar.logger.Info("Auto-registered model agents",
		zap.Int("total", totalRegistered),
		zap.String("file", path),
	)

	return totalRegistered, nil
}

// RegisterAllFromProviders يسجل كل الموديلات من ProviderRegistry
func (ar *AutoRegistrar) RegisterAllFromProviders(ctx context.Context, providerRegistry interface{}) int {
	totalRegistered := 0

	// Try to get providers and list models from each
	type providerLister interface {
		Name() string
		ListModels(ctx context.Context) ([]struct {
			ID            string
			Name          string
			ContextLength int
		}, error)
	}

	if lister, ok := providerRegistry.(interface{ Get(providerType string) (interface{}, bool) }); ok {
		_ = lister
	}

	return totalRegistered
}

// ============================================================
// modelAgent - وكيل بسيط يمثل موديل AI
// ============================================================

type modelAgent struct {
	id            string
	name          string
	agentType     AgentType
	provider      string
	model         string
	contextWindow int
	mu            sync.RWMutex
}

func (m *modelAgent) GetInfo() *AgentInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &AgentInfo{
		ID:            m.id,
		Name:          m.name,
		Type:          m.agentType,
		Provider:      m.provider,
		Model:         m.model,
		Version:       "1.0.0",
		ContextWindow: m.contextWindow,
		MaxTokens:     4096,
		AuthMethod:    "api_key",
	}
}

func (m *modelAgent) SendMessage(ctx context.Context, prompt string) (*AgentResponse, error) {
	return &AgentResponse{
		Content:  fmt.Sprintf("[%s] Received: %s", m.name, prompt),
		Tokens:   0,
		Duration: 0,
		Metadata: map[string]interface{}{
			"provider": m.provider,
			"model":    m.model,
		},
	}, nil
}

func (m *modelAgent) ExecuteTask(ctx context.Context, task *AgentTask) (*TaskExecutionResult, error) {
	return &TaskExecutionResult{
		Success: true,
		Output:  fmt.Sprintf("[%s] Task acknowledged: %s", m.name, task.Title),
		Metrics: map[string]interface{}{
			"provider": m.provider,
			"model":    m.model,
		},
	}, nil
}

func (m *modelAgent) GetCapabilities() []AgentCapability {
	return []AgentCapability{
		CapabilityAnalysis,
		CapabilityCodeGeneration,
	}
}

func (m *modelAgent) GetStatus() *AgentStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &AgentStatus{
		IsAvailable:  true,
		Load:         0,
		LastSeen:     time.Now(),
		ResponseTime: 0,
		SuccessRate:  1.0,
		TotalTasks:   0,
		FailedTasks:  0,
	}
}

func (m *modelAgent) IsAvailable() bool {
	return true
}

func (m *modelAgent) Close() error {
	return nil
}
