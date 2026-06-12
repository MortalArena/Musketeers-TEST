package runtime

import (
	"fmt"

	"github.com/MortalArena/Musketeers/pkg/runtime/events"
	"github.com/MortalArena/Musketeers/pkg/runtime/knowledge"
	"github.com/MortalArena/Musketeers/pkg/runtime/lifecycle"
	"github.com/MortalArena/Musketeers/pkg/runtime/observability"
	"github.com/MortalArena/Musketeers/pkg/runtime/scheduler"
	"github.com/MortalArena/Musketeers/pkg/runtime/state"
)

type AgentRuntime interface {
	Events() events.EventBus
	State() state.StateStore
	Knowledge() knowledge.KnowledgeStore
	Scheduler() scheduler.Scheduler
	Logger() observability.Logger
	Tracer() observability.Tracer
	Metrics() observability.Metrics
	Audit() observability.AuditLog
	Lifecycle() *lifecycle.AgentLifecycle
	Start() error
	Stop() error
}

type AgentRuntimeImpl struct {
	events    events.EventBus
	state     state.StateStore
	knowledge knowledge.KnowledgeStore
	scheduler scheduler.Scheduler
	logger    observability.Logger
	tracer    observability.Tracer
	metrics   observability.Metrics
	audit     observability.AuditLog
	lifecycle *lifecycle.AgentLifecycle
}

func NewAgentRuntime(
	events events.EventBus,
	state state.StateStore,
	knowledge knowledge.KnowledgeStore,
	scheduler scheduler.Scheduler,
	logger observability.Logger,
	tracer observability.Tracer,
	metrics observability.Metrics,
	audit observability.AuditLog,
) *AgentRuntimeImpl {
	if logger == nil {
		logger, _ = observability.NewZapLogger("info")
	}
	if audit == nil {
		audit = observability.NewMemoryAuditLog()
	}
	return &AgentRuntimeImpl{
		events:    events,
		state:     state,
		knowledge: knowledge,
		scheduler: scheduler,
		logger:    logger,
		tracer:    tracer,
		metrics:   metrics,
		audit:     audit,
		lifecycle: lifecycle.NewAgentLifecycle(),
	}
}

func (r *AgentRuntimeImpl) Events() events.EventBus              { return r.events }
func (r *AgentRuntimeImpl) State() state.StateStore              { return r.state }
func (r *AgentRuntimeImpl) Knowledge() knowledge.KnowledgeStore  { return r.knowledge }
func (r *AgentRuntimeImpl) Scheduler() scheduler.Scheduler       { return r.scheduler }
func (r *AgentRuntimeImpl) Logger() observability.Logger         { return r.logger }
func (r *AgentRuntimeImpl) Tracer() observability.Tracer         { return r.tracer }
func (r *AgentRuntimeImpl) Metrics() observability.Metrics       { return r.metrics }
func (r *AgentRuntimeImpl) Audit() observability.AuditLog        { return r.audit }
func (r *AgentRuntimeImpl) Lifecycle() *lifecycle.AgentLifecycle { return r.lifecycle }

func (r *AgentRuntimeImpl) Start() error {
	if r == nil {
		return fmt.Errorf("runtime is nil")
	}
	if err := r.lifecycle.Start(); err != nil {
		return err
	}
	if r.scheduler != nil {
		if err := r.scheduler.Start(); err != nil {
			r.lifecycle.Fail(err)
			return err
		}
	}
	if r.logger != nil {
		r.logger.Info("Agent runtime started", map[string]any{"did": ""})
	}
	return nil
}

func (r *AgentRuntimeImpl) Stop() error {
	if r == nil {
		return fmt.Errorf("runtime is nil")
	}
	var firstErr error
	if r.scheduler != nil {
		if err := r.scheduler.Stop(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if r.state != nil {
		if err := r.state.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if r.audit != nil {
		if err := r.audit.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		r.lifecycle.Fail(firstErr)
		return firstErr
	}
	if err := r.lifecycle.Stop(); err != nil {
		return err
	}
	if r.logger != nil {
		r.logger.Info("Agent runtime stopped", map[string]any{"did": ""})
	}
	return nil
}
