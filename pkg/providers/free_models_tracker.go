package providers

import (
	"sync"
	"time"
)

// FreeModelUsage tracks usage statistics for free models
type FreeModelUsage struct {
	ModelID        string
	Provider      ProviderType
	RequestCount  int
	TokenCount    int
	LastUsed      time.Time
	SuccessCount  int
	FailureCount  int
	AverageLatency time.Duration
}

// FreeModelsTracker tracks usage and availability of free models
type FreeModelsTracker struct {
	usage      map[string]*FreeModelUsage
	models     map[string]bool // modelID -> isFree
	mu         sync.RWMutex
	catalog    *ModelCatalog
}

// NewFreeModelsTracker creates a new free models tracker
func NewFreeModelsTracker(catalog *ModelCatalog) *FreeModelsTracker {
	tracker := &FreeModelsTracker{
		usage:   make(map[string]*FreeModelUsage),
		models:  make(map[string]bool),
		catalog: catalog,
	}
	
	tracker.loadFreeModels()
	return tracker
}

// loadFreeModels loads free models from the catalog
func (t *FreeModelsTracker) loadFreeModels() {
	if t.catalog == nil {
		return
	}
	
	freeModels := t.catalog.GetFreeModels()
	t.mu.Lock()
	defer t.mu.Unlock()
	
	for _, model := range freeModels {
		t.models[model.ID] = true
		if _, exists := t.usage[model.ID]; !exists {
			t.usage[model.ID] = &FreeModelUsage{
				ModelID:   model.ID,
				Provider: model.Provider,
			}
		}
	}
}

// IsFreeModel checks if a model is free
func (t *FreeModelsTracker) IsFreeModel(modelID string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	return t.models[modelID]
}

// GetFreeModels returns all free model IDs
func (t *FreeModelsTracker) GetFreeModels() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	models := make([]string, 0, len(t.models))
	for modelID := range t.models {
		models = append(models, modelID)
	}
	return models
}

// RecordUsage records usage of a free model
func (t *FreeModelsTracker) RecordUsage(modelID string, tokens int, latency time.Duration, success bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	usage, exists := t.usage[modelID]
	if !exists {
		usage = &FreeModelUsage{
			ModelID: modelID,
		}
		t.usage[modelID] = usage
	}
	
	usage.RequestCount++
	usage.TokenCount += tokens
	usage.LastUsed = time.Now()
	
	if success {
		usage.SuccessCount++
	} else {
		usage.FailureCount++
	}
	
	// Update average latency
	if latency > 0 {
		if usage.AverageLatency == 0 {
			usage.AverageLatency = latency
		} else {
			usage.AverageLatency = (usage.AverageLatency + latency) / 2
		}
	}
}

// GetUsage returns usage statistics for a model
func (t *FreeModelsTracker) GetUsage(modelID string) (*FreeModelUsage, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	usage, exists := t.usage[modelID]
	if !exists {
		return nil, false
	}
	
	// Return a copy to avoid race conditions
	copy := *usage
	return &copy, true
}

// GetAllUsage returns usage statistics for all free models
func (t *FreeModelsTracker) GetAllUsage() map[string]*FreeModelUsage {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	result := make(map[string]*FreeModelUsage)
	for modelID, usage := range t.usage {
		copy := *usage
		result[modelID] = &copy
	}
	return result
}

// GetBestFreeModel returns the best free model based on usage statistics
func (t *FreeModelsTracker) GetBestFreeModel() (string, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	var bestModel string
	var bestScore float64
	
	for modelID, usage := range t.usage {
		if !t.models[modelID] {
			continue
		}
		
		// Calculate score based on success rate and latency
		if usage.RequestCount == 0 {
			continue
		}
		
		successRate := float64(usage.SuccessCount) / float64(usage.RequestCount)
		latencyScore := 1.0
		if usage.AverageLatency > 0 {
			latencyScore = 1.0 / float64(usage.AverageLatency.Milliseconds())
		}
		
		score := successRate * 0.7 + latencyScore * 0.3
		
		if score > bestScore {
			bestScore = score
			bestModel = modelID
		}
	}
	
	if bestModel == "" {
		// Return first free model if no usage data
		for modelID := range t.models {
			return modelID, true
		}
		return "", false
	}
	
	return bestModel, true
}

// ResetUsage resets usage statistics for a model
func (t *FreeModelsTracker) ResetUsage(modelID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if usage, exists := t.usage[modelID]; exists {
		usage.RequestCount = 0
		usage.TokenCount = 0
		usage.SuccessCount = 0
		usage.FailureCount = 0
		usage.AverageLatency = 0
	}
}

// ClearAll clears all usage data
func (t *FreeModelsTracker) ClearAll() {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.usage = make(map[string]*FreeModelUsage)
	t.loadFreeModels()
}

// Refresh reloads free models from the catalog
func (t *FreeModelsTracker) Refresh() {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.models = make(map[string]bool)
	t.loadFreeModels()
}

// GetStats returns overall statistics
func (t *FreeModelsTracker) GetStats() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	totalRequests := 0
	totalTokens := 0
	totalSuccess := 0
	totalFailures := 0
	
	for _, usage := range t.usage {
		totalRequests += usage.RequestCount
		totalTokens += usage.TokenCount
		totalSuccess += usage.SuccessCount
		totalFailures += usage.FailureCount
	}
	
	return map[string]interface{}{
		"free_models_count": len(t.models),
		"total_requests":    totalRequests,
		"total_tokens":      totalTokens,
		"total_success":     totalSuccess,
		"total_failures":    totalFailures,
	}
}
