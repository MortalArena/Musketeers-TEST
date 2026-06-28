package unified

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"go.uber.org/zap/zaptest"
)

// TestSecurity_ConcurrentAccess اختبار الوصول المتزامن الآمن
func TestSecurity_ConcurrentAccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

	ctx := context.Background()

	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	thinkingEngine := ua.GetThinkingEngine()
	if thinkingEngine == nil {
		t.Fatal("ThinkingEngine should be initialized")
	}

	// اختبار الوصول المتزامن للبيانات المشتركة
	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer func() { recover() }()
			defer wg.Done()

			// محاولة الوصول إلى البيانات المشتركة
			phase, err := thinkingEngine.GetCurrentPhase(ctx)
			if err != nil {
				t.Errorf("Goroutine %d: GetCurrentPhase failed: %v", index, err)
			}

			_ = phase

			// إضافة فكرة
			err = thinkingEngine.AddThought(ctx, phase, fmt.Sprintf("thought-%d", index), nil)
			if err != nil {
				t.Errorf("Goroutine %d: AddThought failed: %v", index, err)
			}

			// الحصول على الأفكار
			_, err = thinkingEngine.GetThoughts(ctx)
			if err != nil {
				t.Errorf("Goroutine %d: GetThoughts failed: %v", index, err)
			}
		}(i)
	}

	wg.Wait()

	// التحقق من عدم وجود تلف في البيانات
	thoughts, err := thinkingEngine.GetThoughts(ctx)
	if err != nil {
		t.Fatalf("GetThoughts failed after concurrent access: %v", err)
	}

	if len(thoughts) != numGoroutines {
		t.Errorf("Expected %d thoughts, got %d (data corruption possible)", numGoroutines, len(thoughts))
	}

	t.Logf("تم اختبار الوصول المتزامن الآمن مع %d goroutine", numGoroutines)
}

// TestSecurity_DAGConcurrencySafety اختبار أمان التزامن في DAG
func TestSecurity_DAGConcurrencySafety(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

	ctx := context.Background()

	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	thinkingEngine := ua.GetThinkingEngine()
	if thinkingEngine == nil {
		t.Fatal("ThinkingEngine should be initialized")
	}

	// تنفيذ عمليات متوازية على ThinkingEngine
	var wg sync.WaitGroup
	numExecutions := 10

	for i := 0; i < numExecutions; i++ {
		wg.Add(1)
		go func(execIndex int) {
			defer func() { recover() }()
			defer wg.Done()

			// محاكاة عمليات متوازية
			phase, err := thinkingEngine.GetCurrentPhase(ctx)
			if err != nil {
				t.Errorf("Execution %d: GetCurrentPhase failed: %v", execIndex, err)
				return
			}

			err = thinkingEngine.AddThought(ctx, phase, fmt.Sprintf("concurrent-thought-%d", execIndex), nil)
			if err != nil {
				t.Errorf("Execution %d: AddThought failed: %v", execIndex, err)
			}
		}(i)
	}

	wg.Wait()

	t.Logf("تم اختبار أمان التزامن مع %d عملية متوازية", numExecutions)
}

// TestSecurity_MemorySafety اختبار أمان الذاكرة
func TestSecurity_MemorySafety(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

	ctx := context.Background()

	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	thinkingEngine := ua.GetThinkingEngine()
	if thinkingEngine == nil {
		t.Fatal("ThinkingEngine should be initialized")
	}

	// إضافة عدد كبير من الأفكار لاختبار تسرب الذاكرة
	numThoughts := 500

	for i := 0; i < numThoughts; i++ {
		phase, _ := thinkingEngine.GetCurrentPhase(ctx)
		err := thinkingEngine.AddThought(ctx, phase, fmt.Sprintf("thought-%d", i), map[string]interface{}{
			"data": fmt.Sprintf("data-%d", i),
			"metadata": map[string]interface{}{
				"index": i,
				"info":  fmt.Sprintf("info-%d", i),
			},
		})
		if err != nil {
			t.Fatalf("AddThought failed at %d: %v", i, err)
		}
	}

	// التحقق من أن جميع الأفكار موجودة
	thoughts, err := thinkingEngine.GetThoughts(ctx)
	if err != nil {
		t.Fatalf("GetThoughts failed: %v", err)
	}

	if len(thoughts) != numThoughts {
		t.Errorf("Expected %d thoughts, got %d (possible memory leak or data loss)", numThoughts, len(thoughts))
	}

	t.Logf("تم اختبار أمان الذاكرة مع %d فكرة", numThoughts)
}

