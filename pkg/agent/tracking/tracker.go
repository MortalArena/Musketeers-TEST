package tracking

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Checkpoint نقطة تفتيش
type Checkpoint struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Status      string                 `json:"status"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Reminder تذكير
type Reminder struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	DueTime     time.Time              `json:"due_time"`
	Priority    string                 `json:"priority"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ProgressTracker متتبع التقدم
type ProgressTracker struct {
	checkpoints  []*Checkpoint
	reminders    []*Reminder
	logger       *zap.Logger
	mu           sync.RWMutex
	sessionID    string
	agentID      string
	currentStep  int
	totalSteps   int
}

// NewProgressTracker ينشئ متتبع تقدم جديد
func NewProgressTracker(sessionID, agentID string, totalSteps int, logger *zap.Logger) *ProgressTracker {
	return &ProgressTracker{
		checkpoints: make([]*Checkpoint, 0),
		reminders:   make([]*Reminder, 0),
		logger:      logger,
		sessionID:   sessionID,
		agentID:     agentID,
		currentStep: 0,
		totalSteps:  totalSteps,
	}
}

// AddCheckpoint يضيف نقطة تفتيش
func (pt *ProgressTracker) AddCheckpoint(ctx context.Context, name, description string, status string, metadata map[string]interface{}) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	checkpoint := &Checkpoint{
		ID:          fmt.Sprintf("checkpoint_%d", time.Now().UnixNano()),
		Name:        name,
		Description: description,
		Status:      status,
		Timestamp:   time.Now(),
		Metadata:    metadata,
	}

	pt.checkpoints = append(pt.checkpoints, checkpoint)

	pt.logger.Info("تم إضافة نقطة تفتيش",
		zap.String("session_id", pt.sessionID),
		zap.String("agent_id", pt.agentID),
		zap.String("checkpoint_id", checkpoint.ID),
		zap.String("name", name),
		zap.String("status", status),
	)

	return nil
}

// GetCheckpoints يرجع جميع نقاط التفتيش
func (pt *ProgressTracker) GetCheckpoints(ctx context.Context) ([]*Checkpoint, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	return pt.checkpoints, nil
}

// GetLastCheckpoint يرجع آخر نقطة تفتيش
func (pt *ProgressTracker) GetLastCheckpoint(ctx context.Context) (*Checkpoint, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if len(pt.checkpoints) == 0 {
		return nil, fmt.Errorf("لا توجد نقاط تفتيش")
	}

	return pt.checkpoints[len(pt.checkpoints)-1], nil
}

// AddReminder يضيف تذكير
func (pt *ProgressTracker) AddReminder(ctx context.Context, title, description string, dueTime time.Time, priority string, metadata map[string]interface{}) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	reminder := &Reminder{
		ID:          fmt.Sprintf("reminder_%d", time.Now().UnixNano()),
		Title:       title,
		Description: description,
		DueTime:     dueTime,
		Priority:    priority,
		Status:      "pending",
		CreatedAt:   time.Now(),
		Metadata:    metadata,
	}

	pt.reminders = append(pt.reminders, reminder)

	pt.logger.Info("تم إضافة تذكير",
		zap.String("session_id", pt.sessionID),
		zap.String("agent_id", pt.agentID),
		zap.String("reminder_id", reminder.ID),
		zap.String("title", title),
		zap.Time("due_time", dueTime),
	)

	return nil
}

// GetReminders يرجع جميع التذكيرات
func (pt *ProgressTracker) GetReminders(ctx context.Context) ([]*Reminder, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	return pt.reminders, nil
}

// GetPendingReminders يرجع التذكيرات المعلقة
func (pt *ProgressTracker) GetPendingReminders(ctx context.Context) ([]*Reminder, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	var result []*Reminder
	for _, reminder := range pt.reminders {
		if reminder.Status == "pending" {
			result = append(result, reminder)
		}
	}

	return result, nil
}

// GetOverdueReminders يرجع التذكيرات المتأخرة
func (pt *ProgressTracker) GetOverdueReminders(ctx context.Context) ([]*Reminder, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	var result []*Reminder
	now := time.Now()
	for _, reminder := range pt.reminders {
		if reminder.Status == "pending" && reminder.DueTime.Before(now) {
			result = append(result, reminder)
		}
	}

	return result, nil
}

// CompleteReminder يكمل تذكير
func (pt *ProgressTracker) CompleteReminder(ctx context.Context, reminderID string) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	for _, reminder := range pt.reminders {
		if reminder.ID == reminderID {
			reminder.Status = "completed"
			now := time.Now()
			reminder.CompletedAt = &now

			pt.logger.Info("تم إكمال التذكير",
				zap.String("session_id", pt.sessionID),
				zap.String("agent_id", pt.agentID),
				zap.String("reminder_id", reminderID),
			)

			return nil
		}
	}

	return fmt.Errorf("تذكير غير موجود: %s", reminderID)
}

