package recraft

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/providers"
	"go.uber.org/zap"
)

const (
	DefaultBaseURL = "https://api.recraft.ai/v1"
	ProviderType   = providers.ProviderRecraft
)

// Provider Recraft provider implementation
type Provider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
	config     providers.ProviderConfig
	status     providers.ProviderStatus
	statusMu   sync.RWMutex
}

// New creates a new Recraft provider
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
	return "Recraft"
}

func (p *Provider) Capabilities() providers.ProviderCapabilities {
	return providers.ProviderCapabilities{
		SupportsImage: true,
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

	rReq := p.convertRequest(req)

	jsonBody, err := json.Marshal(rReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/images/generations", bytes.NewReader(jsonBody))
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

	var rResp RecraftResponse
	if err := json.Unmarshal(body, &rResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &providers.CompletionResponse{
		ID:       rResp.ID,
		Provider: ProviderType,
		Model:    req.Model,
		Content:  rResp.ImageURL,
		Usage: providers.TokenUsage{
			TotalTokens: 1,
		},
		Latency: time.Since(startTime),
	}, nil
}

func (p *Provider) StreamComplete(ctx context.Context, req *providers.CompletionRequest, callback providers.StreamingCallback) error {
	return fmt.Errorf("streaming not supported for image generation")
}

func (p *Provider) ListModels(ctx context.Context) ([]providers.ModelInfo, error) {
	models := []providers.ModelInfo{
		{
			ID:            "recraft-v3",
			Name:          "Recraft v3",
			Provider:      ProviderType,
			Owner:         "Recraft",
			Description:   "Image generation model",
			ContextLength: 0,
			IsAvailable:   true,
			Capabilities: []providers.ModelCapability{
				providers.CapabilityImage,
			},
			Categories: []string{"image"},
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
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
}

func (p *Provider) convertRequest(req *providers.CompletionRequest) *RecraftRequest {
	return &RecraftRequest{
		Model:  req.Model,
		Prompt: req.Messages[0].Content,
	}
}

func (p *Provider) handleErrorResponse(statusCode int, body []byte) error {
	var errResp RecraftError
	if err := json.Unmarshal(body, &errResp); err != nil {
		return providers.NewProviderError(ProviderType, statusCode, "unknown", string(body))
	}

	pErr := providers.NewProviderError(ProviderType, statusCode, errResp.Error.Code, errResp.Error.Message)

	if statusCode == 429 {
		pErr.Retryable = true
	}

	return pErr
}

// Recraft types
type RecraftRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type RecraftResponse struct {
	ID       string `json:"id"`
	ImageURL string `json:"image_url"`
}

type RecraftError struct {
	Error struct {
		Message string `json:"message"`
		Code    string `json:"code"`
		Type    string `json:"type"`
	} `json:"error"`
}
