package ollama

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
	DefaultBaseURL = "http://localhost:11434"
	ProviderType   = providers.ProviderOllama
)

// Provider Ollama provider implementation for local models
type Provider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
	config     providers.ProviderConfig
	status     providers.ProviderStatus
	statusMu   sync.RWMutex
}

// New creates a new Ollama provider
func New() providers.Provider {
	return &Provider{
		baseURL: DefaultBaseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
		logger: zap.NewNop(),
	}
}

func (p *Provider) Type() providers.ProviderType {
	return ProviderType
}

func (p *Provider) Name() string {
	return "Ollama"
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

	return p.Ping(ctx)
}

func (p *Provider) Close() error {
	return nil
}

func (p *Provider) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/api/tags", nil)
	if err != nil {
		return err
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

func (p *Provider) Complete(ctx context.Context, req *providers.CompletionRequest) (*providers.CompletionResponse, error) {
	startTime := time.Now()

	oReq := p.convertRequest(req)

	jsonBody, err := json.Marshal(oReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(jsonBody))
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

	var oResp OllamaResponse
	if err := json.Unmarshal(body, &oResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &providers.CompletionResponse{
		ID:           oResp.ID,
		Provider:     ProviderType,
		Model:        oResp.Model,
		Content:      oResp.Message.Content,
		FinishReason: oResp.DoneReason,
		Usage: providers.TokenUsage{
			PromptTokens:     oResp.PromptEvalCount,
			CompletionTokens: oResp.EvalCount,
			TotalTokens:      oResp.PromptEvalCount + oResp.EvalCount,
		},
		Latency: time.Since(startTime),
	}, nil
}

func (p *Provider) StreamComplete(ctx context.Context, req *providers.CompletionRequest, callback providers.StreamingCallback) error {
	oReq := p.convertRequest(req)
	oReq.Stream = true

	jsonBody, err := json.Marshal(oReq)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(jsonBody))
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
		line := scanner.Bytes()

		var chunk OllamaStreamChunk
		if err := json.Unmarshal(line, &chunk); err != nil {
			continue
		}

		streamChunk := providers.StreamChunk{
			ID:           chunk.ID,
			Provider:     ProviderType,
			Model:        chunk.Model,
			Delta:        chunk.Message.Content,
			FinishReason: chunk.DoneReason,
		}

		if err := callback(streamChunk); err != nil {
			return err
		}
	}

	return scanner.Err()
}

func (p *Provider) ListModels(ctx context.Context) ([]providers.ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list models: %d", resp.StatusCode)
	}

	var modelsResp OllamaModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, err
	}

	models := make([]providers.ModelInfo, 0, len(modelsResp.Models))
	for _, m := range modelsResp.Models {
		model := providers.ModelInfo{
			ID:            m.Name,
			Name:          m.Name,
			Provider:      ProviderType,
			Owner:         "Ollama",
			Description:   m.Details.Families,
			ContextLength: p.getContextLength(m.Name),
			IsAvailable:   true,
			Capabilities:  p.getModelCapabilities(m),
			Categories:    []string{"local", "chat"},
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
	req.Header.Set("Content-Type", "application/json")
}

func (p *Provider) convertRequest(req *providers.CompletionRequest) *OllamaRequest {
	messages := make([]OllamaMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = OllamaMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	return &OllamaRequest{
		Model:       req.Model,
		Messages:    messages,
		Stream:      req.Stream,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		NumPredict:  req.MaxTokens,
	}
}

func (p *Provider) handleErrorResponse(statusCode int, body []byte) error {
	return providers.NewProviderError(ProviderType, statusCode, "ollama_error", string(body))
}

func (p *Provider) getContextLength(modelID string) int {
	contextLengths := map[string]int{
		"llama3":            8192,
		"llama3:8b":         8192,
		"llama3:70b":        8192,
		"mistral":           32768,
		"mistral:7b":        32768,
		"codellama":         16384,
		"codellama:7b":      16384,
		"deepseek-coder":    16384,
		"deepseek-coder:6b": 16384,
		"phi3":              128000,
		"phi3:mini":         128000,
		"gemma":             8192,
		"gemma:2b":          8192,
		"gemma:7b":          8192,
	}

	for model, cl := range contextLengths {
		if strings.HasPrefix(modelID, model) {
			return cl
		}
	}
	return 8192
}

func (p *Provider) getModelCapabilities(m OllamaModel) []providers.ModelCapability {
	capabilities := []providers.ModelCapability{
		providers.CapabilityText,
		providers.CapabilityStreaming,
	}

	if strings.Contains(m.Name, "vision") || strings.Contains(m.Name, "llava") {
		capabilities = append(capabilities, providers.CapabilityVision)
	}

	if strings.Contains(m.Name, "code") || strings.Contains(m.Name, "coder") {
		capabilities = append(capabilities, providers.CapabilityCode)
	}

	if p.getContextLength(m.Name) >= 100000 {
		capabilities = append(capabilities, providers.CapabilityLongContext)
	}

	return capabilities
}

// Ollama types
type OllamaRequest struct {
	Model       string          `json:"model"`
	Messages    []OllamaMessage `json:"messages"`
	Stream      bool            `json:"stream"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	NumPredict  int             `json:"num_predict,omitempty"`
}

type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OllamaResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	DoneReason      string `json:"done_reason"`
	PromptEvalCount int    `json:"prompt_eval_count"`
	EvalCount       int    `json:"eval_count"`
}

type OllamaStreamChunk struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	DoneReason string `json:"done_reason,omitempty"`
}

type OllamaModelsResponse struct {
	Models []OllamaModel `json:"models"`
}

type OllamaModel struct {
	Name    string `json:"name"`
	Details struct {
		Families string `json:"families"`
	} `json:"details"`
}
