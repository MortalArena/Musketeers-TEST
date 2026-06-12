package policy

import (
	"fmt"
	"sync"
	"time"
)

type ApprovalState string

const (
	ApprovalStatePending   ApprovalState = "pending"
	ApprovalStateApproved  ApprovalState = "approved"
	ApprovalStateDenied    ApprovalState = "denied"
	ApprovalStateCancelled ApprovalState = "cancelled"
)

type ApprovalRequest struct {
	ID        string         `json:"id"`
	Actor     string         `json:"actor"`
	Action    string         `json:"action"`
	Resource  string         `json:"resource"`
	Context   map[string]any `json:"context,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type ApprovalStatus struct {
	Request   ApprovalRequest `json:"request"`
	State     ApprovalState   `json:"state"`
	Decision  string          `json:"decision,omitempty"`
	DecidedBy string          `json:"decided_by,omitempty"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type ApprovalEngine struct {
	mu        sync.RWMutex
	approvals map[string]ApprovalStatus
}

func NewApprovalEngine() *ApprovalEngine {
	return &ApprovalEngine{approvals: make(map[string]ApprovalStatus)}
}

func (e *ApprovalEngine) Request(request ApprovalRequest) error {
	if request.ID == "" {
		return fmt.Errorf("approval id is required")
	}
	if request.Actor == "" {
		return fmt.Errorf("approval actor is required")
	}
	if request.Action == "" || request.Resource == "" {
		return fmt.Errorf("approval action and resource are required")
	}
	if request.Context == nil {
		request.Context = map[string]any{}
	}
	if request.CreatedAt.IsZero() {
		request.CreatedAt = time.Now().UTC()
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, exists := e.approvals[request.ID]; exists {
		return fmt.Errorf("approval already exists: %s", request.ID)
	}
	e.approvals[request.ID] = ApprovalStatus{
		Request:   request,
		State:     ApprovalStatePending,
		UpdatedAt: request.CreatedAt,
	}
	return nil
}

func (e *ApprovalEngine) Approve(id, approver, decision string) error {
	return e.decide(id, approver, decision, ApprovalStateApproved)
}

func (e *ApprovalEngine) Deny(id, approver, decision string) error {
	return e.decide(id, approver, decision, ApprovalStateDenied)
}

func (e *ApprovalEngine) Cancel(id, reason string) error {
	return e.decide(id, "system", reason, ApprovalStateCancelled)
}

func (e *ApprovalEngine) Status(id string) (ApprovalStatus, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	status, exists := e.approvals[id]
	if !exists {
		return ApprovalStatus{}, fmt.Errorf("approval not found: %s", id)
	}
	return status, nil
}

func (e *ApprovalEngine) decide(id, approver, decision string, state ApprovalState) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	status, exists := e.approvals[id]
	if !exists {
		return fmt.Errorf("approval not found: %s", id)
	}
	if status.State != ApprovalStatePending {
		return fmt.Errorf("approval is not pending: %s", id)
	}
	status.State = state
	status.Decision = decision
	status.DecidedBy = approver
	status.UpdatedAt = time.Now().UTC()
	e.approvals[id] = status
	return nil
}
