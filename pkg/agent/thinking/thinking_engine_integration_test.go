package thinking

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"
)

// MockCollectiveMemory محاكاة للذاكرة الجماعية
type MockCollectiveMemory struct {
	events []MemoryEvent
}

func (m *MockCollectiveMemory) RecordEvent(event MemoryEvent) error {
	m.events = append(m.events, event)
	return nil
}

func (m *MockCollectiveMemory) LearnFact(fact MemoryFact) error {
	return nil
}

func (m *MockCollectiveMemory) DiscoverWorkflow(workflow MemoryWorkflow) error {
	return nil
}

func (m *MockCollectiveMemory) DevelopStrategy(strategy MemoryStrategy) error {
	return nil
}

func (m *MockCollectiveMemory) GetBestWorkflow(taskType string) *MemoryWorkflow {
	return nil
}

func (m *MockCollectiveMemory) QueryEvents(filters map[string]interface{}) []MemoryEvent {
	return m.events
}

func (m *MockCollectiveMemory) AddKnowledge(item KnowledgeItem) error {
	return nil
}

func (m *MockCollectiveMemory) GetKnowledgeByCategory(category string) []KnowledgeItem {
	return nil
}

func (m *MockCollectiveMemory) SearchKnowledge(query string) []KnowledgeItem {
	return nil
}

// MockSkillsManager محاكاة لمدير المهارات
type MockSkillsManager struct {
	skills map[string]*AgentSkill
}

func (m *MockSkillsManager) RegisterAgent(agentDID, agentType string) error {
	if m.skills == nil {
		m.skills = make(map[string]*AgentSkill)
	}
	m.skills[agentDID] = &AgentSkill{
		AgentDID:     agentDID,
		AgentType:    agentType,
		OverallLevel: 50,
		Skills:       make(map[string]*Skill),
	}
	return nil
}

func (m *MockSkillsManager) RecordTaskCompletion(agentDID string, task SkillTask) error {
	if m.skills == nil {
		return nil
	}
	if skill, exists := m.skills[agentDID]; exists {
		if task.Success {
			for _, skillName := range task.SkillsUsed {
				if s, ok := skill.Skills[skillName]; ok {
					s.Level++
					s.Experience += task.XPGained
				} else {
					skill.Skills[skillName] = &Skill{
						Name:       skillName,
						Level:      1,
						Experience: task.XPGained,
					}
				}
			}
		}
	}
	return nil
}

func (m *MockSkillsManager) GetAgentSkill(agentDID string) (*AgentSkill, error) {
	if m.skills == nil {
		return nil, fmt.Errorf("agent not found")
	}
	skill, exists := m.skills[agentDID]
	if !exists {
		return nil, fmt.Errorf("agent not found: %s", agentDID)
	}
	return skill, nil
}

// MockSessionMemory محاكاة للذاكرة المحلية
type MockSessionMemory struct {
	data map[string]interface{}
}

func (m *MockSessionMemory) Store(key string, value interface{}) error {
	if m.data == nil {
		m.data = make(map[string]interface{})
	}
	m.data[key] = value
	return nil
}

func (m *MockSessionMemory) Retrieve(key string) (interface{}, error) {
	return m.data[key], nil
}

func (m *MockSessionMemory) Delete(key string) error {
	delete(m.data, key)
	return nil
}

// MockMemorySync محاكاة لمزامنة الذاكرة
type MockMemorySync struct{}

func (m *MockMemorySync) SyncWithPeers() error {
	return nil
}

func (m *MockMemorySync) GetSyncStatus() map[string]interface{} {
	return map[string]interface{}{"synced": true}
}

// MockSkillSync محاكاة لمزامنة المهارات
type MockSkillSync struct{}

func (m *MockSkillSync) SyncSkills() error {
	return nil
}

func (m *MockSkillSync) GetSkillSyncStatus() map[string]interface{} {
	return map[string]interface{}{"synced": true}
}

// MockBridgeManager محاكاة لمدير الجسور
type MockBridgeManager struct{}

func (m *MockBridgeManager) CreateBridge(sourceID, targetID string, bridgeType BridgeType) (*SessionBridge, error) {
	return &SessionBridge{
		ID:       "test-bridge",
		SourceID: sourceID,
		TargetID: targetID,
		Type:     bridgeType,
		Status:   BridgeStatusActive,
	}, nil
}

