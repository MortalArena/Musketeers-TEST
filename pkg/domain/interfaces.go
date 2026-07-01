package domain

// ISession واجهة الجلسة - Domain Interface
type ISession interface {
	GetID() string
	GetStatus() SessionStatus
	GetAgentIDs() []string
	AddAgentID(agentID string)
	RemoveAgentID(agentID string)
	HasAgentID(agentID string) bool
	IsActive() bool
	IsClosed() bool
	CanAddTask() bool
}

// IAgent واجهة الوكيل - Domain Interface
type IAgent interface {
	GetID() string
	GetName() string
	GetType() AgentType
	GetStatus() AgentStatus
	GetSessionID() string
	GetCapabilities() []string
	IsActive() bool
	IsBusy() bool
	CanAssignToSession() bool
	AssignToSession(sessionID string)
	ReleaseFromSession()
	AddCapability(capability string)
	HasCapability(capability string) bool
}

// ITask واجهة المهمة - Domain Interface
type ITask interface {
	GetID() string
	GetTitle() string
	GetDescription() string
	GetStatus() TaskStatus
	GetPriority() TaskPriority
	GetSessionID() string
	GetAssignedTo() string
	IsCompleted() bool
	IsInProgress() bool
	CanAssign() bool
	Assign(agentID string)
	Complete()
	Fail()
}

// IHumanClient واجهة العميل البشري - Domain Interface
type IHumanClient interface {
	GetID() string
	GetName() string
	GetStatus() HumanClientStatus
	GetSessionID() string
	IsOnline() bool
}
