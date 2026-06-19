package custom

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/providers"
	"go.uber.org/zap"
)

const (
	DefaultBaseURL = ""
	ProviderType   = providers.ProviderCustom
)

// Provider Custom provider implementation for user-defined models and APIs
type Provider struct {
	apiKey       string
	baseURL      string
	httpClient   *http.Client
	logger       *zap.Logger
	config       providers.ProviderConfig
	status       providers.ProviderStatus
	statusMu     sync.RWMutex
	customModels map[string]CustomModelConfig
}

// CustomModelConfig represents configuration for a custom model
type CustomModelConfig struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	BaseURL       string            `json:"base_url"`
	APIKey        string            `json:"api_key"`
	Headers       map[string]string `json:"headers"`
	ContextLength int               `json:"context_length"`
	Capabilities  []string          `json:"capabilities"`
	APIFormat     string            `json:"api_format"` // "openai", "anthropic", "custom"
}

// New creates a new Custom provider
func New() providers.Provider {
	return &Provider{
		baseURL: DefaultBaseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
		logger:       zap.NewNop(),
		customModels: make(map[string]CustomModelConfig),
	}
}

func (p *Provider) Type() providers.ProviderType {
	return ProviderType
}

func (p *Provider) Name() string {
	return "Custom"
}

func (p *Provider) Capabilities() providers.ProviderCapabilities {
	return providers.ProviderCapabilities{
		SupportsChat:        true,
		SupportsStreaming:   true,
		SupportsVision:      true,
		SupportsFunctions:   true,
		SupportsLongContext: true,
	}
}

func (p *Provider) Initialize(ctx context.Context, config providers.ProviderConfig) error {
	p.config = config

	if config.BaseURL != "" {
		p.baseURL = config.BaseURL
	}

	if config.Timeout > 0 {
		p.httpClient.Timeout = config.Timeout
	}

	if config.APIKey != "" {
		p.apiKey = config.APIKey
	}

	return p.Ping(ctx)
}

func (p *Provider) Close() error {
	return nil
}

func (p *Provider) Ping(ctx context.Context) error {
	if p.baseURL == "" {
		p.updateStatus(false, fmt.Errorf("no base URL configured"))
		return fmt.Errorf("no base URL configured")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return err
	}

	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		p.updateStatus(false, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("ping failed with status: %d", resp.StatusCode)
		p.updateStatus(false, err)
		return err
	}

	p.updateStatus(true, nil)
	return nil
}

func (p *Provider) updateStatus(available bool, err error) {
	p.statusMu.Lock()
	defer p.statusMu.Unlock()

	p.status.Provider = ProviderType
	p.status.IsAvailable = available
	p.status.LastCheck = time.Now()
	if err != nil {
		p.status.Error = err.Error()
	} else {
		p.status.Error = ""
	}
}

// AddCustomModel adds a custom model configuration
func (p *Provider) AddCustomModel(config CustomModelConfig) error {
	p.statusMu.Lock()
	defer p.statusMu.Unlock()

	if config.ID == "" {
		return fmt.Errorf("model ID is required")
	}
	if config.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}

	p.customModels[config.ID] = config
	return nil
}

// RemoveCustomModel removes a custom model configuration
func (p *Provider) RemoveCustomModel(modelID string) error {
	p.statusMu.Lock()
	defer p.statusMu.Unlock()

	delete(p.customModels, modelID)
	return nil
}

// GetCustomModel returns a custom model configuration
func (p *Provider) GetCustomModel(modelID string) (CustomModelConfig, bool) {
	p.statusMu.RLock()
	defer p.statusMu.RUnlock()

	config, exists := p.customModels[modelID]
	return config, exists
}

