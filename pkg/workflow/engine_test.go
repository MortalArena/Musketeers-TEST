package workflow

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/MortalArena/Musketeers/pkg/capability"
	"github.com/MortalArena/Musketeers/pkg/policy"
)

type workflowCommand struct{}

func (workflowCommand) Name() string         { return "noop" }
func (workflowCommand) Args() map[string]any { return map[string]any{} }

func TestDefaultWorkflowEngineExecuteSteps(t *testing.T) {
	var calls int32
	engine := NewDefaultWorkflowEngine(func(ctx context.Context, principal policy.Principal, cmd capability.Command) (*capability.Result, error) {
		atomic.AddInt32(&calls, 1)
		return capability.NewResult(cmd.Name(), nil), nil
	})
	wf := Workflow{Name: "wf", Steps: []Step{{Name: "step1", Type: StepCapability, Capability: "noop"}}}
	if err := engine.Register(wf); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	execution, err := engine.Execute(context.Background(), policy.Principal{DID: "did:ia:test"}, "wf", map[string]any{})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if execution.State != ExecutionStateCompleted || atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("unexpected execution: %#v calls=%d", execution, calls)
	}
}

func TestWorkflowEngineCancel(t *testing.T) {
	engine := NewDefaultWorkflowEngine(func(ctx context.Context, principal policy.Principal, cmd capability.Command) (*capability.Result, error) {
		return capability.NewResult(cmd.Name(), nil), nil
	})
	wf := Workflow{Name: "wf", Steps: []Step{{Name: "step1", Type: StepCapability, Capability: "noop"}}}
	if err := engine.Register(wf); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := engine.Execute(ctx, policy.Principal{DID: "did:ia:test"}, "wf", map[string]any{})
	if err == nil {
		t.Fatal("expected cancellation error")
	}
}
