package workflow

import (
	"fmt"
	"testing"
	"time"

	"github.com/MortalArena/Musketeers/pkg/content"
	"github.com/MortalArena/Musketeers/pkg/storage"
)

func TestCheckpointManager_Save(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"
	nodeID := "node-456"
	state := map[string]interface{}{
		"var1": "value1",
		"var2": 42,
	}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
}

func TestCheckpointManager_Save_EmptyState(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"
	nodeID := "node-456"
	state := map[string]interface{}{}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with empty state failed: %v", err)
	}
}

func TestCheckpointManager_Save_MultipleCheckpoints(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"

	for i := 0; i < 3; i++ {
		nodeID := fmt.Sprintf("node-%d", i)
		state := map[string]interface{}{
			"step": i,
		}
		err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
		if err != nil {
			t.Fatalf("Save failed for checkpoint %d: %v", i, err)
		}
	}
}

func TestCheckpointManager_GetLatest(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	// في الإصدار الحالي، GetLatest غير منفذ
	_, err := cm.GetLatest("workflow-123")
	if err == nil {
		t.Error("Expected error for not implemented feature")
	}
}

func TestCheckpointManager_GetLatest_NotFound(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	_, err := cm.GetLatest("nonexistent-workflow")
	if err == nil {
		t.Error("Expected error for nonexistent workflow")
	}
}

func TestNewCheckpointManager(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	if cm == nil {
		t.Fatal("NewCheckpointManager returned nil")
	}

	if cm.store != store {
		t.Error("NewCheckpointManager did not set store correctly")
	}
}

func TestCheckpoint_Struct(t *testing.T) {
	cp := &Checkpoint{
		ID:         "test-id",
		WorkflowID: "workflow-123",
		NodeID:     "node-456",
		State:      map[string]interface{}{"key": "value"},
		Hash:       "test-hash",
		Timestamp:  time.Now(),
	}

	if cp.ID != "test-id" {
		t.Errorf("Expected ID test-id, got %s", cp.ID)
	}

	if cp.WorkflowID != "workflow-123" {
		t.Errorf("Expected WorkflowID workflow-123, got %s", cp.WorkflowID)
	}

	if cp.NodeID != "node-456" {
		t.Errorf("Expected NodeID node-456, got %s", cp.NodeID)
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	if id1 == "" {
		t.Error("generateID returned empty string")
	}

	if id1 == id2 {
		t.Error("generateID returned duplicate IDs")
	}

	if len(id1) != 32 { // 16 bytes = 32 hex characters
		t.Errorf("Expected ID length 32, got %d", len(id1))
	}
}

func TestCheckpointManager_Save_ComplexState(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"
	nodeID := "node-456"
	state := map[string]interface{}{
		"var1": "value1",
		"var2": 42,
		"var3": true,
		"var4": []string{"a", "b", "c"},
		"var5": map[string]int{"x": 1, "y": 2},
	}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with complex state failed: %v", err)
	}
}

func TestCheckpointManager_Save_LargeState(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"
	nodeID := "node-456"
	state := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		state[fmt.Sprintf("key-%d", i)] = fmt.Sprintf("value-%d", i)
	}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with large state failed: %v", err)
	}
}

func TestCheckpointManager_Save_NilState(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"
	nodeID := "node-456"
	var state map[string]interface{} = nil

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with nil state failed: %v", err)
	}
}

func TestCheckpointManager_Save_EmptyWorkflowID(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := ""
	nodeID := "node-456"
	state := map[string]interface{}{"key": "value"}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with empty workflow ID failed: %v", err)
	}
}

func TestCheckpointManager_Save_EmptyNodeID(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"
	nodeID := ""
	state := map[string]interface{}{"key": "value"}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with empty node ID failed: %v", err)
	}
}

func TestCheckpointManager_Save_WithSpecialCharsInIDs(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123!@#$%^&*()"
	nodeID := "node-456-<>?/\\"
	state := map[string]interface{}{"key": "value"}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with special chars in IDs failed: %v", err)
	}
}

func TestCheckpointManager_Save_WithUnicodeInIDs(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123-مربا-世界"
	nodeID := "node-456-مرحبا"
	state := map[string]interface{}{"key": "value"}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with unicode in IDs failed: %v", err)
	}
}

func TestCheckpointManager_Save_WithLongIDs(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	longID := ""
	for i := 0; i < 1000; i++ {
		longID += "a"
	}

	workflowID := longID
	nodeID := longID
	state := map[string]interface{}{"key": "value"}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with long IDs failed: %v", err)
	}
}

func TestCheckpointManager_Save_WithStateContainingNil(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"
	nodeID := "node-456"
	state := map[string]interface{}{
		"key1": "value1",
		"key2": nil,
		"key3": "value3",
	}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with state containing nil failed: %v", err)
	}
}

func TestCheckpointManager_Save_WithStateContainingFloats(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"
	nodeID := "node-456"
	state := map[string]interface{}{
		"pi":    3.14159,
		"e":     2.71828,
		"sqrt2": 1.41421,
	}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with state containing floats failed: %v", err)
	}
}

