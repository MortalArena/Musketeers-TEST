package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/MortalArena/Musketeers/pkg/capability"
	"github.com/MortalArena/Musketeers/pkg/policy"
	"github.com/MortalArena/Musketeers/pkg/runtime/observability"
)

type simpleCommand struct{}

func (simpleCommand) Name() string         { return "noop" }
func (simpleCommand) Args() map[string]any { return map[string]any{} }

func TestPipelineRunsMiddlewaresAndExecutor(t *testing.T) {
	var calls []string
	p := New(
		MiddlewareFunc(func(ctx context.Context, principal policy.Principal, cmd capability.Command, next Executor) (*capability.Result, error) {
			calls = append(calls, "before")
			result, err := next(ctx, principal, cmd)
			calls = append(calls, "after")
			return result, err
		}),
	)
	result, err := p.Execute(context.Background(), policy.Principal{DID: "did:ia:test"}, simpleCommand{}, func(ctx context.Context, principal policy.Principal, cmd capability.Command) (*capability.Result, error) {
		calls = append(calls, "execute")
		return &capability.Result{Name: cmd.Name(), Output: map[string]any{"did": principal.DID}}, nil
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Output["did"] != "did:ia:test" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if len(calls) != 3 || calls[0] != "before" || calls[1] != "execute" || calls[2] != "after" {
		t.Fatalf("unexpected calls: %v", calls)
	}
}

func TestPipelineStopsOnMiddlewareError(t *testing.T) {
	p := New(MiddlewareFunc(func(context.Context, policy.Principal, capability.Command, Executor) (*capability.Result, error) {
		return nil, errors.New("blocked")
	}))
	_, err := p.Execute(context.Background(), policy.Principal{DID: "did:ia:test"}, simpleCommand{}, func(context.Context, policy.Principal, capability.Command) (*capability.Result, error) {
		return &capability.Result{Name: "noop"}, nil
	})
	if err == nil {
		t.Fatal("expected middleware error")
	}
}

func TestAuthorizationMiddleware(t *testing.T) {
	engine := policy.NewEngine()
	if err := engine.AddRule(policy.Rule{Name: "allow", Effect: policy.EffectAllow, Principals: []policy.Principal{{DID: "did:ia:test"}}, Resources: []policy.Resource{{Type: "capability", Action: "noop"}}}); err != nil {
		t.Fatalf("AddRule returned error: %v", err)
	}
	p := New(AuthorizationMiddleware(engine))
	_, err := p.Execute(context.Background(), policy.Principal{DID: "did:ia:other"}, simpleCommand{}, func(context.Context, policy.Principal, capability.Command) (*capability.Result, error) {
		return &capability.Result{Name: "noop"}, nil
	})
	if err == nil {
		t.Fatal("expected authorization error")
	}
}

func TestAuditMiddleware(t *testing.T) {
	audit := observability.NewMemoryAuditLog()
	p := New(AuditMiddleware(audit, "did:ia:test"))
	_, err := p.Execute(context.Background(), policy.Principal{DID: "did:ia:test"}, simpleCommand{}, func(context.Context, policy.Principal, capability.Command) (*capability.Result, error) {
		return &capability.Result{Name: "noop"}, nil
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	entries, err := audit.Query(observability.AuditFilter{Action: "capability.noop"})
	if err != nil {
		t.Fatalf("Audit Query returned error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one audit entry, got %d", len(entries))
	}
}