func (m *MockBridgeManager) GetBridge(bridgeID string) (*SessionBridge, error) {
	return &SessionBridge{}, nil
}

func (m *MockBridgeManager) CloseBridge(bridgeID string) error {
	return nil
}

// MockSessionContainer محاكاة للحاوية الكاملة
type MockSessionContainer struct{}

func (m *MockSessionContainer) GetID() string {
	return "test-session"
}

func (m *MockSessionContainer) GetState() UnifiedSessionState {
	return UnifiedSessionState{
		SessionID: "test-session",
		Status:    "active",
		Agents: []AgentInfo{
			{DID: "agent1", Name: "Agent 1", Status: "idle", Role: "worker"},
		},
		Tasks: []TaskInfo{
			{ID: "task1", Title: "Task 1", Status: "pending", AssignedTo: "agent1", Priority: "high"},
		},
		Progress: ProgressInfo{
			TotalTasks:     1,
			CompletedTasks: 0,
			Progress:       0.0,
		},
	}
}

// TestThinkingEngineMemoryIntegration يختبر التكامل مع الذاكرة الجماعية
func TestThinkingEngineMemoryIntegration(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	te := NewThinkingEngine("test-session", "agent-1", logger)

	// ربط الذاكرة الجماعية
	mockMemory := &MockCollectiveMemory{}
	te.SetCollectiveMemory(mockMemory)

	// اختبار تسجيل حدث
	err := te.RememberEvent(ctx, "test_action", map[string]interface{}{"key": "value"}, "success", []string{"lesson1"})
	if err != nil {
		t.Fatalf("فشل تسجيل الحدث: %v", err)
	}

	// اختبار استرجاع الأحداث
	events, err := te.RecallEvents(ctx, "")
	if err != nil {
		t.Fatalf("فشل استرجاع الأحداث: %v", err)
	}

	if len(events) == 0 {
		t.Fatal("لم يتم استرجاع أي أحداث")
	}

	t.Log("✅ التكامل مع الذاكرة الجماعية يعمل بنجاح")
}

// TestThinkingEngineSkillsIntegration يختبر التكامل مع المهارات الجماعية
func TestThinkingEngineSkillsIntegration(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	te := NewThinkingEngine("test-session", "agent-1", logger)

	// ربط مدير المهارات
	mockSkills := &MockSkillsManager{}
	te.SetSkillsManager(mockSkills)

	// اختبار التعلم من مهارة
	err := te.LearnFromSkill(ctx, "coding", true, time.Second*10)
	if err != nil {
		t.Fatalf("فشل التعلم من المهارة: %v", err)
	}

	// اختبار الحصول على مستوى المهارة
	level, err := te.GetSkillLevel(ctx, "coding")
	if err != nil {
		t.Fatalf("فشل الحصول على مستوى المهارة: %v", err)
	}

	if level != 1 {
		t.Fatalf("المستوى المتوقع 1، حصلت على %d", level)
	}

	t.Log("✅ التكامل مع المهارات الجماعية يعمل بنجاح")
}

// TestThinkingEngineBridgeIntegration يختبر التكامل مع الجسور
func TestThinkingEngineBridgeIntegration(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	te := NewThinkingEngine("test-session", "agent-1", logger)

	// ربط مدير الجسور
	mockBridgeManager := &MockBridgeManager{}
	te.SetBridgeManager(mockBridgeManager)

	// اختبار إنشاء جسر
	err := te.BridgeToSession(ctx, "target-session", "two_way")
	if err != nil {
		t.Fatalf("فشل إنشاء الجسر: %v", err)
	}

	// اختبار إرسال رسالة
	err = te.SendBridgeMessage(ctx, "test-bridge", "test", "content", map[string]interface{}{})
	if err != nil {
		t.Fatalf("فشل إرسال الرسالة: %v", err)
	}

	// اختبار استقبال رسالة
	_, err = te.ReceiveBridgeMessage(ctx, "test-bridge")
	if err != nil {
		t.Fatalf("فشل استقبال الرسالة: %v", err)
	}

	t.Log("✅ التكامل مع الجسور يعمل بنجاح")
}

