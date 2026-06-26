package capability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/policy"
	"go.uber.org/zap"
)

type PolicyMode int

const (
	PolicyModeDisabled PolicyMode = iota
	PolicyModeAudit
	PolicyModeEnforce
)

type Manager struct {
	mu           sync.RWMutex
	capabilities map[string]Capability
	policy       *policy.Engine
	policyMode   PolicyMode
	logger       *zap.Logger
}

func NewManager(engine *policy.Engine) *Manager {
	if engine == nil {
		engine = policy.NewEngine()
	}
	return &Manager{
		capabilities: make(map[string]Capability),
		policy:       engine,
		policyMode:   PolicyModeDisabled,
		logger:       zap.NewNop(),
	}
}

func (m *Manager) SetPolicyMode(mode PolicyMode) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.policyMode = mode
}

func (m *Manager) PolicyMode() PolicyMode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.policyMode
}

func (m *Manager) SetLogger(logger *zap.Logger) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if logger == nil {
		logger = zap.NewNop()
	}
	m.logger = logger
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
	policyMode := m.policyMode
	m.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("capability not registered: %s", name)
	}

	m.evaluatePolicy(ctx, principal, name, cmd, policyMode)

	return capability.Execute(ctx, principal, cmd)
}

func (m *Manager) evaluatePolicy(ctx context.Context, principal policy.Principal, capName string, cmd Command, mode PolicyMode) {
	if mode == PolicyModeDisabled {
		return
	}

	req := policy.Request{
		Principal: principal,
		Resource: policy.Resource{
			Type:   "capability",
			Action: capName,
			Attributes: map[string]string{
				"command": cmd.Name(),
			},
		},
		Context: map[string]any{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	}

	result, err := m.policy.Evaluate(req)
	if err != nil {
		m.logger.Warn("Policy evaluation error",
			zap.String("capability", capName),
			zap.String("principal", principal.DID),
			zap.Error(err),
		)
		return
	}

	if result.Effect == policy.EffectDeny {
		m.logger.Warn("POLICY AUDIT: denied capability execution",
			zap.String("capability", capName),
			zap.String("principal", principal.DID),
			zap.String("rule", result.Rule),
			zap.String("mode", map[PolicyMode]string{
				PolicyModeAudit:   "audit",
				PolicyModeEnforce: "enforce",
			}[mode]),
		)
	}
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
