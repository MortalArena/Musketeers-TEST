package providers

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

// RouterConfig configuration for the smart router
type RouterConfig struct {
	PreferFreeModels    bool
	PreferLocalModels   bool
	MaxRetries          int
	Timeout             time.Duration
	FallbackEnabled     bool
	CostOptimization    bool
	LatencyOptimization bool
}

// Router smart router for intelligent model selection
type Router struct {
	registry *ProviderRegistry
	config   RouterConfig
	logger   *zap.Logger

	usageTracker map[string]*UsageStats
	usageMu      sync.RWMutex

	modelCache map[string][]ModelInfo
	cacheMu    sync.RWMutex
}

// UsageStats tracks usage statistics for models
type UsageStats struct {
	ModelID            string
	Provider           ProviderType
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	TotalTokens        int64
	TotalCost          float64
	AvgLatency         time.Duration
	LastUsed           time.Time
}

// NewRouter creates a new smart router
func NewRouter(registry *ProviderRegistry, config RouterConfig) *Router {
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &Router{
		registry:     registry,
		config:       config,
		logger:       zap.NewNop(),
		usageTracker: make(map[string]*UsageStats),
		modelCache:   make(map[string][]ModelInfo),
	}
}

// Route intelligently routes a completion request to the best available model
func (r *Router) Route(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	candidates, err := r.findCandidateModels(req)
	if err != nil {
		return nil, fmt.Errorf("failed to find candidate models: %w", err)
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no available models for request")
	}

	sortedCandidates := r.rankCandidates(candidates, req)

	var lastErr error
	for i, candidate := range sortedCandidates {
		if i > 0 && !r.config.FallbackEnabled {
			break
		}

		provider, exists := r.registry.Get(candidate.Provider)
		if !exists {
			continue
		}

		if !provider.IsAvailable() {
			continue
		}

		resp, err := r.executeWithRetry(ctx, provider, req, candidate.ID)
		if err != nil {
			lastErr = err
			r.recordFailure(candidate.ID, candidate.Provider, err)
			continue
		}

		r.recordSuccess(candidate.ID, candidate.Provider, resp)
		return resp, nil
	}

	return nil, fmt.Errorf("all routing attempts failed: %w", lastErr)
}

// RouteStream intelligently routes a streaming completion request
func (r *Router) RouteStream(ctx context.Context, req *CompletionRequest, callback StreamingCallback) error {
	candidates, err := r.findCandidateModels(req)
	if err != nil {
		return fmt.Errorf("failed to find candidate models: %w", err)
	}

	if len(candidates) == 0 {
		return fmt.Errorf("no available models for request")
	}

	sortedCandidates := r.rankCandidates(candidates, req)

	var lastErr error
	for i, candidate := range sortedCandidates {
		if i > 0 && !r.config.FallbackEnabled {
			break
		}

		provider, exists := r.registry.Get(candidate.Provider)
		if !exists {
			continue
		}

		if !provider.IsAvailable() {
			continue
		}

		err := r.executeStreamWithRetry(ctx, provider, req, candidate.ID, callback)
		if err != nil {
			lastErr = err
			r.recordFailure(candidate.ID, candidate.Provider, err)
			continue
		}

		return nil
	}

	return fmt.Errorf("all routing attempts failed: %w", lastErr)
}

// findCandidateModels finds all models that can handle the request
func (r *Router) findCandidateModels(req *CompletionRequest) ([]ModelInfo, error) {
	var candidates []ModelInfo

	providers := r.registry.List()
	for _, provider := range providers {
		if !provider.IsAvailable() {
			continue
		}

		models, err := provider.ListModels(context.Background())
		if err != nil {
			continue
		}

		for _, model := range models {
			if !model.IsAvailable {
				continue
			}

			if !r.modelMatchesRequirements(model, req) {
				continue
			}

			candidates = append(candidates, model)
		}
	}

	return candidates, nil
}

// modelMatchesRequirements checks if a model matches the request requirements
func (r *Router) modelMatchesRequirements(model ModelInfo, req *CompletionRequest) bool {
	requiredCaps := r.getRequiredCapabilities(req)

	for _, requiredCap := range requiredCaps {
		hasCapability := false
		for _, cap := range model.Capabilities {
			if cap == requiredCap {
				hasCapability = true
				break
			}
		}
		if !hasCapability {
			return false
		}
	}

	if req.MaxTokens > 0 && model.ContextLength < req.MaxTokens {
		return false
	}

	return true
}

// getRequiredCapabilities extracts required capabilities from the request
func (r *Router) getRequiredCapabilities(req *CompletionRequest) []ModelCapability {
	var caps []ModelCapability

	if req.Stream {
		caps = append(caps, CapabilityStreaming)
	}

	for _, msg := range req.Messages {
		for _, part := range msg.MultiModal {
			if part.Type == "image_url" {
				caps = append(caps, CapabilityVision)
			}
			if part.Type == "audio" {
				caps = append(caps, CapabilityAudio)
			}
		}
	}

	return caps
}

// rankCandidates ranks candidate models based on router configuration
func (r *Router) rankCandidates(candidates []ModelInfo, req *CompletionRequest) []ModelInfo {
	sorted := make([]ModelInfo, len(candidates))
	copy(sorted, candidates)

	sort.Slice(sorted, func(i, j int) bool {
		scoreI := r.calculateScore(sorted[i], req)
		scoreJ := r.calculateScore(sorted[j], req)
		return scoreI > scoreJ
	})

	return sorted
}

