package direction

import (
	"context"
	"fmt"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/agent/skills"
	"go.uber.org/zap"
)

// SkillDirector يوجه الوكيل لاستخدام المهارات المناسبة
type SkillDirector struct {
	skillManager    *skills.SkillManager
	contextAnalyzer *ContextAnalyzer
	decisionEngine  *DecisionEngine
	logger          *zap.Logger
	mu              sync.RWMutex
}

// ContextAnalyzer يحلل السياق
type ContextAnalyzer struct {
	logger *zap.Logger
}

// DecisionEngine محرك القرار
type DecisionEngine struct {
	logger *zap.Logger
}

// Guidance توجيه للوكيل
type Guidance struct {
	RecommendedSkills []string               `json:"recommended_skills"`
	ExecutionOrder    []string               `json:"execution_order"`
	Parameters        map[string]interface{} `json:"parameters"`
	ValidationRules   []ValidationRule       `json:"validation_rules"`
	Confidence        float64                `json:"confidence"`
	Reasoning         string                 `json:"reasoning"`
}

// ValidationRule قاعدة تحقق
type ValidationRule struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// NewSkillDirector ينشئ مدير توجيه مهارات جديد
func NewSkillDirector(skillManager *skills.SkillManager, logger *zap.Logger) *SkillDirector {
	return &SkillDirector{
		skillManager:    skillManager,
		contextAnalyzer: NewContextAnalyzer(logger),
		decisionEngine:  NewDecisionEngine(logger),
		logger:          logger,
	}
}

// NewContextAnalyzer ينشئ محلل سياق جديد
func NewContextAnalyzer(logger *zap.Logger) *ContextAnalyzer {
	return &ContextAnalyzer{
		logger: logger,
	}
}

// NewDecisionEngine ينشئ محرك قرار جديد
func NewDecisionEngine(logger *zap.Logger) *DecisionEngine {
	return &DecisionEngine{
		logger: logger,
	}
}

// GuideAgent يوجه الوكيل لاستخدام المهارات المناسبة
func (sd *SkillDirector) GuideAgent(ctx context.Context, task *Task, agentCtx *skills.AgentContext) (*Guidance, error) {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	// [WHY] توجيه الوكيل لاستخدام المهارات المناسبة
	// [HOW] يحلل السياق ويحدد المهارات المناسبة
	// [SAFETY] يتحقق من صحة المدخلات والمهارات

	guidance := &Guidance{
		RecommendedSkills: []string{},
		ExecutionOrder:    []string{},
		Parameters:        make(map[string]interface{}),
		ValidationRules:   []ValidationRule{},
		Confidence:        0.0,
		Reasoning:         "",
	}

	// تحليل السياق
	contextAnalysis := sd.contextAnalyzer.AnalyzeContext(ctx, task, agentCtx)
	guidance.Parameters["context_analysis"] = contextAnalysis

	// البحث عن المهارات المناسبة
	relevantSkills := sd.skillManager.SearchSkills(task.Description)
	if len(relevantSkills) == 0 {
		guidance.Reasoning = "لم يتم العثور على مهارات مناسبة للمهمة"
		guidance.Confidence = 0.5
		return guidance, nil
	}

	// تحديد المهارات الموصى بها
	for _, skill := range relevantSkills {
		guidance.RecommendedSkills = append(guidance.RecommendedSkills, skill.Name)
	}

	// تحديد ترتيب التنفيذ
	guidance.ExecutionOrder = sd.decisionEngine.DetermineExecutionOrder(relevantSkills, task)

	// إضافة قواعد التحقق
	guidance.ValidationRules = sd.generateValidationRules(task)

	// حساب الثقة
	guidance.Confidence = sd.decisionEngine.CalculateConfidence(relevantSkills, task)
	guidance.Reasoning = sd.generateReasoning(relevantSkills, task)

	sd.logger.Info("تم توجيه الوكيل",
		zap.String("task", task.ID),
		zap.Int("recommended_skills", len(guidance.RecommendedSkills)),
		zap.Float64("confidence", guidance.Confidence))

	return guidance, nil
}

// AnalyzeContext يحلل السياق
func (ca *ContextAnalyzer) AnalyzeContext(ctx context.Context, task *Task, agentCtx *skills.AgentContext) map[string]interface{} {
	// [WHY] تحليل السياق
	// [HOW] يستخرج المعلومات من السياق والمهمة
	// [SAFETY] يتحقق من صحة البيانات

	analysis := map[string]interface{}{
		"session_id":  agentCtx.SessionID,
		"agent_id":    agentCtx.AgentID,
		"task_id":     task.ID,
		"task_type":   ca.determineTaskType(task),
		"complexity":  ca.assessComplexity(task),
		"environment": agentCtx.Environment,
		"metadata":    agentCtx.Metadata,
	}

	return analysis
}

