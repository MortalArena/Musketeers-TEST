package anthropic

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
	DefaultBaseURL = "https://api.anthropic.com/v1"
	ProviderType   = providers.ProviderAnthropic
)

// Provider Anthropic provider implementation
type Provider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
	config     providers.ProviderConfig
	status     providers.ProviderStatus
	statusMu   sync.RWMutex
}

// New creates a new Anthropic provider
func New() providers.Provider {
	return &Provider{
		baseURL: DefaultBaseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
		logger: zap.NewNop(),
	}
}

func (p *Provider) Type() providers.ProviderType {
	return ProviderType
}

func (p *Provider) Name() string {
	return "Anthropic"
}

func (p *Provider) Capabilities() providers.ProviderCapabilities {
	return providers.ProviderCapabilities{
		SupportsChat:        true,
		SupportsStreaming:   true,
		SupportsVision:      true,
		SupportsFunctions:   true,
		SupportsReasoning:   true,
		SupportsLongContext: true,
	}
}

func (p *Provider) Initialize(ctx context.Context, config providers.ProviderConfig) error {
	if config.APIKey == "" {
		return providers.ErrAPIKeyMissing
	}

	p.apiKey = config.APIKey
	p.config = config

	if config.BaseURL != "" {
		p.baseURL = config.BaseURL
	}
	if config.Timeout > 0 {
		p.httpClient.Timeout = config.Timeout
	}

	return p.Ping(ctx)
}

func (p *Provider) Close() error {
	return nil
}

func (p *Provider) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return err
	}

	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

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

func (p *Provider) Complete(ctx context.Context, req *providers.CompletionRequest) (*providers.CompletionResponse, error) {
	startTime := time.Now()

	anReq := p.convertRequest(req)

	jsonBody, err := json.Marshal(anReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

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

	var anResp AnthropicResponse
	if err := json.Unmarshal(body, &anResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(anResp.Content) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	content := ""
	for _, block := range anResp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &providers.CompletionResponse{
		ID:           anResp.ID,
		Provider:     ProviderType,
		Model:        anResp.Model,
		Content:      content,
		FinishReason: anResp.StopReason,
		ToolCalls:    p.convertToolCalls(anResp.Content),
		Usage: providers.TokenUsage{
			PromptTokens:     anResp.Usage.InputTokens,
			CompletionTokens: anResp.Usage.OutputTokens,
			TotalTokens:      anResp.Usage.InputTokens + anResp.Usage.OutputTokens,
		},
		Latency: time.Since(startTime),
	}, nil
}

func (p *Provider) StreamComplete(ctx context.Context, req *providers.CompletionRequest, callback providers.StreamingCallback) error {
	anReq := p.convertRequest(req)
	anReq.Stream = true

	jsonBody, err := json.Marshal(anReq)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}

	p.setHeaders(httpReq)

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

		var chunk AnthropicStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if chunk.Type == "content_block_delta" && chunk.Delta.Type == "text_delta" {
			streamChunk := providers.StreamChunk{
				ID:       chunk.MessageID,
				Provider: ProviderType,
				Model:    chunk.Model,
				Delta:    chunk.Delta.Text,
			}

			if err := callback(streamChunk); err != nil {
				return err
			}
		}
	}

	return scanner.Err()
}

