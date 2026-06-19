package perplexity

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
	DefaultBaseURL = "https://api.perplexity.ai"
	ProviderType   = providers.ProviderPerplexity
)

// Provider Perplexity provider implementation
type Provider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
	config     providers.ProviderConfig
	status     providers.ProviderStatus
	statusMu   sync.RWMutex
}

// New creates a new Perplexity provider
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
	return "Perplexity"
}

func (p *Provider) Capabilities() providers.ProviderCapabilities {
	return providers.ProviderCapabilities{
		SupportsChat:          true,
		SupportsStreaming:     true,
		SupportsVision:        false,
		SupportsAudio:         false,
		SupportsVideo:         false,
		SupportsImage:         false,
		SupportsEmbeddings:    false,
		SupportsFunctions:     true,
		SupportsJSON:          true,
		SupportsReasoning:     true,
		SupportsLongContext:   true,
		SupportsTranscription: false,
		SupportsTTS:           false,
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

	ppxReq := p.convertRequest(req)

	jsonBody, err := json.Marshal(ppxReq)
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

	var ppxResp PerplexityResponse
	if err := json.Unmarshal(body, &ppxResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(ppxResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := ppxResp.Choices[0]

	return &providers.CompletionResponse{
		ID:           ppxResp.ID,
		Provider:     ProviderType,
		Model:        ppxResp.Model,
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
		ToolCalls:    p.convertToolCalls(choice.Message.ToolCalls),
		Usage: providers.TokenUsage{
			PromptTokens:     ppxResp.Usage.PromptTokens,
			CompletionTokens: ppxResp.Usage.CompletionTokens,
			TotalTokens:      ppxResp.Usage.TotalTokens,
		},
		Latency: time.Since(startTime),
	}, nil
}

func (p *Provider) StreamComplete(ctx context.Context, req *providers.CompletionRequest, callback providers.StreamingCallback) error {
	req.Stream = true
	ppxReq := p.convertRequest(req)

	jsonBody, err := json.Marshal(ppxReq)
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

		var chunk PerplexityStreamChunk
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

		if chunk.Usage != nil {
			streamChunk.Usage = &providers.TokenUsage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
			}
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

	var modelsResp PerplexityModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, err
	}

	models := make([]providers.ModelInfo, 0, len(modelsResp.Data))
	for _, m := range modelsResp.Data {
		model := providers.ModelInfo{
			ID:            m.ID,
			Name:          m.ID,
			Provider:      ProviderType,
			Owner:         "Perplexity",
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

func (p *Provider) convertRequest(req *providers.CompletionRequest) *PerplexityRequest {
	return &PerplexityRequest{
		Model:       req.Model,
		Messages:    p.convertMessages(req.Messages),
		Temperature: req.Temperature,
		TopP:        req.TopP,
		MaxTokens:   req.MaxTokens,
		Stop:        req.Stop,
		Stream:      req.Stream,
		Tools:       p.convertTools(req.Tools),
	}
}

func (p *Provider) convertMessages(messages []providers.Message) []PerplexityMessage {
	result := make([]PerplexityMessage, len(messages))
	for i, msg := range messages {
		result[i] = PerplexityMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}
	return result
}

func (p *Provider) convertTools(tools []providers.Tool) []PerplexityTool {
	result := make([]PerplexityTool, len(tools))
	for i, tool := range tools {
		result[i] = PerplexityTool{
			Type: tool.Type,
			Function: PerplexityFunction{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			},
		}
	}
	return result
}

func (p *Provider) convertToolCalls(calls []PerplexityToolCall) []providers.ToolCall {
	if len(calls) == 0 {
		return nil
	}
	result := make([]providers.ToolCall, len(calls))
	for i, call := range calls {
		result[i] = providers.ToolCall{
			ID:   call.ID,
			Type: call.Type,
			Function: providers.FunctionCall{
				Name:      call.Function.Name,
				Arguments: call.Function.Arguments,
			},
		}
	}
	return result
}

func (p *Provider) handleErrorResponse(statusCode int, body []byte) error {
	var errResp PerplexityError
	if err := json.Unmarshal(body, &errResp); err != nil {
		return providers.NewProviderError(ProviderType, statusCode, "unknown", string(body))
	}

	pErr := providers.NewProviderError(ProviderType, statusCode, errResp.Error.Code, errResp.Error.Message)
	pErr.Type = errResp.Error.Type

	if statusCode == 429 {
		pErr.Retryable = true
	}

	return pErr
}

func (p *Provider) getContextLength(modelID string) int {
	contextLengths := map[string]int{
		"llama-3.1-sonar-small-128k-online": 127072,
		"llama-3.1-sonar-large-128k-online": 127072,
		"llama-3.1-sonar-huge-128k-online":  127072,
		"llama-3.1-8b-instruct":             8192,
		"llama-3.1-70b-instruct":            8192,
		"llama-3-8b-instruct":               8192,
		"llama-3-70b-instruct":              8192,
		"mixtral-8x7b-instruct":             32768,
	}

	if cl, ok := contextLengths[modelID]; ok {
		return cl
	}
	return 8192
}

func (p *Provider) getModelCapabilities(modelID string) []providers.ModelCapability {
	capabilities := []providers.ModelCapability{
		providers.CapabilityText,
		providers.CapabilityStreaming,
	}

	if strings.Contains(modelID, "online") {
		capabilities = append(capabilities, providers.CapabilitySearch)
	}

	if strings.Contains(modelID, "128k") {
		capabilities = append(capabilities, providers.CapabilityLongContext)
	}

	return capabilities
}

// Perplexity types
type PerplexityRequest struct {
	Model       string              `json:"model"`
	Messages    []PerplexityMessage `json:"messages"`
	Temperature float64             `json:"temperature,omitempty"`
	TopP        float64             `json:"top_p,omitempty"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Stop        []string            `json:"stop,omitempty"`
	Stream      bool                `json:"stream,omitempty"`
	Tools       []PerplexityTool    `json:"tools,omitempty"`
}

type PerplexityMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type PerplexityTool struct {
	Type     string             `json:"type"`
	Function PerplexityFunction `json:"function"`
}

type PerplexityFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type PerplexityResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role      string               `json:"role"`
			Content   string               `json:"content"`
			ToolCalls []PerplexityToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type PerplexityStreamChunk struct {
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
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

type PerplexityToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type PerplexityModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type PerplexityError struct {
	Error struct {
		Message string `json:"message"`
		Code    string `json:"code"`
		Type    string `json:"type"`
	} `json:"error"`
}
