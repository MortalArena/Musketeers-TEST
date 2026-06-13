package workflow

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MortalArena/Musketeers/pkg/content"
)

// Checkpoint يمثل لقطة لحالة سير العمل في نقطة زمنية محددة
type Checkpoint struct {
	ID         string                 `json:"id"`
	WorkflowID string                 `json:"workflow_id"`
	NodeID     string                 `json:"node_id"` // آخر عقدة تم إكمالها بنجاح
	State      map[string]interface{} `json:"state"`   // حالة المتغيرات والبيانات
	Hash       string                 `json:"hash"`    // للتأكد من عدم التلاعب
	Timestamp  time.Time              `json:"timestamp"`
}

// CheckpointManager يدير عمليات حفظ واستعادة النقاط
type CheckpointManager struct {
	store content.BlockStore
}

// NewCheckpointManager ينشئ مدير نقاط حفظ جديد
func NewCheckpointManager(store content.BlockStore) *CheckpointManager {
	return &CheckpointManager{store: store}
}

// Save يحفظ حالة سير العمل بشكل آمن
func (cm *CheckpointManager) Save(workflowID, nodeID string, state map[string]interface{}) error {
	cp := &Checkpoint{
		ID:         generateID(),
		WorkflowID: workflowID,
		NodeID:     nodeID,
		State:      state,
		Timestamp:  time.Now(),
	}

	// 1. حساب Hash للحالة لمنع التلاعب
	stateBytes, err := json.Marshal(cp.State)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}
	hash := sha256.Sum256(stateBytes)
	cp.Hash = hex.EncodeToString(hash[:])

	// 2. حفظ النقطة
	data, err := json.Marshal(cp)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	cid := content.CIDFromData(data)
	if err := cm.store.Put(cid, data); err != nil {
		return fmt.Errorf("failed to store checkpoint: %w", err)
	}

	// 3. تحديث مؤشر "آخر نقطة حفظ" لهذا الـ Workflow
	lastData := []byte(cid)
	lastCID := content.CIDFromData(lastData)
	if err := cm.store.Put(lastCID, lastData); err != nil {
		return fmt.Errorf("failed to update latest checkpoint pointer: %w", err)
	}

	return nil
}

// GetLatest يسترجع آخر حالة محفوظة بنجاح
func (cm *CheckpointManager) GetLatest(workflowID string) (*Checkpoint, error) {
	// في التنفيذ الحالي، سنستخدم تخزين بسيط في الذاكرة
	// في الإنتاج، يجب استخدام قاعدة بيانات حقيقية
	return nil, fmt.Errorf("not implemented in current version")
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
