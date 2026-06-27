package session

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// StepExecutor واجهة لتنفيذ خطوات الـ 16 workflow فعلياً عبر ThinkingEngine
type StepExecutor interface {
	ExecuteStep(ctx context.Context, stepIndex int, task string, workflowState map[string]interface{}) (map[string]interface{}, error)
}

// WorkflowStepType أنواع الخطوات القياسية الـ 16
type WorkflowStepType string

const (
	StepUnderstandRequest   WorkflowStepType = "understand_request"    // 1. فهم الطلب
	StepAnalyzeContext      WorkflowStepType = "analyze_context"       // 2. تحليل السياق
	StepIdentifyTools       WorkflowStepType = "identify_tools"        // 3. تحديد الأدوات المطلوبة
	StepPlanExecution       WorkflowStepType = "plan_execution"        // 4. التخطيط للتنفيذ
	StepExecuteTools        WorkflowStepType = "execute_tools"         // 5. تنفيذ الأدوات بالترتيب
	StepVerifyResults       WorkflowStepType = "verify_results"        // 6. التحقق من النتائج
	StepHandleErrors        WorkflowStepType = "handle_errors"         // 7. معالجة الأخطاء
	StepRetryOnFailure      WorkflowStepType = "retry_on_failure"      // 8. إعادة المحاولة عند الفشل
	StepIntegrateComponents WorkflowStepType = "integrate_components"  // 9. التكامل مع المكونات الأخرى
	StepSyncState           WorkflowStepType = "sync_state"            // 10. مزامنة الحالة
	StepSendUpdates         WorkflowStepType = "send_updates"          // 11. إرسال التحديثات
	StepReceiveResponses    WorkflowStepType = "receive_responses"     // 12. استقبال الاستجابات
	StepAnalyzeFinalResults WorkflowStepType = "analyze_final_results" // 13. تحليل النتائج النهائية
	StepReflectAndLearn     WorkflowStepType = "reflect_and_learn"     // 14. التفكير والتعلم
	StepSaveLessons         WorkflowStepType = "save_lessons"          // 15. حفظ الدروس
	StepCleanupAndComplete  WorkflowStepType = "cleanup_and_complete"  // 16. الإنهاء والتنظيف
)

// WorkflowEngine محرك سير العمل - يدير الـ 16 مرحلة
type WorkflowEngine struct {
	SessionID        string            `json:"session_id"`
	SessionContainer *SessionContainer `json:"-"` // مرجع للحاوية للتكامل
	Phases           []WorkflowPhase   `json:"phases"`
	CurrentPhase     int               `json:"current_phase"`
	Progress         float64           `json:"progress"` // 0-100
	State            string            `json:"state"`    // idle, running, paused, completed
	StartedAt        time.Time         `json:"started_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	StepExecutor     StepExecutor      `json:"-"` // منفذ الخطوات الفعلي (ThinkingEngine)
	mu               sync.RWMutex
}

// WorkflowPhase مرحلة في سير العمل
type WorkflowPhase struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // pending, active, completed, failed
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Tasks       []Task    `json:"tasks"`
	Progress    float64   `json:"progress"` // 0-100
}

// Task مهمة في المرحلة
type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`      // pending, assigned, in_progress, completed, failed
	AssignedTo  string    `json:"assigned_to"` // Agent DID
	Priority    int       `json:"priority"`    // 1-10
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Progress    float64   `json:"progress"` // 0-100
	Result      string    `json:"result,omitempty"`
	DependsOn   []string  `json:"depends_on"` // Task IDs
}

// NewWorkflowEngine ينشئ محرك وورك فلو جديد
func NewWorkflowEngine(sessionID string) *WorkflowEngine {
	return &WorkflowEngine{
		SessionID: sessionID,
		Phases:    make([]WorkflowPhase, 0),
		State:     "idle",
	}
}

// SetSessionContainer يضبط مرجع الحاوية للتكامل
func (we *WorkflowEngine) SetSessionContainer(container *SessionContainer) {
	we.mu.Lock()
	defer we.mu.Unlock()
	we.SessionContainer = container
}

