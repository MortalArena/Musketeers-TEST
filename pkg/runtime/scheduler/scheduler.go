package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/runtime/events"
	"github.com/robfig/cron/v3"
)

type Scheduler interface {
	Schedule(taskID, cronExpr string, metadata map[string]any) error
	Cancel(taskID string) error
	Start() error
	Stop() error
}

type EventDrivenScheduler struct {
	mu       sync.RWMutex
	cron     *cron.Cron
	eventBus events.EventBus
	tasks    map[string]cron.EntryID
	started  bool
}

func NewEventDrivenScheduler(eventBus events.EventBus) *EventDrivenScheduler {
	return &EventDrivenScheduler{
		cron:     cron.New(cron.WithSeconds()),
		eventBus: eventBus,
		tasks:    make(map[string]cron.EntryID),
	}
}

func (s *EventDrivenScheduler) Schedule(taskID, cronExpr string, metadata map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tasks[taskID]; exists {
		return fmt.Errorf("task already scheduled: %s", taskID)
	}
	entryID, err := s.cron.AddFunc(cronExpr, func() {
		event := events.Event{
			ID:        fmt.Sprintf("schedule-%d", time.Now().UnixNano()),
			Type:      events.EventScheduleTriggered,
			Source:    "scheduler",
			Target:    taskID,
			Timestamp: time.Now().UTC(),
			Data: map[string]any{
				"task_id":  taskID,
				"metadata": metadata,
			},
		}
		if s.eventBus != nil {
			if err := s.eventBus.Publish(event); err != nil {
				fmt.Printf("Error publishing schedule event: %v\n", err)
			}
		}
	})
	if err != nil {
		return err
	}
	s.tasks[taskID] = entryID
	return nil
}

func (s *EventDrivenScheduler) Cancel(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entryID, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}
	s.cron.Remove(entryID)
	delete(s.tasks, taskID)
	return nil
}

func (s *EventDrivenScheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		s.cron.Start()
		s.started = true
	}
	return nil
}

func (s *EventDrivenScheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		return nil
	}
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.started = false
	return nil
}

func NewImmediateScheduler(eventBus events.EventBus) *ImmediateScheduler {
	return &ImmediateScheduler{eventBus: eventBus}
}

type ImmediateScheduler struct {
	eventBus events.EventBus
}

func (s *ImmediateScheduler) Schedule(taskID, cronExpr string, metadata map[string]any) error {
	if s.eventBus == nil {
		return nil
	}
	return s.eventBus.Publish(events.Event{
		ID:        fmt.Sprintf("schedule-%d", time.Now().UnixNano()),
		Type:      events.EventScheduleTriggered,
		Source:    "scheduler",
		Target:    taskID,
		Timestamp: time.Now().UTC(),
		Data: map[string]any{
			"task_id":         taskID,
			"cron_expression": cronExpr,
			"metadata":        metadata,
		},
	})
}

func (s *ImmediateScheduler) Cancel(taskID string) error { return nil }
func (s *ImmediateScheduler) Start() error               { return nil }
func (s *ImmediateScheduler) Stop() error                { return nil }

func StopWithContext(ctx context.Context, stop func() error) error {
	done := make(chan error, 1)
	go func() { done <- stop() }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
