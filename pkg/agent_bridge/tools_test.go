package agent_bridge

import (
	"testing"
)

func TestGetAvailableTools(t *testing.T) {
	// اختبار مع صلاحيات كاملة
	permissions := []string{"workflow:edit", "workflow:read"}
	tools := GetAvailableTools(permissions)

	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}

	// اختبار مع صلاحيات محدودة
	permissions = []string{"workflow:read"}
	tools = GetAvailableTools(permissions)

	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}

	if tools[0].Function.Name != "get_workflow_context" {
		t.Errorf("Expected get_workflow_context, got %s", tools[0].Function.Name)
	}

	// اختبار بدون صلاحيات
	permissions = []string{}
	tools = GetAvailableTools(permissions)

	if len(tools) != 1 {
		t.Errorf("Expected 1 tool (get_workflow_context is public), got %d", len(tools))
	}
}

func TestContains(t *testing.T) {
	slice := []string{"a", "b", "c"}

	if !contains(slice, "a") {
		t.Error("Expected true for 'a'")
	}

	if contains(slice, "d") {
		t.Error("Expected false for 'd'")
	}
}
