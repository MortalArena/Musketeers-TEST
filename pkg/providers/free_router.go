package providers

import (
	"context"
	"fmt"
	"time"
)

// FreeRouterConfig configures the free router
type FreeRouterConfig struct {
	PreferLocal    bool
	MaxRetries     int
	Timeout        time.Duration
	EnableFallback bool
	Tracker        *FreeModelsTracker
}

// DefaultFreeRouterConfig returns default configuration
func DefaultFreeRouterConfig() *FreeRouterConfig {
	return &FreeRouterConfig{
		PreferLocal:    true,
		MaxRetries:     3,
		Timeout:        30 * time.Second,
		EnableFallback: true,
	}
}

// FreeRouter routes requests only to free models
type FreeRouter struct {
	config   *FreeRouterConfig
	registry *ProviderRegistry
	tracker  *FreeModelsTracker
	catalog  *ModelCatalog
}

// NewFreeRouter creates a new free router
func NewFreeRouter(config *FreeRouterConfig, registry *ProviderRegistry, tracker *FreeModelsTracker, catalog *ModelCatalog) *FreeRouter {
	if config == nil {
		config = DefaultFreeRouterConfig()
	}

	return &FreeRouter{
		config:   config,
		registry: registry,
		tracker:  tracker,
		catalog:  catalog,
	}
}

// Complete routes a completion request to a free model
func (r *FreeRouter) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	// Select best free model
	modelID, err := r.selectFreeModel(req)
	if err != nil {
		return nil, fmt.Errorf("failed to select free model: %w", err)
	}

	// Update request with selected model
	req.Model = modelID

	// Get provider for the model
	model, exists := r.catalog.GetModel(modelID)
	if !exists {
		return nil, fmt.Errorf("model not found: %s", modelID)
	}

	provider, exists := r.registry.Get(model.Provider)
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", model.Provider)
	}

	// Execute with retry
	return r.executeWithRetry(ctx, provider, req, modelID)
}

// StreamComplete routes a streaming completion request to a free model
func (r *FreeRouter) StreamComplete(ctx context.Context, req *CompletionRequest, callback StreamingCallback) error {
	// Select best free model
	modelID, err := r.selectFreeModel(req)
	if err != nil {
		return fmt.Errorf("failed to select free model: %w", err)
	}

	// Update request with selected model
	req.Model = modelID

	// Get provider for the model
	model, exists := r.catalog.GetModel(modelID)
	if !exists {
		return fmt.Errorf("model not found: %s", modelID)
	}

	provider, exists := r.registry.Get(model.Provider)
	if !exists {
		return fmt.Errorf("provider not found: %s", model.Provider)
	}

	// Execute with retry
	return r.executeStreamWithRetry(ctx, provider, req, callback, modelID)
}

// selectFreeModel selects the best free model for the request
func (r *FreeRouter) selectFreeModel(req *CompletionRequest) (string, error) {
	// If model is specified, check if it's free
	if req.Model != "" {
		if r.tracker != nil && r.tracker.IsFreeModel(req.Model) {
			return req.Model, nil
		}
		return "", fmt.Errorf("specified model is not free: %s", req.Model)
	}

	// Get best free model from tracker
	if r.tracker != nil {
		if modelID, exists := r.tracker.GetBestFreeModel(); exists {
			return modelID, nil
		}
	}

	// Fallback to catalog
	if r.catalog != nil {
		freeModels := r.catalog.GetFreeModels()
		if len(freeModels) > 0 {
			// Prefer local models if configured
			if r.config.PreferLocal {
				for _, model := range freeModels {
					for _, cat := range model.Categories {
						if cat == "local" {
							return model.ID, nil
						}
					}
				}
			}
			// Return first free model
			return freeModels[0].ID, nil
		}
	}

	return "", fmt.Errorf("no free models available")
}

// executeWithRetry executes completion with retry logic
func (r *FreeRouter) executeWithRetry(ctx context.Context, provider Provider, req *CompletionRequest, modelID string) (*CompletionResponse, error) {
	var lastErr error
	startTime := time.Now()

	for attempt := 0; attempt < r.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Add delay before retry
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		// Set timeout if configured
		if r.config.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, r.config.Timeout)
			defer cancel()
		}

		resp, err := provider.Complete(ctx, req)
		if err == nil {
			// Record successful usage
			if r.tracker != nil {
				tokens := resp.Usage.TotalTokens
				r.tracker.RecordUsage(modelID, tokens, resp.Latency, true)
			}
			return resp, nil
		}

		lastErr = err

		// Check if error is retryable
		if pErr, ok := err.(*ProviderError); ok && !pErr.IsRetryable() {
			break
		}
	}

	// Record failed usage
	if r.tracker != nil {
		r.tracker.RecordUsage(modelID, 0, time.Since(startTime), false)
	}

	return nil, lastErr
}

// executeStreamWithRetry executes streaming completion with retry logic
func (r *FreeRouter) executeStreamWithRetry(ctx context.Context, provider Provider, req *CompletionRequest, callback StreamingCallback, modelID string) error {
	var lastErr error
	startTime := time.Now()

	for attempt := 0; attempt < r.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Add delay before retry
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		// Set timeout if configured
		if r.config.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, r.config.Timeout)
			defer cancel()
		}

		err := provider.StreamComplete(ctx, req, callback)
		if err == nil {
			// Record successful usage (streaming doesn't provide token count)
			if r.tracker != nil {
				r.tracker.RecordUsage(modelID, 0, time.Since(startTime), true)
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if pErr, ok := err.(*ProviderError); ok && !pErr.IsRetryable() {
			break
		}
	}

	// Record failed usage
	if r.tracker != nil {
		r.tracker.RecordUsage(modelID, 0, time.Since(startTime), false)
	}

	return lastErr
}

// GetFreeModels returns all available free models
func (r *FreeRouter) GetFreeModels() []ModelInfo {
	if r.catalog == nil {
		return []ModelInfo{}
	}
	return r.catalog.GetFreeModels()
}

// GetStats returns usage statistics
func (r *FreeRouter) GetStats() map[string]interface{} {
	if r.tracker == nil {
		return map[string]interface{}{}
	}
	return r.tracker.GetStats()
}

// UpdateConfig updates the router configuration
func (r *FreeRouter) UpdateConfig(config *FreeRouterConfig) {
	if config != nil {
		r.config = config
	}
}
