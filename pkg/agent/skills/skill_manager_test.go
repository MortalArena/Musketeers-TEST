package skills

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestSkillManager_NewSkillManager(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sm := NewSkillManager(logger)

	if sm == nil {
		t.Fatal("NewSkillManager returned nil")
	}

	if sm.skills == nil {
		t.Error("skills map is nil")
	}

	if sm.loader == nil {
		t.Error("loader is nil")
	}

	if sm.executor == nil {
		t.Error("executor is nil")
	}
}

func TestSkillManager_AddSkillDir(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sm := NewSkillManager(logger)

	// إضافة مهارة مباشرة للاختبار
	sm.skills["test-skill"] = &Skill{
		Name:        "test-skill",
		Description: "Test skill for testing",
		Disabled:    false,
	}

	// التحقق من تحميل المهارة
	skill, err := sm.GetSkill("test-skill")
	if err != nil {
		t.Fatalf("GetSkill failed: %v", err)
	}

	if skill.Name != "test-skill" {
		t.Errorf("Expected skill name 'test-skill', got '%s'", skill.Name)
	}

	if skill.Description != "Test skill for testing" {
		t.Errorf("Expected description 'Test skill for testing', got '%s'", skill.Description)
	}
}

func TestSkillManager_GetSkill(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sm := NewSkillManager(logger)

	// اختبار الحصول على مهارة غير موجودة
	_, err := sm.GetSkill("non-existent-skill")
	if err == nil {
		t.Error("Expected error for non-existent skill, got nil")
	}
}

func TestSkillManager_SearchSkills(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sm := NewSkillManager(logger)

	// إضافة مهارة للاختبار
	sm.skills["test-skill"] = &Skill{
		Name:        "test-skill",
		Description: "Test skill for testing",
		Disabled:    false,
	}

	// البحث عن مهارة
	results := sm.SearchSkills("test")
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if results[0].Name != "test-skill" {
		t.Errorf("Expected skill 'test-skill', got '%s'", results[0].Name)
	}
}

func TestSkillManager_ExecuteSkill(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sm := NewSkillManager(logger)

	// إضافة مهارة للاختبار
	sm.skills["test-skill"] = &Skill{
		Name:         "test-skill",
		Description:  "Test skill for testing",
		Instructions: "Execute this test skill",
		Disabled:     false,
	}

	ctx := context.Background()
	agentCtx := &AgentContext{
		SessionID:   "test-session",
		AgentID:     "test-agent",
		TaskID:      "test-task",
		Metadata:    make(map[string]interface{}),
		Environment: make(map[string]string),
	}

	// تنفيذ المهارة
	result, err := sm.ExecuteSkill(ctx, "test-skill", agentCtx)
	if err != nil {
		t.Fatalf("ExecuteSkill failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success, got failure")
	}

	if result.SkillName != "test-skill" {
		t.Errorf("Expected skill name 'test-skill', got '%s'", result.SkillName)
	}
}

func TestSkillManager_ExecuteSkill_Disabled(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sm := NewSkillManager(logger)

	// إضافة مهارة معطلة
	sm.skills["disabled-skill"] = &Skill{
		Name:     "disabled-skill",
		Disabled: true,
	}

	ctx := context.Background()
	agentCtx := &AgentContext{
		SessionID:   "test-session",
		AgentID:     "test-agent",
		TaskID:      "test-task",
		Metadata:    make(map[string]interface{}),
		Environment: make(map[string]string),
	}

	// محاولة تنفيذ مهارة معطلة
	_, err := sm.ExecuteSkill(ctx, "disabled-skill", agentCtx)
	if err == nil {
		t.Error("Expected error for disabled skill, got nil")
	}
}

func TestSkillManager_GetAllSkills(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sm := NewSkillManager(logger)

	// إضافة مهارات للاختبار
	sm.skills["skill1"] = &Skill{Name: "skill1"}
	sm.skills["skill2"] = &Skill{Name: "skill2"}

	// الحصول على جميع المهارات
	skills := sm.GetAllSkills()
	if len(skills) != 2 {
		t.Errorf("Expected 2 skills, got %d", len(skills))
	}
}

func TestSkillManager_GetSkillSummary(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sm := NewSkillManager(logger)

	// إضافة مهارات للاختبار
	sm.skills["skill1"] = &Skill{Name: "skill1", Disabled: false}
	sm.skills["skill2"] = &Skill{Name: "skill2", Disabled: true}

	// الحصول على ملخص المهارات
	summary := sm.GetSkillSummary()
	if summary["total_skills"] != 2 {
		t.Errorf("Expected total_skills 2, got %v", summary["total_skills"])
	}

	if summary["enabled_skills"] != 1 {
		t.Errorf("Expected enabled_skills 1, got %v", summary["enabled_skills"])
	}

	if summary["disabled_skills"] != 1 {
		t.Errorf("Expected disabled_skills 1, got %v", summary["disabled_skills"])
	}
}

