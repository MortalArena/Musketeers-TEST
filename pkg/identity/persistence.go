package identity

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// IdentityMetadata بيانات وصفية للهوية
type IdentityMetadata struct {
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	LastUsed     time.Time `json:"last_used"`
	IdentityType string    `json:"identity_type"` // "human", "agent", "model"
	NodeID       string    `json:"node_id"`
	AgentName    string    `json:"agent_name,omitempty"`
	AgentVersion string    `json:"agent_version,omitempty"`
	Provider     string    `json:"provider,omitempty"`     // OpenAI, Anthropic, etc.
	APIKeyHash   string    `json:"api_key_hash,omitempty"` // Hash of API key for binding
	SessionCount int       `json:"session_count"`
	IsActive     bool      `json:"is_active"`
}

// PersistentIdentity هوية محفوظة بشكل دائم
type PersistentIdentity struct {
	DID      string           `json:"did"`
	Metadata IdentityMetadata `json:"metadata"`
}

// IdentityStore مخزن الهويات الدائم
type IdentityStore struct {
	mu         sync.RWMutex
	storageDir string
	identities map[string]*PersistentIdentity // DID -> PersistentIdentity
}

// NewIdentityStore ينشئ مخزن هويات جديد
func NewIdentityStore(storageDir string) (*IdentityStore, error) {
	if err := os.MkdirAll(storageDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	store := &IdentityStore{
		storageDir: storageDir,
		identities: make(map[string]*PersistentIdentity),
	}

	// [SAFETY] Load existing identities from disk
	if err := store.loadFromDisk(); err != nil {
		return nil, fmt.Errorf("failed to load identities: %w", err)
	}

	return store, nil
}

// loadFromDisk يحمل الهويات من القرص
func (s *IdentityStore) loadFromDisk() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.storageDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(s.storageDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			// [SAFETY] Log error but continue loading other identities
			continue
		}

		var identity PersistentIdentity
		if err := json.Unmarshal(data, &identity); err != nil {
			// [SAFETY] Log error but continue loading other identities
			continue
		}

		s.identities[identity.DID] = &identity
	}

	return nil
}

// saveToDisk يحفظ الهويات على القرص (للاستخدام الخارجي)
func (s *IdentityStore) saveToDisk() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.saveToDiskInternal()
}

// SaveIdentity يحفظ هوية جديدة أو يحدث هوية موجودة
func (s *IdentityStore) SaveIdentity(did string, metadata IdentityMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	// [SAFETY] Update timestamps
	if existing, exists := s.identities[did]; exists {
		metadata.CreatedAt = existing.Metadata.CreatedAt
		metadata.SessionCount = existing.Metadata.SessionCount + 1
	} else {
		metadata.CreatedAt = now
		metadata.SessionCount = 1
	}
	metadata.UpdatedAt = now
	metadata.LastUsed = now

	identity := &PersistentIdentity{
		DID:      did,
		Metadata: metadata,
	}

	s.identities[did] = identity

	// [SAFETY] Save to disk immediately (without lock, already locked)
	return s.saveToDiskInternal()
}

// saveToDiskInternal يحفظ الهويات على القرص (بدون قفل - يجب استدعاؤه من داخل قفل)
func (s *IdentityStore) saveToDiskInternal() error {
	for did, identity := range s.identities {
		// [SAFETY] Replace colons with underscores for Windows compatibility
		safeDID := strings.ReplaceAll(did, ":", "_")
		filePath := filepath.Join(s.storageDir, safeDID+".json")
		data, err := json.MarshalIndent(identity, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal identity %s: %w", did, err)
		}

		if err := os.WriteFile(filePath, data, 0600); err != nil {
			return fmt.Errorf("failed to save identity %s: %w", did, err)
		}
	}

	return nil
}

// GetIdentity يحصل على هوية محفوظة
func (s *IdentityStore) GetIdentity(did string) (*PersistentIdentity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	identity, exists := s.identities[did]
	if !exists {
		return nil, fmt.Errorf("identity not found: %s", did)
	}

	return identity, nil
}

// FindIdentitiesByType يجد الهويات حسب النوع
func (s *IdentityStore) FindIdentitiesByType(identityType string) []*PersistentIdentity {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*PersistentIdentity
	for _, identity := range s.identities {
		if identity.Metadata.IdentityType == identityType {
			result = append(result, identity)
		}
	}

	return result
}

