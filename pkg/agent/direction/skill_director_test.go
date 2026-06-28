package direction

import (
	"context"
	"testing"

	"github.com/MortalArena/Musketeers/pkg/agent/skills"
	"go.uber.org/zap/zaptest"
)

func TestSkillDirector_NewSkillDirector(t *testing.T) {
	logger := zaptest.NewLogger(t)
	skillManager := skills.NewSkillManager(logger)
	sd := NewSkillDirector(skillManager, logger)

	if sd == nil {
		t.Fatal("NewSkillDirector returned nil")
	}

	if sd.skillManager == nil {
		t.Error("skillManager is nil")
	}

	if sd.contextAnalyzer == nil {
		t.Error("contextAnalyzer is nil")
	}

	if sd.decisionEngine == nil {
		t.Error("decisionEngine is nil")
	}
}

func TestSkillDirector_GuideAgent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	skillManager := skills.NewSkillManager(logger)
	sd := NewSkillDirector(skillManager, logger)

	ctx := context.Background()
	task := &Task{
		ID:          "test-task",
		Description: "test task",
		Parameters:  make(map[string]interface{}),
		Priority:    1,
	}
	agentCtx := &skills.AgentContext{
		SessionID:   "test-session",
		AgentID:     "test-agent",
		TaskID:      "test-task",
		Metadata:    make(map[string]interface{}),
		Environment: make(map[string]string),
	}

	// توجيه الوكيل
	guidance, err := sd.GuideAgent(ctx, task, agentCtx)
	if err != nil {
		t.Fatalf("GuideAgent failed: %v", err)
	}

	if guidance == nil {
		t.Fatal("Expected guidance, got nil")
	}

	if guidance.Reasoning == "" {
		t.Error("Expected reasoning, got empty string")
	}
}

func TestContextAnalyzer_AnalyzeContext(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ca := NewContextAnalyzer(logger)

	ctx := context.Background()
	task := &Task{
		ID:          "test-task",
		Description: "test task",
		Parameters:  make(map[string]interface{}),
		Priority:    1,
	}
	agentCtx := &skills.AgentContext{
		SessionID:   "test-session",
		AgentID:     "test-agent",
		TaskID:      "test-task",
		Metadata:    make(map[string]interface{}),
		Environment: make(map[string]string),
	}

	// تحليل السياق
	analysis := ca.AnalyzeContext(ctx, task, agentCtx)
	if analysis == nil {
		t.Fatal("Expected analysis, got nil")
	}

	if analysis["session_id"] != "test-session" {
		t.Errorf("Expected session_id 'test-session', got '%v'", analysis["session_id"])
	}

	if analysis["agent_id"] != "test-agent" {
		t.Errorf("Expected agent_id 'test-agent', got '%v'", analysis["agent_id"])
	}
}

func TestDecisionEngine_DetermineExecutionOrder(t *testing.T) {
	logger := zaptest.NewLogger(t)
	de := NewDecisionEngine(logger)

	skillList := []*skills.Skill{
		{Name: "skill1"},
		{Name: "skill2"},
		{Name: "skill3"},
	}
	task := &Task{
		ID:          "test-task",
		Description: "test task",
		Parameters:  make(map[string]interface{}),
		Priority:    1,
	}

	// تحديد ترتيب التنفيذ
	order := de.DetermineExecutionOrder(skillList, task)
	if order == nil {
		t.Fatal("Expected order, got nil")
	}

	if len(order) != len(skillList) {
		t.Errorf("Expected %d skills in order, got %d", len(skillList), len(order))
	}
}

func TestDecisionEngine_CalculateConfidence(t *testing.T) {
	logger := zaptest.NewLogger(t)
	de := NewDecisionEngine(logger)

	skillList := []*skills.Skill{
		{Name: "skill1"},
		{Name: "skill2"},
	}
	task := &Task{
		ID:          "test-task",
		Description: "test task",
		Parameters:  make(map[string]interface{}),
		Priority:    1,
	}

	// حساب الثقة
	confidence := de.CalculateConfidence(skillList, task)
	if confidence < 0 || confidence > 1 {
		t.Errorf("Expected confidence between 0 and 1, got %f", confidence)
	}
}

func TestSkillDirector_GenerateValidationRules(t *testing.T) {
	logger := zaptest.NewLogger(t)
	skillManager := skills.NewSkillManager(logger)
	sd := NewSkillDirector(skillManager, logger)

	task := &Task{
		ID:          "test-task",
		Description: "test task",
		Parameters:  make(map[string]interface{}),
		Priority:    1,
	}

	// توليد قواعد التحقق
	rules := sd.generateValidationRules(task)
	if rules == nil {
		t.Fatal("Expected rules, got nil")
	}

	if len(rules) == 0 {
		t.Error("Expected at least one rule, got 0")
	}
}

// اختبارات الأمان
func TestSkillDirector_Security_ConcurrentGuidance(t *testing.T) {
	logger := zaptest.NewLogger(t)
	skillManager := skills.NewSkillManager(logger)
	sd := NewSkillDirector(skillManager, logger)

	ctx := context.Background()
	task := &Task{
		ID:          "test-task",
		Description: "test task",
		Parameters:  make(map[string]interface{}),
		Priority:    1,
	}
	agentCtx := &skills.AgentContext{
		SessionID:   "test-session",
		AgentID:     "test-agent",
		TaskID:      "test-task",
		Metadata:    make(map[string]interface{}),
		Environment: make(map[string]string),
	}

	// اختبار التوجيه المتزامن
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { recover() }()
			sd.GuideAgent(ctx, task, agentCtx)
			done <- true
		}()
	}

	// انتظار جميع goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
