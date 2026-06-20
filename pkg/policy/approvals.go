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
	// [SAFETY] Multi-level approval support
	RequiredLevel int                     `json:"required_level"`
	CurrentLevel  int                     `json:"current_level"`
	Approvals     map[string]ApprovalInfo `json:"approvals"` // approver -> info
}

type ApprovalInfo struct {
	Approver  string    `json:"approver"`
	Level     int       `json:"level"`
	Decision  string    `json:"decision"`
	Timestamp time.Time `json:"timestamp"`
}

type ApprovalEngine struct {
	mu        sync.RWMutex
	approvals map[string]ApprovalStatus
}

func NewApprovalEngine() *ApprovalEngine {
	return &ApprovalEngine{approvals: make(map[string]ApprovalStatus)}
}

func (e *ApprovalEngine) Request(request ApprovalRequest, requiredLevel int) error {
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
	// [SAFETY] Default to single-level approval if not specified
	if requiredLevel < 1 {
		requiredLevel = 1
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, exists := e.approvals[request.ID]; exists {
		return fmt.Errorf("approval already exists: %s", request.ID)
	}
	e.approvals[request.ID] = ApprovalStatus{
		Request:       request,
		State:         ApprovalStatePending,
		UpdatedAt:     request.CreatedAt,
		RequiredLevel: requiredLevel,
		CurrentLevel:  0,
		Approvals:     make(map[string]ApprovalInfo),
	}
	return nil
}

func (e *ApprovalEngine) Approve(id, approver, decision string, level int) error {
	return e.decide(id, approver, decision, ApprovalStateApproved, level)
}

func (e *ApprovalEngine) Deny(id, approver, decision string, level int) error {
	return e.decide(id, approver, decision, ApprovalStateDenied, level)
}

func (e *ApprovalEngine) Cancel(id, reason string) error {
	return e.decide(id, "system", reason, ApprovalStateCancelled, 0)
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

func (e *ApprovalEngine) decide(id, approver, decision string, state ApprovalState, level int) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	status, exists := e.approvals[id]
	if !exists {
		return fmt.Errorf("approval not found: %s", id)
	}
	if status.State != ApprovalStatePending {
		return fmt.Errorf("approval is not pending: %s", id)
	}

	// [SAFETY] Record approval at the specified level
	if state == ApprovalStateApproved || state == ApprovalStateDenied {
		status.Approvals[approver] = ApprovalInfo{
			Approver:  approver,
			Level:     level,
			Decision:  decision,
			Timestamp: time.Now().UTC(),
		}

		// [SAFETY] Update current level if this is a higher level approval
		if level > status.CurrentLevel {
			status.CurrentLevel = level
		}

		// [SAFETY] If denied at any level, mark as denied immediately
		if state == ApprovalStateDenied {
			status.State = ApprovalStateDenied
			status.Decision = decision
			status.DecidedBy = approver
			status.UpdatedAt = time.Now().UTC()
			e.approvals[id] = status
			return nil
		}

		// [SAFETY] Check if all required levels have been approved
		if state == ApprovalStateApproved && status.CurrentLevel >= status.RequiredLevel {
			status.State = ApprovalStateApproved
			status.Decision = "All required levels approved"
			status.DecidedBy = approver
			status.UpdatedAt = time.Now().UTC()
			e.approvals[id] = status
			return nil
		}

		// [SAFETY] Still pending more approvals
		status.UpdatedAt = time.Now().UTC()
		e.approvals[id] = status
		return nil
	}

	// [SAFETY] Handle cancellation
	status.State = state
	status.Decision = decision
	status.DecidedBy = approver
	status.UpdatedAt = time.Now().UTC()
	e.approvals[id] = status
	return nil
}
