package session

import (
	"fmt"
	"sync"

	"github.com/dgraph-io/badger/v4"
)

// ArtifactsStore مخزن القطع الأثرية - يدير القطع الأثرية للجلسة
type ArtifactsStore struct {
	SessionID string
	DB        *badger.DB
	artifacts map[string]*Artifact // artifactID -> artifact
	mu        sync.RWMutex
}

// [SAFETY] حدود الموارد لمنع استهلاك غير محدود
const (
	// [SAFETY] الحد الأقصى لعدد القطع الأثرية في المخزن
	MaxArtifactsInStore = 1000
)

// NewArtifactsStore ينشئ مخزن قطع أثرية جديد
func NewArtifactsStore(sessionID string, db *badger.DB) *ArtifactsStore {
	return &ArtifactsStore{
		SessionID: sessionID,
		DB:        db,
		artifacts: make(map[string]*Artifact),
	}
}

// AddArtifact يضيف قطعة أثرية
func (as *ArtifactsStore) AddArtifact(artifact *Artifact) error {
	// [SAFETY] التحقق من صحة المدخلات
	if artifact == nil {
		return fmt.Errorf("artifact cannot be nil")
	}
	if artifact.ID == "" {
		return fmt.Errorf("artifact ID cannot be empty")
	}
	if artifact.Name == "" {
		return fmt.Errorf("artifact name cannot be empty")
	}

	as.mu.Lock()
	defer as.mu.Unlock()

	// [SAFETY] التحقق من الحد الأقصى للقطع الأثرية
	if len(as.artifacts) >= MaxArtifactsInStore {
		return fmt.Errorf("maximum artifacts limit reached (%d)", MaxArtifactsInStore)
	}

	as.artifacts[artifact.ID] = artifact
	return nil
}

// GetArtifact يحصل على قطعة أثرية
func (as *ArtifactsStore) GetArtifact(artifactID string) (*Artifact, error) {
	as.mu.RLock()
	defer as.mu.RUnlock()

	artifact, exists := as.artifacts[artifactID]
	if !exists {
		return nil, fmt.Errorf("artifact not found: %s", artifactID)
	}
	return artifact, nil
}

// GetAllArtifacts يحصل على جميع القطع الأثرية
func (as *ArtifactsStore) GetAllArtifacts() map[string]*Artifact {
	as.mu.RLock()
	defer as.mu.RUnlock()

	result := make(map[string]*Artifact)
	for k, v := range as.artifacts {
		result[k] = v
	}
	return result
}

// GetArtifactsByAgent يحصل على قطع أثرية لوكيل معين
func (as *ArtifactsStore) GetArtifactsByAgent(agentID string) []*Artifact {
	as.mu.RLock()
	defer as.mu.RUnlock()

	var result []*Artifact
	for _, artifact := range as.artifacts {
		if artifact.CreatedBy == agentID {
			result = append(result, artifact)
		}
	}
	return result
}

// DeleteArtifact يحذف قطعة أثرية
func (as *ArtifactsStore) DeleteArtifact(artifactID string) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	if _, exists := as.artifacts[artifactID]; !exists {
		return fmt.Errorf("artifact not found: %s", artifactID)
	}

	delete(as.artifacts, artifactID)
	return nil
}
