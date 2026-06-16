package automation

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestAutomationManager_NewAutomationManager(t *testing.T) {
	logger := zaptest.NewLogger(t)
	am := NewAutomationManager(logger)

	if am == nil {
		t.Fatal("NewAutomationManager returned nil")
	}

	if am.automations == nil {
		t.Error("automations map is nil")
	}

	if am.triggerManager == nil {
		t.Error("triggerManager is nil")
	}

	if am.actionManager == nil {
		t.Error("actionManager is nil")
	}

	if am.mcpManager == nil {
		t.Error("mcpManager is nil")
	}
}

func TestAutomationManager_CreateAutomation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	am := NewAutomationManager(logger)

	config := &AutomationConfig{
		Name:          "test-automation",
		Description:   "Test automation for testing",
		Triggers:      []Trigger{&MockTrigger{shouldFire: true}},
		Actions:       []Action{&MockAction{shouldSucceed: true}},
		Prompts:       []string{"Test prompt"},
		Model:         "claude",
		MemoryEnabled: true,
	}

	// إنشاء أتمتة
	automation, err := am.CreateAutomation(config)
	if err != nil {
		t.Fatalf("CreateAutomation failed: %v", err)
	}

	if automation.Name != "test-automation" {
		t.Errorf("Expected name 'test-automation', got '%s'", automation.Name)
	}

	if !automation.Enabled {
		t.Error("Expected automation to be enabled")
	}
}

func TestAutomationManager_CreateAutomation_MissingName(t *testing.T) {
	logger := zaptest.NewLogger(t)
	am := NewAutomationManager(logger)

	config := &AutomationConfig{
		Description: "Test automation",
		Triggers:    []Trigger{},
		Actions:     []Action{},
	}

	// محاولة إنشاء أتمتة بدون اسم
	_, err := am.CreateAutomation(config)
	if err == nil {
		t.Error("Expected error for missing name, got nil")
	}
}

func TestAutomationManager_CreateAutomation_NoTriggers(t *testing.T) {
	logger := zaptest.NewLogger(t)
	am := NewAutomationManager(logger)

	config := &AutomationConfig{
		Name:     "test-automation",
		Triggers: []Trigger{},
		Actions:  []Action{},
	}

	// محاولة إنشاء أتمتة بدون تشغيلات
	_, err := am.CreateAutomation(config)
	if err == nil {
		t.Error("Expected error for no triggers, got nil")
	}
}

func TestAutomationManager_GetAutomation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	am := NewAutomationManager(logger)

	// إنشاء أتمتة للاختبار
	config := &AutomationConfig{
		Name:     "test-automation",
		Triggers: []Trigger{&MockTrigger{}},
		Actions:  []Action{&MockAction{}},
	}

	_, err := am.CreateAutomation(config)
	if err != nil {
		t.Fatalf("CreateAutomation failed: %v", err)
	}

	// الحصول على الأتمتة
	automation, err := am.GetAutomation("test-automation")
	if err != nil {
		t.Fatalf("GetAutomation failed: %v", err)
	}

	if automation.Name != "test-automation" {
		t.Errorf("Expected name 'test-automation', got '%s'", automation.Name)
	}
}

func TestAutomationManager_GetAutomation_NotFound(t *testing.T) {
	logger := zaptest.NewLogger(t)
	am := NewAutomationManager(logger)

	// اختبار الحصول على أتمتة غير موجودة
	_, err := am.GetAutomation("non-existent-automation")
	if err == nil {
		t.Error("Expected error for non-existent automation, got nil")
	}
}

func TestAutomationManager_ExecuteAutomation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	am := NewAutomationManager(logger)

	// إنشاء أتمتة للاختبار
	config := &AutomationConfig{
		Name:     "test-automation",
		Triggers: []Trigger{&MockTrigger{shouldFire: true}},
		Actions:  []Action{&MockAction{shouldSucceed: true}},
	}

	_, err := am.CreateAutomation(config)
	if err != nil {
		t.Fatalf("CreateAutomation failed: %v", err)
	}

	ctx := context.Background()

	// تنفيذ الأتمتة
	result, err := am.ExecuteAutomation(ctx, "test-automation")
	if err != nil {
		t.Fatalf("ExecuteAutomation failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success, got failure")
	}
}

func TestAutomationManager_EnableAutomation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	am := NewAutomationManager(logger)

	// إنشاء أتمتة للاختبار
	config := &AutomationConfig{
		Name:     "test-automation",
		Triggers: []Trigger{&MockTrigger{}},
		Actions:  []Action{&MockAction{}},
	}

	_, err := am.CreateAutomation(config)
	if err != nil {
		t.Fatalf("CreateAutomation failed: %v", err)
	}

	// تعطيل الأتمتة
	if err := am.DisableAutomation("test-automation"); err != nil {
		t.Fatalf("DisableAutomation failed: %v", err)
	}

	// تفعيل الأتمتة
	if err := am.EnableAutomation("test-automation"); err != nil {
		t.Fatalf("EnableAutomation failed: %v", err)
	}

	// التحقق من التفعيل
	automation, _ := am.GetAutomation("test-automation")
	if !automation.Enabled {
		t.Error("Expected automation to be enabled")
	}
}

func TestAutomationManager_DisableAutomation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	am := NewAutomationManager(logger)

	// إنشاء أتمتة للاختبار
	config := &AutomationConfig{
		Name:     "test-automation",
		Triggers: []Trigger{&MockTrigger{}},
		Actions:  []Action{&MockAction{}},
	}

	_, err := am.CreateAutomation(config)
	if err != nil {
		t.Fatalf("CreateAutomation failed: %v", err)
	}

	// تعطيل الأتمتة
	if err := am.DisableAutomation("test-automation"); err != nil {
		t.Fatalf("DisableAutomation failed: %v", err)
	}

	// التحقق من التعطيل
	automation, _ := am.GetAutomation("test-automation")
	if automation.Enabled {
		t.Error("Expected automation to be disabled")
	}
}

