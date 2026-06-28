package session

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

// HandoffStatus حالة التسليم
type HandoffStatus string

const (
	HandoffStatusPending   HandoffStatus = "pending"
	HandoffStatusAccepted  HandoffStatus = "accepted"
	HandoffStatusRejected  HandoffStatus = "rejected"
	HandoffStatusCompleted HandoffStatus = "completed"
	HandoffStatusFailed    HandoffStatus = "failed"
)

// Artifact قطعة أثرية (ملف أو بيانات)
type Artifact struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"` // file, data, url
	Path      string                 `json:"path,omitempty"`
	Data      []byte                 `json:"data,omitempty"`
	Checksum  string                 `json:"checksum"`
	Size      int64                  `json:"size"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	CreatedBy string                 `json:"created_by"` // Agent ID
}

// HandoffRequest طلب تسليم
type HandoffRequest struct {
	ID           string                 `json:"id"`
	FromAgentID  string                 `json:"from_agent_id"`
	ToAgentID    string                 `json:"to_agent_id"`
	TaskID       string                 `json:"task_id"`
	Artifacts    []Artifact             `json:"artifacts"`
	Status       HandoffStatus          `json:"status"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	AcceptedAt   *time.Time             `json:"accepted_at,omitempty"`
	RejectedAt   *time.Time             `json:"rejected_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	RejectReason string                 `json:"reject_reason,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// HandoffManager مدير التسليم - يدير تسليم القطع الأثرية بين الوكلاء
type HandoffManager struct {
	sessionID   string
	logger      *zap.Logger
	eventBus    *eventbus.EventBus
	artifactDir string

	// بيانات التسليم
	handoffs map[string]*HandoffRequest // handoffID -> request

	mu sync.RWMutex
}

// [SAFETY] حدود الموارد لمنع استهلاك غير محدود
const (
	// [SAFETY] الحد الأقصى لعدد طلبات التسليم
	MaxHandoffs = 1000
	// [SAFETY] الحد الأقصى لعدد القطع الأثرية لكل طلب
	MaxArtifactsPerHandoff = 100
	// [SAFETY] الحد الأقصى لحجم القطعة الأثرية (100MB)
	MaxArtifactSize = 100 * 1024 * 1024
	// [SAFETY] الحد الأقصى لاسم القطعة الأثرية
	MaxArtifactNameLength = 200
)

// NewHandoffManager ينشئ مدير تسليم جديد
func NewHandoffManager(sessionID, artifactDir string) *HandoffManager {
	return &HandoffManager{
		sessionID:   sessionID,
		logger:      zap.NewNop(), // سيتم استبداله بـ logger حقيقي
		eventBus:    nil,          // سيتم تعيينه لاحقاً
		artifactDir: artifactDir,
		handoffs:    make(map[string]*HandoffRequest),
	}
}

// SetLogger يضبط logger
func (hm *HandoffManager) SetLogger(logger *zap.Logger) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.logger = logger
}

// SetEventBus يضبط event bus
func (hm *HandoffManager) SetEventBus(eb *eventbus.EventBus) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.eventBus = eb
}

