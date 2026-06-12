package capability

import (
	"context"
	"fmt"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/policy"
)

type Manager struct {
	mu           sync.RWMutex
	capabilities map[string]Capability
	policy       *policy.Engine
}

func NewManager(engine *policy.Engine) *Manager {
	if engine == nil {
		engine = policy.NewEngine()
	}
	return &Manager{capabilities: make(map[string]Capability), policy: engine}
}

func (m *Manager) Register(capability Capability) error {
	if capability == nil {
		return fmt.Errorf("capability is nil")
	}
	name := capability.Name()
	if name == "" {
		return fmt.Errorf("capability name is required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.capabilities[name]; exists {
		return fmt.Errorf("capability already registered: %s", name)
	}
	m.capabilities[name] = capability
	return nil
}

func (m *Manager) Execute(ctx context.Context, principal policy.Principal, cmd Command) (*Result, error) {
	if cmd == nil {
		return nil, fmt.Errorf("command is nil")
	}
	name := cmd.Name()
	m.mu.RLock()
	capability, exists := m.capabilities[name]
	m.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("capability not registered: %s", name)
	}
	return capability.Execute(ctx, principal, cmd)
}

func (m *Manager) Names() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.capabilities))
	for name := range m.capabilities {
		names = append(names, name)
	}
	return names
}

func (m *Manager) Policy() *policy.Engine {
	return m.policy
}
