package scheduler

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/MortalArena/Musketeers/pkg/runtime/events"
)

func TestEventDrivenScheduler(t *testing.T) {
	bus := events.NewMemoryEventBus(10)
	defer bus.Close()

	var calls int32
	_, err := bus.Subscribe(events.EventScheduleTriggered, func(event events.Event) error {
		if event.Data["task_id"] != "tick" {
			t.Fatalf("unexpected task_id: %v", event.Data["task_id"])
		}
		atomic.AddInt32(&calls, 1)
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}

	s := NewEventDrivenScheduler(bus)
	if err := s.Schedule("tick", "* * * * * *", map[string]any{"task_id": "tick"}); err != nil {
		t.Fatalf("Schedule returned error: %v", err)
	}
	if err := s.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer s.Stop()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && atomic.LoadInt32(&calls) == 0 {
		time.Sleep(20 * time.Millisecond)
	}
	if atomic.LoadInt32(&calls) == 0 {
		t.Fatal("expected scheduled event")
	}
	if err := s.Cancel("tick"); err != nil {
		t.Fatalf("Cancel returned error: %v", err)
	}
}
