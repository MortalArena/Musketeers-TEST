package knowledge

import (
	"testing"
	"time"
)

func TestDefaultKnowledgeStore(t *testing.T) {
	store := NewDefaultKnowledgeStore()

	if err := store.Working().Set("current-task", "translate", time.Second); err != nil {
		t.Fatalf("Working.Set returned error: %v", err)
	}
	if got, ok := store.Working().Get("current-task"); !ok || got != "translate" {
		t.Fatalf("unexpected working memory: %v, %v", got, ok)
	}

	if err := store.Semantic().Store("Musketeers agents communicate over ACP", []float32{1, 0, 0}, nil); err != nil {
		t.Fatalf("Semantic.Store returned error: %v", err)
	}
	results, err := store.Semantic().Search("agents", 5)
	if err != nil {
		t.Fatalf("Semantic.Search returned error: %v", err)
	}
	if len(results) != 1 || results[0].Text == "" {
		t.Fatalf("unexpected semantic results: %#v", results)
	}

	episode := Episode{Context: map[string]any{"event": "task.completed"}, Outcome: "ok"}
	if err := store.Episodic().Record(episode); err != nil {
		t.Fatalf("Episodic.Record returned error: %v", err)
	}
	episodes, err := store.Episodic().Recall("task.completed", nil)
	if err != nil {
		t.Fatalf("Episodic.Recall returned error: %v", err)
	}
	if len(episodes) != 1 {
		t.Fatalf("unexpected episodes: %#v", episodes)
	}

	if err := store.Procedural().StoreProcedure("ping", []ProcedureStep{{Action: "respond", Description: "return pong"}}); err != nil {
		t.Fatalf("Procedural.StoreProcedure returned error: %v", err)
	}
	steps, err := store.Procedural().GetProcedure("ping")
	if err != nil {
		t.Fatalf("Procedural.GetProcedure returned error: %v", err)
	}
	if len(steps) != 1 || steps[0].Action != "respond" {
		t.Fatalf("unexpected procedure: %#v", steps)
	}
}

func TestWorkingMemoryTTL(t *testing.T) {
	mem := NewInMemoryWorkingMemory()
	if err := mem.Set("short", "value", 20*time.Millisecond); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if _, ok := mem.Get("short"); !ok {
		t.Fatal("expected value before ttl")
	}
	time.Sleep(40 * time.Millisecond)
	if _, ok := mem.Get("short"); ok {
		t.Fatal("expected ttl expiry")
	}
}