// calculateScore calculates a routing score for a model
func (r *Router) calculateScore(model ModelInfo, req *CompletionRequest) float64 {
	score := 0.0

	if r.config.PreferFreeModels {
		for _, category := range model.Categories {
			if category == "free" {
				score += 100
				break
			}
		}
	}

	if r.config.PreferLocalModels {
		for _, category := range model.Categories {
			if category == "local" {
				score += 80
				break
			}
		}
	}

	if r.config.CostOptimization {
		if model.PriceInput > 0 && model.PriceOutput > 0 {
			costScore := 1.0 / (model.PriceInput + model.PriceOutput)
			score += costScore * 50
		}
	}

	if r.config.LatencyOptimization {
		stats := r.getUsageStats(model.ID)
		if stats != nil && stats.AvgLatency > 0 {
			latencyScore := 1.0 / float64(stats.AvgLatency.Milliseconds())
			score += latencyScore * 30
		}
	}

	successRate := r.getSuccessRate(model.ID)
	score += successRate * 20

	if model.ContextLength >= 100000 {
		score += 10
	}

	return score
}

// executeWithRetry executes a completion with retry logic
func (r *Router) executeWithRetry(ctx context.Context, provider Provider, req *CompletionRequest, modelID string) (*CompletionResponse, error) {
	req.Model = modelID

	for attempt := 0; attempt < r.config.MaxRetries; attempt++ {
		ctxWithTimeout, cancel := context.WithTimeout(ctx, r.config.Timeout)

		resp, err := provider.Complete(ctxWithTimeout, req)
		cancel()

		if err == nil {
			return resp, nil
		}

		if attempt < r.config.MaxRetries-1 {
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}

	return nil, fmt.Errorf("max retries exceeded")
}

// executeStreamWithRetry executes a streaming completion with retry logic
func (r *Router) executeStreamWithRetry(ctx context.Context, provider Provider, req *CompletionRequest, modelID string, callback StreamingCallback) error {
	req.Model = modelID

	for attempt := 0; attempt < r.config.MaxRetries; attempt++ {
		ctxWithTimeout, cancel := context.WithTimeout(ctx, r.config.Timeout)

		err := provider.StreamComplete(ctxWithTimeout, req, callback)
		cancel()

		if err == nil {
			return nil
		}

		if attempt < r.config.MaxRetries-1 {
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}

	return fmt.Errorf("max retries exceeded")
}

// recordSuccess records a successful request
func (r *Router) recordSuccess(modelID string, providerType ProviderType, resp *CompletionResponse) {
	r.usageMu.Lock()
	defer r.usageMu.Unlock()

	stats, exists := r.usageTracker[modelID]
	if !exists {
		stats = &UsageStats{
			ModelID:  modelID,
			Provider: providerType,
		}
		r.usageTracker[modelID] = stats
	}

	stats.TotalRequests++
	stats.SuccessfulRequests++
	stats.TotalTokens += int64(resp.Usage.TotalTokens)
	stats.AvgLatency = (stats.AvgLatency*time.Duration(stats.TotalRequests-1) + resp.Latency) / time.Duration(stats.TotalRequests)
	stats.LastUsed = time.Now()
}

// recordFailure records a failed request
func (r *Router) recordFailure(modelID string, providerType ProviderType, err error) {
	r.usageMu.Lock()
	defer r.usageMu.Unlock()

	stats, exists := r.usageTracker[modelID]
	if !exists {
		stats = &UsageStats{
			ModelID:  modelID,
			Provider: providerType,
		}
		r.usageTracker[modelID] = stats
	}

	stats.TotalRequests++
	stats.FailedRequests++
	stats.LastUsed = time.Now()
}

// getUsageStats returns usage statistics for a model
func (r *Router) getUsageStats(modelID string) *UsageStats {
	r.usageMu.RLock()
	defer r.usageMu.RUnlock()

	return r.usageTracker[modelID]
}

// getSuccessRate returns the success rate for a model
func (r *Router) getSuccessRate(modelID string) float64 {
	stats := r.getUsageStats(modelID)
	if stats == nil || stats.TotalRequests == 0 {
		return 0.5
	}

	return float64(stats.SuccessfulRequests) / float64(stats.TotalRequests)
}

// GetUsageStats returns all usage statistics
func (r *Router) GetUsageStats() map[string]*UsageStats {
	r.usageMu.RLock()
	defer r.usageMu.RUnlock()

	result := make(map[string]*UsageStats)
	for k, v := range r.usageTracker {
		result[k] = v
	}
	return result
}

// GetFreeModels returns all free models
func (r *Router) GetFreeModels(ctx context.Context) ([]ModelInfo, error) {
	var freeModels []ModelInfo

	providers := r.registry.List()
	for _, provider := range providers {
		if !provider.IsAvailable() {
			continue
		}

		models, err := provider.ListModels(ctx)
		if err != nil {
			continue
		}

		for _, model := range models {
			for _, category := range model.Categories {
				if category == "free" {
					freeModels = append(freeModels, model)
					break
				}
			}
		}
	}

	return freeModels, nil
}

// GetLocalModels returns all local models
func (r *Router) GetLocalModels(ctx context.Context) ([]ModelInfo, error) {
	var localModels []ModelInfo

	providers := r.registry.List()
	for _, provider := range providers {
		if !provider.IsAvailable() {
			continue
		}

		models, err := provider.ListModels(ctx)
		if err != nil {
			continue
		}

		for _, model := range models {
			for _, category := range model.Categories {
				if category == "local" {
					localModels = append(localModels, model)
					break
				}
			}
		}
	}

	return localModels, nil
}

// ClearCache clears the model cache
func (r *Router) ClearCache() {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	r.modelCache = make(map[string][]ModelInfo)
}

// UpdateConfig updates the router configuration
func (r *Router) UpdateConfig(config RouterConfig) {
	r.config = config
}
