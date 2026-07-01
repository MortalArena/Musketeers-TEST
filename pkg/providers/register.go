package providers

import (
	"context"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/lifecycle"
)

// ProviderRegistry manages all available providers
type ProviderRegistry struct {
	providers map[ProviderType]Provider
	mu        sync.RWMutex
	lifecycle *lifecycle.LifecycleMixin
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	registry := &ProviderRegistry{
		providers: make(map[ProviderType]Provider),
		lifecycle: lifecycle.NewLifecycleMixin(),
	}

	return registry
}

// Start يبدأ ProviderRegistry
func (r *ProviderRegistry) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lifecycle.SetStatus(lifecycle.LifecycleStatusStarting)
	r.lifecycle.SetStatus(lifecycle.LifecycleStatusRunning)
	return nil
}

// Stop يوقف ProviderRegistry
func (r *ProviderRegistry) Stop(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lifecycle.SetStatus(lifecycle.LifecycleStatusStopping)
	r.lifecycle.SetStatus(lifecycle.LifecycleStatusStopped)
	return nil
}

// Close يغلق ProviderRegistry
func (r *ProviderRegistry) Close() error {
	return r.Stop(r.lifecycle.Context())
}

// Shutdown يوقف ProviderRegistry بشكل آمن
func (r *ProviderRegistry) Shutdown(ctx context.Context) error {
	return r.Stop(ctx)
}

// Cancel يلغي العمليات الجارية
func (r *ProviderRegistry) Cancel() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lifecycle.CancelContext()
	return nil
}

// IsRunning يتحقق مما إذا كان يعمل
func (r *ProviderRegistry) IsRunning() bool {
	return r.lifecycle.IsRunningMixin()
}

// Status يرجع الحالة
func (r *ProviderRegistry) Status() lifecycle.LifecycleStatus {
	return r.lifecycle.GetStatus()
}

// Register registers a provider with the given type
func (r *ProviderRegistry) Register(providerType ProviderType, provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[providerType] = provider
}

// Get returns a provider by type
func (r *ProviderRegistry) Get(providerType ProviderType) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	provider, exists := r.providers[providerType]
	return provider, exists
}

// List returns all registered providers
func (r *ProviderRegistry) List() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	providers := make([]Provider, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}
	return providers
}

// ListByType returns all provider types
func (r *ProviderRegistry) ListByType() []ProviderType {
	r.mu.RLock()
	defer r.mu.RUnlock()
	types := make([]ProviderType, 0, len(r.providers))
	for providerType := range r.providers {
		types = append(types, providerType)
	}
	return types
}

// GetProviderByName returns a provider by name
func (r *ProviderRegistry) GetProviderByName(name string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, provider := range r.providers {
		if provider.Name() == name {
			return provider, true
		}
	}
	return nil, false
}

// Global registry instance
var globalRegistry = NewProviderRegistry()

// GetProvider returns a provider from the global registry
func GetProvider(providerType ProviderType) (Provider, bool) {
	return globalRegistry.Get(providerType)
}

// GlobalRegistry returns the global registry instance
func GlobalRegistry() *ProviderRegistry {
	return globalRegistry
}

// RegisterProvider registers a provider in the global registry
func RegisterProvider(providerType ProviderType, provider Provider) {
	globalRegistry.Register(providerType, provider)
}

// ListProviders returns all providers from the global registry
func ListProviders() []Provider {
	return globalRegistry.List()
}

// ListProviderTypes returns all provider types from the global registry
func ListProviderTypes() []ProviderType {
	return globalRegistry.ListByType()
}