// CreateHandoffRequest ينشئ طلب تسليم جديد
func (hm *HandoffManager) CreateHandoffRequest(ctx context.Context, fromAgentID, toAgentID, taskID string, artifacts []Artifact) (*HandoffRequest, error) {
	// [SAFETY] التحقق من صحة المدخلات
	if fromAgentID == "" {
		return nil, fmt.Errorf("from agent ID cannot be empty")
	}
	if toAgentID == "" {
		return nil, fmt.Errorf("to agent ID cannot be empty")
	}
	if taskID == "" {
		return nil, fmt.Errorf("task ID cannot be empty")
	}

	// [SAFETY] التحقق من الحد الأقصى للقطع الأثرية
	if len(artifacts) > MaxArtifactsPerHandoff {
		return nil, fmt.Errorf("maximum artifacts per handoff limit reached (%d)", MaxArtifactsPerHandoff)
	}

	// [SAFETY] التحقق من صحة القطع الأثرية
	for i := range artifacts {
		if artifacts[i].Name == "" {
			return nil, fmt.Errorf("artifact name cannot be empty")
		}
		if len(artifacts[i].Name) > MaxArtifactNameLength {
			return nil, fmt.Errorf("artifact name too long (max %d characters)", MaxArtifactNameLength)
		}
		if artifacts[i].Type == "" {
			return nil, fmt.Errorf("artifact type cannot be empty")
		}
		if int64(len(artifacts[i].Data)) > MaxArtifactSize {
			return nil, fmt.Errorf("artifact size too large (max %d bytes)", MaxArtifactSize)
		}
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()

	// [SAFETY] التحقق من الحد الأقصى لطلبات التسليم
	if len(hm.handoffs) >= MaxHandoffs {
		return nil, fmt.Errorf("maximum handoffs limit reached (%d)", MaxHandoffs)
	}

	// حساب checksum لكل قطعة أثرية
	for i := range artifacts {
		checksum, err := hm.calculateChecksum(&artifacts[i])
		if err != nil {
			return nil, fmt.Errorf("failed to calculate checksum for artifact %s: %w", artifacts[i].Name, err)
		}
		artifacts[i].Checksum = checksum
	}

	request := &HandoffRequest{
		ID:          fmt.Sprintf("handoff_%d", time.Now().UnixNano()),
		FromAgentID: fromAgentID,
		ToAgentID:   toAgentID,
		TaskID:      taskID,
		Artifacts:   artifacts,
		Status:      HandoffStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	hm.handoffs[request.ID] = request

	hm.logger.Info("Handoff request created",
		zap.String("handoff_id", request.ID),
		zap.String("from_agent", fromAgentID),
		zap.String("to_agent", toAgentID),
		zap.String("task_id", taskID),
		zap.Int("artifacts_count", len(artifacts)),
	)

	if hm.eventBus != nil {
		hm.eventBus.Publish(eventbus.Event{
			Type:      "handoff.created",
			Payload:   request,
			Source:    "handoff_manager",
			SessionID: hm.sessionID,
		})
	}

	return request, nil
}

// AcceptHandoff يقبل طلب تسليم
func (hm *HandoffManager) AcceptHandoff(ctx context.Context, handoffID string) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	request, exists := hm.handoffs[handoffID]
	if !exists {
		return fmt.Errorf("handoff request not found: %s", handoffID)
	}

	if request.Status != HandoffStatusPending {
		return fmt.Errorf("handoff request is not pending: %s", handoffID)
	}

	request.Status = HandoffStatusAccepted
	now := time.Now()
	request.AcceptedAt = &now
	request.UpdatedAt = now

	hm.logger.Info("Handoff accepted",
		zap.String("handoff_id", handoffID),
		zap.String("to_agent", request.ToAgentID),
	)

	if hm.eventBus != nil {
		hm.eventBus.Publish(eventbus.Event{
			Type:      "handoff.accepted",
			Payload:   request,
			Source:    "handoff_manager",
			SessionID: hm.sessionID,
		})
	}

	return nil
}

// RejectHandoff يرفض طلب تسليم
func (hm *HandoffManager) RejectHandoff(ctx context.Context, handoffID, reason string) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	request, exists := hm.handoffs[handoffID]
	if !exists {
		return fmt.Errorf("handoff request not found: %s", handoffID)
	}

	if request.Status != HandoffStatusPending {
		return fmt.Errorf("handoff request is not pending: %s", handoffID)
	}

	request.Status = HandoffStatusRejected
	request.RejectReason = reason
	now := time.Now()
	request.RejectedAt = &now
	request.UpdatedAt = now

	hm.logger.Warn("Handoff rejected",
		zap.String("handoff_id", handoffID),
		zap.String("to_agent", request.ToAgentID),
		zap.String("reason", reason),
	)

	if hm.eventBus != nil {
		hm.eventBus.Publish(eventbus.Event{
			Type:      "handoff.rejected",
			Payload:   request,
			Source:    "handoff_manager",
			SessionID: hm.sessionID,
		})
	}

	return nil
}

