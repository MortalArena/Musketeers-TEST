package eventbus

import (
	"sync"
	"testing"
	"time"
)

func TestEventBusSubscribePublish(t *testing.T) {
	eb := NewEventBus()

	var wg sync.WaitGroup
	wg.Add(1)

	handlerCalled := false
	handler := func(event Event) {
		handlerCalled = true
		wg.Done()
	}

	eb.Subscribe("test.event", handler)
	eb.Publish(Event{Type: "test.event", Payload: "test data"})

	wg.Wait()

	if !handlerCalled {
		t.Error("Handler لم يتم استدعاؤه")
	}
}

func TestEventBusWildcard(t *testing.T) {
	eb := NewEventBus()

	var wg sync.WaitGroup
	wg.Add(2)

	wildcardCalled := false
	specificCalled := false

	wildcardHandler := func(event Event) {
		wildcardCalled = true
		wg.Done()
	}

	specificHandler := func(event Event) {
		specificCalled = true
		wg.Done()
	}

	eb.Subscribe("*", wildcardHandler)
	eb.Subscribe("test.event", specificHandler)
	eb.Publish(Event{Type: "test.event", Payload: "test data"})

	wg.Wait()

	if !wildcardCalled {
		t.Error("Wildcard Handler لم يتم استدعاؤه")
	}
	if !specificCalled {
		t.Error("Specific Handler لم يتم استدعاؤه")
	}
}

func TestEventBusUnsubscribe(t *testing.T) {
	eb := NewEventBus()

	handlerCalled := false
	handler := func(event Event) {
		handlerCalled = true
	}

	eb.Subscribe("test.event", handler)
	eb.Unsubscribe("test.event")
	eb.Publish(Event{Type: "test.event", Payload: "test data"})

	time.Sleep(100 * time.Millisecond) // انتظار للمعالجة

	if handlerCalled {
		t.Error("Handler تم استدعاؤه بعد Unsubscribe")
	}
}

func TestEventBusClear(t *testing.T) {
	eb := NewEventBus()

	handlerCalled := false
	handler := func(event Event) {
		handlerCalled = true
	}

	eb.Subscribe("test.event", handler)
	eb.Clear()
	eb.Publish(Event{Type: "test.event", Payload: "test data"})

	time.Sleep(100 * time.Millisecond) // انتظار للمعالجة

	if handlerCalled {
		t.Error("Handler تم استدعاؤه بعد Clear")
	}
}

func TestEventBusMultipleHandlers(t *testing.T) {
	eb := NewEventBus()

	var wg sync.WaitGroup
	wg.Add(3)

	callCount := 0
	handler := func(event Event) {
		callCount++
		wg.Done()
	}

	eb.Subscribe("test.event", handler)
	eb.Subscribe("test.event", handler)
	eb.Subscribe("test.event", handler)
	eb.Publish(Event{Type: "test.event", Payload: "test data"})

	wg.Wait()

	if callCount != 3 {
		t.Errorf("تم استدعاء Handler %d مرة، متوقع 3", callCount)
	}
}

func TestEventBusTimestamp(t *testing.T) {
	eb := NewEventBus()

	var wg sync.WaitGroup
	wg.Add(1)

	var receivedEvent Event
	handler := func(event Event) {
		receivedEvent = event
		wg.Done()
	}

	eb.Subscribe("test.event", handler)
	eb.Publish(Event{Type: "test.event", Payload: "test data"})

	wg.Wait()

	if receivedEvent.Timestamp.IsZero() {
		t.Error("Timestamp لم يتم تعيينه")
	}
}
