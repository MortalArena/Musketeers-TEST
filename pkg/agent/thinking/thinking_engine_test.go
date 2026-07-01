package thinking

import (
	"context"
	"testing"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent/tools"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

func TestNewThinkingEngine(t *testing.T) {
	logger := zap.NewNop()
	te := NewThinkingEngine("test-session", "test-agent", logger)

	if te == nil {
		t.Fatal("ThinkingEngine should not be nil")
	}

	if te.sessionID != "test-session" {
		t.Errorf("Expected sessionID 'test-session', got '%s'", te.sessionID)
	}

	if te.agentID != "test-agent" {
		t.Errorf("Expected agentID 'test-agent', got '%s'", te.agentID)
	}

	if te.contextMemory == nil {
		t.Error("ContextMemory should be initialized")
	}

	if te.toolRegistry == nil {
		t.Error("ToolRegistry should be initialized")
	}

	if te.errorRecovery == nil {
		t.Error("ErrorRecovery should be initialized")
	}

	if te.agentCoordination == nil {
		t.Error("AgentCoordination should be initialized")
	}

	if te.collectiveLearning == nil {
		t.Error("CollectiveLearning should be initialized")
	}

	if te.dagExecutor == nil {
		t.Error("DAGExecutor should be initialized")
	}
}

func TestThinkingEngineAnalyzeTask(t *testing.T) {
	logger := zap.NewNop()
	te := NewThinkingEngine("test-session", "test-agent", logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// اختبار تحليل مهمة بسيطة
	task := "What is 2 + 2?"
	result, err := te.AnalyzeTask(ctx, task)

	if err != nil {
		t.Logf("AnalyzeTask returned error (expected without provider): %v", err)
		// هذا متوقع بدون provider حقيقي
		return
	}

	if result == nil {
		t.Error("AnalyzeTask should return a result even without provider")
	}

	t.Logf("AnalyzeTask result: %+v", result)
}

func TestContextMemory(t *testing.T) {
	logger := zap.NewNop()
	te := NewThinkingEngine("test-session", "test-agent", logger)

	// Test context memory initialization
	if te.contextMemory == nil {
		t.Error("ContextMemory should be initialized")
	}
}

func TestToolExecution(t *testing.T) {
	logger := zap.NewNop()
	te := NewThinkingEngine("test-session", "test-agent", logger)

	// إنشاء ToolExecutor مع ToolRegistry
	registry := tools.NewToolRegistry()
	executor := tools.NewToolExecutorWithRegistry(".", registry, tools.RoleRegular, logger)
	te.SetToolExecutor(executor)

	// تسجيل أداة بسيطة (echo)
	registry.Register(tools.ToolDefinition{
		Name:         "echo",
		Description:  "يعيد النص المدخل",
		Category:     tools.CategoryExecution,
		Action:       tools.ActionRead,
		RequiredRole: tools.RoleRegular,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			text, _ := params["text"].(string)
			return map[string]interface{}{
				"echoed": text,
			}, nil
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// اختبار تنفيذ أداة echo
	params := map[string]interface{}{
		"text": "Hello, World!",
	}

	result, err := executor.ExecuteTool(ctx, "test-task", "echo", params)
	if err != nil {
		t.Fatalf("ExecuteTool failed: %v", err)
	}

	if result == nil {
		t.Fatal("ExecuteTool should return a result")
	}

	t.Logf("ExecuteTool result: %+v", result)
}

func TestMemoryUpdate(t *testing.T) {
	logger := zap.NewNop()
	te := NewThinkingEngine("test-session", "test-agent", logger)

	// اختبار ContextMemory - إضافة كيان
	te.contextMemory.mu.RLock()
	initialEntityCount := len(te.contextMemory.entities)
	te.contextMemory.mu.RUnlock()

	entity := &Entity{
		ID:         "test-entity-1",
		Type:       "test_type",
		Attributes: map[string]interface{}{"test": "data"},
		Confidence: 1.0,
	}

	te.contextMemory.mu.Lock()
	te.contextMemory.entities[entity.ID] = entity
	te.contextMemory.mu.Unlock()

	// التحقق من أن الذاكرة تغيرت
	te.contextMemory.mu.RLock()
	newEntityCount := len(te.contextMemory.entities)
	te.contextMemory.mu.RUnlock()

	if newEntityCount <= initialEntityCount {
		t.Errorf("Memory entity count should increase. Initial: %d, New: %d", initialEntityCount, newEntityCount)
	}

	t.Logf("Memory updated successfully. Initial entities: %d, New entities: %d", initialEntityCount, newEntityCount)
}

func TestEventBusPublish(t *testing.T) {
	// إنشاء EventBus بسيط
	eb := eventbus.NewEventBus()
	defer eb.Stop()

	// إنشاء قناة لاستقبال الأحداث
	eventChan := make(chan eventbus.Event, 10)

	// اشتراك في الأحداث
	eb.Subscribe("test.event", func(evt eventbus.Event) {
		eventChan <- evt
	})

	// نشر حدث
	testEvent := eventbus.Event{
		Type: "test.event",
		Payload: map[string]interface{}{
			"message": "Hello, EventBus!",
		},
	}

	eb.Publish(testEvent)

	// انتظار استقبال الحدث
	select {
	case receivedEvent := <-eventChan:
		t.Logf("Event received successfully: %+v", receivedEvent)
		if receivedEvent.Type != "test.event" {
			t.Errorf("Expected event type 'test.event', got '%s'", receivedEvent.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}

func TestShutdown(t *testing.T) {
	logger := zap.NewNop()
	te := NewThinkingEngine("test-session", "test-agent", logger)

	// اختبار أن ThinkingEngine يمكن إيقافه
	// لا توجد دالة Shutdown مباشرة في ThinkingEngine، لكن يمكننا اختبار أن المكونات تعمل بشكل صحيح
	if te.contextMemory == nil {
		t.Error("ContextMemory should be initialized before shutdown")
	}

	if te.toolRegistry == nil {
		t.Error("ToolRegistry should be initialized before shutdown")
	}

	// اختبار أن EventBus يمكن إيقافه
	eb := eventbus.NewEventBus()
	eb.Stop()

	t.Log("Shutdown test completed - components can be stopped gracefully")
}

func TestErrorRecovery(t *testing.T) {
	logger := zap.NewNop()
	te := NewThinkingEngine("test-session", "test-agent", logger)
	ctx := context.Background()

	// Test error recovery initialization
	if te.errorRecovery == nil {
		t.Error("ErrorRecovery should be initialized")
	}

	// Test RecordError
	te.RecordError(ctx, "test-error", "Test error message", []string{"context1"})

	// Test LearnFromLesson
	te.LearnFromLesson(ctx, "test-task", "test-context", "test-lesson", 0.8)
}

func TestAgentCoordination(t *testing.T) {
	logger := zap.NewNop()
	te := NewThinkingEngine("test-session", "test-agent", logger)
	ctx := context.Background()

	// Test RegisterAgent
	err := te.RegisterAgent(ctx, "agent-1", []string{"capability1", "capability2"}, 5)
	if err != nil {
		t.Errorf("RegisterAgent failed: %v", err)
	}

	// Test AssignTaskToAgents
	agents, err := te.AssignTaskToAgents(ctx, "test-task", []string{"capability1"})
	if err != nil {
		t.Errorf("AssignTaskToAgents failed: %v", err)
	}

	if len(agents) == 0 {
		t.Error("Expected at least one agent to be assigned")
	}

	// Test GetAgentStatus
	status := te.GetAgentStatus(ctx)
	if len(status) == 0 {
		t.Error("Expected at least one agent in status")
	}

	// Test UpdateAgentLoad
	err = te.UpdateAgentLoad(ctx, "agent-1", -1)
	if err != nil {
		t.Errorf("UpdateAgentLoad failed: %v", err)
	}

	// Test DetectConflicts
	conflicts := te.DetectConflicts(ctx)
	if conflicts == nil {
		t.Error("Expected conflicts slice, got nil")
	}
}

func TestCollectiveLearning(t *testing.T) {
	logger := zap.NewNop()
	te := NewThinkingEngine("test-session", "test-agent", logger)
	ctx := context.Background()

	// Test ShareLesson
	err := te.ShareLesson(ctx, "test-shared-lesson", 0.9)
	if err != nil {
		t.Errorf("ShareLesson failed: %v", err)
	}

	// Test FindSimilarLessons
	lessons, err := te.FindSimilarLessons(ctx, "test", 5)
	if err != nil {
		t.Errorf("FindSimilarLessons failed: %v", err)
	}

	if lessons == nil {
		t.Error("Expected lessons slice, got nil")
	}
}

func TestDAGExecutor(t *testing.T) {
	logger := zap.NewNop()
	te := NewThinkingEngine("test-session", "test-agent", logger)
	ctx := context.Background()

	// Test CreateDAG
	nodes := map[string]*DAGNode{
		"node1": {
			ID:           "node1",
			Task:         "task1",
			Status:       "pending",
			Dependencies: []string{},
		},
		"node2": {
			ID:           "node2",
			Task:         "task2",
			Status:       "pending",
			Dependencies: []string{"node1"},
		},
	}

	edges := []DAGEdge{
		{From: "node1", To: "node2"},
	}

	err := te.CreateDAG(ctx, "test-dag", nodes, edges)
	if err != nil {
		t.Errorf("CreateDAG failed: %v", err)
	}

	// Test ExecuteDAG
	results, err := te.ExecuteDAG(ctx, "test-dag", func(nodeID string, task string) (interface{}, error) {
		return "result-" + nodeID, nil
	})

	if err != nil {
		t.Errorf("ExecuteDAG failed: %v", err)
	}

	if results == nil {
		t.Error("Expected results, got nil")
	}

	// Test GetDAGStatus
	status, err := te.GetDAGStatus(ctx, "test-dag")
	if err != nil {
		t.Errorf("GetDAGStatus failed: %v", err)
	}

	if status == nil {
		t.Error("Expected status, got nil")
	}
}

func TestSessionGovernor(t *testing.T) {
	logger := zap.NewNop()
	te := NewThinkingEngine("test-session", "test-agent", logger)
	ctx := context.Background()

	// Test RegisterSession
	err := te.RegisterSession(ctx, "session-1", []string{"agent-1"}, 1)
	if err != nil {
		t.Errorf("RegisterSession failed: %v", err)
	}

	// Test DetectSessionConflicts
	conflicts := te.DetectSessionConflicts(ctx)
	if conflicts == nil {
		t.Error("Expected conflicts slice, got nil")
	}

	// Test AcquireResource
	err = te.AcquireResource(ctx, "session-1", "resource-1")
	if err != nil {
		t.Errorf("AcquireResource failed: %v", err)
	}

	// Test ReleaseResource
	err = te.ReleaseResource(ctx, "session-1", "resource-1")
	if err != nil {
		t.Errorf("ReleaseResource failed: %v", err)
	}

	// Test GetSessionStatus
	status, err := te.GetSessionStatus(ctx, "session-1")
	if err != nil {
		t.Errorf("GetSessionStatus failed: %v", err)
	}

	if status == nil {
		t.Error("Expected status, got nil")
	}
}

func TestDeepThink(t *testing.T) {
	logger := zap.NewNop()
	te := NewThinkingEngine("test-session", "test-agent", logger)
	ctx := context.Background()

	// Test DeepThink with heuristics (no provider set)
	result, err := te.DeepThink(ctx, "test-task", 3)
	if err != nil {
		t.Errorf("DeepThink failed: %v", err)
	}

	if result == nil {
		t.Error("Expected result, got nil")
	}

	if len(result.Stages) == 0 {
		t.Error("Expected at least one stage")
	}

	if result.FinalAnswer == "" {
		t.Error("Expected final answer")
	}
}

func TestLearnFromSession(t *testing.T) {
	logger := zap.NewNop()
	te := NewThinkingEngine("test-session", "test-agent", logger)
	ctx := context.Background()

	// Test LearnFromSession with heuristics
	tasks := []string{"task1", "task2"}
	results := []interface{}{"result1", "result2"}

	result, err := te.LearnFromSession(ctx, "session-1", tasks, results)
	if err != nil {
		t.Errorf("LearnFromSession failed: %v", err)
	}

	if result == nil {
		t.Error("Expected result, got nil")
	}

	if result.SessionID != "session-1" {
		t.Errorf("Expected session ID 'session-1', got '%s'", result.SessionID)
	}

	// Test GetSessionLearningSummary
	summary, err := te.GetSessionLearningSummary(ctx, "session-1")
	if err != nil {
		t.Errorf("GetSessionLearningSummary failed: %v", err)
	}

	if summary == nil {
		t.Error("Expected summary, got nil")
	}
}

func TestMassAgentCoordination(t *testing.T) {
	logger := zap.NewNop()
	te := NewThinkingEngine("test-session", "test-agent", logger)
	ctx := context.Background()

	// Test CreateAgentPool
	err := te.CreateAgentPool(ctx, "pool-1", 10)
	if err != nil {
		t.Errorf("CreateAgentPool failed: %v", err)
	}

	// Test AssignTaskToPool
	err = te.AssignTaskToPool(ctx, "pool-1", "test-task", 1, []string{"capability1"})
	if err != nil {
		t.Errorf("AssignTaskToPool failed: %v", err)
	}

	// Test GetPoolStatus
	status, err := te.GetPoolStatus(ctx, "pool-1")
	if err != nil {
		t.Errorf("GetPoolStatus failed: %v", err)
	}

	if status == nil {
		t.Error("Expected status, got nil")
	}
}

func TestAnalyzeTask(t *testing.T) {
	logger := zap.NewNop()
	te := NewThinkingEngine("test-session", "test-agent", logger)
	ctx := context.Background()

	// Test AnalyzeTask with heuristics (no provider set)
	analysis, err := te.AnalyzeTask(ctx, "test-task")
	if err != nil {
		t.Errorf("AnalyzeTask failed: %v", err)
	}

	if analysis == nil {
		t.Error("Expected analysis, got nil")
	}

	if analysis.TaskType == "" {
		t.Error("Expected task type")
	}
}

func TestPlanTask(t *testing.T) {
	logger := zap.NewNop()
	te := NewThinkingEngine("test-session", "test-agent", logger)
	ctx := context.Background()

	// First analyze the task
	analysis, err := te.AnalyzeTask(ctx, "test-task")
	if err != nil {
		t.Errorf("AnalyzeTask failed: %v", err)
	}

	// Test PlanTask with heuristics
	subtasks, err := te.PlanTask(ctx, analysis)
	if err != nil {
		t.Errorf("PlanTask failed: %v", err)
	}

	if subtasks == nil {
		t.Error("Expected subtasks, got nil")
	}
}

func TestVerifyResult(t *testing.T) {
	logger := zap.NewNop()
	te := NewThinkingEngine("test-session", "test-agent", logger)
	ctx := context.Background()

	// Test VerifyResult with heuristics
	result := "test-result"
	verification, err := te.VerifyResult(ctx, "test-task", result)
	if err != nil {
		t.Errorf("VerifyResult failed: %v", err)
	}

	if verification == nil {
		t.Error("Expected verification, got nil")
	}
}

func TestReflect(t *testing.T) {
	logger := zap.NewNop()
	te := NewThinkingEngine("test-session", "test-agent", logger)
	ctx := context.Background()

	// Test Reflect with heuristics
	result := "test-result"
	reflection, err := te.Reflect(ctx, "test-task", result, time.Second)
	if err != nil {
		t.Errorf("Reflect failed: %v", err)
	}

	if reflection == nil {
		t.Error("Expected reflection, got nil")
	}
}

func TestCollaborationIntegration(t *testing.T) {
	logger := zap.NewNop()
	te := NewThinkingEngine("test-session", "test-agent", logger)
	ctx := context.Background()

	// Test that collaboration engine is initialized
	if te.collaborationEngine == nil {
		t.Error("CollaborationEngine should be initialized")
	}

	// Test CreateWorkflow
	workflow, err := te.collaborationEngine.CreateWorkflow(ctx, "test-workflow", "Test workflow description")
	if err != nil {
		t.Errorf("CreateWorkflow failed: %v", err)
	}

	if workflow == nil {
		t.Error("Expected workflow, got nil")
	}

	// Test AddStep with correct arguments
	err = te.collaborationEngine.AddStep(ctx, workflow.ID, "agent1", "step1", "Step 1 description", []string{})
	if err != nil {
		t.Errorf("AddStep failed: %v", err)
	}

	// Test StartWorkflow
	err = te.collaborationEngine.StartWorkflow(ctx, workflow.ID)
	if err != nil {
		t.Errorf("StartWorkflow failed: %v", err)
	}
}

func TestThoughtTracking(t *testing.T) {
	logger := zap.NewNop()
	te := NewThinkingEngine("test-session", "test-agent", logger)
	ctx := context.Background()

	// Test AddThought
	te.AddThought(ctx, PhaseAnalysis, "Test thought", map[string]interface{}{
		"key": "value",
	})

	// Test GetThoughts
	thoughts, err := te.GetThoughts(ctx)
	if err != nil {
		t.Errorf("GetThoughts failed: %v", err)
	}

	if len(thoughts) == 0 {
		t.Error("Expected at least one thought")
	}

	// Test GetThoughtsByPhase
	analysisThoughts, err := te.GetThoughtsByPhase(ctx, PhaseAnalysis)
	if err != nil {
		t.Errorf("GetThoughtsByPhase failed: %v", err)
	}

	if len(analysisThoughts) == 0 {
		t.Error("Expected at least one analysis phase thought")
	}
}

func TestPhaseTransitions(t *testing.T) {
	logger := zap.NewNop()
	te := NewThinkingEngine("test-session", "test-agent", logger)
	ctx := context.Background()

	// Test initial phase
	if te.currentPhase.Load().(ThinkingPhase) != PhaseAnalysis {
		t.Errorf("Expected initial phase %s, got %s", PhaseAnalysis, te.currentPhase.Load().(ThinkingPhase))
	}

	// Test SetPhase
	te.SetPhase(ctx, PhasePlanning)
	if te.currentPhase.Load().(ThinkingPhase) != PhasePlanning {
		t.Errorf("Expected phase %s, got %s", PhasePlanning, te.currentPhase.Load().(ThinkingPhase))
	}

	// Test GetCurrentPhase
	phase, err := te.GetCurrentPhase(ctx)
	if err != nil {
		t.Errorf("GetCurrentPhase failed: %v", err)
	}

	if phase != PhasePlanning {
		t.Errorf("Expected current phase %s, got %s", PhasePlanning, phase)
	}
}
