package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/providers"
)

// ProviderAdapter محول يربط UnifiedAgent بـ Provider حقيقي
// يجعل أي model من أي provider يصبح وكيل حقيقي
type ProviderAdapter struct {
	info           *agent.AgentInfo
	provider       providers.Provider
	providerType   providers.ProviderType
	model          string
	initialized    bool
	providerConfig *providers.ProviderConfig
}

// NewProviderAdapter ينشئ محول provider
func NewProviderAdapter(agentID, name string, agentType agent.AgentType, providerType providers.ProviderType, model string) *ProviderAdapter {
	info := &agent.AgentInfo{
		ID:         agentID,
		Name:       name,
		Type:       agentType,
		Provider:   string(providerType),
		Model:      model,
		AuthMethod: "provider",
		CreatedAt:  time.Now(),
	}

	return &ProviderAdapter{
		info:         info,
		providerType: providerType,
		model:        model,
		initialized:  false,
	}
}

// SetProvider يضبط الـ Provider الحقيقي
func (a *ProviderAdapter) SetProvider(provider providers.Provider) {
	a.provider = provider
}

// SetProviderConfig يضبط إعدادات Provider
func (a *ProviderAdapter) SetProviderConfig(config *providers.ProviderConfig) {
	a.providerConfig = config
}

// Initialize يهيئ المحول
func (a *ProviderAdapter) Initialize(ctx context.Context, config *providers.ProviderConfig) error {
	if a.provider == nil {
		return fmt.Errorf("provider not set")
	}

	// Store the config regardless
	a.providerConfig = config

	// If config provides an API key (or it's the first initialization and we have a key),
	// pass it to the underlying provider. If no API key is given, we assume the
	// provider was already configured from environment variables and skip re-init.
	if config != nil && config.APIKey != "" {
		if err := a.provider.Initialize(ctx, *config); err != nil {
			return fmt.Errorf("failed to initialize provider: %w", err)
		}
	}

	a.initialized = true
	return nil
}

func (a *ProviderAdapter) GetInfo() *agent.AgentInfo {
	return a.info
}

func (a *ProviderAdapter) SendMessage(ctx context.Context, prompt string) (*agent.AgentResponse, error) {
	startTime := time.Now()

	if !a.initialized {
		return nil, fmt.Errorf("agent not initialized")
	}

	if a.provider == nil {
		return nil, fmt.Errorf("provider not set")
	}

	// استخدام Provider الحقيقي لتوليد الرد
	resp, err := a.provider.Complete(ctx, &providers.CompletionRequest{
		Model: a.model,
		Messages: []providers.Message{
			{Role: providers.RoleUser, Content: prompt},
		},
		MaxTokens: 2000,
	})

	if err != nil {
		return nil, fmt.Errorf("provider completion failed: %w", err)
	}

	return &agent.AgentResponse{
		Content:  resp.Content,
		Tokens:   resp.Usage.TotalTokens,
		Duration: time.Since(startTime),
		Metadata: map[string]interface{}{
			"provider": resp.Provider,
			"model":    resp.Model,
		},
	}, nil
}

func (a *ProviderAdapter) ExecuteTask(ctx context.Context, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	if !a.initialized {
		return nil, fmt.Errorf("agent not initialized")
	}

	if a.provider == nil {
		return nil, fmt.Errorf("provider not set")
	}

	// تحويل المهمة إلى رسالة
	prompt := task.Title
	if task.Description != "" {
		prompt = fmt.Sprintf("%s\n\n%s", task.Title, task.Description)
	}

	// استخدام Provider الحقيقي
	resp, err := a.provider.Complete(ctx, &providers.CompletionRequest{
		Model: a.model,
		Messages: []providers.Message{
			{Role: providers.RoleUser, Content: prompt},
		},
		MaxTokens: 4000,
	})

	if err != nil {
		return &agent.TaskExecutionResult{
			Success: false,
			Output:  "",
			Error:   err.Error(),
		}, nil
	}

	return &agent.TaskExecutionResult{
		Success:  true,
		Output:   resp.Content,
		Metrics:  map[string]interface{}{"tokens": resp.Usage.TotalTokens},
		Duration: resp.Latency,
	}, nil
}

func (a *ProviderAdapter) GetProvider() (providers.Provider, string) {
	return a.provider, a.model
}

func (a *ProviderAdapter) GetCapabilities() []agent.AgentCapability {
	// قدرات افتراضية لـ provider-based agents
	return []agent.AgentCapability{
		agent.CapabilityCodeGeneration,
		agent.CapabilityCodeReview,
		agent.CapabilityAnalysis,
		agent.CapabilityDocumentation,
	}
}

func (a *ProviderAdapter) GetStatus() *agent.AgentStatus {
	available := a.initialized && a.provider != nil && a.provider.IsAvailable()

	return &agent.AgentStatus{
		IsAvailable:  available,
		LastSeen:     time.Now(),
		ResponseTime: 1 * time.Second,
		SuccessRate:  95.0,
	}
}

func (a *ProviderAdapter) IsAvailable() bool {
	return a.initialized && a.provider != nil && a.provider.IsAvailable()
}

func (a *ProviderAdapter) Close() error {
	a.initialized = false
	if a.provider != nil {
		return a.provider.Close()
	}
	return nil
}

// NewProviderAgent ينشئ وكيل provider بسيط
func NewProviderAgent(agentID, name string, providerType providers.ProviderType, model string) *ProviderAdapter {
	return NewProviderAdapter(agentID, name, agent.AgentTypeAPI, providerType, model)
}
