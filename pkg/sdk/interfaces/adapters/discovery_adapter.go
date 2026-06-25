package adapters

import (
	"context"

	"github.com/MortalArena/Musketeers/pkg/discovery"
	"github.com/MortalArena/Musketeers/pkg/registry"
	"github.com/MortalArena/Musketeers/pkg/sdk/interfaces"
)

type DiscoveryAdapter struct {
	disc   discovery.Discovery
	reg    registry.Registry
}

func NewDiscoveryAdapter(disc discovery.Discovery, reg registry.Registry) *DiscoveryAdapter {
	return &DiscoveryAdapter{disc: disc, reg: reg}
}

func (a *DiscoveryAdapter) Index(manifest *interfaces.AgentManifest) error {
	return a.disc.Index(registry.AgentManifest{
		ID:          manifest.ID,
		DID:         manifest.DID,
		Name:        manifest.Name,
		Description: manifest.Description,
		Category:    manifest.Category,
	})
}

func (a *DiscoveryAdapter) Search(query string) ([]*interfaces.AgentManifest, error) {
	results, err := a.disc.Search(discovery.SearchQuery{Text: query})
	if err != nil {
		return nil, err
	}
	out := make([]*interfaces.AgentManifest, len(results))
	for i, r := range results {
		out[i] = &interfaces.AgentManifest{
			ID:          r.ID,
			DID:         r.DID,
			Name:        r.Name,
			Description: r.Description,
			Category:    r.Category,
		}
	}
	return out, nil
}

func (a *DiscoveryAdapter) Recommend(tags ...string) ([]*interfaces.AgentManifest, error) {
	results := a.disc.Recommend(tags...)
	out := make([]*interfaces.AgentManifest, len(results))
	for i, r := range results {
		out[i] = &interfaces.AgentManifest{
			ID:          r.ID,
			DID:         r.DID,
			Name:        r.Name,
			Description: r.Description,
			Category:    r.Category,
		}
	}
	return out, nil
}

func (a *DiscoveryAdapter) FindPeers(ctx context.Context, topic string) ([]string, error) {
	return nil, nil
}

var _ interfaces.DiscoveryInterface = (*DiscoveryAdapter)(nil)
