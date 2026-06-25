package metrics

import (
	"sync"

	"go.uber.org/zap"
)

// Metrics مقاييس بسيطة لمراقبة الأداء
type Metrics struct {
	logger *zap.Logger

	// Task metrics
	taskSuccess map[string]int
	taskFailure map[string]int

	// Agent metrics
	agentActive map[string]int
	agentTotal  map[string]int

	// Session metrics
	sessionActive map[string]int
	sessionTotal  map[string]int

	// Error metrics
	errorCount map[string]int

	mu sync.RWMutex
}

// NewMetrics ينشئ مقاييس جديدة
func NewMetrics(logger *zap.Logger) *Metrics {
	return &Metrics{
		logger: logger,

		// Task metrics
		taskSuccess: make(map[string]int),
		taskFailure: make(map[string]int),

		// Agent metrics
		agentActive: make(map[string]int),
		agentTotal:  make(map[string]int),

		// Session metrics
		sessionActive: make(map[string]int),
		sessionTotal:  make(map[string]int),

		// Error metrics
		errorCount: make(map[string]int),
	}
}

// RecordTaskSuccess يسجل نجاح المهمة
func (m *Metrics) RecordTaskSuccess(taskType, agentID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := taskType + ":" + agentID
	m.taskSuccess[key]++
}

// RecordTaskFailure يسجل فشل المهمة
func (m *Metrics) RecordTaskFailure(taskType, agentID, errorType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := taskType + ":" + agentID
	m.taskFailure[key]++
}

// SetAgentActive يضبط عدد الوكلاء النشطين
func (m *Metrics) SetAgentActive(agentType, sessionID string, count int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := agentType + ":" + sessionID
	m.agentActive[key] = count
}

// IncrementAgentTotal يزيد عدد الوكلاء الإجمالي
func (m *Metrics) IncrementAgentTotal(agentType, sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := agentType + ":" + sessionID
	m.agentTotal[key]++
}

// SetSessionActive يضبط عدد الجلسات النشطة
func (m *Metrics) SetSessionActive(sessionType string, count int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessionActive[sessionType] = count
}

// IncrementSessionTotal يزيد عدد الجلسات الإجمالي
func (m *Metrics) IncrementSessionTotal(sessionType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessionTotal[sessionType]++
}

// RecordError يسجل خطأ
func (m *Metrics) RecordError(errorType, component string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := errorType + ":" + component
	m.errorCount[key]++
}

// GetSummary يحصل على ملخص المقاييس
func (m *Metrics) GetSummary() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"task_success":   m.taskSuccess,
		"task_failure":   m.taskFailure,
		"agent_active":   m.agentActive,
		"agent_total":    m.agentTotal,
		"session_active": m.sessionActive,
		"session_total":  m.sessionTotal,
		"error_count":    m.errorCount,
	}
}