// CompleteHandoff يكمل تسليم
func (hm *HandoffManager) CompleteHandoff(ctx context.Context, handoffID string) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	request, exists := hm.handoffs[handoffID]
	if !exists {
		return fmt.Errorf("handoff request not found: %s", handoffID)
	}

	if request.Status != HandoffStatusAccepted {
		return fmt.Errorf("handoff request is not accepted: %s", handoffID)
	}

	// التحقق من سلامة القطع الأثرية
	for _, artifact := range request.Artifacts {
		if err := hm.verifyArtifact(&artifact); err != nil {
			request.Status = HandoffStatusFailed
			request.UpdatedAt = time.Now()
			return fmt.Errorf("artifact verification failed: %w", err)
		}
	}

	request.Status = HandoffStatusCompleted
	now := time.Now()
	request.CompletedAt = &now
	request.UpdatedAt = now

	hm.logger.Info("Handoff completed",
		zap.String("handoff_id", handoffID),
		zap.String("from_agent", request.FromAgentID),
		zap.String("to_agent", request.ToAgentID),
	)

	if hm.eventBus != nil {
		hm.eventBus.Publish(eventbus.Event{
			Type:      "handoff.completed",
			Payload:   request,
			Source:    "handoff_manager",
			SessionID: hm.sessionID,
		})
	}

	return nil
}

// GetHandoffRequest يحصل على طلب تسليم
func (hm *HandoffManager) GetHandoffRequest(handoffID string) (*HandoffRequest, error) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	request, exists := hm.handoffs[handoffID]
	if !exists {
		return nil, fmt.Errorf("handoff request not found: %s", handoffID)
	}

	// إنشاء نسخة لتجنب التعديل الخارجي
	requestCopy := *request
	return &requestCopy, nil
}

// GetHandoffsByTask يحصل على طلبات التسليم لمهمة
func (hm *HandoffManager) GetHandoffsByTask(taskID string) []*HandoffRequest {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	var result []*HandoffRequest
	for _, request := range hm.handoffs {
		if request.TaskID == taskID {
			requestCopy := *request
			result = append(result, &requestCopy)
		}
	}

	return result
}

// GetHandoffsByAgent يحصل على طلبات التسليم لوكيل
func (hm *HandoffManager) GetHandoffsByAgent(agentID string) []*HandoffRequest {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	var result []*HandoffRequest
	for _, request := range hm.handoffs {
		if request.FromAgentID == agentID || request.ToAgentID == agentID {
			requestCopy := *request
			result = append(result, &requestCopy)
		}
	}

	return result
}

// SaveArtifact يحفظ قطعة أثرية على القرص
func (hm *HandoffManager) SaveArtifact(artifact *Artifact) error {
	if artifact.Type != "file" {
		return nil
	}

	if hm.artifactDir == "" {
		return fmt.Errorf("artifact directory not set")
	}

	// إنشاء الدليل إذا لم يكن موجوداً
	if err := os.MkdirAll(hm.artifactDir, 0755); err != nil {
		return fmt.Errorf("failed to create artifact directory: %w", err)
	}

	// حفظ الملف
	filePath := filepath.Join(hm.artifactDir, artifact.ID+"_"+artifact.Name)
	if err := os.WriteFile(filePath, artifact.Data, 0644); err != nil {
		return fmt.Errorf("failed to write artifact file: %w", err)
	}

	// تحديث المسار
	artifact.Path = filePath
	artifact.Size = int64(len(artifact.Data))

	return nil
}

