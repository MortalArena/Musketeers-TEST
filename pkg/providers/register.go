package providers

// ProviderRegistry manages all available providers
type ProviderRegistry struct {
	providers map[ProviderType]Provider
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	registry := &ProviderRegistry{
		providers: make(map[ProviderType]Provider),
	}

	return registry
}

// Register registers a provider with the given type
func (r *ProviderRegistry) Register(providerType ProviderType, provider Provider) {
	r.providers[providerType] = provider
}

// Get returns a provider by type
func (r *ProviderRegistry) Get(providerType ProviderType) (Provider, bool) {
	provider, exists := r.providers[providerType]
	return provider, exists
}

// List returns all registered providers
func (r *ProviderRegistry) List() []Provider {
	providers := make([]Provider, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}
	return providers
}

// ListByType returns all provider types
func (r *ProviderRegistry) ListByType() []ProviderType {
	types := make([]ProviderType, 0, len(r.providers))
	for providerType := range r.providers {
		types = append(types, providerType)
	}
	return types
}

// GetProviderByName returns a provider by name
func (r *ProviderRegistry) GetProviderByName(name string) (Provider, bool) {
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