func TestAutomationManager_DeleteAutomation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	am := NewAutomationManager(logger)

	// إنشاء أتمتة للاختبار
	config := &AutomationConfig{
		Name:     "test-automation",
		Triggers: []Trigger{&MockTrigger{}},
		Actions:  []Action{&MockAction{}},
	}

	_, err := am.CreateAutomation(config)
	if err != nil {
		t.Fatalf("CreateAutomation failed: %v", err)
	}

	// حذف الأتمتة
	if err := am.DeleteAutomation("test-automation"); err != nil {
		t.Fatalf("DeleteAutomation failed: %v", err)
	}

	// التحقق من الحذف
	_, err = am.GetAutomation("test-automation")
	if err == nil {
		t.Error("Expected error for deleted automation, got nil")
	}
}

func TestAutomationManager_GetAllAutomations(t *testing.T) {
	logger := zaptest.NewLogger(t)
	am := NewAutomationManager(logger)

	// إنشاء أتمتات للاختبار
	for i := 0; i < 3; i++ {
		config := &AutomationConfig{
			Name:     fmt.Sprintf("automation-%d", i),
			Triggers: []Trigger{&MockTrigger{}},
			Actions:  []Action{&MockAction{}},
		}
		_, err := am.CreateAutomation(config)
		if err != nil {
			t.Fatalf("CreateAutomation failed: %v", err)
		}
	}

	// الحصول على جميع الأتمتات
	automations := am.GetAllAutomations()
	if len(automations) != 3 {
		t.Errorf("Expected 3 automations, got %d", len(automations))
	}
}

func TestAutomationManager_GetAutomationSummary(t *testing.T) {
	logger := zaptest.NewLogger(t)
	am := NewAutomationManager(logger)

	// إنشاء أتمتات للاختبار
	config1 := &AutomationConfig{
		Name:     "enabled-automation",
		Triggers: []Trigger{&MockTrigger{}},
		Actions:  []Action{&MockAction{}},
	}
	config2 := &AutomationConfig{
		Name:     "disabled-automation",
		Triggers: []Trigger{&MockTrigger{}},
		Actions:  []Action{&MockAction{}},
	}

	am.CreateAutomation(config1)
	am.CreateAutomation(config2)
	am.DisableAutomation("disabled-automation")

	// الحصول على ملخص الأتمتات
	summary := am.GetAutomationSummary()
	if summary["total_automations"] != 2 {
		t.Errorf("Expected total_automations 2, got %v", summary["total_automations"])
	}

	if summary["enabled_automations"] != 1 {
		t.Errorf("Expected enabled_automations 1, got %v", summary["enabled_automations"])
	}

	if summary["disabled_automations"] != 1 {
		t.Errorf("Expected disabled_automations 1, got %v", summary["disabled_automations"])
	}
}

// MockTrigger تشغيل وهمي للاختبار
type MockTrigger struct {
	shouldFire bool
}

func (mt *MockTrigger) Type() string {
	return "mock"
}

func (mt *MockTrigger) Evaluate(ctx context.Context) bool {
	return mt.shouldFire
}

func (mt *MockTrigger) GetPayload() map[string]interface{} {
	return map[string]interface{}{"mock": true}
}

// MockAction إجراء وهمي للاختبار
type MockAction struct {
	shouldSucceed bool
}

func (ma *MockAction) Type() string {
	return "mock"
}

func (ma *MockAction) Execute(ctx context.Context, payload map[string]interface{}) error {
	if !ma.shouldSucceed {
		return fmt.Errorf("mock action failed")
	}
	return nil
}

// اختبارات الأمان
func TestAutomationManager_Security_ConcurrentAccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	am := NewAutomationManager(logger)

	// إنشاء أتمتة
	config := &AutomationConfig{
		Name:     "test-automation",
		Triggers: []Trigger{&MockTrigger{}},
		Actions:  []Action{&MockAction{}},
	}
	am.CreateAutomation(config)

	// اختبار الوصول المتزامن
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			am.GetAutomation("test-automation")
			done <- true
		}()
	}

	// انتظار جميع goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// اختبارات التكامل
func TestAutomationManager_Integration_CompleteWorkflow(t *testing.T) {
	logger := zaptest.NewLogger(t)
	am := NewAutomationManager(logger)

	// إنشاء أتمتة كاملة
	config := &AutomationConfig{
		Name:          "integration-test-automation",
		Description:   "Integration test automation",
		Triggers:      []Trigger{&MockTrigger{shouldFire: true}},
		Actions:       []Action{&MockAction{shouldSucceed: true}},
		Prompts:       []string{"Test prompt"},
		Model:         "claude",
		MemoryEnabled: true,
	}

	// إنشاء الأتمتة
	automation, err := am.CreateAutomation(config)
	if err != nil {
		t.Fatalf("CreateAutomation failed: %v", err)
	}

	// تنفيذ الأتمتة
	ctx := context.Background()
	result, err := am.ExecuteAutomation(ctx, automation.Name)
	if err != nil {
		t.Fatalf("ExecuteAutomation failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success, got failure")
	}

	// الحصول على ملخص
	summary := am.GetAutomationSummary()
	if summary["total_automations"] != 1 {
		t.Errorf("Expected total_automations 1, got %v", summary["total_automations"])
	}

	// حذف الأتمتة
	if err := am.DeleteAutomation(automation.Name); err != nil {
		t.Fatalf("DeleteAutomation failed: %v", err)
	}
}
