package google

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
	DefaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"
	ProviderType   = providers.ProviderGoogle
)

// Provider Google provider implementation
type Provider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
	config     providers.ProviderConfig
	status     providers.ProviderStatus
	statusMu   sync.RWMutex
}

// New creates a new Google provider
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
	return "Google"
}

func (p *Provider) Capabilities() providers.ProviderCapabilities {
	return providers.ProviderCapabilities{
		SupportsChat:          true,
		SupportsStreaming:     true,
		SupportsVision:        true,
		SupportsAudio:         true,
		SupportsVideo:         true,
		SupportsImage:         true,
		SupportsEmbeddings:    true,
		SupportsFunctions:     true,
		SupportsReasoning:     true,
		SupportsLongContext:   true,
		SupportsTranscription: true,
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

// setHeaders sets the required headers for Google API requests
func (p *Provider) setHeaders(req *http.Request) {
	req.Header.Set("X-Goog-Api-Key", p.apiKey)
	req.Header.Set("Content-Type", "application/json")
}

func (p *Provider) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	p.setHeaders(req)

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

	gReq := p.convertRequest(req)

	jsonBody, err := json.Marshal(gReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/models/"+req.Model+":generateContent", bytes.NewReader(jsonBody))
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

	var gResp GoogleResponse
	if err := json.Unmarshal(body, &gResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(gResp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	candidate := gResp.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return nil, fmt.Errorf("no content parts in response")
	}

	content := ""
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			content += part.Text
		}
	}

	return &providers.CompletionResponse{
		ID:           gResp.ID,
		Provider:     ProviderType,
		Model:        req.Model,
		Content:      content,
		FinishReason: candidate.FinishReason,
		Usage: providers.TokenUsage{
			PromptTokens:     gResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: gResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      gResp.UsageMetadata.TotalTokenCount,
		},
		Latency: time.Since(startTime),
	}, nil
}

func (p *Provider) StreamComplete(ctx context.Context, req *providers.CompletionRequest, callback providers.StreamingCallback) error {
	gReq := p.convertRequest(req)

	jsonBody, err := json.Marshal(gReq)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/models/"+req.Model+":streamGenerateContent", bytes.NewReader(jsonBody))
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

		var chunk GoogleStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Candidates) > 0 && len(chunk.Candidates[0].Content.Parts) > 0 {
			delta := chunk.Candidates[0].Content.Parts[0].Text

			streamChunk := providers.StreamChunk{
				ID:       chunk.ID,
				Provider: ProviderType,
				Model:    req.Model,
				Delta:    delta,
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
	p.setHeaders(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list models: %d", resp.StatusCode)
	}

	var modelsResp GoogleModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, err
	}

	models := make([]providers.ModelInfo, 0, len(modelsResp.Models))
	for _, m := range modelsResp.Models {
		model := providers.ModelInfo{
			ID:            m.Name,
			Name:          m.DisplayName,
			Provider:      ProviderType,
			Owner:         "Google",
			Description:   m.Description,
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

func (p *Provider) convertRequest(req *providers.CompletionRequest) *GoogleRequest {
	contents := make([]GoogleContent, len(req.Messages))
	for i, msg := range req.Messages {
		contents[i] = GoogleContent{
			Role:  string(msg.Role),
			Parts: []GooglePart{{Text: msg.Content}},
		}
	}

	return &GoogleRequest{
		Contents: contents,
	}
}

func (p *Provider) handleErrorResponse(statusCode int, body []byte) error {
	var errResp GoogleError
	if err := json.Unmarshal(body, &errResp); err != nil {
		return providers.NewProviderError(ProviderType, statusCode, "unknown", string(body))
	}

	pErr := providers.NewProviderError(ProviderType, statusCode, fmt.Sprintf("%d", errResp.Error.Code), errResp.Error.Message)

	if statusCode == 429 {
		pErr.Retryable = true
	}

	return pErr
}

func (p *Provider) getContextLength(modelID string) int {
	contextLengths := map[string]int{
		"gemini-2.5-pro":   2000000,
		"gemini-2.5-flash": 2000000,
		"gemini-2.0-pro":   1000000,
		"gemini-2.0-flash": 1000000,
		"gemini-1.5-pro":   2000000,
		"gemini-1.5-flash": 1000000,
		"gemini-1.0-pro":   1000000,
	}

	for model, cl := range contextLengths {
		if strings.Contains(modelID, model) {
			return cl
		}
	}
	return 128000
}

func (p *Provider) getModelCapabilities(modelID string) []providers.ModelCapability {
	capabilities := []providers.ModelCapability{
		providers.CapabilityText,
		providers.CapabilityStreaming,
	}

	if strings.Contains(modelID, "vision") || strings.Contains(modelID, "pro") {
		capabilities = append(capabilities, providers.CapabilityVision)
	}

	if strings.Contains(modelID, "flash") {
		capabilities = append(capabilities, providers.CapabilityCode)
	}

	if strings.Contains(modelID, "2.5") || strings.Contains(modelID, "2.0") {
		capabilities = append(capabilities, providers.CapabilityReasoning, providers.CapabilityLongContext)
	}

	return capabilities
}

// Google types
type GoogleRequest struct {
	Contents []GoogleContent `json:"contents"`
}

type GoogleContent struct {
	Role  string       `json:"role"`
	Parts []GooglePart `json:"parts"`
}

type GooglePart struct {
	Text string `json:"text"`
}

type GoogleResponse struct {
	ID         string `json:"id"`
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

type GoogleStreamChunk struct {
	ID         string `json:"id"`
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

type GoogleModelsResponse struct {
	Models []struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		Description string `json:"description"`
	} `json:"models"`
}

type GoogleError struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}