// SetStepExecutor يضبط منفذ الخطوات الفعلي (ThinkingEngine)
func (we *WorkflowEngine) SetStepExecutor(executor StepExecutor) {
	we.mu.Lock()
	defer we.mu.Unlock()
	we.StepExecutor = executor
}

// GetStepExecutor يرجع منفذ الخطوات الفعلي
func (we *WorkflowEngine) GetStepExecutor() StepExecutor {
	we.mu.RLock()
	defer we.mu.RUnlock()
	return we.StepExecutor
}

// GetSessionContainer يرجع مرجع الحاوية
func (we *WorkflowEngine) GetSessionContainer() *SessionContainer {
	we.mu.RLock()
	defer we.mu.RUnlock()
	return we.SessionContainer
}

// InitializePhases يهيئ المراحل
func (we *WorkflowEngine) InitializePhases(phases []WorkflowPhase) {
	we.mu.Lock()
	defer we.mu.Unlock()

	we.Phases = phases
	we.CurrentPhase = 0
	we.State = "initialized"
	we.StartedAt = time.Now()
	we.UpdatedAt = time.Now()
}

// StartPhase يبدأ مرحلة
func (we *WorkflowEngine) StartPhase(phaseIndex int) error {
	we.mu.Lock()
	defer we.mu.Unlock()

	if phaseIndex >= len(we.Phases) {
		return fmt.Errorf("phase index out of range")
	}

	we.Phases[phaseIndex].Status = "active"
	we.Phases[phaseIndex].StartedAt = time.Now()
	we.CurrentPhase = phaseIndex
	we.State = "running"
	we.UpdatedAt = time.Now()

	return nil
}

// CompletePhase يكمل مرحلة
func (we *WorkflowEngine) CompletePhase(phaseIndex int) error {
	we.mu.Lock()
	defer we.mu.Unlock()

	if phaseIndex >= len(we.Phases) {
		return fmt.Errorf("phase index out of range")
	}

	we.Phases[phaseIndex].Status = "completed"
	we.Phases[phaseIndex].CompletedAt = time.Now()
	we.Phases[phaseIndex].Progress = 100

	we.UpdatedAt = time.Now()
	we.calculateProgress()

	return nil
}

// AddTask يضيف مهمة لمرحلة
func (we *WorkflowEngine) AddTask(phaseIndex int, task Task) error {
	we.mu.Lock()
	defer we.mu.Unlock()

	if phaseIndex >= len(we.Phases) {
		return fmt.Errorf("phase index out of range")
	}

	task.ID = fmt.Sprintf("task_%d_%d", phaseIndex, len(we.Phases[phaseIndex].Tasks)+1)
	task.Status = "pending"
	task.StartedAt = time.Now()

	we.Phases[phaseIndex].Tasks = append(we.Phases[phaseIndex].Tasks, task)
	we.UpdatedAt = time.Now()

	return nil
}

// UpdateTaskStatus يحدث حالة مهمة
func (we *WorkflowEngine) UpdateTaskStatus(phaseIndex, taskIndex int, status string, progress float64) error {
	we.mu.Lock()
	defer we.mu.Unlock()

	if phaseIndex >= len(we.Phases) {
		return fmt.Errorf("phase index out of range")
	}

	if taskIndex >= len(we.Phases[phaseIndex].Tasks) {
		return fmt.Errorf("task index out of range")
	}

	we.Phases[phaseIndex].Tasks[taskIndex].Status = status
	we.Phases[phaseIndex].Tasks[taskIndex].Progress = progress

	if status == "completed" {
		we.Phases[phaseIndex].Tasks[taskIndex].CompletedAt = time.Now()
	}

	we.UpdatedAt = time.Now()
	we.calculateProgress()

	return nil
}

// calculateProgress يحسب التقدم العام
func (we *WorkflowEngine) calculateProgress() {
	totalTasks := 0
	completedTasks := 0

	for _, phase := range we.Phases {
		for _, task := range phase.Tasks {
			totalTasks++
			if task.Status == "completed" {
				completedTasks++
			}
		}
	}

	if totalTasks > 0 {
		we.Progress = float64(completedTasks) / float64(totalTasks) * 100
	}
}