// FindIdentitiesByNode يجد الهويات حسب العقدة
func (s *IdentityStore) FindIdentitiesByNode(nodeID string) []*PersistentIdentity {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*PersistentIdentity
	for _, identity := range s.identities {
		if identity.Metadata.NodeID == nodeID {
			result = append(result, identity)
		}
	}

	return result
}

// FindIdentitiesByProvider يجد الهويات حسب المزود
func (s *IdentityStore) FindIdentitiesByProvider(provider string) []*PersistentIdentity {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*PersistentIdentity
	for _, identity := range s.identities {
		if identity.Metadata.Provider == provider {
			result = append(result, identity)
		}
	}

	return result
}

// FindIdentityByAPIKeyHash يجد هوية حسب hash of API key
// [SAFETY] This allows identity reuse even if API key changes
func (s *IdentityStore) FindIdentityByAPIKeyHash(apiKeyHash string, agentName string, nodeID string) *PersistentIdentity {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, identity := range s.identities {
		if identity.Metadata.APIKeyHash == apiKeyHash &&
			identity.Metadata.AgentName == agentName &&
			identity.Metadata.NodeID == nodeID {
			return identity
		}
	}

	return nil
}

// UpdateIdentityMetadata يحدث بيانات الهوية الوصفية
func (s *IdentityStore) UpdateIdentityMetadata(did string, metadata IdentityMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	identity, exists := s.identities[did]
	if !exists {
		return fmt.Errorf("identity not found: %s", did)
	}

	// [SAFETY] Preserve created_at only
	metadata.CreatedAt = identity.Metadata.CreatedAt
	// [SAFETY] Don't preserve session_count - it should be set by caller
	metadata.UpdatedAt = time.Now().UTC()

	identity.Metadata = metadata
	s.identities[did] = identity

	return s.saveToDiskInternal()
}

// MarkIdentityUsed يحدد أن الهوية استُخدمت
func (s *IdentityStore) MarkIdentityUsed(did string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	identity, exists := s.identities[did]
	if !exists {
		return fmt.Errorf("identity not found: %s", did)
	}

	identity.Metadata.LastUsed = time.Now().UTC()
	identity.Metadata.SessionCount++

	return s.saveToDiskInternal()
}

// DeactivateIdentity يوقف هوية
func (s *IdentityStore) DeactivateIdentity(did string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	identity, exists := s.identities[did]
	if !exists {
		return fmt.Errorf("identity not found: %s", did)
	}

	identity.Metadata.IsActive = false
	identity.Metadata.UpdatedAt = time.Now().UTC()

	return s.saveToDiskInternal()
}

// ActivateIdentity يفعل هوية
func (s *IdentityStore) ActivateIdentity(did string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	identity, exists := s.identities[did]
	if !exists {
		return fmt.Errorf("identity not found: %s", did)
	}

	identity.Metadata.IsActive = true
	identity.Metadata.UpdatedAt = time.Now().UTC()

	return s.saveToDiskInternal()
}

// DeleteIdentity يحذف هوية
func (s *IdentityStore) DeleteIdentity(did string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.identities[did]; !exists {
		return fmt.Errorf("identity not found: %s", did)
	}

	delete(s.identities, did)

	// [SAFETY] Delete from disk (use safe filename)
	safeDID := strings.ReplaceAll(did, ":", "_")
	filePath := filepath.Join(s.storageDir, safeDID+".json")
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete identity file: %w", err)
	}

	return nil
}

// ListAllIdentities يسرد جميع الهويات
func (s *IdentityStore) ListAllIdentities() []*PersistentIdentity {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*PersistentIdentity
	for _, identity := range s.identities {
		result = append(result, identity)
	}

	return result
}

// GetActiveIdentitiesCount يحصل على عدد الهويات النشطة
func (s *IdentityStore) GetActiveIdentitiesCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, identity := range s.identities {
		if identity.Metadata.IsActive {
			count++
		}
	}

	return count
}

// HashAPIKey يحسب hash of API key
// [SAFETY] We never store the actual API key, only its hash
func HashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return fmt.Sprintf("%x", hash)
}
