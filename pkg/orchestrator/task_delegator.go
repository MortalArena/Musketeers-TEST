package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/agent/unified"
	"go.uber.org/zap"
)

type DelegationStatus string

const (
	DelegationPending   DelegationStatus = "pending"
	DelegationRunning   DelegationStatus = "running"
	DelegationCompleted DelegationStatus = "completed"
	DelegationFailed    DelegationStatus = "failed"
)

type AgentDelegation struct {
	TaskID     string            `json:"task_id"`
	AgentID    string            `json:"agent_id"`
	Status     DelegationStatus  `json:"status"`
	DelegatedAt time.Time        `json:"delegated_at"`
	CompletedAt *time.Time       `json:"completed_at,omitempty"`
	Error      string            `json:"error,omitempty"`
}

type TaskDelegator struct {
	mu         sync.RWMutex
	agentPool  *unified.AgentPool
	eventBus   *unified.SessionEventBus
	delegations map[string]*AgentDelegation
	logger     *zap.Logger
}

func NewTaskDelegator(agentPool *unified.AgentPool, eventBus *unified.SessionEventBus, logger *zap.Logger) *TaskDelegator {
	return &TaskDelegator{
		agentPool:   agentPool,
		eventBus:    eventBus,
		delegations: make(map[string]*AgentDelegation),
		logger:      logger,
	}
}

func (td *TaskDelegator) SetAgentPool(ap *unified.AgentPool) {
	td.mu.Lock()
	defer td.mu.Unlock()
	td.agentPool = ap
}

func (td *TaskDelegator) SetEventBus(eb *unified.SessionEventBus) {
	td.mu.Lock()
	defer td.mu.Unlock()
	td.eventBus = eb
}

func (td *TaskDelegator) SelectAgent(task *agent.AgentTask) (string, error) {
	td.mu.RLock()
	ap := td.agentPool
	td.mu.RUnlock()

	if ap == nil {
		return "", fmt.Errorf("agent pool not available")
	}

	agents := ap.ListAgents()
	if len(agents) == 0 {
		return "", fmt.Errorf("no agents available in pool")
	}

	if task.Inputs != nil {
		if assigned, ok := task.Inputs["assigned_agent"].(string); ok && assigned != "" {
			for _, inst := range agents {
				if inst.AgentID == assigned && inst.GetStatus() != unified.PoolAgentStatusError {
					return inst.AgentID, nil
				}
			}
		}
	}

	modelAgents := make([]*unified.AgentInstance, 0)
	externalAgents := make([]*unified.AgentInstance, 0)

	for _, inst := range agents {
		status := inst.GetStatus()
		if status == unified.PoolAgentStatusError || status == unified.PoolAgentStatusParked {
			continue
		}
	// --- الرجاء تنفيذ التنفيذ في الخارجي أو النموذج
		if unified.IsExternalAgentType(inst.AgentType) {
			externalAgents = append(externalAgents, inst)
		} else {
			modelAgents = append(modelAgents, inst)
		}
	}

	if len(modelAgents) > 0 {
		for _, inst := range modelAgents {
			if inst.GetStatus() == unified.PoolAgentStatusActive {
				return inst.AgentID, nil
			}
		}
		return modelAgents[0].AgentID, nil
	}

	if len(externalAgents) > 0 {
		return externalAgents[0].AgentID, nil
	}

	return "", fmt.Errorf("no available agents for task")
}

func (td *TaskDelegator) DelegateTask(ctx context.Context, task *agent.AgentTask, agentID string, oe *OrchestratorEngine) (*agent.TaskExecutionResult, error) {
	td.mu.Lock()
	del := &AgentDelegation{
		TaskID:      task.ID,
		AgentID:     agentID,
		Status:      DelegationRunning,
		DelegatedAt: time.Now(),
	}
	td.delegations[task.ID] = del
	td.mu.Unlock()

	defer func() {
		td.mu.Lock()
		if del.Status == DelegationRunning {
			del.Status = DelegationFailed
			del.Error = "execution terminated"
		}
		td.mu.Unlock()
	}()

	td.publishTaskEvent(unified.TaskAssigned, task, agentID)

	td.publishTaskEvent(unified.TaskStarted, task, agentID)

	ap := td.agentPool
	var result *agent.TaskExecutionResult
	var err error

	if ap != nil {
		inst, getErr := ap.GetAgent(agentID)
		if getErr == nil && inst != nil {
			if unified.IsExternalAgentType(inst.AgentType) {
				result, err = td.executeOnExternalAgent(ctx, inst, task)
			} else {
				result, err = oe.executeTaskViaThinkingEngine(ctx, ap, agentID, task)
			}
		} else {
			result, err = oe.executeTaskViaThinkingEngine(ctx, ap, agentID, task)
		}
	} else {
		err = fmt.Errorf("agent pool not available")
	}

	td.mu.Lock()
	if err != nil {
		del.Status = DelegationFailed
		del.Error = err.Error()
		del.CompletedAt = timePtr(time.Now())
		td.mu.Unlock()
		td.publishTaskEvent(unified.TaskFailed, task, agentID)
		return nil, err
	}

	del.Status = DelegationCompleted
	del.CompletedAt = timePtr(time.Now())
	td.mu.Unlock()

	td.publishTaskEvent(unified.TaskCompleted, task, agentID)

	return result, nil
}

func (td *TaskDelegator) executeOnExternalAgent(ctx context.Context, inst *unified.AgentInstance, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	td.logger.Info("Executing task on external agent via adapter",
		zap.String("agent_id", inst.AgentID),
		zap.String("type", inst.AgentType),
		zap.String("task", task.Title))

	result, err := inst.Adapter.ExecuteTask(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("external agent execution failed: %w", err)
	}
	return result, nil
}

func (td *TaskDelegator) publishTaskEvent(eventType unified.SessionEventType, task *agent.AgentTask, agentID string) {
	td.mu.RLock()
	eb := td.eventBus
	td.mu.RUnlock()

	if eb == nil {
		return
	}

	data := map[string]interface{}{
		"task_id":    task.ID,
		"task_title": task.Title,
		"agent_id":   agentID,
		"timestamp":  time.Now(),
	}

	event := &unified.SessionEvent{
		ID:          fmt.Sprintf("task-%s-%d", task.ID, time.Now().UnixNano()),
		SessionID:   "",
		SourceAgent: agentID,
		TargetAgent: "",
		EventType:   eventType,
		Timestamp:   time.Now(),
		Priority:    unified.PriorityHigh,
		Data:        data,
		Metadata:    map[string]interface{}{"source": "task_delegator"},
	}

	_ = eb.PublishEvent(context.Background(), event)
}

func (td *TaskDelegator) GetDelegation(taskID string) (*AgentDelegation, bool) {
	td.mu.RLock()
	defer td.mu.RUnlock()
	del, ok := td.delegations[taskID]
	return del, ok
}

func (td *TaskDelegator) GetDelegationsByAgent(agentID string) []*AgentDelegation {
	td.mu.RLock()
	defer td.mu.RUnlock()

	result := make([]*AgentDelegation, 0)
	for _, del := range td.delegations {
		if del.AgentID == agentID {
			result = append(result, del)
		}
	}
	return result
}

func (td *TaskDelegator) GetAllDelegations() []*AgentDelegation {
	td.mu.RLock()
	defer td.mu.RUnlock()

	result := make([]*AgentDelegation, 0, len(td.delegations))
	for _, del := range td.delegations {
		result = append(result, del)
	}
	return result
}

func timePtr(t time.Time) *time.Time {
	return &t
}