// GetProgress يعيد التقدم العام
func (we *WorkflowEngine) GetProgress() float64 {
	we.mu.RLock()
	defer we.mu.RUnlock()
	return we.Progress
}

// GetCurrentPhase يعيد المرحلة الحالية
func (we *WorkflowEngine) GetCurrentPhase() int {
	we.mu.RLock()
	defer we.mu.RUnlock()
	return we.CurrentPhase
}

// Execute16StepWorkflow ينفذ ورك فلو من 16 خطوة
// [SAFETY] لا يحمل we.mu أثناء استدعاء addTaskLocked لتجنب Deadlock (sync.RWMutex غير قابل لإعادة الدخول)
func (we *WorkflowEngine) Execute16StepWorkflow(ctx context.Context, task string, thinkingEngine interface{}) (map[string]interface{}, error) {
	we.mu.Lock()
	if len(we.Phases) != 16 {
		we.mu.Unlock()
		return nil, fmt.Errorf("يجب أن يكون هناك 16 مرحلة")
	}
	we.State = "running"
	we.StartedAt = time.Now()
	we.mu.Unlock()

	// تهيئة حالة الورك فلو
	workflowState := make(map[string]interface{})
	workflowState["task"] = task

	// تنفيذ الخطوات الـ 16 بشكل متسلسل
	// [SAFETY] لا نحمل القفل خلال التنفيذ الطويل — نقفل فقط لتحديث الحالة
	for i := 0; i < 16; i++ {
		we.mu.Lock()
		we.Phases[i].Status = "active"
		we.Phases[i].StartedAt = time.Now()
		we.CurrentPhase = i
		we.mu.Unlock()

		// تنفيذ الخطوة (خارج القفل)
		stepResult, err := we.executeStep(ctx, i, workflowState, thinkingEngine)
		if err != nil {
			we.mu.Lock()
			we.Phases[i].Status = "failed"
			we.State = "failed"
			we.mu.Unlock()
			return nil, fmt.Errorf("فشل في الخطوة %d: %w", i+1, err)
		}

		// حفظ النتيجة في الحالة
		workflowState[fmt.Sprintf("step_%d", i+1)] = stepResult

		// إكمال الخطوة
		we.mu.Lock()
		we.Phases[i].Status = "completed"
		we.Phases[i].CompletedAt = time.Now()
		we.Phases[i].Progress = 100

		// [FIX] استخدام الدالة الداخلية لتجنب Deadlock (لا تستدعي mu.Lock)
		we.addTaskLocked(i, Task{
			Description: we.getStepDescription(i),
			Status:      "completed",
			Progress:    100,
			CompletedAt: time.Now(),
		})
		we.calculateProgress()
		we.UpdatedAt = time.Now()
		we.mu.Unlock()
	}

	we.mu.Lock()
	we.State = "completed"
	we.UpdatedAt = time.Now()
	we.mu.Unlock()

	return workflowState, nil
}

// addTaskLocked يضيف مهمة لمرحلة — يجب أن يُستدعى داخل we.mu.Lock()
// [WHY] نسخة داخلية لا تقفل بنفسها لتجنب Deadlock من Execute16StepWorkflow
func (we *WorkflowEngine) addTaskLocked(phaseIndex int, task Task) {
	if phaseIndex >= len(we.Phases) {
		return
	}
	task.ID = fmt.Sprintf("task_%d_%d", phaseIndex, len(we.Phases[phaseIndex].Tasks)+1)
	task.StartedAt = time.Now()
	we.Phases[phaseIndex].Tasks = append(we.Phases[phaseIndex].Tasks, task)
}

// executeStep ينفذ خطوة معينة عبر StepExecutor إذا كان متاحاً
func (we *WorkflowEngine) executeStep(ctx context.Context, stepIndex int, workflowState map[string]interface{}, thinkingEngine interface{}) (map[string]interface{}, error) {
	// استخدام StepExecutor للتنفيذ الفعلي إذا كان متاحاً
	if we.StepExecutor != nil {
		task, _ := workflowState["task"].(string)
		return we.StepExecutor.ExecuteStep(ctx, stepIndex, task, workflowState)
	}

	stepName := we.getStepName(stepIndex)
	stepDescription := we.getStepDescription(stepIndex)

	result := map[string]interface{}{
		"step":        stepIndex + 1,
		"name":        stepName,
		"description": stepDescription,
		"status":      "completed",
		"timestamp":   time.Now(),
	}

	// استخدام نتائج الخطوة السابقة
	if stepIndex > 0 {
		prevStepResult := workflowState[fmt.Sprintf("step_%d", stepIndex)]
		result["previous_step_result"] = prevStepResult
	}

	return result, nil
}

