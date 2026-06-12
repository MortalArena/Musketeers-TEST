package registry

import "testing"

func TestMemoryRegistryCRUD(t *testing.T) {
	reg := NewMemoryRegistry()
	manifest := AgentManifest{ID: "agent-1", DID: "did:ia:agent", Name: "agent", Version: "1.0.0"}
	if err := reg.Register(manifest); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	got, err := reg.Get("agent-1")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.Name != "agent" {
		t.Fatalf("unexpected manifest: %#v", got)
	}
	manifest.Version = "1.0.1"
	if err := reg.Update(manifest); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	got, err = reg.Get("agent-1")
	if err != nil {
		t.Fatalf("Get after update returned error: %v", err)
	}
	if got.Version != "1.0.1" {
		t.Fatalf("unexpected version: %s", got.Version)
	}
	agents, err := reg.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("unexpected list: %#v", agents)
	}
	if err := reg.Unregister("agent-1"); err != nil {
		t.Fatalf("Unregister returned error: %v", err)
	}
	if _, err := reg.Get("agent-1"); err == nil {
		t.Fatal("expected missing agent after unregister")
	}
}

func TestMemoryRegistryRejectsDuplicate(t *testing.T) {
	reg := NewMemoryRegistry()
	manifest := AgentManifest{ID: "agent-1", DID: "did:ia:agent", Name: "agent"}
	if err := reg.Register(manifest); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if err := reg.Register(manifest); err == nil {
		t.Fatal("expected duplicate error")
	}
}
