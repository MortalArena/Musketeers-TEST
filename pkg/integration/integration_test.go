package integration

import (
	"context"
	"testing"
	"time"

	"github.com/MortalArena/Musketeers/pkg/capability"
	"github.com/MortalArena/Musketeers/pkg/policy"
	"github.com/MortalArena/Musketeers/pkg/registry"
	"github.com/MortalArena/Musketeers/pkg/runtime/events"
	"github.com/MortalArena/Musketeers/pkg/runtime/observability"
	"github.com/MortalArena/Musketeers/pkg/runtime/state"
	"github.com/MortalArena/Musketeers/pkg/vault"
	"github.com/MortalArena/Musketeers/pkg/vault/keyprovider"
	"github.com/MortalArena/Musketeers/pkg/workflow"
)

type integrationCommand struct{}

func (integrationCommand) Name() string         { return "noop" }
func (integrationCommand) Args() map[string]any { return map[string]any{} }

func TestRuntimePolicyVaultCapabilityWorkflowIntegration(t *testing.T) {
	bus := events.NewMemoryEventBus(10)
	defer bus.Close()
	logger, err := observability.NewZapLogger("debug")
	if err != nil {
		t.Fatalf("NewZapLogger returned error: %v", err)
	}
	policyEngine := policy.NewEngine()
	if err := policyEngine.AddRule(policy.Rule{Name: "allow-noop", Effect: policy.EffectAllow, Principals: []policy.Principal{{DID: "did:ia:test"}}, Resources: []policy.Resource{{Type: "capability", Action: "noop"}}}); err != nil {
		t.Fatalf("AddRule returned error: %v", err)
	}
	manager := capability.NewManager(policyEngine)
	if err := manager.Register(&noopCapability{}); err != nil {
		t.Fatalf("Register capability returned error: %v", err)
	}
	executor := func(ctx context.Context, principal policy.Principal, cmd capability.Command) (*capability.Result, error) {
		return manager.Execute(ctx, principal, cmd)
	}
	engine := workflow.NewDefaultWorkflowEngine(executor)
	if err := engine.Register(workflow.Workflow{Name: "integration", Steps: []workflow.Step{{Name: "noop", Type: workflow.StepCapability, Capability: "noop"}}}); err != nil {
		t.Fatalf("Register workflow returned error: %v", err)
	}
	_, err = engine.Execute(context.Background(), policy.Principal{DID: "did:ia:test"}, "integration", map[string]any{})
	if err != nil {
		t.Fatalf("workflow integration failed: %v", err)
	}
	reg := registry.NewMemoryRegistry()
	if err := reg.Register(registry.AgentManifest{ID: "agent", DID: "did:ia:test", Name: "test"}); err != nil {
		t.Fatalf("Register manifest returned error: %v", err)
	}
	if _, err := reg.Get("agent"); err != nil {
		t.Fatalf("Get manifest returned error: %v", err)
	}
	v := vault.New(keyprovider.NewFileKeyProvider(t.TempDir()))
	if err := v.Store("token", []byte("secret"), nil); err != nil {
		t.Fatalf("Vault Store returned error: %v", err)
	}
	secret, err := v.Retrieve("token")
	if err != nil {
		t.Fatalf("Vault Retrieve returned error: %v", err)
	}
	if string(secret) != "secret" {
		t.Fatalf("unexpected secret: %s", secret)
	}
	store := state.NewMemoryStateStore()
	if err := store.Set("runtime", []byte("ok")); err != nil {
		t.Fatalf("State Set returned error: %v", err)
	}
	_ = logger
}

type noopCapability struct{}

func (noopCapability) Name() string { return "noop" }
func (noopCapability) Execute(ctx context.Context, principal policy.Principal, cmd capability.Command) (*capability.Result, error) {
	return capability.NewResult(cmd.Name(), map[string]any{"did": principal.DID}), nil
}

func BenchmarkMemoryEventBusThroughput(b *testing.B) {
	bus := events.NewMemoryEventBus(100000)
	defer bus.Close()
	for i := 0; i < b.N; i++ {
		if err := bus.Publish(events.Event{Type: events.EventTaskCompleted, Source: "benchmark", Timestamp: time.Now()}); err != nil {
			b.Fatalf("Publish returned error: %v", err)
		}
	}
}
