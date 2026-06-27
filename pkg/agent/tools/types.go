package tools

import "context"

// AgentRole دور الوكيل في الجلسة - يحدد الصلاحيات
type AgentRole string

const (
	RoleManager AgentRole = "manager" // مدير الجلسة - صلاحيات كاملة (قراءة، كتابة، حذف، إدارة)
	RoleRegular AgentRole = "regular" // وكيل عادي - مشاركة معرفة بدون حذف
	RoleAny     AgentRole = "any"     // أي دور
)

// ToolCategory تصنيف الأداة
type ToolCategory string

const (
	CategoryMemory    ToolCategory = "memory"     // أدوات الذاكرة الجماعية
	CategorySkills    ToolCategory = "skills"     // أدوات المهارات
	CategoryKnowledge ToolCategory = "knowledge"  // أدوات المعرفة
	CategoryChannel   ToolCategory = "channel"    // أدوات القنوات والرسائل
	CategoryFile      ToolCategory = "file"       // أدوات الملفات
	CategoryExecution ToolCategory = "execution"  // أدوات التنفيذ (terminal, browser, http)
	CategoryAgent     ToolCategory = "agent"      // أدوات الوكيل (تسجيل، معلومات)
	CategorySession   ToolCategory = "session"    // أدوات الجلسة (مهام، تقدم)
	CategoryIntegration ToolCategory = "integration" // أدوات التكامل (github, docker, email)
)

// ToolAction نوع العملية على المورد
type ToolAction string

const (
	ActionRead    ToolAction = "read"    // قراءة
	ActionWrite   ToolAction = "write"   // كتابة
	ActionDelete  ToolAction = "delete"  // حذف
	ActionAdmin   ToolAction = "admin"   // إدارة
	ActionExecute ToolAction = "execute" // تنفيذ
)

// ToolDefinition تعريف الأداة كاملاً
type ToolDefinition struct {
	Name         string       // اسم الأداة (معرف فريد)
	Description  string       // وصف الأداة
	Category     ToolCategory // تصنيف الأداة
	Action       ToolAction   // نوع العملية
	RequiredRole AgentRole    // الحد الأدنى من الدور المطلوب
	Handler      ToolHandler  // دالة التنفيذ
}

// ToolHandler دالة تنفيذ الأداة
// تتلقى السياق والبارامترات، وتعيد النتيجة أو الخطأ
type ToolHandler func(ctx context.Context, params map[string]interface{}) (interface{}, error)

// ToolInfo معلومات الأداة للعرض (بدون Handler)
type ToolInfo struct {
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	Category     ToolCategory `json:"category"`
	Action       ToolAction   `json:"action"`
	RequiredRole AgentRole    `json:"required_role"`
}

// ToolResult نتيجة تنفيذ الأداة الموحدة
type ToolResult struct {
	Success  bool                   `json:"success"`
	Data     interface{}            `json:"data,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewToolResult ينشئ نتيجة نجاح
func NewToolResult(data interface{}) *ToolResult {
	return &ToolResult{Success: true, Data: data}
}

// NewToolError ينشئ نتيجة خطأ
func NewToolError(err error) *ToolResult {
	if err == nil {
		return &ToolResult{Success: false, Error: "unknown error"}
	}
	return &ToolResult{Success: false, Error: err.Error()}
}

// PermissionSet مجموعة الصلاحيات لكل دور
type PermissionSet struct {
	ManagerTools map[string]bool // الأدوات المسموحة للمدير
	RegularTools map[string]bool // الأدوات المسموحة للوكيل العادي
}

// HasPermission يتحقق مما إذا كان الدور لديه صلاحية استخدام الأداة
func (td *ToolDefinition) HasPermission(role AgentRole) bool {
	switch td.RequiredRole {
	case RoleAny:
		return true
	case RoleManager:
		return role == RoleManager
	case RoleRegular:
		return role == RoleRegular || role == RoleManager
	default:
		// الأدوار المخصصة: المدير لديه صلاحية كاملة، أو التطابق التام
		return role == RoleManager || role == td.RequiredRole
	}
}
