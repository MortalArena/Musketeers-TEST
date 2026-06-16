package subagents

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestSubagentManager_NewSubagentManager(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sm := NewSubagentManager(logger)

	if sm == nil {
		t.Fatal("NewSubagentManager returned nil")
	}

	if sm.subagents == nil {
		t.Error("subagents map is nil")
	}

	if sm.factory == nil {
		t.Error("factory is nil")
	}

	if sm.executor == nil {
		t.Error("executor is nil")
	}
}

func TestSubagentManager_AddAgentDir(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sm := NewSubagentManager(logger)

	// إنشاء دليل مؤقت للاختبار
	tempDir := t.TempDir()
	agentDir := filepath.Join(tempDir, "test_agent")
	if err := os.Mkdir(agentDir, 0755); err != nil {
		t.Fatalf("Failed to create test agent directory: %v", err)
	}

	// إنشاء ملف .md بسيط
	agentFile := filepath.Join(agentDir, "test_agent.md")
	content := `---
name: test-agent
description: Test agent for testing
---
This is a test agent.`
	if err := os.WriteFile(agentFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create agent file: %v", err)
	}

	// إضافة دليل الوكلاء
	if err := sm.AddAgentDir(agentDir); err != nil {
		t.Fatalf("AddAgentDir failed: %v", err)
	}

	// التحقق من تحميل الوكيل
	agent, err := sm.GetSubagent("test-agent")
	if err != nil {
		t.Fatalf("GetSubagent failed: %v", err)
	}

	if agent.Name != "test-agent" {
		t.Errorf("Expected agent name 'test-agent', got '%s'", agent.Name)
	}

	if agent.Description != "Test agent for testing" {
		t.Errorf("Expected description 'Test agent for testing', got '%s'", agent.Description)
	}
}

func TestSubagentManager_GetSubagent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sm := NewSubagentManager(logger)

	// اختبار الحصول على وكيل غير موجود
	_, err := sm.GetSubagent("non-existent-agent")
	if err == nil {
		t.Error("Expected error for non-existent agent, got nil")
	}
}

func TestSubagentManager_SearchSubagents(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sm := NewSubagentManager(logger)

	// إضافة وكيل للاختبار
	sm.subagents["test-agent"] = &Subagent{
		Name:        "test-agent",
		Description: "Test agent for testing",
	}

	// البحث عن وكيل
	results := sm.SearchSubagents("test")
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if results[0].Name != "test-agent" {
		t.Errorf("Expected agent 'test-agent', got '%s'", results[0].Name)
	}
}

func TestSubagentManager_CreateSubagent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sm := NewSubagentManager(logger)

	config := &SubagentConfig{
		Name:           "new-agent",
		Description:    "New test agent",
		SystemPrompt:   "You are a test agent",
		Specialization: "coder",
		Capabilities:   []string{"coding", "debugging"},
		Priority:       1,
	}

	// إنشاء وكيل
	agent, err := sm.CreateSubagent(config)
	if err != nil {
		t.Fatalf("CreateSubagent failed: %v", err)
	}

	if agent.Name != "new-agent" {
		t.Errorf("Expected name 'new-agent', got '%s'", agent.Name)
	}

	if agent.Specialization != "coder" {
		t.Errorf("Expected specialization 'coder', got '%s'", agent.Specialization)
	}
}

func TestSubagentManager_CreateSubagent_MissingName(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sm := NewSubagentManager(logger)

	config := &SubagentConfig{
		Description:  "New test agent",
		SystemPrompt: "You are a test agent",
	}

	// محاولة إنشاء وكيل بدون اسم
	_, err := sm.CreateSubagent(config)
	if err == nil {
		t.Error("Expected error for missing name, got nil")
	}
}

func TestSubagentManager_DelegateTask(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sm := NewSubagentManager(logger)

	// إضافة وكيل للاختبار
	sm.subagents["test-agent"] = &Subagent{
		Name:         "test-agent",
		SystemPrompt: "You are a test agent",
	}

	ctx := context.Background()
	task := &Task{
		ID:          "test-task",
		Description: "Test task",
		Parameters:  make(map[string]interface{}),
		Priority:    1,
	}

	// تفويض المهمة
	result, err := sm.DelegateTask(ctx, task, "test-agent")
	if err != nil {
		t.Fatalf("DelegateTask failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success, got failure")
	}

	if result.SubagentName != "test-agent" {
		t.Errorf("Expected subagent name 'test-agent', got '%s'", result.SubagentName)
	}
}

func TestSubagentManager_GetAllSubagents(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sm := NewSubagentManager(logger)

	// إضافة وكلاء للاختبار
	sm.subagents["agent1"] = &Subagent{Name: "agent1"}
	sm.subagents["agent2"] = &Subagent{Name: "agent2"}

	// الحصول على جميع الوكلاء
	agents := sm.GetAllSubagents()
	if len(agents) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(agents))
	}
}

func TestSubagentManager_GetSubagentSummary(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sm := NewSubagentManager(logger)

	// إضافة وكلاء للاختبار
	sm.subagents["agent1"] = &Subagent{
		Name:           "agent1",
		Specialization: "coder",
	}
	sm.subagents["agent2"] = &Subagent{
		Name:           "agent2",
		Specialization: "designer",
	}

	// الحصول على ملخص الوكلاء
	summary := sm.GetSubagentSummary()
	if summary["total_subagents"] != 2 {
		t.Errorf("Expected total_subagents 2, got %v", summary["total_subagents"])
	}
}

// اختبارات الأمان
func TestSubagentManager_Security_PathValidation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sm := NewSubagentManager(logger)

	// محاولة إضافة دليل غير موجود
	err := sm.AddAgentDir("/non/existent/path")
	if err == nil {
		t.Error("Expected error for non-existent directory, got nil")
	}
}

func TestSubagentManager_Security_ConcurrentAccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sm := NewSubagentManager(logger)

	// إضافة وكيل
	sm.subagents["test-agent"] = &Subagent{Name: "test-agent"}

	// اختبار الوصول المتزامن
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			sm.GetSubagent("test-agent")
			done <- true
		}()
	}

	// انتظار جميع goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
