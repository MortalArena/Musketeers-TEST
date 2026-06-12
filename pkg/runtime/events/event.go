package events

import "time"

type Event struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Source    string            `json:"source"`
	Target    string            `json:"target,omitempty"`
	Data      map[string]any    `json:"data"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

const (
	EventAgentStarted       = "agent.started"
	EventAgentStopped       = "agent.stopped"
	EventAgentFailed        = "agent.failed"
	EventMessageReceived    = "message.received"
	EventMessageSent        = "message.sent"
	EventTaskReceived       = "task.received"
	EventTaskStarted        = "task.started"
	EventTaskCompleted      = "task.completed"
	EventTaskFailed         = "task.failed"
	EventScheduleTriggered  = "schedule.triggered"
	EventWebhookReceived    = "webhook.received"
	EventDomainUpdated      = "domain.updated"
	EventChannelJoined      = "channel.joined"
	EventChannelLeft        = "channel.left"
	EventCapabilityGranted  = "capability.granted"
	EventCapabilityRevoked  = "capability.revoked"
	EventCapabilityExecuted = "capability.executed"
	EventWorkflowStarted    = "workflow.started"
	EventWorkflowCompleted  = "workflow.completed"
	EventWorkflowFailed     = "workflow.failed"
	EventStepStarted        = "step.started"
	EventStepCompleted      = "step.completed"
	EventStepFailed         = "step.failed"
	EventPolicyEvaluated    = "policy.evaluated"
	EventApprovalRequested  = "approval.requested"
	EventApprovalGranted    = "approval.granted"
	EventApprovalDenied     = "approval.denied"
)
