package agent_bridge

// ToolDefinition يحدد أداة متاحة للوكيل الخارجي (متوافق مع OpenAI Schema)
type ToolDefinition struct {
	Type        string          `json:"type"` // دائماً "function"
	Function    FunctionDetails `json:"function"`
}

type FunctionDetails struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// GetAvailableTools يعيد قائمة الأدوات المسموح بها للوكيل الحالي بناءً على صلاحياته
func GetAvailableTools(permissions []string) []ToolDefinition {
	var tools []ToolDefinition

	// أداة إضافة عقدة (تتطلب صلاحية workflow:edit)
	if contains(permissions, "workflow:edit") {
		tools = append(tools, ToolDefinition{
			Type: "function",
			Function: FunctionDetails{
				Name:        "add_workflow_node",
				Description: "إضافة عقدة جديدة إلى سير العمل الحالي",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"workflow_id": map[string]string{"type": "string", "description": "معرف سير العمل"},
						"node_type":   map[string]string{"type": "string", "description": "نوع العقدة (مثال: github_trigger, code_analyzer)"},
						"config":      map[string]string{"type": "object", "description": "إعدادات العقدة بصيغة JSON"},
					},
					"required": []string{"workflow_id", "node_type"},
				},
			},
		})
	}

	// أداة الحصول على السياق (متاحة للجميع)
	tools = append(tools, ToolDefinition{
		Type: "function",
		Function: FunctionDetails{
			Name:        "get_workflow_context",
			Description: "الحصول على الحالة الحالية وسجل التنفيذ لسير عمل محدد",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"workflow_id": map[string]string{"type": "string"},
				},
				"required": []string{"workflow_id"},
			},
		},
	})

	return tools
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
