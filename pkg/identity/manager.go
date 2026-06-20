package identity

import (
	"fmt"
	"sync"
)

// IdentityType نوع الهوية
type IdentityType string

const (
	IdentityTypeHuman IdentityType = "human" // هوية بشرية
	IdentityTypeAgent IdentityType = "agent" // هوية وكيل/نموذج AI
)

// IdentityManager يدير دورة حياة الهويات
type IdentityManager struct {
	store   *IdentityStore
	limiter *IdentityLimiter
	mu      sync.RWMutex
}

// NewIdentityManager ينشئ مدير هويات جديد
func NewIdentityManager(storageDir string) (*IdentityManager, error) {
	store, err := NewIdentityStore(storageDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity store: %w", err)
	}

	return &IdentityManager{
		store:   store,
		limiter: NewIdentityLimiter(),
	}, nil
}

// CreateOrUpdateIdentity ينشئ هوية جديدة أو يحدث هوية موجودة
// [SAFETY] This is the main entry point for identity management
// It handles:
// - Identity reuse across sessions
// - API key changes (via hash matching)
// - Agent updates (via version tracking)
func (im *IdentityManager) CreateOrUpdateIdentity(did, nodeID, identityType, agentName, agentVersion, provider, apiKey string) (*PersistentIdentity, error) {
	im.mu.Lock()
	defer im.mu.Unlock()

	// [SAFETY] Check if identity already exists
	existing, err := im.store.GetIdentity(did)
	if err == nil {
		// Identity exists - update it
		metadata := existing.Metadata

		// [SAFETY] Update mutable fields
		if agentVersion != "" {
			metadata.AgentVersion = agentVersion
		}
		if provider != "" {
			metadata.Provider = provider
		}
		if apiKey != "" {
			metadata.APIKeyHash = HashAPIKey(apiKey)
		}

		// [SAFETY] Increment session count
		metadata.SessionCount++

		err = im.store.UpdateIdentityMetadata(did, metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to update identity: %w", err)
		}

		return im.store.GetIdentity(did)
	}

	// [SAFETY] Check if we can reuse an existing identity by API key hash
	// This handles API key changes gracefully
	if apiKey != "" && agentName != "" {
		apiKeyHash := HashAPIKey(apiKey)
		existingByHash := im.store.FindIdentityByAPIKeyHash(apiKeyHash, agentName, nodeID)
		if existingByHash != nil {
			// Reuse existing identity
			metadata := existingByHash.Metadata

			// Update DID if it changed (shouldn't happen normally)
			if existingByHash.DID != did {
				// Delete old identity
				im.store.DeleteIdentity(existingByHash.DID)
			} else {
				// Update metadata
				if agentVersion != "" {
					metadata.AgentVersion = agentVersion
				}
				if provider != "" {
					metadata.Provider = provider
				}

				metadata.SessionCount++
				err = im.store.UpdateIdentityMetadata(did, metadata)
				if err != nil {
					return nil, fmt.Errorf("failed to update reused identity: %w", err)
				}

				return im.store.GetIdentity(did)
			}
		}
	}

	// [SAFETY] Check identity limits
	idType := IdentityTypeHuman
	if identityType == "agent" {
		idType = IdentityTypeAgent
	}

	err = im.limiter.CanCreateIdentity(nodeID, idType)
	if err != nil {
		return nil, fmt.Errorf("identity limit reached: %w", err)
	}

	// [SAFETY] Create new identity
	metadata := IdentityMetadata{
		IdentityType: identityType,
		NodeID:       nodeID,
		AgentName:    agentName,
		AgentVersion: agentVersion,
		Provider:     provider,
		APIKeyHash:   HashAPIKey(apiKey),
		IsActive:     true,
	}

	err = im.store.SaveIdentity(did, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to save identity: %w", err)
	}

	// [SAFETY] Record creation in limiter
	im.limiter.RecordIdentityCreation(nodeID, idType)

	return im.store.GetIdentity(did)
}

// GetIdentity يحصل على هوية
func (im *IdentityManager) GetIdentity(did string) (*PersistentIdentity, error) {
	return im.store.GetIdentity(did)
}

