package unified

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"go.uber.org/zap"
)

// UnifiedMemoryManager نظام الذاكرة الشامل الذي يدمج جميع وظائف الذاكرة
type UnifiedMemoryManager struct {
	sessionID string
	logger    *zap.Logger
	mu        sync.RWMutex

	// قاعدة البيانات (من sessionMemory)
	db *badger.DB

	// الذاكرة العرضية (من sessionMemory)
	episodic []MemoryEvent

	// الذاكرة الدلالية (من sessionMemory)
	semantic []MemoryFact

	// الذاكرة الإجرائية (من sessionMemory)
	procedural []MemoryWorkflow

	// الذاكرة الاستراتيجية (من sessionMemory)
	meta []MemoryStrategy

	// الذاكرة الداخلية (من agentMemory)
	internalMemory map[string]*MemoryItem

	// إحصائيات
	totalEvents     int
	totalFacts      int
	totalWorkflows  int
	totalStrategies int
}

// MemoryEvent حدث في الذاكرة العرضية
type MemoryEvent struct {
	ID         string                 `json:"id"`
	Timestamp  time.Time              `json:"timestamp"`
	AgentDID   string                 `json:"agent_did"`
	Action     string                 `json:"action"`
	Context    map[string]interface{} `json:"context"`
	Outcome    string                 `json:"outcome"`
	Lessons    []string               `json:"lessons"`
	Confidence float64                `json:"confidence"`
	Tags       []string               `json:"tags"`
}

// MemoryFact حقيقة في الذاكرة الدلالية
type MemoryFact struct {
	ID         string    `json:"id"`
	Statement  string    `json:"statement"`
	Category   string    `json:"category"`
	Confidence float64   `json:"confidence"`
	Source     string    `json:"source"`
	Timestamp  time.Time `json:"timestamp"`
	Tags       []string  `json:"tags"`
}

// MemoryWorkflow طريقة في الذاكرة الإجرائية
type MemoryWorkflow struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Steps       []string               `json:"steps"`
	Parameters  map[string]interface{} `json:"parameters"`
	SuccessRate float64                `json:"success_rate"`
	UsageCount  int                    `json:"usage_count"`
	Timestamp   time.Time              `json:"timestamp"`
}

// MemoryStrategy استراتيجية في الذاكرة الاستراتيجية
type MemoryStrategy struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Context     []string               `json:"context"`
	Effectiveness float64              `json:"effectiveness"`
	UsageCount  int                    `json:"usage_count"`
	Timestamp   time.Time              `json:"timestamp"`
}

// MemoryItem عنصر ذاكرة داخلي (من agentMemory)
type MemoryItem struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Content    string                 `json:"content"`
	Source     string                 `json:"source"`
	Importance float64                `json:"importance"`
	Tags       []string               `json:"tags"`
	Metadata   map[string]interface{} `json:"metadata"`
	Timestamp  time.Time              `json:"timestamp"`
}

// NewUnifiedMemoryManager ينشئ مدير ذاكرة شامل جديد
func NewUnifiedMemoryManager(sessionID string, db *badger.DB, logger *zap.Logger) *UnifiedMemoryManager {
	return &UnifiedMemoryManager{
		sessionID:      sessionID,
		logger:         logger,
		db:             db,
		episodic:       []MemoryEvent{},
		semantic:       []MemoryFact{},
		procedural:     []MemoryWorkflow{},
		meta:           []MemoryStrategy{},
		internalMemory: make(map[string]*MemoryItem),
	}
}

// RecordEvent يسجل حدث في الذاكرة العرضية
func (umm *UnifiedMemoryManager) RecordEvent(event MemoryEvent) error {
	umm.mu.Lock()
	defer umm.mu.Unlock()

	// إنشاء معرف فريد
	event.ID = fmt.Sprintf("event_%d", time.Now().UnixNano())
	event.Timestamp = time.Now()

	// إضافة إلى الذاكرة العرضية
	umm.episodic = append(umm.episodic, event)
	umm.totalEvents++

	// حفظ في قاعدة البيانات
	if umm.db != nil {
		key := []byte(fmt.Sprintf("episodic:%s", event.ID))
		value, _ := json.Marshal(event)
		if err := umm.db.Update(func(txn *badger.Txn) error {
			return txn.Set(key, value)
		}); err != nil {
			umm.logger.Error("فشل حفظ الحدث في قاعدة البيانات", zap.Error(err))
		}
	}

	umm.logger.Info("تم تسجيل حدث في الذاكرة العرضية",
		zap.String("event_id", event.ID),
		zap.String("action", event.Action))

	return nil
}

// AddFact يضيف حقيقة إلى الذاكرة الدلالية
func (umm *UnifiedMemoryManager) AddFact(fact MemoryFact) error {
	umm.mu.Lock()
	defer umm.mu.Unlock()

	// إنشاء معرف فريد
	fact.ID = fmt.Sprintf("fact_%d", time.Now().UnixNano())
	fact.Timestamp = time.Now()

	// إضافة إلى الذاكرة الدلالية
	umm.semantic = append(umm.semantic, fact)
	umm.totalFacts++

	// حفظ في قاعدة البيانات
	if umm.db != nil {
		key := []byte(fmt.Sprintf("semantic:%s", fact.ID))
		value, _ := json.Marshal(fact)
		if err := umm.db.Update(func(txn *badger.Txn) error {
			return txn.Set(key, value)
		}); err != nil {
			umm.logger.Error("فشل حفظ الحقيقة في قاعدة البيانات", zap.Error(err))
		}
	}

	umm.logger.Info("تم إضافة حقيقة إلى الذاكرة الدلالية",
		zap.String("fact_id", fact.ID),
		zap.String("statement", fact.Statement))

	return nil
}

