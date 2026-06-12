package lifecycle

import (
	"errors"
	"testing"
)

func TestAgentLifecycleTransitions(t *testing.T) {
	lc := NewAgentLifecycle()
	if err := lc.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if lc.State() != StateRunning {
		t.Fatalf("expected running, got %s", lc.State())
	}
	if err := lc.Stop(); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	if lc.State() != StateStopped {
		t.Fatalf("expected stopped, got %s", lc.State())
	}
}

func TestAgentLifecycleRecordsFailure(t *testing.T) {
	lc := NewAgentLifecycle()
	lc.Fail(errors.New("boom"))
	if lc.State() != StateFailed {
		t.Fatalf("expected failed, got %s", lc.State())
	}
	if lc.LastError() == nil || lc.LastError().Error() != "boom" {
		t.Fatalf("unexpected last error: %v", lc.LastError())
	}
}
