package registry

import (
	"fmt"
	"sort"
	"sync"
)

type Registry interface {
	Register(manifest AgentManifest) error
	Update(manifest AgentManifest) error
	Unregister(id string) error
	Get(id string) (AgentManifest, error)
	List() ([]AgentManifest, error)
}

type MemoryRegistry struct {
	mu     sync.RWMutex
	agents map[string]AgentManifest
}

func NewMemoryRegistry() *MemoryRegistry {
	return &MemoryRegistry{agents: make(map[string]AgentManifest)}
}

func (r *MemoryRegistry) Register(manifest AgentManifest) error {
	manifest = manifest.Normalize()
	if manifest.ID == "" || manifest.DID == "" || manifest.Name == "" {
		return fmt.Errorf("id, did and name are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.agents[manifest.ID]; exists {
		return fmt.Errorf("agent already registered: %s", manifest.ID)
	}
	r.agents[manifest.ID] = manifest
	return nil
}

func (r *MemoryRegistry) Update(manifest AgentManifest) error {
	manifest = manifest.Normalize()
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.agents[manifest.ID]; !exists {
		return fmt.Errorf("agent not found: %s", manifest.ID)
	}
	r.agents[manifest.ID] = manifest
	return nil
}

func (r *MemoryRegistry) Unregister(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.agents[id]; !exists {
		return fmt.Errorf("agent not found: %s", id)
	}
	delete(r.agents, id)
	return nil
}

func (r *MemoryRegistry) Get(id string) (AgentManifest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	manifest, exists := r.agents[id]
	if !exists {
		return AgentManifest{}, fmt.Errorf("agent not found: %s", id)
	}
	return manifest, nil
}

func (r *MemoryRegistry) List() ([]AgentManifest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agents := make([]AgentManifest, 0, len(r.agents))
	for _, manifest := range r.agents {
		agents = append(agents, manifest)
	}
	sort.Slice(agents, func(i, j int) bool { return agents[i].Name < agents[j].Name })
	return agents, nil
}

type DHTRegistry struct {
	memory *MemoryRegistry
}

func NewDHTRegistry() *DHTRegistry {
	return &DHTRegistry{memory: NewMemoryRegistry()}
}

func (r *DHTRegistry) Register(manifest AgentManifest) error { return r.memory.Register(manifest) }
func (r *DHTRegistry) Update(manifest AgentManifest) error   { return r.memory.Update(manifest) }
func (r *DHTRegistry) Unregister(id string) error            { return r.memory.Unregister(id) }
func (r *DHTRegistry) Get(id string) (AgentManifest, error)  { return r.memory.Get(id) }
func (r *DHTRegistry) List() ([]AgentManifest, error)        { return r.memory.List() }