// AddWorkflow يضيف طريقة إلى الذاكرة الإجرائية
func (umm *UnifiedMemoryManager) AddWorkflow(workflow MemoryWorkflow) error {
	umm.mu.Lock()
	defer umm.mu.Unlock()

	// إنشاء معرف فريد
	workflow.ID = fmt.Sprintf("workflow_%d", time.Now().UnixNano())
	workflow.Timestamp = time.Now()

	// إضافة إلى الذاكرة الإجرائية
	umm.procedural = append(umm.procedural, workflow)
	umm.totalWorkflows++

	// حفظ في قاعدة البيانات
	if umm.db != nil {
		key := []byte(fmt.Sprintf("procedural:%s", workflow.ID))
		value, _ := json.Marshal(workflow)
		if err := umm.db.Update(func(txn *badger.Txn) error {
			return txn.Set(key, value)
		}); err != nil {
			umm.logger.Error("فشل حفظ الطريقة في قاعدة البيانات", zap.Error(err))
		}
	}

	umm.logger.Info("تم إضافة طريقة إلى الذاكرة الإجرائية",
		zap.String("workflow_id", workflow.ID),
		zap.String("name", workflow.Name))

	return nil
}

// AddStrategy يضيف استراتيجية إلى الذاكرة الاستراتيجية
func (umm *UnifiedMemoryManager) AddStrategy(strategy MemoryStrategy) error {
	umm.mu.Lock()
	defer umm.mu.Unlock()

	// إنشاء معرف فريد
	strategy.ID = fmt.Sprintf("strategy_%d", time.Now().UnixNano())
	strategy.Timestamp = time.Now()

	// إضافة إلى الذاكرة الاستراتيجية
	umm.meta = append(umm.meta, strategy)
	umm.totalStrategies++

	// حفظ في قاعدة البيانات
	if umm.db != nil {
		key := []byte(fmt.Sprintf("meta:%s", strategy.ID))
		value, _ := json.Marshal(strategy)
		if err := umm.db.Update(func(txn *badger.Txn) error {
			return txn.Set(key, value)
		}); err != nil {
			umm.logger.Error("فشل حفظ الاستراتيجية في قاعدة البيانات", zap.Error(err))
		}
	}

	umm.logger.Info("تم إضافة استراتيجية إلى الذاكرة الاستراتيجية",
		zap.String("strategy_id", strategy.ID),
		zap.String("name", strategy.Name))

	return nil
}

// AddMemory يضيف ذاكرة داخلية (من agentMemory)
func (umm *UnifiedMemoryManager) AddMemory(ctx context.Context, memoryType, content, source string, importance float64, tags []string, metadata map[string]interface{}) error {
	umm.mu.Lock()
	defer umm.mu.Unlock()

	// إنشاء معرف فريد
	id := fmt.Sprintf("memory_%d", time.Now().UnixNano())

	item := &MemoryItem{
		ID:         id,
		Type:       memoryType,
		Content:    content,
		Source:     source,
		Importance: importance,
		Tags:       tags,
		Metadata:   metadata,
		Timestamp:  time.Now(),
	}

	// إضافة إلى الذاكرة الداخلية
	umm.internalMemory[id] = item

	// حفظ في قاعدة البيانات
	if umm.db != nil {
		key := []byte(fmt.Sprintf("internal:%s", id))
		value, _ := json.Marshal(item)
		if err := umm.db.Update(func(txn *badger.Txn) error {
			return txn.Set(key, value)
		}); err != nil {
			umm.logger.Error("فشل حفظ الذاكرة الداخلية في قاعدة البيانات", zap.Error(err))
		}
	}

	umm.logger.Info("تم إضافة ذاكرة داخلية",
		zap.String("memory_id", id),
		zap.String("type", memoryType),
		zap.String("content", content))

	return nil
}

// SearchEvents يبحث عن أحداث
func (umm *UnifiedMemoryManager) SearchEvents(query string) []MemoryEvent {
	umm.mu.RLock()
	defer umm.mu.RUnlock()

	results := []MemoryEvent{}
	for _, event := range umm.episodic {
		if containsQuery(event.Action, query) ||
			containsQuery(event.Outcome, query) {
			results = append(results, event)
		}
	}

	return results
}

// SearchFacts يبحث عن حقائق
func (umm *UnifiedMemoryManager) SearchFacts(query string) []MemoryFact {
	umm.mu.RLock()
	defer umm.mu.RUnlock()

	results := []MemoryFact{}
	for _, fact := range umm.semantic {
		if containsQuery(fact.Statement, query) ||
			containsQuery(fact.Category, query) {
			results = append(results, fact)
		}
	}

	return results
}

// GetMemorySummary يحصل على ملخص الذاكرة
func (umm *UnifiedMemoryManager) GetMemorySummary() map[string]interface{} {
	umm.mu.RLock()
	defer umm.mu.RUnlock()

	return map[string]interface{}{
		"total_events":     umm.totalEvents,
		"total_facts":      umm.totalFacts,
		"total_workflows":  umm.totalWorkflows,
		"total_strategies": umm.totalStrategies,
		"total_internal":   len(umm.internalMemory),
	}
}

// containsQuery يتحقق من وجود استعلام في نص
func containsQuery(text, query string) bool {
	return len(text) > 0 && len(query) > 0 && len(text) >= len(query) && (text == query || len(text) > len(query) && (text[:len(query)] == query || text[len(text)-len(query):] == query))
}
