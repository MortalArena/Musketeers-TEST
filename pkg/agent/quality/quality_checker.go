package quality

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// QualityCheck فحص جودة
type QualityCheck struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Status      string                 `json:"status"` // "pending", "passed", "failed"
	Score       float64                `json:"score"` // 0.0 to 1.0
	Details     string                 `json:"details"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// QualityChecker مفتش الجودة
type QualityChecker struct {
	checks     []*QualityCheck
	logger     *zap.Logger
	mu         sync.RWMutex
	sessionID  string
	agentID    string
}

// NewQualityChecker ينشئ مفتش جودة جديد
func NewQualityChecker(sessionID, agentID string, logger *zap.Logger) *QualityChecker {
	return &QualityChecker{
		checks:    make([]*QualityCheck, 0),
		logger:    logger,
		sessionID: sessionID,
		agentID:   agentID,
	}
}

// AddCheck يضيف فحص جودة
func (qc *QualityChecker) AddCheck(ctx context.Context, name, description string) error {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	check := &QualityCheck{
		ID:          fmt.Sprintf("check_%d", time.Now().UnixNano()),
		Name:        name,
		Description: description,
		Status:      "pending",
		Score:       0.0,
		Timestamp:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	qc.checks = append(qc.checks, check)

	qc.logger.Info("تم إضافة فحص جودة",
		zap.String("session_id", qc.sessionID),
		zap.String("agent_id", qc.agentID),
		zap.String("check_id", check.ID),
		zap.String("name", name),
	)

	return nil
}

// PassCheck ينجح فحص جودة
func (qc *QualityChecker) PassCheck(ctx context.Context, checkID string, score float64, details string) error {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	for _, check := range qc.checks {
		if check.ID == checkID {
			check.Status = "passed"
			check.Score = score
			check.Details = details
			check.Timestamp = time.Now()

			qc.logger.Info("تم نجاح فحص الجودة",
				zap.String("session_id", qc.sessionID),
				zap.String("agent_id", qc.agentID),
				zap.String("check_id", checkID),
				zap.Float64("score", score),
			)

			return nil
		}
	}

	return fmt.Errorf("فحص غير موجود: %s", checkID)
}

// FailCheck يفشل فحص جودة
func (qc *QualityChecker) FailCheck(ctx context.Context, checkID string, details string) error {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	for _, check := range qc.checks {
		if check.ID == checkID {
			check.Status = "failed"
			check.Score = 0.0
			check.Details = details
			check.Timestamp = time.Now()

			qc.logger.Warn("فشل فحص الجودة",
				zap.String("session_id", qc.sessionID),
				zap.String("agent_id", qc.agentID),
				zap.String("check_id", checkID),
				zap.String("details", details),
			)

			return nil
		}
	}

	return fmt.Errorf("فحص غير موجود: %s", checkID)
}

// GetChecks يرجع جميع الفحوصات
func (qc *QualityChecker) GetChecks(ctx context.Context) ([]*QualityCheck, error) {
	qc.mu.RLock()
	defer qc.mu.RUnlock()

	return qc.checks, nil
}

// GetPendingChecks يرجع الفحوصات المعلقة
func (qc *QualityChecker) GetPendingChecks(ctx context.Context) ([]*QualityCheck, error) {
	qc.mu.RLock()
	defer qc.mu.RUnlock()

	var result []*QualityCheck
	for _, check := range qc.checks {
		if check.Status == "pending" {
			result = append(result, check)
		}
	}

	return result, nil
}

// GetPassedChecks يرجع الفحوصات الناجحة
func (qc *QualityChecker) GetPassedChecks(ctx context.Context) ([]*QualityCheck, error) {
	qc.mu.RLock()
	defer qc.mu.RUnlock()

	var result []*QualityCheck
	for _, check := range qc.checks {
		if check.Status == "passed" {
			result = append(result, check)
		}
	}

	return result, nil
}

// GetFailedChecks يرجع الفحوصات الفاشلة
func (qc *QualityChecker) GetFailedChecks(ctx context.Context) ([]*QualityCheck, error) {
	qc.mu.RLock()
	defer qc.mu.RUnlock()

	var result []*QualityCheck
	for _, check := range qc.checks {
		if check.Status == "failed" {
			result = append(result, check)
		}
	}

	return result, nil
}

// GetOverallQualityScore يحسب درجة الجودة الإجمالية
func (qc *QualityChecker) GetOverallQualityScore(ctx context.Context) (float64, error) {
	qc.mu.RLock()
	defer qc.mu.RUnlock()

	if len(qc.checks) == 0 {
		return 0.0, nil
	}

	totalScore := 0.0
	completedChecks := 0

	for _, check := range qc.checks {
		if check.Status == "passed" || check.Status == "failed" {
			totalScore += check.Score
			completedChecks++
		}
	}

	if completedChecks == 0 {
		return 0.0, nil
	}

	return totalScore / float64(completedChecks), nil
}

// RunStandardChecks ينفذ فحوصات قياسية
func (qc *QualityChecker) RunStandardChecks(ctx context.Context, task string, result interface{}, metadata map[string]interface{}) error {
	// [WHY] تنفيذ فحوصات قياسية للجودة
	// [HOW] يفحص النتائج للتأكد من صحتها
	// [SAFETY] يضمن عدم وجود أخطاء واضحة

	// فحص 1: التحقق من أن النتيجة ليست nil
	qc.AddCheck(ctx, "nil_check", "التحقق من أن النتيجة ليست nil")
	if result == nil {
		qc.FailCheck(ctx, "nil_check", "النتيجة nil")
	} else {
		qc.PassCheck(ctx, "nil_check", 1.0, "النتيجة ليست nil")
	}

	// فحص 2: التحقق من أن المهمة ليست فارغة
	qc.AddCheck(ctx, "task_check", "التحقق من أن المهمة ليست فارغة")
	if task == "" {
		qc.FailCheck(ctx, "task_check", "المهمة فارغة")
	} else {
		qc.PassCheck(ctx, "task_check", 1.0, "المهمة ليست فارغة")
	}

	// فحص 3: التحقق من أن البيانات الوصفية موجودة
	qc.AddCheck(ctx, "metadata_check", "التحقق من وجود البيانات الوصفية")
	if metadata == nil {
		qc.PassCheck(ctx, "metadata_check", 0.5, "البيانات الوصفية اختيارية")
	} else {
		qc.PassCheck(ctx, "metadata_check", 1.0, "البيانات الوصفية موجودة")
	}

	return nil
}

// GetQualitySummary يرجع ملخص الجودة
func (qc *QualityChecker) GetQualitySummary(ctx context.Context) (map[string]interface{}, error) {
	qc.mu.RLock()
	defer qc.mu.RUnlock()

	totalChecks := len(qc.checks)
	passedChecks := 0
	failedChecks := 0
	pendingChecks := 0

	for _, check := range qc.checks {
		switch check.Status {
		case "passed":
			passedChecks++
		case "failed":
			failedChecks++
		case "pending":
			pendingChecks++
		}
	}

	overallScore, _ := qc.GetOverallQualityScore(ctx)

	summary := map[string]interface{}{
		"session_id":      qc.sessionID,
		"agent_id":        qc.agentID,
		"total_checks":    totalChecks,
		"passed_checks":   passedChecks,
		"failed_checks":   failedChecks,
		"pending_checks":  pendingChecks,
		"overall_score":   overallScore,
		"quality_rating":  qc.getQualityRating(overallScore),
	}

	return summary, nil
}

// getQualityRating يرجع تقييم الجودة
func (qc *QualityChecker) getQualityRating(score float64) string {
	if score >= 0.9 {
		return "excellent"
	}
	if score >= 0.7 {
		return "good"
	}
	if score >= 0.5 {
		return "acceptable"
	}
	if score >= 0.3 {
		return "poor"
	}
	return "critical"
}

// ResetChecks يعيد تعيين جميع الفحوصات
func (qc *QualityChecker) ResetChecks(ctx context.Context) error {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	qc.checks = make([]*QualityCheck, 0)

	qc.logger.Info("تم إعادة تعيين فحوصات الجودة",
		zap.String("session_id", qc.sessionID),
		zap.String("agent_id", qc.agentID),
	)

	return nil
}
