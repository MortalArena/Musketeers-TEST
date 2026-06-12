package lifecycle

import (
	"errors"
	"sync"
	"time"
)

type State string

const (
	StateCreated  State = "created"
	StateStarting State = "starting"
	StateRunning  State = "running"
	StateStopping State = "stopping"
	StateStopped  State = "stopped"
	StateFailed   State = "failed"
)

type AgentLifecycle struct {
	mu        sync.RWMutex
	state     State
	startedAt time.Time
	stoppedAt time.Time
	lastError error
}

func NewAgentLifecycle() *AgentLifecycle {
	return &AgentLifecycle{state: StateCreated}
}

func (l *AgentLifecycle) State() State {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.state
}

func (l *AgentLifecycle) Start() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	switch l.state {
	case StateCreated, StateStopped:
		l.state = StateStarting
		l.startedAt = time.Now().UTC()
		l.lastError = nil
		l.state = StateRunning
		return nil
	case StateRunning:
		return nil
	case StateStarting:
		return errors.New("agent is already starting")
	case StateStopping:
		return errors.New("agent is stopping")
	case StateFailed:
		return errors.New("agent failed; restart required")
	default:
		return errors.New("unknown lifecycle state")
	}
}

func (l *AgentLifecycle) Stop() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	switch l.state {
	case StateRunning, StateStarting:
		l.state = StateStopping
		l.stoppedAt = time.Now().UTC()
		l.state = StateStopped
		return nil
	case StateStopped:
		return nil
	case StateFailed:
		return l.lastError
	default:
		return errors.New("agent cannot stop from state " + string(l.state))
	}
}

func (l *AgentLifecycle) Fail(err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err == nil {
		err = errors.New("unknown failure")
	}
	l.state = StateFailed
	l.stoppedAt = time.Now().UTC()
	l.lastError = err
}

func (l *AgentLifecycle) LastError() error {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.lastError
}

func (l *AgentLifecycle) StartedAt() time.Time {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.startedAt
}

func (l *AgentLifecycle) StoppedAt() time.Time {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.stoppedAt
}
