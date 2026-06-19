package sourceful

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
	DefaultBaseURL = "https://api.sourceful.ai/v1"
	ProviderType   = providers.ProviderSourceful
)

// Provider Sourceful provider implementation
type Provider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
	config     providers.ProviderConfig
	status     providers.ProviderStatus
	statusMu   sync.RWMutex
}

// New creates a new Sourceful provider
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
	return "Sourceful"
}

func (p *Provider) Capabilities() providers.ProviderCapabilities {
	return providers.ProviderCapabilities{
		SupportsChat:        true,
		SupportsStreaming:   true,
		SupportsFunctions:   true,
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

	req.Header.Set("Authorization", "Bearer "+p.apiKey)

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

	sReq := p.convertRequest(req)

	jsonBody, err := json.Marshal(sReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
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

	var sResp SourcefulResponse
	if err := json.Unmarshal(body, &sResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(sResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := sResp.Choices[0]

	return &providers.CompletionResponse{
		ID:           sResp.ID,
		Provider:     ProviderType,
		Model:        sResp.Model,
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
		Usage: providers.TokenUsage{
			PromptTokens:     sResp.Usage.PromptTokens,
			CompletionTokens: sResp.Usage.CompletionTokens,
			TotalTokens:      sResp.Usage.TotalTokens,
		},
		Latency: time.Since(startTime),
	}, nil
}

func (p *Provider) StreamComplete(ctx context.Context, req *providers.CompletionRequest, callback providers.StreamingCallback) error {
	req.Stream = true
	sReq := p.convertRequest(req)

	jsonBody, err := json.Marshal(sReq)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
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

		var chunk SourcefulStreamChunk
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
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list models: %d", resp.StatusCode)
	}

	var modelsResp SourcefulModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, err
	}

	models := make([]providers.ModelInfo, 0, len(modelsResp.Data))
	for _, m := range modelsResp.Data {
		model := providers.ModelInfo{
			ID:            m.ID,
			Name:          m.ID,
			Provider:      ProviderType,
			Owner:         "Sourceful",
			ContextLength: p.getContextLength(m.ID),
			IsAvailable:   true,
			Capabilities:  p.getModelCapabilities(m.ID),
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

func (p *Provider) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
}

func (p *Provider) convertRequest(req *providers.CompletionRequest) *SourcefulRequest {
	return &SourcefulRequest{
		Model:       req.Model,
		Messages:    p.convertMessages(req.Messages),
		Temperature: req.Temperature,
		TopP:        req.TopP,
		MaxTokens:   req.MaxTokens,
		Stop:        req.Stop,
		Stream:      req.Stream,
	}
}

func (p *Provider) convertMessages(messages []providers.Message) []SourcefulMessage {
	result := make([]SourcefulMessage, len(messages))
	for i, msg := range messages {
		result[i] = SourcefulMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}
	return result
}

func (p *Provider) handleErrorResponse(statusCode int, body []byte) error {
	var errResp SourcefulError
	if err := json.Unmarshal(body, &errResp); err != nil {
		return providers.NewProviderError(ProviderType, statusCode, "unknown", string(body))
	}

	pErr := providers.NewProviderError(ProviderType, statusCode, errResp.Error.Code, errResp.Error.Message)

	if statusCode == 429 {
		pErr.Retryable = true
	}

	return pErr
}

func (p *Provider) getContextLength(modelID string) int {
	return 32000
}

func (p *Provider) getModelCapabilities(modelID string) []providers.ModelCapability {
	capabilities := []providers.ModelCapability{
		providers.CapabilityText,
		providers.CapabilityStreaming,
		providers.CapabilityCode,
	}
	return capabilities
}

// Sourceful types
type SourcefulRequest struct {
	Model       string             `json:"model"`
	Messages    []SourcefulMessage `json:"messages"`
	Temperature float64            `json:"temperature,omitempty"`
	TopP        float64            `json:"top_p,omitempty"`
	MaxTokens   int                `json:"max_tokens,omitempty"`
	Stop        []string           `json:"stop,omitempty"`
	Stream      bool               `json:"stream,omitempty"`
}

type SourcefulMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type SourcefulResponse struct {
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

type SourcefulStreamChunk struct {
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

type SourcefulModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type SourcefulError struct {
	Error struct {
		Message string `json:"message"`
		Code    string `json:"code"`
		Type    string `json:"type"`
	} `json:"error"`
}
