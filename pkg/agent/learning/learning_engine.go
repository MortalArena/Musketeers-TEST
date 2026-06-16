package learning

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Skill مهارة
type Skill struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Level       float64                `json:"level"` // 0.0 to 1.0
	UsageCount  int                    `json:"usage_count"`
	SuccessRate float64                `json:"success_rate"`
	LastUsed    time.Time              `json:"last_used"`
	CreatedAt   time.Time              `json:"created_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Lesson درس مستفاد
type Lesson struct {
	ID           string                 `json:"id"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	Category     string                 `json:"category"`
	Importance   float64                `json:"importance"` // 0.0 to 1.0
	LearnedAt    time.Time              `json:"learned_at"`
	AppliedCount int                    `json:"applied_count"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// LearningEngine محرك التعلم المستمر
type LearningEngine struct {
	skills    map[string]*Skill
	lessons   map[string]*Lesson
	logger    *zap.Logger
	mu        sync.RWMutex
	sessionID string
	agentID   string
}

// NewLearningEngine ينشئ محرك تعلم جديد
func NewLearningEngine(sessionID, agentID string, logger *zap.Logger) *LearningEngine {
	return &LearningEngine{
		skills:    make(map[string]*Skill),
		lessons:   make(map[string]*Lesson),
		logger:    logger,
		sessionID: sessionID,
		agentID:   agentID,
	}
}

// AddSkill يضيف مهارة جديدة
func (le *LearningEngine) AddSkill(ctx context.Context, name, description string) error {
	le.mu.Lock()
	defer le.mu.Unlock()

	skill := &Skill{
		ID:          fmt.Sprintf("skill_%d", time.Now().UnixNano()),
		Name:        name,
		Description: description,
		Level:       0.0,
		UsageCount:  0,
		SuccessRate: 0.0,
		CreatedAt:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	le.skills[skill.ID] = skill

	le.logger.Info("تم إضافة مهارة جديدة",
		zap.String("session_id", le.sessionID),
		zap.String("agent_id", le.agentID),
		zap.String("skill_id", skill.ID),
		zap.String("name", name),
	)

	return nil
}

// UseSkill يستخدم مهارة
func (le *LearningEngine) UseSkill(ctx context.Context, skillID string, success bool) error {
	le.mu.Lock()
	defer le.mu.Unlock()

	skill, ok := le.skills[skillID]
	if !ok {
		return fmt.Errorf("مهارة غير موجودة: %s", skillID)
	}

	skill.UsageCount++
	skill.LastUsed = time.Now()

	// تحديث معدل النجاح
	if success {
		skill.SuccessRate = (skill.SuccessRate*float64(skill.UsageCount-1) + 1.0) / float64(skill.UsageCount)
		skill.Level = min(1.0, skill.Level+0.05)
	} else {
		skill.SuccessRate = (skill.SuccessRate * float64(skill.UsageCount-1)) / float64(skill.UsageCount)
		skill.Level = max(0.0, skill.Level-0.02)
	}

	le.logger.Info("تم استخدام المهارة",
		zap.String("session_id", le.sessionID),
		zap.String("agent_id", le.agentID),
		zap.String("skill_id", skillID),
		zap.Bool("success", success),
		zap.Float64("new_level", skill.Level),
	)

	return nil
}

// GetSkills يرجع جميع المهارات
func (le *LearningEngine) GetSkills(ctx context.Context) ([]*Skill, error) {
	le.mu.RLock()
	defer le.mu.RUnlock()

	skills := make([]*Skill, 0, len(le.skills))
	for _, skill := range le.skills {
		skills = append(skills, skill)
	}

	return skills, nil
}

// GetSkill يرجع مهارة محددة
func (le *LearningEngine) GetSkill(ctx context.Context, skillID string) (*Skill, error) {
	le.mu.RLock()
	defer le.mu.RUnlock()

	skill, ok := le.skills[skillID]
	if !ok {
		return nil, fmt.Errorf("مهارة غير موجودة: %s", skillID)
	}

	return skill, nil
}

// GetTopSkills يرجع المهارات الأعلى مستوى
func (le *LearningEngine) GetTopSkills(ctx context.Context, limit int) ([]*Skill, error) {
	le.mu.RLock()
	defer le.mu.RUnlock()

	skills := make([]*Skill, 0, len(le.skills))
	for _, skill := range le.skills {
		skills = append(skills, skill)
	}

	// ترتيب المهارات حسب المستوى
	for i := 0; i < len(skills); i++ {
		for j := i + 1; j < len(skills); j++ {
			if skills[j].Level > skills[i].Level {
				skills[i], skills[j] = skills[j], skills[i]
			}
		}
	}

	if limit > len(skills) {
		limit = len(skills)
	}

	return skills[:limit], nil
}

// AddLesson يضيف درساً مستفاداً
func (le *LearningEngine) AddLesson(ctx context.Context, title, description, category string, importance float64) error {
	le.mu.Lock()
	defer le.mu.Unlock()

	lesson := &Lesson{
		ID:           fmt.Sprintf("lesson_%d", time.Now().UnixNano()),
		Title:        title,
		Description:  description,
		Category:     category,
		Importance:   importance,
		LearnedAt:    time.Now(),
		AppliedCount: 0,
		Metadata:     make(map[string]interface{}),
	}

	le.lessons[lesson.ID] = lesson

	le.logger.Info("تم إضافة درس مستفاد",
		zap.String("session_id", le.sessionID),
		zap.String("agent_id", le.agentID),
		zap.String("lesson_id", lesson.ID),
		zap.String("title", title),
		zap.String("category", category),
	)

	return nil
}

// ApplyLesson يطبق درساً مستفاداً
func (le *LearningEngine) ApplyLesson(ctx context.Context, lessonID string) error {
	le.mu.Lock()
	defer le.mu.Unlock()

	lesson, ok := le.lessons[lessonID]
	if !ok {
		return fmt.Errorf("درس غير موجود: %s", lessonID)
	}

	lesson.AppliedCount++

	le.logger.Info("تم تطبيق الدرس المستفاد",
		zap.String("session_id", le.sessionID),
		zap.String("agent_id", le.agentID),
		zap.String("lesson_id", lessonID),
		zap.Int("applied_count", lesson.AppliedCount),
	)

	return nil
}

// GetLessons يرجع جميع الدروس المستفادة
func (le *LearningEngine) GetLessons(ctx context.Context) ([]*Lesson, error) {
	le.mu.RLock()
	defer le.mu.RUnlock()

	lessons := make([]*Lesson, 0, len(le.lessons))
	for _, lesson := range le.lessons {
		lessons = append(lessons, lesson)
	}

	return lessons, nil
}

// GetLessonsByCategory يرجع الدروس حسب الفئة
func (le *LearningEngine) GetLessonsByCategory(ctx context.Context, category string) ([]*Lesson, error) {
	le.mu.RLock()
	defer le.mu.RUnlock()

	var result []*Lesson
	for _, lesson := range le.lessons {
		if lesson.Category == category {
			result = append(result, lesson)
		}
	}

	return result, nil
}

// GetImportantLessons يرجع الدروس الأهم
func (le *LearningEngine) GetImportantLessons(ctx context.Context, limit int) ([]*Lesson, error) {
	le.mu.RLock()
	defer le.mu.RUnlock()

	lessons := make([]*Lesson, 0, len(le.lessons))
	for _, lesson := range le.lessons {
		lessons = append(lessons, lesson)
	}

	// ترتيب الدروس حسب الأهمية
	for i := 0; i < len(lessons); i++ {
		for j := i + 1; j < len(lessons); j++ {
			if lessons[j].Importance > lessons[i].Importance {
				lessons[i], lessons[j] = lessons[j], lessons[i]
			}
		}
	}

	if limit > len(lessons) {
		limit = len(lessons)
	}

	return lessons[:limit], nil
}

// LearnFromTask يتعلم من مهمة منجزة
func (le *LearningEngine) LearnFromTask(ctx context.Context, task string, success bool, duration time.Duration, metadata map[string]interface{}) error {
	le.mu.Lock()
	defer le.mu.Unlock()

	// [WHY] التعلم من المهام المنجزة
	// [HOW] يحلل النتائج ويستخلص الدروس
	// [SAFETY] يخزن الدروس للاستخدام المستقبلي

	lessonTitle := fmt.Sprintf("درس من مهمة: %s", task)
	var lessonDescription string
	var importance float64

	if success {
		lessonDescription = fmt.Sprintf("تم تنفيذ المهمة بنجاح في %v", duration)
		importance = 0.7
	} else {
		lessonDescription = fmt.Sprintf("فشل تنفيذ المهمة بعد %v", duration)
		importance = 0.9 // الدروس من الفشل أكثر أهمية
	}

	lesson := &Lesson{
		ID:           fmt.Sprintf("lesson_%d", time.Now().UnixNano()),
		Title:        lessonTitle,
		Description:  lessonDescription,
		Category:     "task_execution",
		Importance:   importance,
		LearnedAt:    time.Now(),
		AppliedCount: 0,
		Metadata:     metadata,
	}

	le.lessons[lesson.ID] = lesson

	le.logger.Info("تم التعلم من المهمة",
		zap.String("session_id", le.sessionID),
		zap.String("agent_id", le.agentID),
		zap.String("lesson_id", lesson.ID),
		zap.Bool("success", success),
		zap.Duration("duration", duration),
	)

	return nil
}

// GetLearningSummary يرجع ملخص التعلم
func (le *LearningEngine) GetLearningSummary(ctx context.Context) (map[string]interface{}, error) {
	le.mu.RLock()
	defer le.mu.RUnlock()

	// حساب متوسط مستوى المهارات
	totalSkillLevel := 0.0
	for _, skill := range le.skills {
		totalSkillLevel += skill.Level
	}
	avgSkillLevel := 0.0
	if len(le.skills) > 0 {
		avgSkillLevel = totalSkillLevel / float64(len(le.skills))
	}

	// حساب متوسط أهمية الدروس
	totalLessonImportance := 0.0
	for _, lesson := range le.lessons {
		totalLessonImportance += lesson.Importance
	}
	avgLessonImportance := 0.0
	if len(le.lessons) > 0 {
		avgLessonImportance = totalLessonImportance / float64(len(le.lessons))
	}

	summary := map[string]interface{}{
		"session_id":            le.sessionID,
		"agent_id":              le.agentID,
		"total_skills":          len(le.skills),
		"total_lessons":         len(le.lessons),
		"avg_skill_level":       avgSkillLevel,
		"avg_lesson_importance": avgLessonImportance,
		"top_skills":            le.getTopSkillsInternal(3),
		"important_lessons":     le.getImportantLessonsInternal(3),
	}

	return summary, nil
}

// getTopSkillsInternal دالة داخلية للحصول على المهارات الأعلى
func (le *LearningEngine) getTopSkillsInternal(limit int) []string {
	skills := make([]*Skill, 0, len(le.skills))
	for _, skill := range le.skills {
		skills = append(skills, skill)
	}

	for i := 0; i < len(skills); i++ {
		for j := i + 1; j < len(skills); j++ {
			if skills[j].Level > skills[i].Level {
				skills[i], skills[j] = skills[j], skills[i]
			}
		}
	}

	result := make([]string, 0)
	maxLimit := limit
	if maxLimit > len(skills) {
		maxLimit = len(skills)
	}
	for i := 0; i < maxLimit; i++ {
		result = append(result, skills[i].Name)
	}

	return result
}

// getImportantLessonsInternal دالة داخلية للحصول على الدروس الأهم
func (le *LearningEngine) getImportantLessonsInternal(limit int) []string {
	lessons := make([]*Lesson, 0, len(le.lessons))
	for _, lesson := range le.lessons {
		lessons = append(lessons, lesson)
	}

	for i := 0; i < len(lessons); i++ {
		for j := i + 1; j < len(lessons); j++ {
			if lessons[j].Importance > lessons[i].Importance {
				lessons[i], lessons[j] = lessons[j], lessons[i]
			}
		}
	}

	result := make([]string, 0)
	maxLimit := limit
	if maxLimit > len(lessons) {
		maxLimit = len(lessons)
	}
	for i := 0; i < maxLimit; i++ {
		result = append(result, lessons[i].Title)
	}

	return result
}

// min دالة مساعدة
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// max دالة مساعدة
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
