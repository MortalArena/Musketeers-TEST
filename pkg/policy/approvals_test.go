package policy

import "testing"

func TestApprovalEngineRequestApproveDeny(t *testing.T) {
	engine := NewApprovalEngine()
	request := ApprovalRequest{ID: "approval-1", Actor: "did:ia:user", Action: "capability.execute", Resource: "github:issue:create"}
	if err := engine.Request(request, 1); err != nil {
		t.Fatalf("Request returned error: %v", err)
	}
	if err := engine.Approve("approval-1", "operator", "ok", 1); err != nil {
		t.Fatalf("Approve returned error: %v", err)
	}
	status, err := engine.Status("approval-1")
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.State != ApprovalStateApproved {
		t.Fatalf("expected approved, got %#v", status)
	}
}

func TestApprovalEngineDeny(t *testing.T) {
	engine := NewApprovalEngine()
	if err := engine.Request(ApprovalRequest{ID: "approval-2", Actor: "did:ia:user", Action: "capability.execute", Resource: "github:issue:create"}, 1); err != nil {
		t.Fatalf("Request returned error: %v", err)
	}
	if err := engine.Deny("approval-2", "operator", "blocked", 1); err != nil {
		t.Fatalf("Deny returned error: %v", err)
	}
	status, err := engine.Status("approval-2")
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.State != ApprovalStateDenied {
		t.Fatalf("expected denied, got %#v", status)
	}
}

func TestApprovalEngineRejectsUnknownApproval(t *testing.T) {
	engine := NewApprovalEngine()
	if err := engine.Approve("missing", "operator", "ok", 1); err == nil {
		t.Fatal("expected unknown approval error")
	}
}