func TestCheckpointManager_Save_WithStateContainingNegativeNumbers(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"
	nodeID := "node-456"
	state := map[string]interface{}{
		"neg1": -1,
		"neg2": -100,
		"neg3": -9999,
	}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with state containing negative numbers failed: %v", err)
	}
}

func TestCheckpointManager_Save_WithStateContainingNestedMaps(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"
	nodeID := "node-456"
	state := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": "deep value",
			},
		},
	}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with state containing nested maps failed: %v", err)
	}
}

func TestCheckpointManager_Save_WithStateContainingArrays(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"
	nodeID := "node-456"
	state := map[string]interface{}{
		"array1": []int{1, 2, 3, 4, 5},
		"array2": []string{"a", "b", "c"},
		"array3": []interface{}{1, "two", 3.0, true},
	}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with state containing arrays failed: %v", err)
	}
}

func TestCheckpointManager_Save_WithStateContainingUnicode(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"
	nodeID := "node-456"
	state := map[string]interface{}{
		"arabic":  "مرحبا بالعالم",
		"chinese": "世界你好",
		"emoji":   "🌍🌎🌏",
	}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with state containing unicode failed: %v", err)
	}
}

func TestCheckpointManager_Save_WithStateContainingSpecialChars(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"
	nodeID := "node-456"
	state := map[string]interface{}{
		"newline": "line1\nline2",
		"tab":     "col1\tcol2",
		"quote":   `he said "hello"`,
	}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with state containing special chars failed: %v", err)
	}
}

func TestCheckpointManager_Save_WithStateContainingZeroValues(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"
	nodeID := "node-456"
	state := map[string]interface{}{
		"zero_int":   0,
		"zero_float": 0.0,
		"empty_str":  "",
		"false_bool": false,
	}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with state containing zero values failed: %v", err)
	}
}

type MockBlockStore struct {
	putError  bool
	callCount int
}

func (m *MockBlockStore) Put(cid string, data []byte, did string) error {
	m.callCount++
	if m.putError && m.callCount >= 2 {
		return fmt.Errorf("mock store error")
	}
	return nil
}

func (m *MockBlockStore) Get(cid string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockBlockStore) Size() int64 {
	return 0
}

func TestCheckpointManager_Save_WithStoreError(t *testing.T) {
	store := &MockBlockStore{putError: true}
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"
	nodeID := "node-456"
	state := map[string]interface{}{"key": "value"}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err == nil {
		t.Error("Expected error when store fails")
	}
}

func TestCheckpointManager_Save_WithVeryLargeState(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"
	nodeID := "node-456"
	state := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		state[fmt.Sprintf("key-%d", i)] = fmt.Sprintf("value-%d", i)
	}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with very large state failed: %v", err)
	}
}

func TestCheckpointManager_Save_WithDeeplyNestedState(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"
	nodeID := "node-456"
	state := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": map[string]interface{}{
					"level4": map[string]interface{}{
						"level5": "deep value",
					},
				},
			},
		},
	}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with deeply nested state failed: %v", err)
	}
}

func TestCheckpointManager_Save_WithMixedTypes(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"
	nodeID := "node-456"
	state := map[string]interface{}{
		"string": "hello",
		"int":    42,
		"float":  3.14,
		"bool":   true,
		"nil":    nil,
		"array":  []int{1, 2, 3},
		"map":    map[string]string{"a": "b"},
	}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with mixed types failed: %v", err)
	}
}

func TestCheckpointManager_Save_WithEmptyStrings(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"
	nodeID := "node-456"
	state := map[string]interface{}{
		"empty1": "",
		"empty2": "",
		"empty3": "",
	}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with empty strings failed: %v", err)
	}
}

func TestCheckpointManager_Save_WithLargeStrings(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"
	nodeID := "node-456"
	largeString := ""
	for i := 0; i < 10000; i++ {
		largeString += "a"
	}
	state := map[string]interface{}{
		"large": largeString,
	}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with large strings failed: %v", err)
	}
}

func TestCheckpointManager_Save_WithBooleanValues(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"
	nodeID := "node-456"
	state := map[string]interface{}{
		"true1":  true,
		"true2":  true,
		"false1": false,
		"false2": false,
	}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with boolean values failed: %v", err)
	}
}

func TestCheckpointManager_Save_WithIntegerValues(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"
	nodeID := "node-456"
	state := map[string]interface{}{
		"small":  1,
		"medium": 1000,
		"large":  1000000,
	}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with integer values failed: %v", err)
	}
}

func TestCheckpointManager_Save_WithFloatValues(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	cm := NewCheckpointManager(store)

	workflowID := "workflow-123"
	nodeID := "node-456"
	state := map[string]interface{}{
		"small":  0.1,
		"medium": 1.5,
		"large":  1000.999,
	}

	err := cm.Save(workflowID, nodeID, state, "did:mskt:test")
	if err != nil {
		t.Fatalf("Save with float values failed: %v", err)
	}
}
