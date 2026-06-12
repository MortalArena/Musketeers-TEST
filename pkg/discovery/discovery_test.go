package discovery

import (
	"testing"

	"github.com/MortalArena/Musketeers/pkg/registry"
)

func TestIndexedDiscoverySearchCategorizeRecommend(t *testing.T) {
	discovery := NewIndexedDiscovery()
	if err := discovery.Index(registry.AgentManifest{ID: "agent-1", DID: "did:ia:one", Name: "Code Review Agent", Category: "coding", Capabilities: []registry.CapabilityManifest{{Name: "review"}, {Name: "git"}}}); err != nil {
		t.Fatalf("Index returned error: %v", err)
	}
	if err := discovery.Index(registry.AgentManifest{ID: "agent-2", DID: "did:ia:two", Name: "Email Agent", Category: "messaging", Capabilities: []registry.CapabilityManifest{{Name: "email"}}}); err != nil {
		t.Fatalf("Index returned error: %v", err)
	}
	results, err := discovery.Search(SearchQuery{Text: "code review"})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 1 || results[0].ID != "agent-1" {
		t.Fatalf("unexpected results: %#v", results)
	}
	categories := discovery.Categorize()
	if categories["coding"] != 1 || categories["messaging"] != 1 {
		t.Fatalf("unexpected categories: %#v", categories)
	}
	recommendations := discovery.Recommend("git")
	if len(recommendations) != 1 || recommendations[0].ID != "agent-1" {
		t.Fatalf("unexpected recommendations: %#v", recommendations)
	}
}
