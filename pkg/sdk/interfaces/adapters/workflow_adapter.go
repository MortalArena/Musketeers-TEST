package adapters

import (
	"context"

	"github.com/MortalArena/Musketeers/pkg/policy"
	"github.com/MortalArena/Musketeers/pkg/sdk/interfaces"
	"github.com/MortalArena/Musketeers/pkg/workflow"
)

type WorkflowAdapter struct {
	engine workflow.WorkflowEngine
}

func NewWorkflowAdapter(engine workflow.WorkflowEngine) *WorkflowAdapter {
	return &WorkflowAdapter{engine: engine}
}

func (a *WorkflowAdapter) Register(name string, wf *interfaces.WorkflowDef) error {
	steps := make([]workflow.Step, len(wf.Steps))
	for i, s := range wf.Steps {
		steps[i] = workflow.Step{
			Name:       s.Name,
			Type:       workflow.StepType(s.Type),
			Capability: s.Capability,
			Command:    s.Input,
		}
	}
	return a.engine.Register(workflow.Workflow{
		Name:        wf.Name,
		Description: wf.Description,
		Steps:       steps,
	})
}

func (a *WorkflowAdapter) Execute(ctx context.Context, workflowName string, input map[string]interface{}) (*interfaces.WorkflowExecution, error) {
	exec, err := a.engine.Execute(ctx, policy.Principal{DID: "system"}, workflowName, input)
	if err != nil {
		return nil, err
	}
	return &interfaces.WorkflowExecution{
		ID:        exec.ID,
		Workflow:  exec.Workflow,
		State:     string(exec.State),
		StartedAt: exec.StartedAt,
		EndedAt:   exec.EndedAt,
		Output:    exec.Output,
		Error:     exec.Error,
	}, nil
}

func (a *WorkflowAdapter) CancelExecution(id string) error {
	return a.engine.CancelExecution(id)
}

var _ interfaces.WorkflowInterface = (*WorkflowAdapter)(nil)
