package orchestrator

import (
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

// FailureStrategy استراتيجية معالجة الفشل
type FailureStrategy string

const (
	StrategyRetry        FailureStrategy = "retry"
	StrategyReassign     FailureStrategy = "reassign"
	StrategyEscalate     FailureStrategy = "escalate"
	StrategyFallback     FailureStrategy = "fallback"
	StrategySkip         FailureStrategy = "skip"
	StrategyManualReview FailureStrategy = "manual_review"
)

// FailureHandler معالج الفشل
type FailureHandler struct {
	Strategies      map[string]FailureStrategy
	RetryLimits     map[string]int
	EscalationRules []EscalationRule
	EventBus        *eventbus.EventBus
	Logger          *zap.Logger
	mu              sync.RWMutex
}

// EscalationRule قاعدة تصعيد
type EscalationRule struct {
	Condition   string        `json:"condition"`
	Action      string        `json:"action"`
	TargetAgent string        `json:"target_agent,omitempty"`
	Timeout     time.Duration `json:"timeout"`
}

// FailureEvent حدث فشل
type FailureEvent struct {
	TaskID      string    `json:"task_id"`
	AgentDID    string    `json:"agent_did"`
	FailureType string    `json:"failure_type"` // timeout, error, quality_fail
	Error       string    `json:"error"`
	Timestamp   time.Time `json:"timestamp"`
	RetryCount  int       `json:"retry_count"`
}

// NewFailureHandler ينشئ معالج فشل
func NewFailureHandler(logger *zap.Logger) *FailureHandler {
	return &FailureHandler{
		Strategies: map[string]FailureStrategy{
			"timeout":       StrategyReassign,
			"error":         StrategyRetry,
			"quality_fail":  StrategyRetry,
			"agent_offline": StrategyReassign,
		},
		RetryLimits: map[string]int{
			"default":  3,
			"critical": 5,
		},
		EscalationRules: []EscalationRule{
			{
				Condition: "retry_count >= 3",
				Action:    "escalate_to_human",
				Timeout:   5 * time.Minute,
			},
			{
				Condition: "all_agents_busy",
				Action:    "queue_and_wait",
				Timeout:   10 * time.Minute,
			},
			{
				Condition: "agent_capability_too_low",
				Action:    "reassign_to_stronger_agent",
				Timeout:   1 * time.Minute,
			},
		},
		Logger: logger,
	}
}

// HandleFailure يعالج فشل مهمة
func (fh *FailureHandler) HandleFailure(event FailureEvent) FailureStrategy {
	fh.mu.Lock()
	defer fh.mu.Unlock()

	fh.Logger.Info("تم اكتشاف فشل",
		zap.String("task_id", event.TaskID),
		zap.String("agent", event.AgentDID),
		zap.String("failure_type", event.FailureType),
		zap.Int("retry_count", event.RetryCount),
	)

	// نشر حدث الفشل عبر EventBus
	if fh.EventBus != nil {
		fh.EventBus.Publish(eventbus.Event{
			Type:      "failure.detected",
			Payload:   event,
			SessionID: event.TaskID,
		})
	}

	strategy, exists := fh.Strategies[event.FailureType]
	if !exists {
		strategy = StrategyRetry
	}

	// تقييم قواعد التصعيد
	for _, rule := range fh.EscalationRules {
		if evaluateRule(rule, event) {
			fh.Logger.Warn("تفعيل قاعدة تصعيد",
				zap.String("rule_condition", rule.Condition),
				zap.String("action", rule.Action),
			)
			if fh.EventBus != nil {
				fh.EventBus.Publish(eventbus.Event{
					Type:      "failure.escalated",
					Payload:   map[string]interface{}{"rule": rule, "event": event},
					SessionID: event.TaskID,
				})
			}
			return StrategyEscalate
		}
	}

	maxRetries := fh.RetryLimits["default"]
	if event.RetryCount >= maxRetries {
		strategy = StrategyEscalate
	}

	return strategy
}

// evaluateRule يقيّم قاعدة تصعيد
func evaluateRule(rule EscalationRule, event FailureEvent) bool {
	switch rule.Condition {
	case "retry_count >= 3":
		return event.RetryCount >= 3
	case "all_agents_busy":
		return false
	case "agent_capability_too_low":
		return false
	default:
		return false
	}
}

// SetStrategy يضبط استراتيجية لنوع فشل معين
func (fh *FailureHandler) SetStrategy(failureType string, strategy FailureStrategy) {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	fh.Strategies[failureType] = strategy
}

// SetRetryLimit يضبط حد إعادة المحاولة
func (fh *FailureHandler) SetRetryLimit(taskType string, limit int) {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	fh.RetryLimits[taskType] = limit
}

// AddEscalationRule يضيف قاعدة تصعيد
func (fh *FailureHandler) AddEscalationRule(rule EscalationRule) {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	fh.EscalationRules = append(fh.EscalationRules, rule)
}