// FindIdentitiesByType يجد الهويات حسب النوع
func (im *IdentityManager) FindIdentitiesByType(identityType string) []*PersistentIdentity {
	return im.store.FindIdentitiesByType(identityType)
}

// FindIdentitiesByNode يجد الهويات حسب العقدة
func (im *IdentityManager) FindIdentitiesByNode(nodeID string) []*PersistentIdentity {
	return im.store.FindIdentitiesByNode(nodeID)
}

// FindIdentitiesByProvider يجد الهويات حسب المزود
func (im *IdentityManager) FindIdentitiesByProvider(provider string) []*PersistentIdentity {
	return im.store.FindIdentitiesByProvider(provider)
}

// FindIdentityForAgent يجد هوية وكيل
// [SAFETY] This handles agent updates by matching on stable metadata
func (im *IdentityManager) FindIdentityForAgent(nodeID, agentName, provider string) *PersistentIdentity {
	// Try to find by exact match first
	identities := im.store.FindIdentitiesByNode(nodeID)
	for _, identity := range identities {
		if identity.Metadata.AgentName == agentName &&
			identity.Metadata.Provider == provider &&
			identity.Metadata.IsActive {
			return identity
		}
	}

	return nil
}

// FindIdentityForAPIKey يجد هوية حسب API key
// [SAFETY] This handles API key changes by matching on hash
func (im *IdentityManager) FindIdentityForAPIKey(apiKey, agentName, nodeID string) *PersistentIdentity {
	apiKeyHash := HashAPIKey(apiKey)
	return im.store.FindIdentityByAPIKeyHash(apiKeyHash, agentName, nodeID)
}

// UpdateAPIKey يحدث API key لهوية
// [SAFETY] This allows API key changes without losing identity binding
func (im *IdentityManager) UpdateAPIKey(did, newAPIKey string) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	identity, err := im.store.GetIdentity(did)
	if err != nil {
		return fmt.Errorf("identity not found: %w", err)
	}

	metadata := identity.Metadata
	metadata.APIKeyHash = HashAPIKey(newAPIKey)

	return im.store.UpdateIdentityMetadata(did, metadata)
}

// UpdateAgentVersion يحدث إصدار الوكيل
// [SAFETY] This allows agent updates without losing identity binding
func (im *IdentityManager) UpdateAgentVersion(did, newVersion string) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	identity, err := im.store.GetIdentity(did)
	if err != nil {
		return fmt.Errorf("identity not found: %w", err)
	}

	metadata := identity.Metadata
	metadata.AgentVersion = newVersion

	return im.store.UpdateIdentityMetadata(did, metadata)
}

// MarkIdentityUsed يحدد أن الهوية استُخدمت
func (im *IdentityManager) MarkIdentityUsed(did string) error {
	return im.store.MarkIdentityUsed(did)
}

// DeactivateIdentity يوقف هوية
func (im *IdentityManager) DeactivateIdentity(did string) error {
	return im.store.DeactivateIdentity(did)
}

// ActivateIdentity يفعل هوية
func (im *IdentityManager) ActivateIdentity(did string) error {
	return im.store.ActivateIdentity(did)
}

// DeleteIdentity يحذف هوية
func (im *IdentityManager) DeleteIdentity(did string) error {
	return im.store.DeleteIdentity(did)
}

// ListAllIdentities يسرد جميع الهويات
func (im *IdentityManager) ListAllIdentities() []*PersistentIdentity {
	return im.store.ListAllIdentities()
}

// GetIdentityCount يحصل على عدد الهويات حسب النوع
func (im *IdentityManager) GetIdentityCount(identityType string) int {
	identities := im.store.FindIdentitiesByType(identityType)
	return len(identities)
}

// GetIdentityLimits يحصل على حدود الهويات
func (im *IdentityManager) GetIdentityLimits() (maxHuman, maxAgent int) {
	return im.limiter.GetLimits()
}

// SetIdentityLimits يحد حدود الهويات
func (im *IdentityManager) SetIdentityLimits(maxHuman, maxAgent int) {
	im.limiter.SetLimits(maxHuman, maxAgent)
}
