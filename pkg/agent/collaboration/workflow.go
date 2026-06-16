package collaboration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// WorkflowStep خطوة في الورك فلو
type WorkflowStep struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	AssignedTo  string                 `json:"assigned_to"`
	Status      string                 `json:"status"`
	Dependencies []string             `json:"dependencies"`
	StartedAt   *time.Time            `json:"started_at,omitempty"`
	CompletedAt *time.Time            `json:"completed_at,omitempty"`
	Result      interface{}            `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Workflow ورك فلو تعاوني
type Workflow struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Status      string                 `json:"status"`
	Steps       []*WorkflowStep        `json:"steps"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time            `json:"started_at,omitempty"`
	CompletedAt *time.Time            `json:"completed_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// CollaborationEngine محرك التعاون
type CollaborationEngine struct {
	workflows   map[string]*Workflow
	logger      *zap.Logger
	mu          sync.RWMutex
	sessionID   string
	agentID     string
}

// NewCollaborationEngine ينشئ محرك تعاون جديد
func NewCollaborationEngine(sessionID, agentID string, logger *zap.Logger) *CollaborationEngine {
	return &CollaborationEngine{
		workflows: make(map[string]*Workflow),
		logger:    logger,
		sessionID: sessionID,
		agentID:   agentID,
	}
}

// CreateWorkflow ينشئ ورك فلو جديد
func (ce *CollaborationEngine) CreateWorkflow(ctx context.Context, name, description string) (*Workflow, error) {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	workflow := &Workflow{
		ID:          fmt.Sprintf("workflow_%d", time.Now().UnixNano()),
		Name:        name,
		Description: description,
		Status:      "pending",
		Steps:       make([]*WorkflowStep, 0),
		CreatedAt:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	ce.workflows[workflow.ID] = workflow

	ce.logger.Info("تم إنشاء ورك فلو جديد",
		zap.String("session_id", ce.sessionID),
		zap.String("agent_id", ce.agentID),
		zap.String("workflow_id", workflow.ID),
		zap.String("name", name),
	)

	return workflow, nil
}

// AddStep يضيف خطوة للورك فلو
func (ce *CollaborationEngine) AddStep(ctx context.Context, workflowID, name, description string, assignedTo string, dependencies []string) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	workflow, ok := ce.workflows[workflowID]
	if !ok {
		return fmt.Errorf("ورك فلو غير موجود: %s", workflowID)
	}

	step := &WorkflowStep{
		ID:           fmt.Sprintf("step_%d", time.Now().UnixNano()),
		Name:         name,
		Description:  description,
		AssignedTo:   assignedTo,
		Status:       "pending",
		Dependencies: dependencies,
		Metadata:     make(map[string]interface{}),
	}

	workflow.Steps = append(workflow.Steps, step)

	ce.logger.Info("تم إضافة خطوة للورك فلو",
		zap.String("session_id", ce.sessionID),
		zap.String("agent_id", ce.agentID),
		zap.String("workflow_id", workflowID),
		zap.String("step_id", step.ID),
		zap.String("assigned_to", assignedTo),
	)

	return nil
}

// StartWorkflow يبدأ الورك فلو
func (ce *CollaborationEngine) StartWorkflow(ctx context.Context, workflowID string) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	workflow, ok := ce.workflows[workflowID]
	if !ok {
		return fmt.Errorf("ورك فلو غير موجود: %s", workflowID)
	}

	workflow.Status = "in_progress"
	now := time.Now()
	workflow.StartedAt = &now

	ce.logger.Info("تم بدء الورك فلو",
		zap.String("session_id", ce.sessionID),
		zap.String("agent_id", ce.agentID),
		zap.String("workflow_id", workflowID),
	)

	return nil
}

// CompleteStep يكمل خطوة في الورك فلو
func (ce *CollaborationEngine) CompleteStep(ctx context.Context, workflowID, stepID string, result interface{}, err error) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	workflow, ok := ce.workflows[workflowID]
	if !ok {
		return fmt.Errorf("ورك فلو غير موجود: %s", workflowID)
	}

	for _, step := range workflow.Steps {
		if step.ID == stepID {
			step.Status = "completed"
			step.Result = result
			if err != nil {
				step.Error = err.Error()
				step.Status = "failed"
			}
			now := time.Now()
			step.CompletedAt = &now

			ce.logger.Info("تم إكمال الخطوة",
				zap.String("session_id", ce.sessionID),
				zap.String("agent_id", ce.agentID),
				zap.String("workflow_id", workflowID),
				zap.String("step_id", stepID),
				zap.Bool("success", err == nil),
			)

			// التحقق من إكمال جميع الخطوات
			allCompleted := true
			for _, s := range workflow.Steps {
				if s.Status != "completed" && s.Status != "failed" {
					allCompleted = false
					break
				}
			}

			if allCompleted {
				workflow.Status = "completed"
				now := time.Now()
				workflow.CompletedAt = &now

				ce.logger.Info("تم إكمال الورك فلو",
					zap.String("session_id", ce.sessionID),
					zap.String("agent_id", ce.agentID),
					zap.String("workflow_id", workflowID),
				)
			}

			return nil
		}
	}

	return fmt.Errorf("خطوة غير موجودة: %s", stepID)
}

// GetNextStep يرجع الخطوة التالية القابلة للتنفيذ
func (ce *CollaborationEngine) GetNextStep(ctx context.Context, workflowID, agentID string) (*WorkflowStep, error) {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	workflow, ok := ce.workflows[workflowID]
	if !ok {
		return nil, fmt.Errorf("ورك فلو غير موجود: %s", workflowID)
	}

	for _, step := range workflow.Steps {
		if step.Status == "pending" && step.AssignedTo == agentID {
			// التحقق من الاعتمادات
			dependenciesMet := true
			for _, depID := range step.Dependencies {
				depMet := false
				for _, s := range workflow.Steps {
					if s.ID == depID && s.Status == "completed" {
						depMet = true
						break
					}
				}
				if !depMet {
					dependenciesMet = false
					break
				}
			}

			if dependenciesMet {
				return step, nil
			}
		}
	}

	return nil, fmt.Errorf("لا توجد خطوات قابلة للتنفيذ")
}

// GetWorkflow يرجع ورك فلو
func (ce *CollaborationEngine) GetWorkflow(ctx context.Context, workflowID string) (*Workflow, error) {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	workflow, ok := ce.workflows[workflowID]
	if !ok {
		return nil, fmt.Errorf("ورك فلو غير موجود: %s", workflowID)
	}

	return workflow, nil
}

// GetAllWorkflows يرجع جميع الورك فلو
func (ce *CollaborationEngine) GetAllWorkflows(ctx context.Context) ([]*Workflow, error) {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	workflows := make([]*Workflow, 0, len(ce.workflows))
	for _, workflow := range ce.workflows {
		workflows = append(workflows, workflow)
	}

	return workflows, nil
}

// GetWorkflowsByStatus يرجع الورك فلو حسب الحالة
func (ce *CollaborationEngine) GetWorkflowsByStatus(ctx context.Context, status string) ([]*Workflow, error) {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	var result []*Workflow
	for _, workflow := range ce.workflows {
		if workflow.Status == status {
			result = append(result, workflow)
		}
	}

	return result, nil
}

// GetStepsAssignedToAgent يرجع الخطوات المخصصة لوكيل
func (ce *CollaborationEngine) GetStepsAssignedToAgent(ctx context.Context, agentID string) ([]*WorkflowStep, error) {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	var result []*WorkflowStep
	for _, workflow := range ce.workflows {
		for _, step := range workflow.Steps {
			if step.AssignedTo == agentID {
				result = append(result, step)
			}
		}
	}

	return result, nil
}

// GetPendingStepsForAgent يرجع الخطوات المعلقة لوكيل
func (ce *CollaborationEngine) GetPendingStepsForAgent(ctx context.Context, agentID string) ([]*WorkflowStep, error) {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	var result []*WorkflowStep
	for _, workflow := range ce.workflows {
		for _, step := range workflow.Steps {
			if step.Status == "pending" && step.AssignedTo == agentID {
				// التحقق من الاعتمادات
				dependenciesMet := true
				for _, depID := range step.Dependencies {
					depMet := false
					for _, s := range workflow.Steps {
						if s.ID == depID && s.Status == "completed" {
							depMet = true
							break
						}
					}
					if !depMet {
						dependenciesMet = false
						break
					}
				}

				if dependenciesMet {
					result = append(result, step)
				}
			}
		}
	}

	return result, nil
}

// GetWorkflowProgress يحسب تقدم الورك فلو
func (ce *CollaborationEngine) GetWorkflowProgress(ctx context.Context, workflowID string) (map[string]interface{}, error) {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	workflow, ok := ce.workflows[workflowID]
	if !ok {
		return nil, fmt.Errorf("ورك فلو غير موجود: %s", workflowID)
	}

	total := len(workflow.Steps)
	completed := 0
	failed := 0
	inProgress := 0
	pending := 0

	for _, step := range workflow.Steps {
		switch step.Status {
		case "completed":
			completed++
		case "failed":
			failed++
		case "in_progress":
			inProgress++
		case "pending":
			pending++
		}
	}

	progress := map[string]interface{}{
		"total":       total,
		"completed":   completed,
		"failed":      failed,
		"in_progress": inProgress,
		"pending":     pending,
		"percentage":  0.0,
	}

	if total > 0 {
		progress["percentage"] = float64(completed) / float64(total) * 100
	}

	return progress, nil
}

// GetCollaborationSummary يرجع ملخص التعاون
func (ce *CollaborationEngine) GetCollaborationSummary(ctx context.Context) (map[string]interface{}, error) {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	// حساب إحصائيات التعاون
	agentSteps := make(map[string]int)
	for _, workflow := range ce.workflows {
		for _, step := range workflow.Steps {
			agentSteps[step.AssignedTo]++
		}
	}

	summary := map[string]interface{}{
		"session_id":       ce.sessionID,
		"agent_id":         ce.agentID,
		"total_workflows":  len(ce.workflows),
		"agent_steps":      agentSteps,
		"completed":        0,
		"in_progress":      0,
		"pending":          0,
	}

	for _, workflow := range ce.workflows {
		switch workflow.Status {
		case "completed":
			summary["completed"] = summary["completed"].(int) + 1
		case "in_progress":
			summary["in_progress"] = summary["in_progress"].(int) + 1
		case "pending":
			summary["pending"] = summary["pending"].(int) + 1
		}
	}

	return summary, nil
}