func (p *Provider) Complete(ctx context.Context, req *providers.CompletionRequest) (*providers.CompletionResponse, error) {
	startTime := time.Now()

	modelConfig, exists := p.GetCustomModel(req.Model)
	if !exists {
		return nil, fmt.Errorf("custom model not found: %s", req.Model)
	}

	baseURL := modelConfig.BaseURL
	if p.baseURL != "" {
		baseURL = p.baseURL
	}

	cReq := p.convertRequest(req, modelConfig.APIFormat)

	jsonBody, err := json.Marshal(cReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq, modelConfig)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleErrorResponse(resp.StatusCode, body)
	}

	var cResp CustomResponse
	if err := json.Unmarshal(body, &cResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(cResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := cResp.Choices[0]

	return &providers.CompletionResponse{
		ID:           cResp.ID,
		Provider:     ProviderType,
		Model:        cResp.Model,
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
		Usage: providers.TokenUsage{
			PromptTokens:     cResp.Usage.PromptTokens,
			CompletionTokens: cResp.Usage.CompletionTokens,
			TotalTokens:      cResp.Usage.TotalTokens,
		},
		Latency: time.Since(startTime),
	}, nil
}

func (p *Provider) StreamComplete(ctx context.Context, req *providers.CompletionRequest, callback providers.StreamingCallback) error {
	modelConfig, exists := p.GetCustomModel(req.Model)
	if !exists {
		return fmt.Errorf("custom model not found: %s", req.Model)
	}

	baseURL := modelConfig.BaseURL
	if p.baseURL != "" {
		baseURL = p.baseURL
	}

	req.Stream = true
	cReq := p.convertRequest(req, modelConfig.APIFormat)

	jsonBody, err := json.Marshal(cReq)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}

	p.setHeaders(httpReq, modelConfig)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return p.handleErrorResponse(resp.StatusCode, body)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk CustomStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta

		streamChunk := providers.StreamChunk{
			ID:           chunk.ID,
			Provider:     ProviderType,
			Model:        chunk.Model,
			Delta:        delta.Content,
			FinishReason: chunk.Choices[0].FinishReason,
		}

		if err := callback(streamChunk); err != nil {
			return err
		}
	}

	return scanner.Err()
}

func (p *Provider) ListModels(ctx context.Context) ([]providers.ModelInfo, error) {
	p.statusMu.RLock()
	defer p.statusMu.RUnlock()

	models := make([]providers.ModelInfo, 0, len(p.customModels))
	for _, config := range p.customModels {
		model := providers.ModelInfo{
			ID:            config.ID,
			Name:          config.Name,
			Provider:      ProviderType,
			Owner:         "Custom",
			ContextLength: config.ContextLength,
			IsAvailable:   true,
			Capabilities:  p.getModelCapabilities(config),
			Categories:    []string{"custom", "chat"},
		}
		models = append(models, model)
	}

	p.statusMu.Lock()
	p.status.ModelsCount = len(models)
	p.statusMu.Unlock()

	return models, nil
}

func (p *Provider) GetModel(ctx context.Context, modelID string) (*providers.ModelInfo, error) {
	models, err := p.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	for i := range models {
		if models[i].ID == modelID {
			return &models[i], nil
		}
	}
	return nil, providers.ErrModelNotFound
}

func (p *Provider) Status() providers.ProviderStatus {
	p.statusMu.RLock()
	defer p.statusMu.RUnlock()
	return p.status
}

func (p *Provider) IsAvailable() bool {
	p.statusMu.RLock()
	defer p.statusMu.RUnlock()
	return p.status.IsAvailable
}

func (p *Provider) setHeaders(req *http.Request, modelConfig CustomModelConfig) {
	req.Header.Set("Content-Type", "application/json")

	if modelConfig.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+modelConfig.APIKey)
	} else if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	for key, value := range modelConfig.Headers {
		req.Header.Set(key, value)
	}
}

func (p *Provider) convertRequest(req *providers.CompletionRequest, apiFormat string) interface{} {
	messages := make([]CustomMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = CustomMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	switch apiFormat {
	case "anthropic":
		return AnthropicRequest{
			Model:       req.Model,
			Messages:    messages,
			Temperature: req.Temperature,
			TopP:        req.TopP,
			MaxTokens:   req.MaxTokens,
			Stream:      req.Stream,
		}
	default:
		return OpenAIRequest{
			Model:       req.Model,
			Messages:    messages,
			Temperature: req.Temperature,
			TopP:        req.TopP,
			MaxTokens:   req.MaxTokens,
			Stop:        req.Stop,
			Stream:      req.Stream,
		}
	}
}

func (p *Provider) handleErrorResponse(statusCode int, body []byte) error {
	return providers.NewProviderError(ProviderType, statusCode, "custom_error", string(body))
}

func (p *Provider) getModelCapabilities(config CustomModelConfig) []providers.ModelCapability {
	capabilities := []providers.ModelCapability{
		providers.CapabilityText,
		providers.CapabilityStreaming,
	}

	for _, cap := range config.Capabilities {
		switch cap {
		case "vision":
			capabilities = append(capabilities, providers.CapabilityVision)
		case "code":
			capabilities = append(capabilities, providers.CapabilityCode)
		case "long_context":
			capabilities = append(capabilities, providers.CapabilityLongContext)
		}
	}

	return capabilities
}

// Custom types
type CustomMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []CustomMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stop        []string        `json:"stop,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type AnthropicRequest struct {
	Model       string          `json:"model"`
	Messages    []CustomMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type CustomResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type CustomStreamChunk struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
}