// IncrementStep يزيد الخطوة الحالية
func (pt *ProgressTracker) IncrementStep(ctx context.Context) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.currentStep < pt.totalSteps {
		pt.currentStep++
		pt.logger.Info("تم زيادة الخطوة",
			zap.String("session_id", pt.sessionID),
			zap.String("agent_id", pt.agentID),
			zap.Int("current_step", pt.currentStep),
			zap.Int("total_steps", pt.totalSteps),
		)
	}

	return nil
}

// GetCurrentStep يرجع الخطوة الحالية
func (pt *ProgressTracker) GetCurrentStep(ctx context.Context) (int, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	return pt.currentStep, nil
}

// GetProgress يرجع التقدم
func (pt *ProgressTracker) GetProgress(ctx context.Context) (map[string]interface{}, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	progress := map[string]interface{}{
		"current_step": pt.currentStep,
		"total_steps":  pt.totalSteps,
		"percentage":   0.0,
	}

	if pt.totalSteps > 0 {
		progress["percentage"] = float64(pt.currentStep) / float64(pt.totalSteps) * 100
	}

	return progress, nil
}

// CheckForGaps يتحقق من الفجوات في التقدم
func (pt *ProgressTracker) CheckForGaps(ctx context.Context) ([]string, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	var gaps []string

	// التحقق من التذكيرات المتأخرة
	overdueReminders, _ := pt.GetOverdueReminders(ctx)
	for _, reminder := range overdueReminders {
		gaps = append(gaps, fmt.Sprintf("تذكير متأخر: %s", reminder.Title))
	}

	// التحقق من نقاط التفتيش المفقودة
	if len(pt.checkpoints) < pt.currentStep {
		gaps = append(gaps, fmt.Sprintf("نقاط تفتيش مفقودة: متوقع %d، موجود %d", pt.currentStep, len(pt.checkpoints)))
	}

	return gaps, nil
}

// CreateAutoReminders ينشئ تذكيرات تلقائية بناءً على التقدم
func (pt *ProgressTracker) CreateAutoReminders(ctx context.Context) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	// [WHY] إنشاء تذكيرات تلقائية لضمان عدم النسيان
	// [HOW] ينشئ تذكيرات للخطوات المتبقية
	// [SAFETY] يضمن عدم نسيان أي خطوة مهمة

	if pt.currentStep < pt.totalSteps {
		remainingSteps := pt.totalSteps - pt.currentStep
		reminder := &Reminder{
			ID:          fmt.Sprintf("auto_reminder_%d", time.Now().UnixNano()),
			Title:       fmt.Sprintf("إكمال الخطوات المتبقية (%d)", remainingSteps),
			Description: fmt.Sprintf("لا تزال هناك %d خطوات متبقية لإكمال المهمة", remainingSteps),
			DueTime:     time.Now().Add(5 * time.Minute),
			Priority:    "high",
			Status:      "pending",
			CreatedAt:   time.Now(),
			Metadata:    map[string]interface{}{"auto": true, "remaining_steps": remainingSteps},
		}

		pt.reminders = append(pt.reminders, reminder)

		pt.logger.Info("تم إنشاء تذكير تلقائي",
			zap.String("session_id", pt.sessionID),
			zap.String("agent_id", pt.agentID),
			zap.String("reminder_id", reminder.ID),
			zap.Int("remaining_steps", remainingSteps),
		)
	}

	return nil
}

// GetSummary يرجع ملخص التتبع
func (pt *ProgressTracker) GetSummary(ctx context.Context) (map[string]interface{}, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	pendingReminders, _ := pt.GetPendingReminders(ctx)
	overdueReminders, _ := pt.GetOverdueReminders(ctx)

	summary := map[string]interface{}{
		"session_id":          pt.sessionID,
		"agent_id":            pt.agentID,
		"current_step":        pt.currentStep,
		"total_steps":         pt.totalSteps,
		"checkpoints_count":   len(pt.checkpoints),
		"reminders_count":     len(pt.reminders),
		"pending_reminders":   len(pendingReminders),
		"overdue_reminders":   len(overdueReminders),
		"progress_percentage": 0.0,
	}

	if pt.totalSteps > 0 {
		summary["progress_percentage"] = float64(pt.currentStep) / float64(pt.totalSteps) * 100
	}

	return summary, nil
}
