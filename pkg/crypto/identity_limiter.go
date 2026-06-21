package crypto

import (
	"fmt"
	"sync"
	"time"
)

// IdentityType identity type
type IdentityType string

const (
	IdentityTypeHuman IdentityType = "human" // human identity
	IdentityTypeAgent IdentityType = "agent" // agent/AI model identity
)

// IdentityLimiter limits number of identities allowed on node
type IdentityLimiter struct {
	mu              sync.RWMutex
	humanIdentities map[string]time.Time // nodeID -> last creation time
	agentIdentities map[string]time.Time // nodeID -> last creation time

	// [SAFETY] Limits per node
	maxHumanIdentities int // Maximum human identities per node
	maxAgentIdentities int // Maximum agent identities per node

	// [SAFETY] Rate limiting
	humanCooldown time.Duration // Time between human identity creation
	agentCooldown time.Duration // Time between agent identity creation
}

// NewIdentityLimiter creates new identity limiter
func NewIdentityLimiter() *IdentityLimiter {
	return &IdentityLimiter{
		humanIdentities: make(map[string]time.Time),
		agentIdentities: make(map[string]time.Time),

		// [SAFETY] Conservative limits for humans (typically 1 person per device)
		// [WHY] Prevent abuse while allowing for multiple accounts on same device
		maxHumanIdentities: 8, // Up to 8 human identities per node (for family/shared devices)

		// [SAFETY] Higher limit for agents (teams, developers, AI workflows)
		// [WHY] Teams may need dozens of agents collaborating in sessions
		// [UPDATE] Increased to 128 for future-proofing as agents/models evolve
		maxAgentIdentities: 128, // Up to 128 agent identities per node

		// [SAFETY] Rate limiting to prevent rapid identity creation
		humanCooldown: 5 * time.Minute, // 5 minutes between human identity creation
		agentCooldown: 1 * time.Minute, // 1 minute between agent identity creation
	}
}

// CanCreateIdentity checks if new identity can be created
func (il *IdentityLimiter) CanCreateIdentity(nodeID string, identityType IdentityType) error {
	il.mu.Lock()
	defer il.mu.Unlock()

	now := time.Now()

	switch identityType {
	case IdentityTypeHuman:
		return il.canCreateHumanIdentity(nodeID, now)
	case IdentityTypeAgent:
		return il.canCreateAgentIdentity(nodeID, now)
	default:
		return fmt.Errorf("unknown identity type: %s", identityType)
	}
}

// canCreateHumanIdentity checks if human identity can be created
func (il *IdentityLimiter) canCreateHumanIdentity(nodeID string, now time.Time) error {
	// [SAFETY] Check count limit (identities persist indefinitely - no cleanup)
	count := len(il.humanIdentities)

	if count >= il.maxHumanIdentities {
		return fmt.Errorf("human identity limit reached: %d/%d", count, il.maxHumanIdentities)
	}

	// [SAFETY] Check rate limit
	if lastCreated, exists := il.humanIdentities[nodeID]; exists {
		if now.Sub(lastCreated) < il.humanCooldown {
			return fmt.Errorf("human identity cooldown: wait %v", il.humanCooldown-now.Sub(lastCreated))
		}
	}

	return nil
}

// canCreateAgentIdentity checks if agent identity can be created
func (il *IdentityLimiter) canCreateAgentIdentity(nodeID string, now time.Time) error {
	// [SAFETY] Check count limit (identities persist indefinitely - no cleanup)
	count := len(il.agentIdentities)

	if count >= il.maxAgentIdentities {
		return fmt.Errorf("agent identity limit reached: %d/%d", count, il.maxAgentIdentities)
	}

	// [SAFETY] Check rate limit
	if lastCreated, exists := il.agentIdentities[nodeID]; exists {
		if now.Sub(lastCreated) < il.agentCooldown {
			return fmt.Errorf("agent identity cooldown: wait %v", il.agentCooldown-now.Sub(lastCreated))
		}
	}

	return nil
}

// RecordIdentityCreation records new identity creation
func (il *IdentityLimiter) RecordIdentityCreation(nodeID string, identityType IdentityType) {
	il.mu.Lock()
	defer il.mu.Unlock()

	now := time.Now()

	switch identityType {
	case IdentityTypeHuman:
		il.humanIdentities[nodeID] = now
	case IdentityTypeAgent:
		il.agentIdentities[nodeID] = now
	}
}

// GetIdentityCount gets current identity count
func (il *IdentityLimiter) GetIdentityCount(identityType IdentityType) int {
	il.mu.RLock()
	defer il.mu.RUnlock()

	switch identityType {
	case IdentityTypeHuman:
		return len(il.humanIdentities)
	case IdentityTypeAgent:
		return len(il.agentIdentities)
	}

	return 0
}

// GetLimits gets current limits
func (il *IdentityLimiter) GetLimits() (maxHuman, maxAgent int) {
	il.mu.RLock()
	defer il.mu.RUnlock()
	return il.maxHumanIdentities, il.maxAgentIdentities
}

// SetLimits sets limits (for use in tests or custom configuration)
func (il *IdentityLimiter) SetLimits(maxHuman, maxAgent int) {
	il.mu.Lock()
	defer il.mu.Unlock()
	il.maxHumanIdentities = maxHuman
	il.maxAgentIdentities = maxAgent
}

// Clear clears all records
func (il *IdentityLimiter) Clear() {
	il.mu.Lock()
	defer il.mu.Unlock()
	il.humanIdentities = make(map[string]time.Time)
	il.agentIdentities = make(map[string]time.Time)
}
