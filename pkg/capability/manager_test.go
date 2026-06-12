package capability

import (
	"context"
	"testing"

	"github.com/MortalArena/Musketeers/pkg/policy"
)

func TestManagerRegisterAndExecute(t *testing.T) {
	manager := NewManager(policy.NewEngine())
	if err := manager.Register(&testCapability{name: "echo"}); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	result, err := manager.Execute(context.Background(), policy.Principal{DID: "did:ia:test"}, testCommand{name: "echo", args: map[string]any{"value": "ok"}})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Output["principal"] != "did:ia:test" || result.Output["value"] != "ok" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestManagerRejectsUnknownCapability(t *testing.T) {
	manager := NewManager(policy.NewEngine())
	_, err := manager.Execute(context.Background(), policy.Principal{DID: "did:ia:test"}, testCommand{name: "missing"})
	if err == nil {
		t.Fatal("expected unknown capability error")
	}
}

func TestManagerRejectsDuplicateCapability(t *testing.T) {
	manager := NewManager(policy.NewEngine())
	cap := &testCapability{name: "dup"}
	if err := manager.Register(cap); err != nil {
		t.Fatalf("first Register returned error: %v", err)
	}
	if err := manager.Register(cap); err == nil {
		t.Fatal("expected duplicate capability error")
	}
}