// TestThinkingEngineContainerIntegration يختبر التكامل مع الحاوية الكاملة
func TestThinkingEngineContainerIntegration(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	te := NewThinkingEngine("test-session", "agent-1", logger)

	// ربط الحاوية الكاملة
	mockContainer := &MockSessionContainer{}
	te.SetSessionContainer(mockContainer)

	// اختبار الحصول على سياق الجلسة
	context, err := te.GetSessionContext(ctx)
	if err != nil {
		t.Fatalf("فشل الحصول على سياق الجلسة: %v", err)
	}

	if !context["has_container"].(bool) {
		t.Fatal("الحاوية غير مرتبطة")
	}

	t.Log("✅ التكامل مع الحاوية الكاملة يعمل بنجاح")
}

// TestThinkingEngineFullIntegration يختبر التكامل الكامل
func TestThinkingEngineFullIntegration(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	te := NewThinkingEngine("test-session", "agent-1", logger)

	// ربط جميع المكونات
	te.SetCollectiveMemory(&MockCollectiveMemory{})
	te.SetSessionMemory(&MockSessionMemory{})
	te.SetMemorySync(&MockMemorySync{})
	te.SetSkillsManager(&MockSkillsManager{})
	te.SetSkillSync(&MockSkillSync{})
	te.SetBridgeManager(&MockBridgeManager{})
	te.SetSessionContainer(&MockSessionContainer{})

	// اختبار حالة التكامل
	status := te.GetSystemIntegrationStatus(ctx)

	if !status["components"].(map[string]interface{})["collective_memory"].(bool) {
		t.Fatal("الذاكرة الجماعية غير مرتبطة")
	}

	if !status["components"].(map[string]interface{})["skills_manager"].(bool) {
		t.Fatal("مدير المهارات غير مرتبط")
	}

	if !status["components"].(map[string]interface{})["session_container"].(bool) {
		t.Fatal("الحاوية الكاملة غير مرتبطة")
	}

	// اختبار فهم البيئة
	env, err := te.UnderstandSessionEnvironment(ctx)
	if err != nil {
		t.Fatalf("فشل فهم البيئة: %v", err)
	}

	if env["session_id"] != "test-session" {
		t.Fatal("معرف الجلسة غير صحيح")
	}

	t.Log("✅ التكامل الكامل يعمل بنجاح")
}

// TestThinkingEngineJoinSession يختبر الانضمام للجلسة
func TestThinkingEngineJoinSession(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	te := NewThinkingEngine("", "agent-1", logger)

	// اختبار الانضمام كوكيل عادي
	err := te.JoinSession(ctx, "test-session", "agent")
	if err != nil {
		t.Fatalf("فشل الانضمام للجلسة: %v", err)
	}

	// اختبار الانضمام كمدير
	err = te.JoinSession(ctx, "test-session-2", "manager")
	if err != nil {
		t.Fatalf("فشل الانضمام كمدير: %v", err)
	}

	if !te.isSessionManager {
		t.Fatal("الوكيل لم يتم تعيينه كمدير")
	}

	t.Log("✅ الانضمام للجلسة يعمل بنجاح")
}

// TestThinkingEdgeCases يختبر الحالات الحدية
func TestThinkingEdgeCases(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	te := NewThinkingEngine("test-session", "agent-1", logger)

	// اختبار محاولة استخدام الذاكرة بدون ربطها
	err := te.RememberEvent(ctx, "test", nil, "success", nil)
	if err == nil {
		t.Fatal("يجب أن يفشل تسجيل الحدث بدون ربط الذاكرة")
	}

	// اختبار محاولة استخدام المهارات بدون ربطها
	_, err = te.GetSkillLevel(ctx, "coding")
	if err == nil {
		t.Fatal("يجب أن يفشل الحصول على مستوى المهارة بدون ربط مدير المهارات")
	}

	// اختبار محاولة إنشاء جسر بدون صلاحية
	err = te.BridgeToSession(ctx, "target", "two_way")
	if err == nil {
		t.Fatal("يجب أن يفشل إنشاء الجسر بدون صلاحية")
	}

	t.Log("✅ معالجة الحالات الحدية تعمل بنجاح")
}
