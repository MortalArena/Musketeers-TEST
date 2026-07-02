package autodiscovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"go.uber.org/zap"
)

// LifecycleManager نظام إدارة دورة حياة الوكلاء
type LifecycleManager struct {
	agentRegistry   *agent.AgentRegistry
	lifecycleStates map[string]*LifecycleState
	mu              sync.RWMutex
	logger          *zap.Logger
}

// LifecycleState حالة دورة حياة الوكيل
type LifecycleState struct {
	AgentID          string                 `json:"agent_id"`
	Name             string                 `json:"name"`
	Type             string                 `json:"type"`
	Status           LifecycleStatus        `json:"status"` // active, paused, frozen, removed
	LastStatusChange time.Time              `json:"last_status_change"`
	Reason           string                 `json:"reason,omitempty"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// LifecycleStatus حالة دورة حياة الوكيل
type LifecycleStatus string

const (
	LifecycleStatusActive  LifecycleStatus = "active"  // نشط - يعمل بشكل طبيعي
	LifecycleStatusPaused  LifecycleStatus = "paused"  // متوقف مؤقتاً - لا يستقبل مهام جديدة
	LifecycleStatusFrozen  LifecycleStatus = "frozen"  // مجمد - لا يستقبل مهام ولا يرسل
	LifecycleStatusRemoved LifecycleStatus = "removed" // محذوف - تم إزالته من النظام
)

// NewLifecycleManager ينشئ مدير دورة حياة جديد
func NewLifecycleManager(agentRegistry *agent.AgentRegistry, logger *zap.Logger) *LifecycleManager {
	return &LifecycleManager{
		agentRegistry:   agentRegistry,
		lifecycleStates: make(map[string]*LifecycleState),
		logger:          logger,
	}
}

// RegisterAgent يسجل وكيل في نظام إدارة دورة الحياة
func (lm *LifecycleManager) RegisterAgent(agentID, name, agentType string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if _, exists := lm.lifecycleStates[agentID]; exists {
		return fmt.Errorf("الوكيل مسجل بالفعل في نظام دورة الحياة: %s", agentID)
	}

	state := &LifecycleState{
		AgentID:          agentID,
		Name:             name,
		Type:             agentType,
		Status:           LifecycleStatusActive,
		LastStatusChange: time.Now(),
		Metadata:         make(map[string]interface{}),
	}

	lm.lifecycleStates[agentID] = state

	lm.logger.Info("تم تسجيل الوكيل في نظام دورة الحياة",
		zap.String("agent_id", agentID),
		zap.String("name", name),
		zap.String("type", agentType),
	)

	return nil
}

// PauseAgent يوقف الوكيل مؤقتاً
func (lm *LifecycleManager) PauseAgent(agentID, reason string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	state, exists := lm.lifecycleStates[agentID]
	if !exists {
		return fmt.Errorf("الوكيل غير موجود في نظام دورة الحياة: %s", agentID)
	}

	if state.Status == LifecycleStatusRemoved {
		return fmt.Errorf("الوكيل محذوف، لا يمكن توقيفه: %s", agentID)
	}

	state.Status = LifecycleStatusPaused
	state.LastStatusChange = time.Now()
	state.Reason = reason

	lm.logger.Info("تم توقيف الوكيل مؤقتاً",
		zap.String("agent_id", agentID),
		zap.String("reason", reason),
	)

	return nil
}

// ResumeAgent يستأنف الوكيل المتوقف
func (lm *LifecycleManager) ResumeAgent(agentID, reason string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	state, exists := lm.lifecycleStates[agentID]
	if !exists {
		return fmt.Errorf("الوكيل غير موجود في نظام دورة الحياة: %s", agentID)
	}

	if state.Status == LifecycleStatusRemoved {
		return fmt.Errorf("الوكيل محذوف، لا يمكن استئنافه: %s", agentID)
	}

	state.Status = LifecycleStatusActive
	state.LastStatusChange = time.Now()
	state.Reason = reason

	lm.logger.Info("تم استئناف الوكيل",
		zap.String("agent_id", agentID),
		zap.String("reason", reason),
	)

	return nil
}

// FreezeAgent يجمد الوكيل
func (lm *LifecycleManager) FreezeAgent(agentID, reason string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	state, exists := lm.lifecycleStates[agentID]
	if !exists {
		return fmt.Errorf("الوكيل غير موجود في نظام دورة الحياة: %s", agentID)
	}

	if state.Status == LifecycleStatusRemoved {
		return fmt.Errorf("الوكيل محذوف، لا يمكن تجميده: %s", agentID)
	}

	state.Status = LifecycleStatusFrozen
	state.LastStatusChange = time.Now()
	state.Reason = reason

	lm.logger.Info("تم تجميد الوكيل",
		zap.String("agent_id", agentID),
		zap.String("reason", reason),
	)

	return nil
}

// UnfreezeAgent يذيب الوكيل المجمد
func (lm *LifecycleManager) UnfreezeAgent(agentID, reason string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	state, exists := lm.lifecycleStates[agentID]
	if !exists {
		return fmt.Errorf("الوكيل غير موجود في نظام دورة الحياة: %s", agentID)
	}

	if state.Status == LifecycleStatusRemoved {
		return fmt.Errorf("الوكيل محذوف، لا يمكن إذابته: %s", agentID)
	}

	state.Status = LifecycleStatusActive
	state.LastStatusChange = time.Now()
	state.Reason = reason

	lm.logger.Info("تم إذابة الوكيل",
		zap.String("agent_id", agentID),
		zap.String("reason", reason),
	)

	return nil
}

// RemoveAgent يزيل الوكيل من النظام
func (lm *LifecycleManager) RemoveAgent(ctx context.Context, agentID, reason string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	state, exists := lm.lifecycleStates[agentID]
	if !exists {
		return fmt.Errorf("الوكيل غير موجود في نظام دورة الحياة: %s", agentID)
	}

	if state.Status == LifecycleStatusRemoved {
		return fmt.Errorf("الوكيل محذوف بالفعل: %s", agentID)
	}

	// إزالة من AgentRegistry
	if err := lm.agentRegistry.Unregister(agentID); err != nil {
		lm.logger.Error("فشل إزالة الوكيل من AgentRegistry",
			zap.String("agent_id", agentID),
			zap.Error(err),
		)
		return fmt.Errorf("فشل إزالة الوكيل من AgentRegistry: %w", err)
	}

	state.Status = LifecycleStatusRemoved
	state.LastStatusChange = time.Now()
	state.Reason = reason

	lm.logger.Info("تم إزالة الوكيل من النظام",
		zap.String("agent_id", agentID),
		zap.String("reason", reason),
	)

	return nil
}

// GetLifecycleState يعيد حالة دورة حياة الوكيل
func (lm *LifecycleManager) GetLifecycleState(agentID string) (*LifecycleState, error) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	state, exists := lm.lifecycleStates[agentID]
	if !exists {
		return nil, fmt.Errorf("الوكيل غير موجود في نظام دورة الحياة: %s", agentID)
	}

	return state, nil
}

// GetAllLifecycleStates يعيد جميع حالات دورة حياة الوكلاء
func (lm *LifecycleManager) GetAllLifecycleStates() []*LifecycleState {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	states := make([]*LifecycleState, 0, len(lm.lifecycleStates))
	for _, state := range lm.lifecycleStates {
		states = append(states, state)
	}

	return states
}

// GetAgentsByStatus يعيد الوكلاء حسب الحالة
func (lm *LifecycleManager) GetAgentsByStatus(status LifecycleStatus) []*LifecycleState {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	states := make([]*LifecycleState, 0)
	for _, state := range lm.lifecycleStates {
		if state.Status == status {
			states = append(states, state)
		}
	}

	return states
}

// IsAgentActive يعيد ما إذا كان الوكيل نشطاً
func (lm *LifecycleManager) IsAgentActive(agentID string) bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	state, exists := lm.lifecycleStates[agentID]
	if !exists {
		return false
	}

	return state.Status == LifecycleStatusActive
}

// IsAgentPaused يعيد ما إذا كان الوكيل متوقفاً مؤقتاً
func (lm *LifecycleManager) IsAgentPaused(agentID string) bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	state, exists := lm.lifecycleStates[agentID]
	if !exists {
		return false
	}

	return state.Status == LifecycleStatusPaused
}

// IsAgentFrozen يعيد ما إذا كان الوكيل مجمداً
func (lm *LifecycleManager) IsAgentFrozen(agentID string) bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	state, exists := lm.lifecycleStates[agentID]
	if !exists {
		return false
	}

	return state.Status == LifecycleStatusFrozen
}

// IsAgentRemoved يعيد ما إذا كان الوكيل محذوفاً
func (lm *LifecycleManager) IsAgentRemoved(agentID string) bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	state, exists := lm.lifecycleStates[agentID]
	if !exists {
		return false
	}

	return state.Status == LifecycleStatusRemoved
}

// GetActiveAgents يعيد جميع الوكلاء النشطين
func (lm *LifecycleManager) GetActiveAgents() []*LifecycleState {
	return lm.GetAgentsByStatus(LifecycleStatusActive)
}

// GetPausedAgents يعيد جميع الوكلاء المتوقفين مؤقتاً
func (lm *LifecycleManager) GetPausedAgents() []*LifecycleState {
	return lm.GetAgentsByStatus(LifecycleStatusPaused)
}

// GetFrozenAgents يعيد جميع الوكلاء المجمدة
func (lm *LifecycleManager) GetFrozenAgents() []*LifecycleState {
	return lm.GetAgentsByStatus(LifecycleStatusFrozen)
}

// GetRemovedAgents يعيد جميع الوكلاء المحذوفة
func (lm *LifecycleManager) GetRemovedAgents() []*LifecycleState {
	return lm.GetAgentsByStatus(LifecycleStatusRemoved)
}

// UpdateMetadata يحدث بيانات الوصفية للوكيل
func (lm *LifecycleManager) UpdateMetadata(agentID string, metadata map[string]interface{}) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	state, exists := lm.lifecycleStates[agentID]
	if !exists {
		return fmt.Errorf("الوكيل غير موجود في نظام دورة الحياة: %s", agentID)
	}

	if state.Metadata == nil {
		state.Metadata = make(map[string]interface{})
	}

	for key, value := range metadata {
		state.Metadata[key] = value
	}

	lm.logger.Info("تم تحديث بيانات الوصفية للوكيل",
		zap.String("agent_id", agentID),
	)

	return nil
}

// GetMetadata يعيد بيانات الوصفية للوكيل
func (lm *LifecycleManager) GetMetadata(agentID string) (map[string]interface{}, error) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	state, exists := lm.lifecycleStates[agentID]
	if !exists {
		return nil, fmt.Errorf("الوكيل غير موجود في نظام دورة الحياة: %s", agentID)
	}

	return state.Metadata, nil
}

// GetSummary يعيد ملخص نظام دورة الحياة
func (lm *LifecycleManager) GetSummary() map[string]interface{} {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	summary := map[string]interface{}{
		"total_agents":   len(lm.lifecycleStates),
		"active_agents":  0,
		"paused_agents":  0,
		"frozen_agents":  0,
		"removed_agents": 0,
	}

	for _, state := range lm.lifecycleStates {
		switch state.Status {
		case LifecycleStatusActive:
			summary["active_agents"] = summary["active_agents"].(int) + 1
		case LifecycleStatusPaused:
			summary["paused_agents"] = summary["paused_agents"].(int) + 1
		case LifecycleStatusFrozen:
			summary["frozen_agents"] = summary["frozen_agents"].(int) + 1
		case LifecycleStatusRemoved:
			summary["removed_agents"] = summary["removed_agents"].(int) + 1
		}
	}

	return summary
}
