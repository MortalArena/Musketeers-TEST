package runtime

import (
	"context"
	"testing"

	"github.com/MortalArena/Musketeers/pkg/runtime/events"
	"github.com/MortalArena/Musketeers/pkg/runtime/knowledge"
	"github.com/MortalArena/Musketeers/pkg/runtime/observability"
	"github.com/MortalArena/Musketeers/pkg/runtime/scheduler"
	"github.com/MortalArena/Musketeers/pkg/runtime/state"
)

type noopNotifier struct{}

func (noopNotifier) Notify(string, string) error { return nil }

func TestAgentRuntimeStartStopAndContext(t *testing.T) {
	bus := events.NewMemoryEventBus(10)
	defer bus.Close()
	logger, err := observability.NewZapLogger("debug")
	if err != nil {
		t.Fatalf("NewZapLogger returned error: %v", err)
	}
	rt := NewAgentRuntime(
		bus,
		state.NewMemoryStateStore(),
		knowledge.NewDefaultKnowledgeStore(),
		scheduler.NewEventDrivenScheduler(bus),
		logger,
		observability.NewOTelTracer("Musketeers-test"),
		observability.NewPrometheusMetrics(),
		observability.NewMemoryAuditLog(),
	)
	if err := rt.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if err := rt.Stop(); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}

	ctx := NewAgentContext("did:ia:test", &AgentMetadata{Name: "test"}, rt, nil, noopNotifier{}, context.Background())
	if ctx.DID() != "did:ia:test" {
		t.Fatalf("unexpected DID: %s", ctx.DID())
	}
	if err := ctx.Notify("title", "message"); err != nil {
		t.Fatalf("Notify returned error: %v", err)
	}
}
