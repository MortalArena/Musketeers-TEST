package state

import "testing"

func TestStateStores(t *testing.T) {
	stores := []struct {
		name  string
		store StateStore
	}{
		{"memory", NewMemoryStateStore()},
	}

	for _, tt := range stores {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.store.Set("agent:1", []byte("active")); err != nil {
				t.Fatalf("Set returned error: %v", err)
			}
			got, err := tt.store.Get("agent:1")
			if err != nil {
				t.Fatalf("Get returned error: %v", err)
			}
			if string(got) != "active" {
				t.Fatalf("unexpected value: %s", got)
			}

			keys, err := tt.store.List("agent:")
			if err != nil {
				t.Fatalf("List returned error: %v", err)
			}
			if len(keys) != 1 || keys[0] != "agent:1" {
				t.Fatalf("unexpected keys: %v", keys)
			}
			if err := tt.store.Delete("agent:1"); err != nil {
				t.Fatalf("Delete returned error: %v", err)
			}
			got, err = tt.store.Get("agent:1")
			if err != nil {
				t.Fatalf("Get after delete returned error: %v", err)
			}
			if got != nil {
				t.Fatalf("expected nil after delete, got %v", got)
			}
			if err := tt.store.Close(); err != nil {
				t.Fatalf("Close returned error: %v", err)
			}
		})
	}
}