// LoadArtifact يحمل قطعة أثرية من القرص
func (hm *HandoffManager) LoadArtifact(artifact *Artifact) error {
	if artifact.Type != "file" || artifact.Path == "" {
		return nil
	}

	data, err := os.ReadFile(artifact.Path)
	if err != nil {
		return fmt.Errorf("failed to read artifact file: %w", err)
	}

	artifact.Data = data
	artifact.Size = int64(len(data))

	return nil
}

// calculateChecksum يحسب checksum لقطعة أثرية
func (hm *HandoffManager) calculateChecksum(artifact *Artifact) (string, error) {
	hash := sha256.New()

	if artifact.Type == "file" && len(artifact.Data) > 0 {
		hash.Write(artifact.Data)
	} else if artifact.Type == "data" && len(artifact.Data) > 0 {
		hash.Write(artifact.Data)
	} else {
		// حساب checksum من البيانات الوصفية
		metadata, err := json.Marshal(artifact.Metadata)
		if err != nil {
			return "", err
		}
		hash.Write(metadata)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// verifyArtifact يتحقق من سلامة قطعة أثرية
func (hm *HandoffManager) verifyArtifact(artifact *Artifact) error {
	expectedChecksum := artifact.Checksum
	actualChecksum, err := hm.calculateChecksum(artifact)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	if expectedChecksum != actualChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

// GetStats يحصل على إحصائيات
func (hm *HandoffManager) GetStats() map[string]interface{} {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	pending := 0
	accepted := 0
	rejected := 0
	completed := 0
	failed := 0

	for _, request := range hm.handoffs {
		switch request.Status {
		case HandoffStatusPending:
			pending++
		case HandoffStatusAccepted:
			accepted++
		case HandoffStatusRejected:
			rejected++
		case HandoffStatusCompleted:
			completed++
		case HandoffStatusFailed:
			failed++
		}
	}

	return map[string]interface{}{
		"total_handoffs": len(hm.handoffs),
		"pending":        pending,
		"accepted":       accepted,
		"rejected":       rejected,
		"completed":      completed,
		"failed":         failed,
	}
}

// Save يحفظ حالة HandoffManager
func (hm *HandoffManager) Save() ([]byte, error) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	data := struct {
		Handoffs map[string]*HandoffRequest `json:"handoffs"`
	}{
		Handoffs: hm.handoffs,
	}

	return json.Marshal(data)
}

// Load يحمل حالة HandoffManager
func (hm *HandoffManager) Load(data []byte) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	var loaded struct {
		Handoffs map[string]*HandoffRequest `json:"handoffs"`
	}

	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}

	hm.handoffs = loaded.Handoffs
	if hm.handoffs == nil {
		hm.handoffs = make(map[string]*HandoffRequest)
	}

	return nil
}

// CleanupOldHandoffs ينظف طلبات التسليم القديمة
func (hm *HandoffManager) CleanupOldHandoffs(ctx context.Context, olderThan time.Duration) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	deletedCount := 0

	for id, request := range hm.handoffs {
		if request.UpdatedAt.Before(cutoff) &&
			(request.Status == HandoffStatusCompleted || request.Status == HandoffStatusRejected || request.Status == HandoffStatusFailed) {
			delete(hm.handoffs, id)
			deletedCount++
		}
	}

	hm.logger.Info("Cleaned up old handoffs",
		zap.Int("deleted_count", deletedCount),
		zap.Duration("older_than", olderThan),
	)

	return nil
}

// StreamArtifact يبث قطعة أثرية
func (hm *HandoffManager) StreamArtifact(artifact *Artifact, writer io.Writer) error {
	if artifact.Type == "file" && artifact.Path != "" {
		file, err := os.Open(artifact.Path)
		if err != nil {
			return fmt.Errorf("failed to open artifact file: %w", err)
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		if err != nil {
			return fmt.Errorf("failed to copy artifact data: %w", err)
		}
	} else if len(artifact.Data) > 0 {
		_, err := writer.Write(artifact.Data)
		if err != nil {
			return fmt.Errorf("failed to write artifact data: %w", err)
		}
	}

	return nil
}