// getStepName يرجع اسم الخطوة
func (we *WorkflowEngine) getStepName(stepIndex int) string {
	stepNames := []string{
		"فهم الطلب",
		"تحليل السياق",
		"تحديد الأدوات المطلوبة",
		"التخطيط للتنفيذ",
		"تنفيذ الأدوات بالترتيب",
		"التحقق من النتائج",
		"معالجة الأخطاء",
		"إعادة المحاولة عند الفشل",
		"التكامل مع المكونات الأخرى",
		"مزامنة الحالة",
		"إرسال التحديثات",
		"استقبال الاستجابات",
		"تحليل النتائج النهائية",
		"التفكير والتعلم",
		"حفظ الدروس",
		"الإنهاء والتنظيف",
	}

	if stepIndex < len(stepNames) {
		return stepNames[stepIndex]
	}
	return fmt.Sprintf("الخطوة %d", stepIndex+1)
}

// getStepDescription يرجع وصف الخطوة
func (we *WorkflowEngine) getStepDescription(stepIndex int) string {
	descriptions := []string{
		"فهم الطلب وتحليل المتطلبات الأساسية",
		"تحليل السياق المحيط بالمهمة والموارد المتاحة",
		"تحديد الأدوات والمكونات المطلوبة للتنفيذ",
		"التخطيط لخطوات التنفيذ وتحديد الأولويات",
		"تنفيذ الأدوات بالترتيب المحدد",
		"التحقق من النتائج والتأكد من صحتها",
		"معالجة الأخطاء التي قد تظهر أثناء التنفيذ",
		"إعادة المحاولة عند الفشل مع تعديل الاستراتيجية",
		"التكامل مع المكونات الأخرى في النظام",
		"مزامنة الحالة مع المكونات الأخرى",
		"إرسال التحديثات للأطراف المعنية",
		"استقبال الاستجابات من الأطراف المعنية",
		"تحليل النتائج النهائية واستخلاص الدروس",
		"التفكير في العملية والتعلم من التجربة",
		"حفظ الدروس المستفادة للاستخدام المستقبلي",
		"الإنهاء والتنظيف وإطلاق الموارد",
	}

	if stepIndex < len(descriptions) {
		return descriptions[stepIndex]
	}
	return fmt.Sprintf("وصف الخطوة %d", stepIndex+1)
}

// GetWorkflowState يرجع حالة الورك فلو
func (we *WorkflowEngine) GetWorkflowState() map[string]interface{} {
	we.mu.RLock()
	defer we.mu.RUnlock()

	state := map[string]interface{}{
		"state":         we.State,
		"progress":      we.Progress,
		"current_phase": we.CurrentPhase,
		"phases":        make([]map[string]interface{}, len(we.Phases)),
	}

	for i, phase := range we.Phases {
		state["phases"].([]map[string]interface{})[i] = map[string]interface{}{
			"id":           phase.ID,
			"name":         phase.Name,
			"status":       phase.Status,
			"progress":     phase.Progress,
			"started_at":   phase.StartedAt,
			"completed_at": phase.CompletedAt,
			"tasks_count":  len(phase.Tasks),
		}
	}

	return state
}

// ResetWorkflow يعيد تعيين الورك فلو
func (we *WorkflowEngine) ResetWorkflow() error {
	we.mu.Lock()
	defer we.mu.Unlock()

	we.State = "pending"
	we.Progress = 0
	we.CurrentPhase = 0
	we.UpdatedAt = time.Now()

	for i := range we.Phases {
		we.Phases[i].Status = "pending"
		we.Phases[i].Progress = 0
		we.Phases[i].StartedAt = time.Time{}
		we.Phases[i].CompletedAt = time.Time{}
		we.Phases[i].Tasks = []Task{}
	}

	return nil
}
