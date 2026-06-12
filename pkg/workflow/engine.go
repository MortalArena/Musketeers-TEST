package workflow

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/capability"
	"github.com/MortalArena/Musketeers/pkg/policy"
)

type WorkflowEngine interface {
	Register(workflow Workflow) error
	Execute(ctx context.Context, principal policy.Principal, workflowName string, input map[string]any) (*Execution, error)
	GetExecution(id string) (Execution, error)
	CancelExecution(id string) error
}

type CommandBuilder func(name string, args map[string]any) capability.Command

type DefaultWorkflowEngine struct {
	mu             sync.RWMutex
	workflows      map[string]Workflow
	executions     map[string]*Execution
	cancels        map[string]context.CancelFunc
	execute        capabilityExecutor
	commandBuilder CommandBuilder
}

type capabilityExecutor func(ctx context.Context, principal policy.Principal, cmd capability.Command) (*capability.Result, error)

func NewDefaultWorkflowEngine(execute capabilityExecutor) *DefaultWorkflowEngine {
	return &DefaultWorkflowEngine{
		workflows:      make(map[string]Workflow),
		executions:     make(map[string]*Execution),
		cancels:        make(map[string]context.CancelFunc),
		execute:        execute,
		commandBuilder: defaultCommandBuilder,
	}
}

func (e *DefaultWorkflowEngine) SetCommandBuilder(builder CommandBuilder) {
	if builder != nil {
		e.commandBuilder = builder
	}
}

func (e *DefaultWorkflowEngine) Register(workflow Workflow) error {
	workflow = workflow.Normalize()
	if workflow.Name == "" || len(workflow.Steps) == 0 {
		return fmt.Errorf("workflow name and at least one step are required")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.workflows[workflow.Name] = workflow
	return nil
}

func (e *DefaultWorkflowEngine) Execute(ctx context.Context, principal policy.Principal, workflowName string, input map[string]any) (*Execution, error) {
	if input == nil {
		input = map[string]any{}
	}
	e.mu.RLock()
	workflow, exists := e.workflows[workflowName]
	e.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("workflow not found: %s", workflowName)
	}
	if e.execute == nil {
		return nil, fmt.Errorf("workflow executor is not configured")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	execCtx, cancel := context.WithCancel(ctx)
	execution := &Execution{ID: fmt.Sprintf("exec-%d", time.Now().UnixNano()), Workflow: workflow.Name, State: ExecutionStateRunning, StartedAt: time.Now().UTC(), Output: map[string]any{}}
	e.mu.Lock()
	e.executions[execution.ID] = execution
	e.cancels[execution.ID] = cancel
	e.mu.Unlock()
	defer cancel()
	for _, step := range workflow.Steps {
		stepExec := StepExecution{Name: step.Name, State: ExecutionStateRunning, StartedAt: time.Now().UTC()}
		if err := e.executeStep(execCtx, principal, input, step, &stepExec); err != nil {
			stepExec.State = ExecutionStateFailed
			stepExec.Error = err.Error()
			execution.Steps = append(execution.Steps, stepExec)
			execution.State = ExecutionStateFailed
			execution.Error = err.Error()
			e.finishExecution(execution.ID, execution)
			return execution, err
		}
		stepExec.State = ExecutionStateCompleted
		stepExec.EndedAt = time.Now().UTC()
		execution.Steps = append(execution.Steps, stepExec)
	}
	execution.State = ExecutionStateCompleted
	execution.EndedAt = time.Now().UTC()
	e.finishExecution(execution.ID, execution)
	return execution, nil
}

func (e *DefaultWorkflowEngine) GetExecution(id string) (Execution, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	execution, exists := e.executions[id]
	if !exists {
		return Execution{}, fmt.Errorf("execution not found: %s", id)
	}
	return *execution, nil
}

func (e *DefaultWorkflowEngine) CancelExecution(id string) error {
	e.mu.RLock()
	cancel, exists := e.cancels[id]
	e.mu.RUnlock()
	if !exists {
		return fmt.Errorf("execution not found: %s", id)
	}
	cancel()
	e.mu.Lock()
	if execution, exists := e.executions[id]; exists {
		execution.State = ExecutionStateCancelled
		execution.EndedAt = time.Now().UTC()
	}
	e.mu.Unlock()
	return nil
}

func (e *DefaultWorkflowEngine) executeStep(ctx context.Context, principal policy.Principal, input map[string]any, step Step, stepExec *StepExecution) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if step.Condition != nil && !evaluateCondition(input, *step.Condition) {
		return nil
	}
	count := 1
	if step.Loop != nil && step.Loop.Count > 0 {
		count = step.Loop.Count
	}
	for i := 0; i < count; i++ {
		switch step.Type {
		case StepCapability:
			cmd := e.commandBuilder(step.Capability, mergeMaps(input, step.Command))
			if cmd == nil {
				return fmt.Errorf("command is nil for step %s", step.Name)
			}
			result, err := e.execute(ctx, principal, cmd)
			if err != nil {
				return err
			}
			stepExec.Output = result.Output
		case StepDelay:
			duration, err := time.ParseDuration(step.Delay)
			if err != nil {
				return err
			}
			timer := time.NewTimer(duration)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return ctx.Err()
			case <-timer.C:
			}
		case StepCondition:
			if step.Condition == nil || !evaluateCondition(input, *step.Condition) {
				return fmt.Errorf("condition failed for step %s", step.Name)
			}
		default:
			return fmt.Errorf("unsupported step type: %s", step.Type)
		}
	}
	return nil
}

func (e *DefaultWorkflowEngine) finishExecution(id string, execution *Execution) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.cancels, id)
}

func defaultCommandBuilder(name string, args map[string]any) capability.Command {
	return mapCommand{name: name, args: args}
}

type mapCommand struct {
	name string
	args map[string]any
}

func (c mapCommand) Name() string         { return c.name }
func (c mapCommand) Args() map[string]any { return c.args }

func evaluateCondition(input map[string]any, condition Condition) bool {
	value := input[condition.Field]
	switch condition.Op {
	case "eq", "==":
		return fmt.Sprint(value) == fmt.Sprint(condition.Value)
	case "ne", "!=":
		return fmt.Sprint(value) != fmt.Sprint(condition.Value)
	case "contains":
		return strings.Contains(fmt.Sprint(value), fmt.Sprint(condition.Value))
	case "gt":
		return asFloat(value) > asFloat(condition.Value)
	case "gte":
		return asFloat(value) >= asFloat(condition.Value)
	case "lt":
		return asFloat(value) < asFloat(condition.Value)
	case "lte":
		return asFloat(value) <= asFloat(condition.Value)
	default:
		return false
	}
}

func asFloat(value any) float64 {
	switch v := value.(type) {
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case float64:
		return v
	case string:
		parsed, _ := strconv.ParseFloat(v, 64)
		return parsed
	default:
		return 0
	}
}

func mergeMaps(base, override map[string]any) map[string]any {
	merged := make(map[string]any, len(base)+len(override))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range override {
		merged[k] = v
	}
	return merged
}
