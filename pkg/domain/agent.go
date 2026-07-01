package domain

import "time"

// Agent Domain Model - الكيان الأساسي للوكيل
type Agent struct {
	ID          string
	Name        string
	Type        AgentType
	Status      AgentStatus
	SessionID   string
	Capabilities []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastActive  time.Time
}

// AgentType Value Object
type AgentType string

const (
	AgentTypeManager  AgentType = "manager"
	AgentTypeRegular  AgentType = "regular"
	AgentTypeSpecial  AgentType = "special"
	AgentTypeHybrid   AgentType = "hybrid"
)

// AgentStatus Value Object
type AgentStatus string

const (
	AgentStatusCreated  AgentStatus = "created"
	AgentStatusActive   AgentStatus = "active"
	AgentStatusIdle     AgentStatus = "idle"
	AgentStatusBusy     AgentStatus = "busy"
	AgentStatusInactive AgentStatus = "inactive"
	AgentStatusError    AgentStatus = "error"
)

// IsValid يتحقق من صحة Agent
func (a *Agent) IsValid() bool {
	if a.ID == "" {
		return false
	}
	if a.Name == "" {
		return false
	}
	if a.Type == "" {
		return false
	}
	return true
}

// IsActive يتحقق مما إذا كان الوكيل نشطاً
func (a *Agent) IsActive() bool {
	return a.Status == AgentStatusActive
}

// IsBusy يتحقق مما إذا كان الوكيل مشغولاً
func (a *Agent) IsBusy() bool {
	return a.Status == AgentStatusBusy
}

// CanAssignToSession يتحقق مما إذا كان يمكن تعيين الوكيل لجلسة
func (a *Agent) CanAssignToSession() bool {
	return a.IsActive() && !a.IsBusy()
}

// AssignToSession يعين الوكيل لجلسة
func (a *Agent) AssignToSession(sessionID string) {
	a.SessionID = sessionID
	a.Status = AgentStatusBusy
	a.UpdatedAt = time.Now()
}

// ReleaseFromSession يفرغ الوكيل من الجلسة
func (a *Agent) ReleaseFromSession() {
	a.SessionID = ""
	a.Status = AgentStatusIdle
	a.UpdatedAt = time.Now()
}

// AddCapability يضيف قدرة
func (a *Agent) AddCapability(capability string) {
	a.Capabilities = append(a.Capabilities, capability)
}

// HasCapability يتحقق من وجود قدرة
func (a *Agent) HasCapability(capability string) bool {
	for _, cap := range a.Capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}
