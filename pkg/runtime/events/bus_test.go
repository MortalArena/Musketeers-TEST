package events

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestMemoryEventBusPublishSubscribeAndReplay(t *testing.T) {
	bus := NewMemoryEventBus(10)
	defer bus.Close()

	var calls int32
	sub, err := bus.Subscribe(EventTaskCompleted, func(event Event) error {
		atomic.AddInt32(&calls, 1)
		if event.Data["task_id"] != "task-1" {
			t.Fatalf("unexpected task_id: %v", event.Data["task_id"])
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}

	event := Event{Type: EventTaskCompleted, Source: "runtime", Data: map[string]any{"task_id": "task-1"}}
	if err := bus.Publish(event); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	waitFor(t, func() bool { return atomic.LoadInt32(&calls) == 1 })
	if err := sub.Unsubscribe(); err != nil {
		t.Fatalf("Unsubscribe returned error: %v", err)
	}

	var replayCalls int32
	if err := bus.Replay(time.Now().Add(-time.Hour), func(event Event) error {
		atomic.AddInt32(&replayCalls, 1)
		return nil
	}); err != nil {
		t.Fatalf("Replay returned error: %v", err)
	}
	if replayCalls != 1 {
		t.Fatalf("expected one replayed event, got %d", replayCalls)
	}
}

func TestMemoryEventBusRejectsPublishAfterClose(t *testing.T) {
	bus := NewMemoryEventBus(1)
	if err := bus.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := bus.Publish(Event{Type: EventAgentStarted}); err == nil {
		t.Fatal("expected closed bus error")
	}
}

func waitFor(t *testing.T, predicate func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if predicate() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !predicate() {
		t.Fatal("condition was not met before timeout")
	}
}

func TestMemoryEventBusHandlerErrorIsReturnedByReplay(t *testing.T) {
	bus := NewMemoryEventBus(1)
	defer bus.Close()
	if err := bus.Publish(Event{Type: EventAgentStarted}); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	err := bus.Replay(time.Now().Add(-time.Hour), func(Event) error { return errors.New("boom") })
	if err == nil {
		t.Fatal("expected replay error")
	}
}
