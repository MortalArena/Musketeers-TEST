package agent_bridge

import (
	"fmt"

	"github.com/MortalArena/Musketeers/pkg/policy"
)

// ValidateToolRequest يتحقق من أن الوكيل مخول لتنفيذ الأداة المطلوبة
func ValidateToolRequest(agentDID, toolName string, policyEngine *policy.Engine) error {
	// خريطة تربط اسم الأداة بالموارد المطلوبة في نظام السياسات
	requiredResources := map[string]string{
		"add_workflow_node":    "workflow:edit:nodes",
		"execute_workflow":     "workflow:execute",
		"get_workflow_context": "workflow:read",
	}

	resource, exists := requiredResources[toolName]
	if !exists {
		return fmt.Errorf("unknown tool: %s", toolName)
	}

	// طلب من محرك السياسات التحقق من الصلاحية
	// ملاحظة: نستخدم Evaluate بدلاً من Check لأن هذا هو المتوفر في الريبو
	req := policy.Request{
		Principal: policy.Principal{DID: agentDID},
		Resource:  policy.Resource{Type: resource},
	}

	result, err := policyEngine.Evaluate(req)
	if err != nil {
		return fmt.Errorf("policy check failed: %w", err)
	}

	if result.Effect != policy.EffectAllow {
		return fmt.Errorf("permission denied: agent %s cannot use tool %s (requires %s)", agentDID, toolName, resource)
	}

	return nil
}