func TestSkillLoader_LoadSkill(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sl := NewSkillLoader(logger)

	// إنشاء ملف SKILL.md مؤقت
	tempDir := t.TempDir()
	skillFile := filepath.Join(tempDir, "SKILL.md")
	content := `---
name: test-skill
description: Test skill
---
Test skill instructions.`
	if err := os.WriteFile(skillFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create SKILL.md: %v", err)
	}

	// تحميل المهارة
	skill, err := sl.LoadSkill(tempDir)
	if err != nil {
		t.Fatalf("LoadSkill failed: %v", err)
	}

	if skill.Name != "test-skill" {
		t.Errorf("Expected name 'test-skill', got '%s'", skill.Name)
	}

	if skill.Description != "Test skill" {
		t.Errorf("Expected description 'Test skill', got '%s'", skill.Description)
	}
}

func TestSkillLoader_LoadSkill_MissingName(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sl := NewSkillLoader(logger)

	// إنشاء ملف SKILL.md بدون اسم
	tempDir := t.TempDir()
	skillFile := filepath.Join(tempDir, "SKILL.md")
	content := `---
description: Test skill
---
Test skill instructions.`
	if err := os.WriteFile(skillFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create SKILL.md: %v", err)
	}

	// محاولة تحميل مهارة بدون اسم
	_, err := sl.LoadSkill(tempDir)
	if err == nil {
		t.Error("Expected error for missing name, got nil")
	}
}

func TestSkillExecutor_ExecuteSkill(t *testing.T) {
	logger := zaptest.NewLogger(t)
	se := NewSkillExecutor(logger)

	skill := &Skill{
		Name:         "test-skill",
		Instructions: "Test instructions",
	}

	ctx := context.Background()
	agentCtx := &AgentContext{
		SessionID:   "test-session",
		AgentID:     "test-agent",
		TaskID:      "test-task",
		Metadata:    make(map[string]interface{}),
		Environment: make(map[string]string),
	}

	// تنفيذ المهارة
	result, err := se.ExecuteSkill(ctx, skill, agentCtx)
	if err != nil {
		t.Fatalf("ExecuteSkill failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success, got failure")
	}
}

// اختبارات التكامل
func TestSkillManager_Integration_CompleteWorkflow(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sm := NewSkillManager(logger)

	// إضافة مهارة مباشرة للاختبار
	sm.skills["integration-test-skill"] = &Skill{
		Name:        "integration-test-skill",
		Description: "Integration test skill",
		Disabled:    false,
	}

	// البحث عن المهارة
	results := sm.SearchSkills("integration")
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	// تنفيذ المهارة
	ctx := context.Background()
	agentCtx := &AgentContext{
		SessionID:   "test-session",
		AgentID:     "test-agent",
		TaskID:      "test-task",
		Metadata:    make(map[string]interface{}),
		Environment: make(map[string]string),
	}

	result, err := sm.ExecuteSkill(ctx, results[0].Name, agentCtx)
	if err != nil {
		t.Fatalf("ExecuteSkill failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success, got failure")
	}

	// الحصول على ملخص
	summary := sm.GetSkillSummary()
	if summary["total_skills"] != 1 {
		t.Errorf("Expected total_skills 1, got %v", summary["total_skills"])
	}
}

// اختبارات الأمان
func TestSkillManager_Security_PathValidation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sm := NewSkillManager(logger)

	// محاولة إضافة دليل غير موجود
	err := sm.AddSkillDir("/non/existent/path")
	if err == nil {
		t.Error("Expected error for non-existent directory, got nil")
	}
}

func TestSkillManager_Security_ConcurrentAccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sm := NewSkillManager(logger)

	// إضافة مهارة
	sm.skills["test-skill"] = &Skill{Name: "test-skill"}

	// اختبار الوصول المتزامن
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			sm.GetSkill("test-skill")
			done <- true
		}()
	}

	// انتظار جميع goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// اختبارات الأداء
func TestSkillManager_Performance_LargeSkills(t *testing.T) {
	logger := zaptest.NewLogger(t)
	sm := NewSkillManager(logger)

	// إضافة عدد كبير من المهارات
	for i := 0; i < 1000; i++ {
		sm.skills[fmt.Sprintf("skill-%d", i)] = &Skill{
			Name:        fmt.Sprintf("skill-%d", i),
			Description: fmt.Sprintf("Description %d", i),
			Disabled:    false,
		}
	}

	// اختبار البحث
	start := time.Now()
	results := sm.SearchSkills("skill")
	duration := time.Since(start)

	if len(results) != 1000 {
		t.Errorf("Expected 1000 results, got %d", len(results))
	}

	if duration > time.Second {
		t.Errorf("Search took too long: %v", duration)
	}
}
