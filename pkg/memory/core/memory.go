package core

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// UnifiedMemory الذاكرة الموحدة للمنصة
type UnifiedMemory struct {
	sessionID string
	logger    *zap.Logger
	mu        sync.RWMutex

	// أنواع الذاكرة
	episodic   []*MemoryEvent
	semantic   []*MemoryFact
	procedural []*MemoryWorkflow
	meta       []*MemoryStrategy

	// إحصائيات
	totalEvents     int
	totalFacts      int
	totalWorkflows  int
	totalStrategies int

	// التخزين
	storage Storage
}

// MemoryEvent حدث في الذاكرة العرضية (ماذا حدث؟)
type MemoryEvent struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Source      string                 `json:"source"`
	Timestamp   time.Time              `json:"timestamp"`
	Confidence  float64                `json:"confidence"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// MemoryFact حقيقة في الذاكرة الدلالية (ماذا نعرف؟)
type MemoryFact struct {
	ID          string                 `json:"id"`
	Subject     string                 `json:"subject"`
	Predicate   string                 `json:"predicate"`
	Object      string                 `json:"object"`
	Confidence  float64                `json:"confidence"`
	AccessCount int                    `json:"access_count"`
	LastAccess  time.Time              `json:"last_access"`
	CreatedAt   time.Time              `json:"created_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// MemoryWorkflow workflow في الذاكرة الإجرائية (كيف نفعل؟)
type MemoryWorkflow struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Steps        []string               `json:"steps"`
	SuccessRate  float64                `json:"success_rate"`
	AvgDuration  time.Duration          `json:"avg_duration"`
	UsageCount   int                    `json:"usage_count"`
	LastUsed     time.Time              `json:"last_used"`
	CreatedAt    time.Time              `json:"created_at"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// MemoryStrategy استراتيجية في الذاكرة الوصفية (كيف نفكر؟)
type MemoryStrategy struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	SuccessRate float64                `json:"success_rate"`
	UsageCount  int                    `json:"usage_count"`
	LastUsed    time.Time              `json:"last_used"`
	CreatedAt   time.Time              `json:"created_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Storage واجهة التخزين
type Storage interface {
	Save(ctx context.Context, key string, data []byte) error
	Load(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}

// NewUnifiedMemory ينشئ ذاكرة موحدة جديدة
func NewUnifiedMemory(sessionID string, logger *zap.Logger, storage Storage) *UnifiedMemory {
	return &UnifiedMemory{
		sessionID:  sessionID,
		logger:     logger,
		episodic:   []*MemoryEvent{},
		semantic:   []*MemoryFact{},
		procedural: []*MemoryWorkflow{},
		meta:       []*MemoryStrategy{},
		storage:    storage,
	}
}

// RecordEvent يسجل حدث في الذاكرة العرضية
func (um *UnifiedMemory) RecordEvent(ctx context.Context, event *MemoryEvent) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	event.ID = generateID()
	event.Timestamp = time.Now()
	um.episodic = append(um.episodic, event)
	um.totalEvents++

	um.logger.Info("تم تسجيل حدث",
		zap.String("session_id", um.sessionID),
		zap.String("event_id", event.ID),
		zap.String("type", event.Type))

	// حفظ في التخزين
	if um.storage != nil {
		data, err := json.Marshal(event)
		if err == nil {
			key := fmt.Sprintf("memory/%s/event/%s", um.sessionID, event.ID)
			um.storage.Save(ctx, key, data)
		}
	}

	return nil
}

// LearnFact يتعلم حقيقة جديدة في الذاكرة الدلالية
func (um *UnifiedMemory) LearnFact(ctx context.Context, fact *MemoryFact) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	fact.ID = generateID()
	fact.CreatedAt = time.Now()
	fact.LastAccess = time.Now()
	um.semantic = append(um.semantic, fact)
	um.totalFacts++

	um.logger.Info("تم تعلم حقيقة",
		zap.String("session_id", um.sessionID),
		zap.String("fact_id", fact.ID),
		zap.String("subject", fact.Subject))

	// حفظ في التخزين
	if um.storage != nil {
		data, err := json.Marshal(fact)
		if err == nil {
			key := fmt.Sprintf("memory/%s/fact/%s", um.sessionID, fact.ID)
			um.storage.Save(ctx, key, data)
		}
	}

	return nil
}

// DiscoverWorkflow يكتشف workflow جديد في الذاكرة الإجرائية
func (um *UnifiedMemory) DiscoverWorkflow(ctx context.Context, workflow *MemoryWorkflow) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	workflow.ID = generateID()
	workflow.CreatedAt = time.Now()
	workflow.LastUsed = time.Now()
	um.procedural = append(um.procedural, workflow)
	um.totalWorkflows++

	um.logger.Info("تم اكتشاف workflow",
		zap.String("session_id", um.sessionID),
		zap.String("workflow_id", workflow.ID),
		zap.String("name", workflow.Name))

	// حفظ في التخزين
	if um.storage != nil {
		data, err := json.Marshal(workflow)
		if err == nil {
			key := fmt.Sprintf("memory/%s/workflow/%s", um.sessionID, workflow.ID)
			um.storage.Save(ctx, key, data)
		}
	}

	return nil
}

// DevelopStrategy يطور استراتيجية جديدة في الذاكرة الوصفية
func (um *UnifiedMemory) DevelopStrategy(ctx context.Context, strategy *MemoryStrategy) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	strategy.ID = generateID()
	strategy.CreatedAt = time.Now()
	strategy.LastUsed = time.Now()
	um.meta = append(um.meta, strategy)
	um.totalStrategies++

	um.logger.Info("تم تطوير استراتيجية",
		zap.String("session_id", um.sessionID),
		zap.String("strategy_id", strategy.ID),
		zap.String("name", strategy.Name))

	// حفظ في التخزين
	if um.storage != nil {
		data, err := json.Marshal(strategy)
		if err == nil {
			key := fmt.Sprintf("memory/%s/strategy/%s", um.sessionID, strategy.ID)
			um.storage.Save(ctx, key, data)
		}
	}

	return nil
}

// GetEvents يحصل على جميع الأحداث
func (um *UnifiedMemory) GetEvents() []*MemoryEvent {
	um.mu.RLock()
	defer um.mu.RUnlock()
	return um.episodic
}

// GetFacts يحصل على جميع الحقائق
func (um *UnifiedMemory) GetFacts() []*MemoryFact {
	um.mu.RLock()
	defer um.mu.RUnlock()
	return um.semantic
}

// GetWorkflows يحصل على جميع الـ workflows
func (um *UnifiedMemory) GetWorkflows() []*MemoryWorkflow {
	um.mu.RLock()
	defer um.mu.RUnlock()
	return um.procedural
}

// GetStrategies يحصل على جميع الاستراتيجيات
func (um *UnifiedMemory) GetStrategies() []*MemoryStrategy {
	um.mu.RLock()
	defer um.mu.RUnlock()
	return um.meta
}

// GetSummary يحصل على ملخص الذاكرة
func (um *UnifiedMemory) GetSummary() map[string]interface{} {
	um.mu.RLock()
	defer um.mu.RUnlock()

	return map[string]interface{}{
		"session_id":       um.sessionID,
		"total_events":     um.totalEvents,
		"total_facts":      um.totalFacts,
		"total_workflows":  um.totalWorkflows,
		"total_strategies": um.totalStrategies,
	}
}

// generateID يولد معرف فريد
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