// determineTaskType يحدد نوع المهمة
func (ca *ContextAnalyzer) determineTaskType(task *Task) string {
	// [WHY] تحديد نوع المهمة
	// [HOW] يحلل وصف المهمة
	// [SAFETY] يستخدم تحليل بسيط

	description := task.Description

	// تحليل بسيط بناءً على الكلمات المفتاحية
	if containsAny(description, []string{"bug", "error", "fix", "debug"}) {
		return "debugging"
	}
	if containsAny(description, []string{"review", "check", "audit"}) {
		return "review"
	}
	if containsAny(description, []string{"create", "build", "implement", "develop"}) {
		return "development"
	}
	if containsAny(description, []string{"test", "verify", "validate"}) {
		return "testing"
	}
	if containsAny(description, []string{"deploy", "release", "publish"}) {
		return "deployment"
	}

	return "general"
}

// assessComplexity يقيّم تعقيد المهمة
func (ca *ContextAnalyzer) assessComplexity(task *Task) string {
	// [WHY] تقييم تعقيد المهمة
	// [HOW] يحلل طول المهمة والمعلمات
	// [SAFETY] يستخدم تقييم بسيط

	description := task.Description
	paramCount := len(task.Parameters)

	if len(description) < 100 && paramCount < 3 {
		return "low"
	}
	if len(description) < 500 && paramCount < 10 {
		return "medium"
	}
	return "high"
}

// DetermineExecutionOrder يحدد ترتيب التنفيذ
func (de *DecisionEngine) DetermineExecutionOrder(skillList []*skills.Skill, task *Task) []string {
	// [WHY] تحديد ترتيب تنفيذ المهارات
	// [HOW] يرتب المهارات حسب الأولوية
	// [SAFETY] يستخدم ترتيب بسيط

	order := make([]string, 0, len(skillList))

	// في التنفيذ الحالي، سنستخدم الترتيب كما هو
	// في المستقبل، يمكن إضافة منطق أكثر تعقيداً
	for _, skill := range skillList {
		order = append(order, skill.Name)
	}

	return order
}

// CalculateConfidence يحسب الثقة
func (de *DecisionEngine) CalculateConfidence(skillList []*skills.Skill, task *Task) float64 {
	// [WHY] حساب الثقة في التوصية
	// [HOW] يحسب بناءً على عدد المهارات المطابقة
	// [SAFETY] يستخدم حساب بسيط

	if len(skillList) == 0 {
		return 0.0
	}

	// كلما زادت المهارات المطابقة، زادت الثقة
	confidence := float64(len(skillList)) * 0.2
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// generateValidationRules يولد قواعد التحقق
func (sd *SkillDirector) generateValidationRules(task *Task) []ValidationRule {
	// [WHY] توليد قواعد التحقق
	// [HOW] يولد قواعد بناءً على المهمة
	// [SAFETY] يستخدم قواعد أساسية

	rules := []ValidationRule{
		{
			Name:        "task_id_required",
			Description: "معرف المهمة مطلوب",
			Required:    true,
		},
		{
			Name:        "description_required",
			Description: "وصف المهمة مطلوب",
			Required:    true,
		},
	}

	return rules
}

// generateReasoning يولد التبرير
func (sd *SkillDirector) generateReasoning(skillList []*skills.Skill, task *Task) string {
	// [WHY] توليد التبرير
	// [HOW] يولد تبريراً بناءً على المهارات والمهمة
	// [SAFETY] يستخدم تبرير بسيط

	if len(skillList) == 0 {
		return "لم يتم العثور على مهارات مناسبة للمهمة"
	}

	return fmt.Sprintf("تم العثور على %d مهارات مناسبة للمهمة '%s'", len(skillList), task.ID)
}

// Task يمثل مهمة
type Task struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Priority    int                    `json:"priority"`
}

// containsAny يتحقق من وجود أي من الكلمات
func containsAny(text string, words []string) bool {
	for _, word := range words {
		if contains(text, word) {
			return true
		}
	}
	return false
}

// contains يتحقق من وجود كلمة
func contains(text, word string) bool {
	return len(text) > 0 && len(word) > 0 &&
		(text == word ||
			len(text) > len(word) &&
				(text[:len(word)] == word ||
					text[len(text)-len(word):] == word ||
					containsSubstring(text, word)))
}

// containsSubstring يتحقق من وجود سلسلة فرعية
func containsSubstring(text, word string) bool {
	for i := 0; i <= len(text)-len(word); i++ {
		if text[i:i+len(word)] == word {
			return true
		}
	}
	return false
}