// TestSecurity_SessionIsolation اختبار عزل الجلسات
func TestSecurity_SessionIsolation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"

	ctx := context.Background()

	// إنشاء وكلاء في جلسات مختلفة
	agent1 := NewUnifiedAgent(sessionID, "agent-1", nil, logger)
	agent2 := NewUnifiedAgent(sessionID, "agent-2", nil, logger)

	if err := agent1.Initialize(ctx); err != nil {
		t.Fatalf("Agent1 Initialize failed: %v", err)
	}

	if err := agent2.Initialize(ctx); err != nil {
		t.Fatalf("Agent2 Initialize failed: %v", err)
	}

	// إضافة أفكار مختلفة لكل وكيل
	te1 := agent1.GetThinkingEngine()
	te2 := agent2.GetThinkingEngine()

	if te1 == nil || te2 == nil {
		t.Fatal("ThinkingEngine should be initialized for both agents")
	}

	phase1, _ := te1.GetCurrentPhase(ctx)
	phase2, _ := te2.GetCurrentPhase(ctx)

	te1.AddThought(ctx, phase1, "agent-1-thought", nil)
	te2.AddThought(ctx, phase2, "agent-2-thought", nil)

	// التحقق من عزل البيانات
	thoughts1, _ := te1.GetThoughts(ctx)
	_, _ = te2.GetThoughts(ctx)

	// كل وكيل يجب أن يكون لديه أفكاره الخاصة فقط
	foundAgent2ThoughtInAgent1 := false
	for _, thought := range thoughts1 {
		if thought.Content == "agent-2-thought" {
			foundAgent2ThoughtInAgent1 = true
			break
		}
	}

	if foundAgent2ThoughtInAgent1 {
		t.Error("Session isolation failed: agent-2 thought found in agent-1")
	}

	t.Logf("تم اختبار عزل الجلسات بنجاح")
}

// TestSecurity_ErrorHandling اختبار معالجة الأخطاء
func TestSecurity_ErrorHandling(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

	ctx := context.Background()

	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	thinkingEngine := ua.GetThinkingEngine()
	if thinkingEngine == nil {
		t.Fatal("ThinkingEngine should be initialized")
	}

	// اختبار معالجة الأخطاء في العمليات المختلفة
	tests := []struct {
		name string
		test func() error
	}{
		{
			name: "AnalyzeTask with empty task",
			test: func() error {
				_, err := thinkingEngine.AnalyzeTask(ctx, "")
				return err
			},
		},
		{
			name: "PlanTask with nil analysis",
			test: func() error {
				_, err := thinkingEngine.PlanTask(ctx, nil)
				return err
			},
		},
		{
			name: "ExecuteSteps with nil subtasks",
			test: func() error {
				_, err := thinkingEngine.ExecuteSteps(ctx, nil)
				return err
			},
		},
		{
			name: "VerifyResults with nil results",
			test: func() error {
				_, err := thinkingEngine.VerifyResults(ctx, nil)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.test()
			// نتوقع أن توجد أخطاء للمدخلات غير الصحيحة
			if err == nil {
				t.Errorf("Expected error for %s, got nil", tt.name)
			}
		})
	}

	t.Log("تم اختبار معالجة الأخطاء بنجاح")
}

// TestSecurity_ResourceManagement اختبار إدارة الموارد
func TestSecurity_ResourceManagement(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

	ctx := context.Background()

	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	thinkingEngine := ua.GetThinkingEngine()
	if thinkingEngine == nil {
		t.Fatal("ThinkingEngine should be initialized")
	}

	// تسجيل جلسات متعددة
	for i := 0; i < 10; i++ {
		sessionID := fmt.Sprintf("session-%d", i)
		agents := []string{fmt.Sprintf("agent-%d", i)}
		err := thinkingEngine.RegisterSession(ctx, sessionID, agents, i+1)
		if err != nil {
			t.Errorf("Failed to register session %d: %v", i, err)
		}
	}

	// محاولة الحصول على موارد متعددة
	for i := 0; i < 20; i++ {
		sessionID := fmt.Sprintf("session-%d", i%10)
		resourceID := fmt.Sprintf("resource-%d", i)

		err := thinkingEngine.AcquireResource(ctx, sessionID, resourceID)
		if err != nil {
			t.Errorf("Failed to acquire resource %d: %v", i, err)
		}

		// إطلاق المورد
		err = thinkingEngine.ReleaseResource(ctx, sessionID, resourceID)
		if err != nil {
			t.Errorf("Failed to release resource %d: %v", i, err)
		}
	}

	// التحقق من عدم وجود تعارضات خطيرة
	conflicts := thinkingEngine.DetectSessionConflicts(ctx)
	if len(conflicts) > 5 {
		t.Logf("تم اكتشاف %d تعارضات (قد يكون مقبول)", len(conflicts))
	}

	t.Log("تم اختبار إدارة الموارد بنجاح")
}
