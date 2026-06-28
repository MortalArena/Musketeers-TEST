package session

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

// ProgressMetric مقياس التقدم
type ProgressMetric struct {
	TaskID    string                 `json:"task_id"`
	AgentID   string                 `json:"agent_id"`
	Phase     string                 `json:"phase"`
	Progress  float64                `json:"progress"` // 0-100
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// DelayInfo معلومات التأخير
type DelayInfo struct {
	TaskID        string        `json:"task_id"`
	AgentID       string        `json:"agent_id"`
	ExpectedTime  time.Time     `json:"expected_time"`
	ActualTime    time.Time     `json:"actual_time"`
	DelayDuration time.Duration `json:"delay_duration"`
	Reason        string        `json:"reason"`
}

// RiskRisk مستوى المخاطرة
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// RiskInfo معلومات المخاطرة
type RiskInfo struct {
	TaskID      string                 `json:"task_id"`
	AgentID     string                 `json:"agent_id"`
	RiskLevel   RiskLevel              `json:"risk_level"`
	Description string                 `json:"description"`
	DetectedAt  time.Time              `json:"detected_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ProgressTracker متتبع التقدم - يراقب تقدم المهام ويكتشف المشاكل
type ProgressTracker struct {
	sessionID string
	logger    *zap.Logger
	eventBus  *eventbus.EventBus

	// بيانات التتبع
	progressMetrics map[string][]ProgressMetric // taskID -> metrics
	delays          map[string]DelayInfo        // taskID -> delay
	risks           map[string]RiskInfo         // taskID -> risk

	// إحصائيات
	totalTasks     int
	completedTasks int
	delayedTasks   int
	atRiskTasks    int

	mu sync.RWMutex
}

// [SAFETY] حدود الموارد لمنع استهلاك غير محدود
const (
	// [SAFETY] الحد الأقصى لعدد مقاييس التقدم لكل مهمة
	MaxProgressMetricsPerTask = 1000
	// [SAFETY] الحد الأقصى لعدد التأخيرات
	MaxDelays = 500
	// [SAFETY] الحد الأقصى لعدد المخاطر
	MaxRisks = 100
	// [SAFETY] الحد الأقصى لقيمة التقدم
	MaxProgress = 100.0
	// [SAFETY] الحد الأدنى لقيمة التقدم
	MinProgress = 0.0
)

// NewProgressTracker ينشئ متتبع تقدم جديد
func NewProgressTracker(sessionID string) *ProgressTracker {
	return &ProgressTracker{
		sessionID:       sessionID,
		logger:          zap.NewNop(), // سيتم استبداله بـ logger حقيقي
		eventBus:        nil,          // سيتم تعيينه لاحقاً
		progressMetrics: make(map[string][]ProgressMetric),
		delays:          make(map[string]DelayInfo),
		risks:           make(map[string]RiskInfo),
	}
}

// SetLogger يضبط logger
func (pt *ProgressTracker) SetLogger(logger *zap.Logger) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.logger = logger
}

// SetEventBus يضبط event bus
func (pt *ProgressTracker) SetEventBus(eb *eventbus.EventBus) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.eventBus = eb

	// الاشتراك في أحداث المهام
	if eb != nil {
		eb.Subscribe("task.created", pt.handleTaskCreated)
		eb.Subscribe("task.started", pt.handleTaskStarted)
		eb.Subscribe("task.completed", pt.handleTaskCompleted)
		eb.Subscribe("task.failed", pt.handleTaskFailed)
		eb.Subscribe("task.assigned", pt.handleTaskAssigned)
	}
}

// RecordProgress يسجل تقدم مهمة
func (pt *ProgressTracker) RecordProgress(ctx context.Context, taskID, agentID, phase string, progress float64, metadata map[string]interface{}) error {
	// [SAFETY] التحقق من صحة المدخلات
	if taskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}
	if agentID == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}
	if phase == "" {
		return fmt.Errorf("phase cannot be empty")
	}
	if progress < MinProgress || progress > MaxProgress {
		return fmt.Errorf("progress must be between %.1f and %.1f", MinProgress, MaxProgress)
	}

	pt.mu.Lock()
	defer pt.mu.Unlock()

	// [SAFETY] التحقق من الحد الأقصى لمقاييس التقدم
	if len(pt.progressMetrics[taskID]) >= MaxProgressMetricsPerTask {
		return fmt.Errorf("maximum progress metrics per task limit reached (%d)", MaxProgressMetricsPerTask)
	}

	metric := ProgressMetric{
		TaskID:    taskID,
		AgentID:   agentID,
		Phase:     phase,
		Progress:  progress,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	pt.progressMetrics[taskID] = append(pt.progressMetrics[taskID], metric)

	pt.logger.Info("Progress recorded",
		zap.String("task_id", taskID),
		zap.String("agent_id", agentID),
		zap.String("phase", phase),
		zap.Float64("progress", progress),
	)

	if pt.eventBus != nil {
		pt.eventBus.Publish(eventbus.Event{
			Type:      "progress.recorded",
			Payload:   metric,
			Source:    "progress_tracker",
			SessionID: pt.sessionID,
		})
	}

	// فحص المخاطر
	pt.checkForRisks(taskID, agentID, progress)

	return nil
}

// RecordDelay يسجل تأخير مهمة
func (pt *ProgressTracker) RecordDelay(ctx context.Context, taskID, agentID string, expectedTime, actualTime time.Time, reason string) error {
	// [SAFETY] التحقق من صحة المدخلات
	if taskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}
	if agentID == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}
	if reason == "" {
		return fmt.Errorf("reason cannot be empty")
	}

	pt.mu.Lock()
	defer pt.mu.Unlock()

	// [SAFETY] التحقق من الحد الأقصى للتأخيرات
	if len(pt.delays) >= MaxDelays {
		return fmt.Errorf("maximum delays limit reached (%d)", MaxDelays)
	}

	delayDuration := actualTime.Sub(expectedTime)
	if delayDuration < 0 {
		delayDuration = 0
	}

	delay := DelayInfo{
		TaskID:        taskID,
		AgentID:       agentID,
		ExpectedTime:  expectedTime,
		ActualTime:    actualTime,
		DelayDuration: delayDuration,
		Reason:        reason,
	}

	pt.delays[taskID] = delay
	pt.delayedTasks++

	pt.logger.Warn("Delay recorded",
		zap.String("task_id", taskID),
		zap.String("agent_id", agentID),
		zap.Duration("delay", delayDuration),
		zap.String("reason", reason),
	)

	if pt.eventBus != nil {
		pt.eventBus.Publish(eventbus.Event{
			Type:      "delay.recorded",
			Payload:   delay,
			Source:    "progress_tracker",
			SessionID: pt.sessionID,
		})
	}

	// تحديد مستوى المخاطرة بناءً على التأخير
	riskLevel := pt.calculateDelayRisk(delayDuration)
	if riskLevel != RiskLevelLow {
		pt.recordRisk(taskID, agentID, riskLevel, fmt.Sprintf("Task delayed by %v: %s", delayDuration, reason), nil)
	}

	return nil
}

// RecordRisk يسجل مخاطرة
func (pt *ProgressTracker) RecordRisk(ctx context.Context, taskID, agentID string, riskLevel RiskLevel, description string, metadata map[string]interface{}) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.recordRisk(taskID, agentID, riskLevel, description, metadata)

	return nil
}

// recordRisk يسجل مخاطرة (داخلي)
func (pt *ProgressTracker) recordRisk(taskID, agentID string, riskLevel RiskLevel, description string, metadata map[string]interface{}) {
	// [SAFETY] التحقق من الحد الأقصى للمخاطر
	if len(pt.risks) >= MaxRisks {
		pt.logger.Warn("Maximum risks limit reached, skipping risk recording",
			zap.String("task_id", taskID),
			zap.String("risk_level", string(riskLevel)),
		)
		return
	}

	risk := RiskInfo{
		TaskID:      taskID,
		AgentID:     agentID,
		RiskLevel:   riskLevel,
		Description: description,
		DetectedAt:  time.Now(),
		Metadata:    metadata,
	}

	pt.risks[taskID] = risk
	pt.atRiskTasks++

	pt.logger.Warn("Risk detected",
		zap.String("task_id", taskID),
		zap.String("agent_id", agentID),
		zap.String("risk_level", string(riskLevel)),
		zap.String("description", description),
	)

	if pt.eventBus != nil {
		pt.eventBus.Publish(eventbus.Event{
			Type:      "risk.detected",
			Payload:   risk,
			Source:    "progress_tracker",
			SessionID: pt.sessionID,
		})
	}
}

// GetProgressMetrics يحصل على مقاييس التقدم لمهمة
func (pt *ProgressTracker) GetProgressMetrics(taskID string) []ProgressMetric {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	metrics, exists := pt.progressMetrics[taskID]
	if !exists {
		return []ProgressMetric{}
	}

	// إنشاء نسخة لتجنب التعديل الخارجي
	result := make([]ProgressMetric, len(metrics))
	copy(result, metrics)
	return result
}

// GetDelayInfo يحصل على معلومات التأخير لمهمة
func (pt *ProgressTracker) GetDelayInfo(taskID string) (DelayInfo, bool) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	delay, exists := pt.delays[taskID]
	return delay, exists
}

// GetRiskInfo يحصل على معلومات المخاطرة لمهمة
func (pt *ProgressTracker) GetRiskInfo(taskID string) (RiskInfo, bool) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	risk, exists := pt.risks[taskID]
	return risk, exists
}

// GetAllRisks يحصل على جميع المخاطر
func (pt *ProgressTracker) GetAllRisks() map[string]RiskInfo {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	result := make(map[string]RiskInfo, len(pt.risks))
	for k, v := range pt.risks {
		result[k] = v
	}
	return result
}

// GetStats يحصل على إحصائيات
func (pt *ProgressTracker) GetStats() map[string]interface{} {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	return map[string]interface{}{
		"total_tasks":     pt.totalTasks,
		"completed_tasks": pt.completedTasks,
		"delayed_tasks":   pt.delayedTasks,
		"at_risk_tasks":   pt.atRiskTasks,
		"active_risks":    len(pt.risks),
		"total_delays":    len(pt.delays),
	}
}

// GetOverallProgress يحسب التقدم العام
func (pt *ProgressTracker) GetOverallProgress() float64 {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if pt.totalTasks == 0 {
		return 0
	}

	return float64(pt.completedTasks) / float64(pt.totalTasks) * 100
}

// checkForRisks يفحص المخاطر بناءً على التقدم
func (pt *ProgressTracker) checkForRisks(taskID, agentID string, progress float64) {
	metrics := pt.progressMetrics[taskID]
	if len(metrics) < 2 {
		return
	}

	// فحص التقدم البطيء
	lastMetric := metrics[len(metrics)-1]
	prevMetric := metrics[len(metrics)-2]
	timeDiff := lastMetric.Timestamp.Sub(prevMetric.Timestamp).Minutes()
	progressDiff := lastMetric.Progress - prevMetric.Progress

	if timeDiff > 5 && progressDiff < 1 {
		// تقدم بطيء جداً خلال 5 دقائق
		pt.recordRisk(taskID, agentID, RiskLevelMedium, "Slow progress detected", map[string]interface{}{
			"time_diff_minutes": timeDiff,
			"progress_diff":     progressDiff,
		})
	}

	// فحص عدم التقدم
	if progressDiff == 0 && timeDiff > 10 {
		pt.recordRisk(taskID, agentID, RiskLevelHigh, "No progress detected", map[string]interface{}{
			"time_diff_minutes": timeDiff,
		})
	}
}

// calculateDelayRisk يحسب مستوى المخاطرة بناءً على التأخير
func (pt *ProgressTracker) calculateDelayRisk(delay time.Duration) RiskLevel {
	if delay < 5*time.Minute {
		return RiskLevelLow
	}
	if delay < 15*time.Minute {
		return RiskLevelMedium
	}
	if delay < 30*time.Minute {
		return RiskLevelHigh
	}
	return RiskLevelCritical
}

// معالجات الأحداث
func (pt *ProgressTracker) handleTaskCreated(e eventbus.Event) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.totalTasks++
}

func (pt *ProgressTracker) handleTaskStarted(e eventbus.Event) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	// يمكن إضافة منطق خاص عند بدء المهمة
}

func (pt *ProgressTracker) handleTaskCompleted(e eventbus.Event) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.completedTasks++

	// إزالة المخاطر والتأخيرات للمهمة المكتملة
	if payload, ok := e.Payload.(map[string]interface{}); ok {
		if taskID, ok := payload["task_id"].(string); ok {
			delete(pt.risks, taskID)
			delete(pt.delays, taskID)
		}
	}
}

func (pt *ProgressTracker) handleTaskFailed(e eventbus.Event) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.completedTasks++

	// تسجيل مخاطرة عالية للمهمة الفاشلة
	if payload, ok := e.Payload.(map[string]interface{}); ok {
		taskID, _ := payload["task_id"].(string)
		errorMsg, _ := payload["error"].(string)
		pt.recordRisk(taskID, "", RiskLevelCritical, fmt.Sprintf("Task failed: %s", errorMsg), nil)
	}
}

func (pt *ProgressTracker) handleTaskAssigned(e eventbus.Event) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	// يمكن إضافة منطق خاص عند تعيين المهمة
}

// Save يحفظ حالة ProgressTracker
func (pt *ProgressTracker) Save() ([]byte, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	data := struct {
		ProgressMetrics map[string][]ProgressMetric `json:"progress_metrics"`
		Delays          map[string]DelayInfo        `json:"delays"`
		Risks           map[string]RiskInfo         `json:"risks"`
		TotalTasks      int                         `json:"total_tasks"`
		CompletedTasks  int                         `json:"completed_tasks"`
		DelayedTasks    int                         `json:"delayed_tasks"`
		AtRiskTasks     int                         `json:"at_risk_tasks"`
	}{
		ProgressMetrics: pt.progressMetrics,
		Delays:          pt.delays,
		Risks:           pt.risks,
		TotalTasks:      pt.totalTasks,
		CompletedTasks:  pt.completedTasks,
		DelayedTasks:    pt.delayedTasks,
		AtRiskTasks:     pt.atRiskTasks,
	}

	return json.Marshal(data)
}

// Load يحمل حالة ProgressTracker
func (pt *ProgressTracker) Load(data []byte) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	var loaded struct {
		ProgressMetrics map[string][]ProgressMetric `json:"progress_metrics"`
		Delays          map[string]DelayInfo        `json:"delays"`
		Risks           map[string]RiskInfo         `json:"risks"`
		TotalTasks      int                         `json:"total_tasks"`
		CompletedTasks  int                         `json:"completed_tasks"`
		DelayedTasks    int                         `json:"delayed_tasks"`
		AtRiskTasks     int                         `json:"at_risk_tasks"`
	}

	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}

	pt.progressMetrics = loaded.ProgressMetrics
	if pt.progressMetrics == nil {
		pt.progressMetrics = make(map[string][]ProgressMetric)
	}
	pt.delays = loaded.Delays
	if pt.delays == nil {
		pt.delays = make(map[string]DelayInfo)
	}
	pt.risks = loaded.Risks
	if pt.risks == nil {
		pt.risks = make(map[string]RiskInfo)
	}
	pt.totalTasks = loaded.TotalTasks
	pt.completedTasks = loaded.CompletedTasks
	pt.delayedTasks = loaded.DelayedTasks
	pt.atRiskTasks = loaded.AtRiskTasks

	return nil
}
