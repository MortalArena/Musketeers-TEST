package cohere

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
	DefaultBaseURL = "https://api.cohere.ai/v1"
	ProviderType   = providers.ProviderCohere
)

// Provider Cohere provider implementation
type Provider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
	config     providers.ProviderConfig
	status     providers.ProviderStatus
	statusMu   sync.RWMutex
}

// New creates a new Cohere provider
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
	return "Cohere"
}

func (p *Provider) Capabilities() providers.ProviderCapabilities {
	return providers.ProviderCapabilities{
		SupportsChat:        true,
		SupportsStreaming:   true,
		SupportsEmbeddings:  true,
		SupportsRerank:      true,
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

	cReq := p.convertRequest(req)

	jsonBody, err := json.Marshal(cReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat", bytes.NewReader(jsonBody))
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

	var cResp CohereResponse
	if err := json.Unmarshal(body, &cResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &providers.CompletionResponse{
		ID:           cResp.GenerationID,
		Provider:     ProviderType,
		Model:        req.Model,
		Content:      cResp.Text,
		FinishReason: cResp.FinishReason,
		Usage: providers.TokenUsage{
			PromptTokens:     cResp.Meta.BilledUnits.InputTokens,
			CompletionTokens: cResp.Meta.BilledUnits.OutputTokens,
			TotalTokens:      cResp.Meta.BilledUnits.InputTokens + cResp.Meta.BilledUnits.OutputTokens,
		},
		Latency: time.Since(startTime),
	}, nil
}

func (p *Provider) StreamComplete(ctx context.Context, req *providers.CompletionRequest, callback providers.StreamingCallback) error {
	cReq := p.convertRequest(req)
	cReq.Stream = true

	jsonBody, err := json.Marshal(cReq)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat", bytes.NewReader(jsonBody))
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

		var chunk CohereStreamChunk
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}

		if chunk.EventType == "text-generation" {
			streamChunk := providers.StreamChunk{
				ID:       chunk.GenerationID,
				Provider: ProviderType,
				Model:    req.Model,
				Delta:    chunk.Text,
			}

			if err := callback(streamChunk); err != nil {
				return err
			}
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

	var modelsResp CohereModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, err
	}

	models := make([]providers.ModelInfo, 0, len(modelsResp.Models))
	for _, m := range modelsResp.Models {
		model := providers.ModelInfo{
			ID:            m.Name,
			Name:          m.Name,
			Provider:      ProviderType,
			Owner:         "Cohere",
			ContextLength: p.getContextLength(m.Name),
			IsAvailable:   true,
			Capabilities:  p.getModelCapabilities(m.Name),
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

func (p *Provider) convertRequest(req *providers.CompletionRequest) *CohereRequest {
	messages := make([]CohereMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = CohereMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	return &CohereRequest{
		Message:     messages,
		Model:       req.Model,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      req.Stream,
	}
}

func (p *Provider) handleErrorResponse(statusCode int, body []byte) error {
	var errResp CohereError
	if err := json.Unmarshal(body, &errResp); err != nil {
		return providers.NewProviderError(ProviderType, statusCode, "unknown", string(body))
	}

	pErr := providers.NewProviderError(ProviderType, statusCode, errResp.Message, errResp.Message)

	if statusCode == 429 {
		pErr.Retryable = true
	}

	return pErr
}

func (p *Provider) getContextLength(modelID string) int {
	contextLengths := map[string]int{
		"command-r-plus": 128000,
		"command-r":      128000,
		"command-light":  32000,
	}

	if cl, ok := contextLengths[modelID]; ok {
		return cl
	}
	return 4000
}

func (p *Provider) getModelCapabilities(modelID string) []providers.ModelCapability {
	capabilities := []providers.ModelCapability{
		providers.CapabilityText,
		providers.CapabilityStreaming,
	}

	if strings.Contains(modelID, "command") {
		capabilities = append(capabilities, providers.CapabilityLongContext)
	}

	return capabilities
}

// Cohere types
type CohereRequest struct {
	Message     []CohereMessage `json:"message"`
	Model       string          `json:"model"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type CohereMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CohereResponse struct {
	GenerationID string `json:"generation_id"`
	Text         string `json:"text"`
	FinishReason string `json:"finish_reason"`
	Meta         struct {
		BilledUnits struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"billed_units"`
	} `json:"meta"`
}

type CohereStreamChunk struct {
	EventType    string `json:"event_type"`
	GenerationID string `json:"generation_id,omitempty"`
	Text         string `json:"text,omitempty"`
}

type CohereModelsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

type CohereError struct {
	Message string `json:"message"`
}
