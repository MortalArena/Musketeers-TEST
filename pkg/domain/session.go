package domain

import "time"

// Session Domain Model - الكيان الأساسي للجلسة
type Session struct {
	ID          string
	Status      SessionStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ClosedAt    *time.Time
	AgentIDs    []string // Aggregate Reference - Agent IDs فقط
	HumanClient *HumanClient
	Tasks       []Task
}

// SessionStatus Value Object
type SessionStatus string

const (
	SessionStatusCreated   SessionStatus = "created"
	SessionStatusActive    SessionStatus = "active"
	SessionStatusPaused    SessionStatus = "paused"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusClosed    SessionStatus = "closed"
	SessionStatusError     SessionStatus = "error"
)

// IsValid يتحقق من صحة Session
func (s *Session) IsValid() bool {
	if s.ID == "" {
		return false
	}
	if s.Status == "" {
		return false
	}
	if s.CreatedAt.IsZero() {
		return false
	}
	return true
}

// IsActive يتحقق مما إذا كانت الجلسة نشطة
func (s *Session) IsActive() bool {
	return s.Status == SessionStatusActive
}

// IsClosed يتحقق مما إذا كانت الجلسة مغلقة
func (s *Session) IsClosed() bool {
	return s.Status == SessionStatusClosed
}

// CanAddTask يتحقق مما إذا كان يمكن إضافة مهمة
func (s *Session) CanAddTask() bool {
	return s.IsActive()
}

// AddAgentID يضيف Agent ID
func (s *Session) AddAgentID(agentID string) {
	s.AgentIDs = append(s.AgentIDs, agentID)
}

// RemoveAgentID يزيل Agent ID
func (s *Session) RemoveAgentID(agentID string) {
	for i, id := range s.AgentIDs {
		if id == agentID {
			s.AgentIDs = append(s.AgentIDs[:i], s.AgentIDs[i+1:]...)
			break
		}
	}
}

// HasAgentID يتحقق من وجود Agent ID
func (s *Session) HasAgentID(agentID string) bool {
	for _, id := range s.AgentIDs {
		if id == agentID {
			return true
		}
	}
	return false
}
