package agent_bridge

import (
	"testing"

	"github.com/MortalArena/Musketeers/pkg/policy"
)

func TestValidateToolRequest(t *testing.T) {
	engine := policy.NewEngine()

	// إضافة قاعدة تسمح بالوصول
	rule := policy.Rule{
		Name:      "allow_workflow_edit",
		Effect:    policy.EffectAllow,
		Priority:  100,
		Principals: []policy.Principal{{DID: "did:mskt:agent1"}},
		Resources: []policy.Resource{{Type: "workflow:edit:nodes"}},
	}

	if err := engine.AddRule(rule); err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// اختبار صلاحية معروفة
	err := ValidateToolRequest("did:mskt:agent1", "add_workflow_node", engine)
	if err != nil {
		t.Errorf("Expected no error for authorized agent, got: %v", err)
	}

	// اختبار صلاحية غير معروفة
	err = ValidateToolRequest("did:mskt:agent2", "add_workflow_node", engine)
	if err == nil {
		t.Error("Expected error for unauthorized agent")
	}

	// اختبار أداة غير موجودة
	err = ValidateToolRequest("did:mskt:agent1", "unknown_tool", engine)
	if err == nil {
		t.Error("Expected error for unknown tool")
	}
}
