package direction

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// SkillDirector مدير توجيه المهارات
type SkillDirector struct {
	logger *zap.Logger
}

// ContextAnalyzer محلل السياق
type ContextAnalyzer struct {
	logger *zap.Logger
}

// DecisionEngine محرك القرار
type DecisionEngine struct {
	logger *zap.Logger
}

// Guidance توجيه للوكيل
type Guidance struct {
	RecommendedSkills []string
	ExecutionOrder   []string
	Confidence       float64
	Reasoning        string
	ValidationRules  []ValidationRule
}

// ValidationRule قاعدة تحقق
type ValidationRule struct {
	Name        string
	Description string
	Required    bool
}

// NewSkillDirector ينشئ مدير توجيه مهارات جديد
func NewSkillDirector(logger *zap.Logger) *SkillDirector {
	return &SkillDirector{
		logger: logger,
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
func (sd *SkillDirector) GuideAgent(ctx context.Context, prompt string, availableSkills []string) (*Guidance, error) {
	analyzer := NewContextAnalyzer(sd.logger)
	engine := NewDecisionEngine(sd.logger)

	// تحليل السياق
	context := analyzer.AnalyzeContext(ctx, prompt)

	// تحديد المهارات المناسبة
	skills := engine.DetermineSkills(context, availableSkills)

	// تحديد ترتيب التنفيذ
	order := engine.DetermineExecutionOrder(context, skills)

	// حساب الثقة
	confidence := engine.CalculateConfidence(context, skills)

	// توليد التبرير
	reasoning := engine.GenerateReasoning(context, skills)

	// توليد قواعد التحقق
	rules := engine.GenerateValidationRules(context)

	guidance := &Guidance{
		RecommendedSkills: skills,
		ExecutionOrder:   order,
		Confidence:       confidence,
		Reasoning:        reasoning,
		ValidationRules:  rules,
	}

	sd.logger.Info("تم توجيه الوكيل",
		zap.String("prompt", prompt),
		zap.Int("recommended_skills", len(skills)),
		zap.Float64("confidence", confidence))

	return guidance, nil
}

// AnalyzeContext يحلل السياق
func (ca *ContextAnalyzer) AnalyzeContext(ctx context.Context, prompt string) *TaskContext {
	// تحديد نوع المهمة
	taskType := ca.determineTaskType(prompt)

	// تقييم التعقيد
	complexity := ca.assessComplexity(prompt)

	context := &TaskContext{
		Prompt:     prompt,
		TaskType:   taskType,
		Complexity: complexity,
		Timestamp:  time.Now(),
	}

	return context
}

// determineTaskType يحدد نوع المهمة
func (ca *ContextAnalyzer) determineTaskType(prompt string) string {
	promptLower := strings.ToLower(prompt)

	if strings.Contains(promptLower, "debug") || strings.Contains(promptLower, "fix bug") {
		return "debugging"
	}
	if strings.Contains(promptLower, "review") || strings.Contains(promptLower, "check") {
		return "review"
	}
	if strings.Contains(promptLower, "create") || strings.Contains(promptLower, "build") || strings.Contains(promptLower, "implement") {
		return "development"
	}
	if strings.Contains(promptLower, "test") || strings.Contains(promptLower, "verify") {
		return "testing"
	}
	if strings.Contains(promptLower, "deploy") || strings.Contains(promptLower, "release") {
		return "deployment"
	}

	return "general"
}

// assessComplexity يقيّم تعقيد المهمة
func (ca *ContextAnalyzer) assessComplexity(prompt string) string {
	length := len(prompt)

	if length < 100 {
		return "low"
	}
	if length < 500 {
		return "medium"
	}
	if length < 1000 {
		return "high"
	}
	return "critical"
}

// DetermineSkills يحدد المهارات المناسبة
func (de *DecisionEngine) DetermineSkills(context *TaskContext, availableSkills []string) []string {
	// في التنفيذ الحالي، سنقوم بإرجاع جميع المهارات المتاحة
	// في المستقبل، يمكن إضافة منطق أكثر تعقيداً
	return availableSkills
}

// DetermineExecutionOrder يحدد ترتيب التنفيذ
func (de *DecisionEngine) DetermineExecutionOrder(context *TaskContext, skills []string) []string {
	// في التنفيذ الحالي، سنقوم بإرجاع المهارات كما هي
	// في المستقبل، يمكن إضافة منطق أكثر تعقيداً
	return skills
}

// CalculateConfidence يحسب الثقة
func (de *DecisionEngine) CalculateConfidence(context *TaskContext, skills []string) float64 {
	// في التنفيذ الحالي، سنقوم بإرجاع ثقة متوسطة
	// في المستقبل، يمكن إضافة منطق أكثر تعقيداً
	return 0.75
}

// GenerateReasoning يولد التبرير
func (de *DecisionEngine) GenerateReasoning(context *TaskContext, skills []string) string {
	return fmt.Sprintf("بناءً على نوع المهمة (%s) وتعقيدها (%s)، يُنصح باستخدام المهارات المتاحة", context.TaskType, context.Complexity)
}

// GenerateValidationRules يولد قواعد التحقق
func (de *DecisionEngine) GenerateValidationRules(context *TaskContext) []ValidationRule {
	return []ValidationRule{
		{
			Name:        "skill_exists",
			Description: "المهارة يجب أن تكون موجودة",
			Required:    true,
		},
		{
			Name:        "skill_enabled",
			Description: "المهارة يجب أن تكون مفعلة",
			Required:    true,
		},
	}
}

// TaskContext سياق المهمة
type TaskContext struct {
	Prompt     string
	TaskType   string
	Complexity string
	Timestamp  time.Time
}
