package recovery

import (
	"context"
	"sync"
	"time"
)

type Handler func(ctx context.Context, err error)

type Manager struct {
	mu        sync.Mutex
	handlers  []Handler
	maxRetries int
	baseDelay time.Duration
}

func NewManager(maxRetries int, baseDelay time.Duration) *Manager {
	return &Manager{
		handlers:   make([]Handler, 0),
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
	}
}

func (m *Manager) AddHandler(h Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = append(m.handlers, h)
}

func (m *Manager) Execute(ctx context.Context, fn func(context.Context) error) error {
	var lastErr error
	for i := 0; i <= m.maxRetries; i++ {
		if i > 0 {
			delay := m.baseDelay * time.Duration(1<<(i-1))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
		if err := fn(ctx); err != nil {
			lastErr = err
			m.notifyHandlers(ctx, err)
			continue
		}
		return nil
	}
	return lastErr
}

func (m *Manager) notifyHandlers(ctx context.Context, err error) {
	m.mu.Lock()
	handlers := make([]Handler, len(m.handlers))
	copy(handlers, m.handlers)
	m.mu.Unlock()
	for _, h := range handlers {
		h(ctx, err)
	}
}
