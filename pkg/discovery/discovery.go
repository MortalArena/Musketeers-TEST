package discovery

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/registry"
)

type Discovery interface {
	Index(manifest registry.AgentManifest) error
	Search(query SearchQuery) ([]registry.AgentManifest, error)
	Categorize() map[string]int
	Recommend(tags ...string) []registry.AgentManifest
}

type SearchQuery struct {
	Text       string
	Category   string
	Capability string
	Limit      int
}

type IndexedDiscovery struct {
	mu     sync.RWMutex
	agents map[string]registry.AgentManifest
	tokens map[string]map[string]bool
}

func NewIndexedDiscovery() *IndexedDiscovery {
	return &IndexedDiscovery{agents: make(map[string]registry.AgentManifest), tokens: make(map[string]map[string]bool)}
}

func (d *IndexedDiscovery) Index(manifest registry.AgentManifest) error {
	if manifest.ID == "" {
		return fmt.Errorf("manifest id is required")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.agents[manifest.ID] = manifest
	d.indexTokens(manifest.ID, manifest)
	return nil
}

func (d *IndexedDiscovery) Search(query SearchQuery) ([]registry.AgentManifest, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	query.Text = strings.ToLower(strings.TrimSpace(query.Text))
	query.Category = strings.ToLower(strings.TrimSpace(query.Category))
	query.Capability = strings.ToLower(strings.TrimSpace(query.Capability))
	results := make([]registry.AgentManifest, 0)
	for _, manifest := range d.agents {
		if query.Category != "" && strings.ToLower(manifest.Category) != query.Category {
			continue
		}
		if query.Capability != "" && !manifestHasCapability(manifest, query.Capability) {
			continue
		}
		if query.Text != "" && !manifestMatchesText(manifest, query.Text) {
			continue
		}
		results = append(results, manifest)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })
	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}
	return results, nil
}

func (d *IndexedDiscovery) Categorize() map[string]int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	counts := make(map[string]int)
	for _, manifest := range d.agents {
		category := manifest.Category
		if category == "" {
			category = "uncategorized"
		}
		counts[category]++
	}
	return counts
}

func (d *IndexedDiscovery) Recommend(tags ...string) []registry.AgentManifest {
	d.mu.RLock()
	defer d.mu.RUnlock()
	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[strings.ToLower(strings.TrimSpace(tag))] = true
	}
	results := make([]registry.AgentManifest, 0)
	for id, manifest := range d.agents {
		if len(tagSet) == 0 || manifestMatchesTokens(d.tokens[id], tagSet) {
			results = append(results, manifest)
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })
	return results
}

func (d *IndexedDiscovery) indexTokens(id string, manifest registry.AgentManifest) {
	tokens := make(map[string]bool)
	addToken := func(value string) {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			return
		}
		for _, part := range strings.Fields(value) {
			tokens[part] = true
		}
	}
	addToken(manifest.Name)
	addToken(manifest.Description)
	addToken(manifest.Category)
	for _, capability := range manifest.Capabilities {
		addToken(capability.Name)
		addToken(capability.Description)
		for _, input := range capability.Inputs {
			addToken(input)
		}
	}
	for _, task := range manifest.Tasks {
		addToken(task.Name)
		addToken(task.Description)
	}
	d.tokens[id] = tokens
}

func manifestMatchesText(manifest registry.AgentManifest, text string) bool {
	haystack := strings.ToLower(strings.Join([]string{manifest.Name, manifest.Description, manifest.Category}, " "))
	return strings.Contains(haystack, text)
}

func manifestHasCapability(manifest registry.AgentManifest, capability string) bool {
	capability = strings.ToLower(capability)
	for _, item := range manifest.Capabilities {
		if strings.ToLower(item.Name) == capability {
			return true
		}
	}
	return false
}

func manifestMatchesTokens(tokens map[string]bool, required map[string]bool) bool {
	for token := range required {
		if !tokens[token] {
			return false
		}
	}
	return true
}
