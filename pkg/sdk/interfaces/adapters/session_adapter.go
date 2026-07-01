package adapters

import (
	"context"

	"github.com/MortalArena/Musketeers/pkg/sdk/interfaces"
	"github.com/MortalArena/Musketeers/pkg/session"
	"github.com/MortalArena/Musketeers/pkg/session/core"
)

type SessionAdapter struct {
	container *session.SessionContainer
	manager   *core.UnifiedSessionManager
}

func NewSessionAdapter(container *session.SessionContainer, manager *core.UnifiedSessionManager) *SessionAdapter {
	return &SessionAdapter{container: container, manager: manager}
}

func (a *SessionAdapter) ID() string {
	if a.container == nil {
		return ""
	}
	return a.container.ID
}

func (a *SessionAdapter) Status() string {
	if a.container == nil {
		return ""
	}
	// استخدام الحقل المباشر بدلاً من الدالة
	// الحقل Status موجود في SessionContainer struct
	return a.container.Status
}

func (a *SessionAdapter) Start(ctx context.Context) error {
	return nil
}

func (a *SessionAdapter) Stop(ctx context.Context) error {
	if a.container != nil && a.container.EventBus != nil {
		a.container.EventBus.Stop()
	}
	return nil
}

func (a *SessionAdapter) Execute(ctx context.Context, task *interfaces.Task) (*interfaces.TaskResult, error) {
	tm := a.container.Tasks
	if tm == nil {
		return nil, nil
	}
	tm.CreateTask(ctx, task.Title, task.Description, session.PriorityMedium, task.Inputs, task.Timeout)
	return &interfaces.TaskResult{}, nil
}

func (a *SessionAdapter) State() map[string]interface{} {
	if a.container == nil {
		return nil
	}
	return map[string]interface{}{
		"id":     a.container.ID,
		"status": a.container.Status,
		"name":   a.container.Name,
	}
}

var _ interfaces.SessionInterface = (*SessionAdapter)(nil)
