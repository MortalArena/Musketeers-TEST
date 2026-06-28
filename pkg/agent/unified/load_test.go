package unified

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"go.uber.org/zap/zaptest"
)

// TestLoad_StressTest_50Agents اختبار إجهاد مع 50 وكيل
func TestLoad_StressTest_50Agents(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sessionID := "test-session"

	ctx := context.Background()

	numAgents := 50
	agents := make([]*UnifiedAgent, numAgents)

	// إنشاء 50 وكيل
	for i := 0; i < numAgents; i++ {
		agentID := fmt.Sprintf("agent-%d", i)
		ua := NewUnifiedAgent(sessionID, agentID, nil, logger)
		
		if err := ua.Initialize(ctx); err != nil {
			t.Fatalf("Failed to initialize agent %d: %v", i, err)
		}
		
		agents[i] = ua
	}

	t.Logf("تم تهيئة %d وكيل بنجاح", numAgents)

	// تنفيذ مهام متعددة بشكل متوازي
	var wg sync.WaitGroup
	tasksPerAgent := 10

	for i, agent := range agents {
		for j := 0; j < tasksPerAgent; j++ {
			wg.Add(1)
			go func(agentIndex, taskIndex int, ua *UnifiedAgent) {
				defer func() { recover() }()
				defer wg.Done()
				
				thinkingEngine := ua.GetThinkingEngine()
				if thinkingEngine == nil {
					return
				}

				task := fmt.Sprintf("task-%d-%d", agentIndex, taskIndex)
				
				// تحليل مهمة
				_, err := thinkingEngine.AnalyzeTask(ctx, task)
				if err != nil {
					t.Errorf("Agent %d, Task %d: AnalyzeTask failed: %v", agentIndex, taskIndex, err)
				}

				// إضافة فكرة
				phase, _ := thinkingEngine.GetCurrentPhase(ctx)
				err = thinkingEngine.AddThought(ctx, phase, "test thought", nil)
				if err != nil {
					t.Errorf("Agent %d, Task %d: AddThought failed: %v", agentIndex, taskIndex, err)
				}
			}(i, j, agent)
		}
	}

	wg.Wait()

	totalTasks := numAgents * tasksPerAgent
	t.Logf("تم تنفيذ %d مهمة بنجاح بواسطة %d وكيل", totalTasks, numAgents)
}

// TestLoad_MemoryStressTest اختبار إجهاد الذاكرة
func TestLoad_MemoryStressTest(t *testing.T) {
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

	// إضافة عدد كبير من الأفكار
	numThoughts := 100
	var wg sync.WaitGroup

	for i := 0; i < numThoughts; i++ {
		wg.Add(1)
		go func(thoughtIndex int) {
			defer func() { recover() }()
			defer wg.Done()

			phase, _ := thinkingEngine.GetCurrentPhase(ctx)
			content := fmt.Sprintf("thought-%d", thoughtIndex)
			
			err := thinkingEngine.AddThought(ctx, phase, content, map[string]interface{}{
				"index": thoughtIndex,
			})
			if err != nil {
				t.Errorf("Thought %d: AddThought failed: %v", thoughtIndex, err)
			}
		}(i)
	}

	wg.Wait()

	thoughts, err := thinkingEngine.GetThoughts(ctx)
	if err != nil {
		t.Fatalf("GetThoughts failed: %v", err)
	}

	if len(thoughts) != numThoughts {
		t.Errorf("Expected %d thoughts, got %d", numThoughts, len(thoughts))
	}

	t.Logf("تم إضافة %d فكرة بنجاح", numThoughts)
}
