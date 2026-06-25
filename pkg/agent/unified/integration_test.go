package unified

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"go.uber.org/zap/zaptest"
)

// TestIntegration_MultipleAgentsInSession اختبار دعم 10-50 وكيل في جلسة واحدة
func TestIntegration_MultipleAgentsInSession(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"

	ctx := context.Background()

	// اختبار مع 10 وكلاء
	testAgents := func(numAgents int) {
		t.Run(fmt.Sprintf("%d_agents", numAgents), func(t *testing.T) {
			agents := make([]*UnifiedAgent, numAgents)

			// إنشاء وكلاء
			for i := 0; i < numAgents; i++ {
				agentID := fmt.Sprintf("agent-%d", i)
				ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

				if err := ua.Initialize(ctx); err != nil {
					t.Fatalf("Failed to initialize agent %d: %v", i, err)
				}

				agents[i] = ua
			}

			// التحقق من أن جميع الوكلاء لديهم ThinkingEngine
			for i, agent := range agents {
				if agent.GetThinkingEngine() == nil {
					t.Errorf("Agent %d should have ThinkingEngine", i)
				}
			}

			// تنفيذ مهام متوازية
			var wg sync.WaitGroup
			tasksPerAgent := 3

			for i, agent := range agents {
				for j := 0; j < tasksPerAgent; j++ {
					wg.Add(1)
					go func(agentIndex, taskIndex int, ua *UnifiedAgent) {
						defer wg.Done()

						thinkingEngine := ua.GetThinkingEngine()
						if thinkingEngine == nil {
							return
						}

						// تحليل مهمة
						task := fmt.Sprintf("task-%d-%d", agentIndex, taskIndex)
						_, err := thinkingEngine.AnalyzeTask(ctx, task)
						if err != nil {
							t.Errorf("Agent %d, Task %d: AnalyzeTask failed: %v", agentIndex, taskIndex, err)
						}
					}(i, j, agent)
				}
			}

			wg.Wait()
			t.Logf("تم تنفيذ %d مهام بنجاح بواسطة %d وكلاء", numAgents*tasksPerAgent, numAgents)
		})
	}

	// اختبار بأعداد مختلفة
	testAgents(10)
	testAgents(25)
	testAgents(50)
}

// TestIntegration_WiringLayerFunctionality اختبار وظائف WiringLayer
func TestIntegration_WiringLayerFunctionality(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"
	agentID := "test-agent"

	ua := NewUnifiedAgent(sessionID, agentID, nil, logger)

	ctx := context.Background()

	if err := ua.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// التحقق من أن التهيئة لم تفشل
	t.Log("تم تهيئة WiringLayer بنجاح")
}

// TestIntegration_CompleteWorkflow اختبار سير العمل الكامل
func TestIntegration_CompleteWorkflow(t *testing.T) {
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

	// تحليل مهمة
	analysis, err := thinkingEngine.AnalyzeTask(ctx, "test-task")
	if err != nil {
		t.Fatalf("AnalyzeTask failed: %v", err)
	}

	if analysis == nil {
		t.Fatal("Expected analysis, got nil")
	}

	// تخطيط مهمة
	subtasks, err := thinkingEngine.PlanTask(ctx, analysis)
	if err != nil {
		t.Fatalf("PlanTask failed: %v", err)
	}

	if subtasks == nil {
		t.Error("Expected subtasks, got nil")
	}

	// تنفيذ خطوات
	results, err := thinkingEngine.ExecuteSteps(ctx, subtasks)
	if err != nil {
		t.Fatalf("ExecuteSteps failed: %v", err)
	}

	if results == nil {
		t.Error("Expected results, got nil")
	}

	// التحقق من النتائج
	verification, err := thinkingEngine.VerifyResults(ctx, results)
	if err != nil {
		t.Fatalf("VerifyResults failed: %v", err)
	}

	if verification == nil {
		t.Error("Expected verification, got nil")
	}

	// الحصول على ملخص
	summary, err := thinkingEngine.GetSummary(ctx)
	if err != nil {
		t.Fatalf("GetSummary failed: %v", err)
	}

	if summary == nil {
		t.Error("Expected summary, got nil")
	}

	t.Log("تم تنفيذ سير العمل الكامل بنجاح")
}
