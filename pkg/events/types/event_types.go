package types

// EventType نوع الحدث
type EventType string

// أنواع الأحداث الأساسية
const (
	EventTypeAgentStarted       EventType = "agent.started"
	EventTypeAgentStopped       EventType = "agent.stopped"
	EventTypeAgentFailed        EventType = "agent.failed"
	EventTypeMessageReceived    EventType = "message.received"
	EventTypeMessageSent        EventType = "message.sent"
	EventTypeTaskReceived       EventType = "task.received"
	EventTypeTaskStarted        EventType = "task.started"
	EventTypeTaskCompleted      EventType = "task.completed"
	EventTypeTaskFailed         EventType = "task.failed"
	EventTypeSessionCreated     EventType = "session.created"
	EventTypeSessionPaused      EventType = "session.paused"
	EventTypeSessionResumed     EventType = "session.resumed"
	EventTypeSessionCompleted   EventType = "session.completed"
	EventTypeMemoryCreated      EventType = "memory.created"
	EventTypeMemoryUpdated      EventType = "memory.updated"
	EventTypeMemoryDeleted      EventType = "memory.deleted"
	EventTypeSkillLearned       EventType = "skill.learned"
	EventTypeSkillImproved      EventType = "skill.improved"
	EventTypeSkillUsed          EventType = "skill.used"
	EventTypeWorkflowStarted    EventType = "workflow.started"
	EventTypeWorkflowCompleted  EventType = "workflow.completed"
	EventTypeWorkflowFailed     EventType = "workflow.failed"
)

// EventPriority أولوية الحدث
type EventPriority string

// أولويات الأحداث
const (
	EventPriorityLow      EventPriority = "low"
	EventPriorityMedium   EventPriority = "medium"
	EventPriorityHigh     EventPriority = "high"
	EventPriorityCritical EventPriority = "critical"
)