func (p *Provider) ListModels(ctx context.Context) ([]providers.ModelInfo, error) {
	// Anthropic doesn't have a models endpoint, so we return static list
	models := []providers.ModelInfo{
		{
			ID:            "claude-opus-4.8",
			Name:          "Claude Opus 4.8",
			Provider:      ProviderType,
			Owner:         "Anthropic",
			Description:   "Most capable Claude model",
			ContextLength: 1000000,
			Capabilities: []providers.ModelCapability{
				providers.CapabilityText,
				providers.CapabilityCode,
				providers.CapabilityVision,
				providers.CapabilityStreaming,
				providers.CapabilityReasoning,
				providers.CapabilityLongContext,
			},
			Categories:  []string{"chat", "code", "reasoning"},
			IsAvailable: true,
		},
		{
			ID:            "claude-opus-4.7",
			Name:          "Claude Opus 4.7",
			Provider:      ProviderType,
			Owner:         "Anthropic",
			Description:   "Previous generation Opus",
			ContextLength: 1000000,
			Capabilities: []providers.ModelCapability{
				providers.CapabilityText,
				providers.CapabilityCode,
				providers.CapabilityVision,
				providers.CapabilityStreaming,
				providers.CapabilityReasoning,
				providers.CapabilityLongContext,
			},
			Categories:  []string{"chat", "code", "reasoning"},
			IsAvailable: true,
		},
		{
			ID:            "claude-sonnet-4.6",
			Name:          "Claude Sonnet 4.6",
			Provider:      ProviderType,
			Owner:         "Anthropic",
			Description:   "Balanced performance and speed",
			ContextLength: 1000000,
			Capabilities: []providers.ModelCapability{
				providers.CapabilityText,
				providers.CapabilityCode,
				providers.CapabilityVision,
				providers.CapabilityStreaming,
				providers.CapabilityLongContext,
			},
			Categories:  []string{"chat", "code"},
			IsAvailable: true,
		},
		{
			ID:            "claude-haiku-4.5",
			Name:          "Claude Haiku 4.5",
			Provider:      ProviderType,
			Owner:         "Anthropic",
			Description:   "Fast and efficient model",
			ContextLength: 200000,
			Capabilities: []providers.ModelCapability{
				providers.CapabilityText,
				providers.CapabilityCode,
				providers.CapabilityVision,
				providers.CapabilityStreaming,
			},
			Categories:  []string{"chat", "fast"},
			IsAvailable: true,
		},
		{
			ID:            "claude-fable-latest",
			Name:          "Claude Fable Latest",
			Provider:      ProviderType,
			Owner:         "Anthropic",
			Description:   "Creative writing model",
			ContextLength: 1000000,
			Capabilities: []providers.ModelCapability{
				providers.CapabilityText,
				providers.CapabilityStreaming,
				providers.CapabilityLongContext,
			},
			Categories:  []string{"chat", "creative"},
			IsAvailable: true,
		},
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

func (p *Provider) setHeaders(req *http.Request) {
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")
}

func (p *Provider) convertRequest(req *providers.CompletionRequest) *AnthropicRequest {
	messages := make([]AnthropicMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = AnthropicMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	return &AnthropicRequest{
		Model:     req.Model,
		Messages:  messages,
		MaxTokens: req.MaxTokens,
		Stream:    req.Stream,
	}
}

func (p *Provider) convertToolCalls(content []AnthropicContentBlock) []providers.ToolCall {
	var calls []providers.ToolCall
	for _, block := range content {
		if block.Type == "tool_use" {
			calls = append(calls, providers.ToolCall{
				ID:   block.ID,
				Type: "tool_use",
				Function: providers.FunctionCall{
					Name:      block.Name,
					Arguments: block.Input,
				},
			})
		}
	}
	return calls
}

func (p *Provider) handleErrorResponse(statusCode int, body []byte) error {
	var errResp AnthropicError
	if err := json.Unmarshal(body, &errResp); err != nil {
		return providers.NewProviderError(ProviderType, statusCode, "unknown", string(body))
	}

	pErr := providers.NewProviderError(ProviderType, statusCode, errResp.Error.Type, errResp.Error.Message)

	if statusCode == 429 {
		pErr.Retryable = true
	}

	return pErr
}

// Anthropic types
type AnthropicRequest struct {
	Model     string             `json:"model"`
	Messages  []AnthropicMessage `json:"messages"`
	MaxTokens int                `json:"max_tokens"`
	Stream    bool               `json:"stream,omitempty"`
}

type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicResponse struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
	Role       string                  `json:"role"`
	Content    []AnthropicContentBlock `json:"content"`
	Model      string                  `json:"model"`
	StopReason string                  `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type AnthropicContentBlock struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Input string `json:"input,omitempty"`
}

type AnthropicStreamChunk struct {
	Type      string `json:"type"`
	MessageID string `json:"message_id,omitempty"`
	Model     string `json:"model,omitempty"`
	Delta     struct {
		Type string `json:"type,omitempty"`
		Text string `json:"text,omitempty"`
	} `json:"delta,omitempty"`
}

type AnthropicError struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}
