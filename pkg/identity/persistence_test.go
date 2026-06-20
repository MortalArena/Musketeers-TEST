package identity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIdentityStoreSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	store, err := NewIdentityStore(dir)
	if err != nil {
		t.Fatalf("Failed to create identity store: %v", err)
	}

	did := "did:test:123"
	metadata := IdentityMetadata{
		IdentityType: "agent",
		NodeID:       "node-1",
		AgentName:    "test-agent",
		AgentVersion: "1.0.0",
		Provider:     "openai",
		APIKeyHash:   HashAPIKey("test-api-key"),
		IsActive:     true,
	}

	err = store.SaveIdentity(did, metadata)
	if err != nil {
		t.Fatalf("Failed to save identity: %v", err)
	}

	// Create new store to test loading from disk
	store2, err := NewIdentityStore(dir)
	if err != nil {
		t.Fatalf("Failed to create second identity store: %v", err)
	}

	identity, err := store2.GetIdentity(did)
	if err != nil {
		t.Fatalf("Failed to get identity: %v", err)
	}

	if identity.DID != did {
		t.Errorf("Expected DID %s, got %s", did, identity.DID)
	}
	if identity.Metadata.IdentityType != "agent" {
		t.Errorf("Expected identity type agent, got %s", identity.Metadata.IdentityType)
	}
	if identity.Metadata.APIKeyHash != HashAPIKey("test-api-key") {
		t.Error("API key hash mismatch")
	}
}

func TestIdentityStoreUpdateMetadata(t *testing.T) {
	dir := t.TempDir()
	store, err := NewIdentityStore(dir)
	if err != nil {
		t.Fatalf("Failed to create identity store: %v", err)
	}

	did := "did:test:456"
	metadata := IdentityMetadata{
		IdentityType: "human",
		NodeID:       "node-2",
		IsActive:     true,
	}

	err = store.SaveIdentity(did, metadata)
	if err != nil {
		t.Fatalf("Failed to save identity: %v", err)
	}

	// Update metadata
	updatedMetadata := IdentityMetadata{
		IdentityType: "human",
		NodeID:       "node-2",
		AgentVersion: "2.0.0",
		IsActive:     true,
	}

	err = store.UpdateIdentityMetadata(did, updatedMetadata)
	if err != nil {
		t.Fatalf("Failed to update metadata: %v", err)
	}

	identity, err := store.GetIdentity(did)
	if err != nil {
		t.Fatalf("Failed to get identity: %v", err)
	}

	if identity.Metadata.AgentVersion != "2.0.0" {
		t.Errorf("Expected version 2.0.0, got %s", identity.Metadata.AgentVersion)
	}
	// CreatedAt should be preserved
	if identity.Metadata.CreatedAt.IsZero() {
		t.Error("CreatedAt should be preserved")
	}
}

func TestIdentityStoreFindByIdentityType(t *testing.T) {
	dir := t.TempDir()
	store, err := NewIdentityStore(dir)
	if err != nil {
		t.Fatalf("Failed to create identity store: %v", err)
	}

	// Save multiple identities
	for i := 0; i < 5; i++ {
		metadata := IdentityMetadata{
			IdentityType: "agent",
			NodeID:       "node-1",
			IsActive:     true,
		}
		store.SaveIdentity("did:test:agent-"+string(rune('a'+i)), metadata)
	}

	for i := 0; i < 3; i++ {
		metadata := IdentityMetadata{
			IdentityType: "human",
			NodeID:       "node-1",
			IsActive:     true,
		}
		store.SaveIdentity("did:test:human-"+string(rune('a'+i)), metadata)
	}

	agents := store.FindIdentitiesByType("agent")
	if len(agents) != 5 {
		t.Errorf("Expected 5 agents, got %d", len(agents))
	}

	humans := store.FindIdentitiesByType("human")
	if len(humans) != 3 {
		t.Errorf("Expected 3 humans, got %d", len(humans))
	}
}

func TestIdentityStoreFindByNode(t *testing.T) {
	dir := t.TempDir()
	store, err := NewIdentityStore(dir)
	if err != nil {
		t.Fatalf("Failed to create identity store: %v", err)
	}

	// Save identities for different nodes
	metadata1 := IdentityMetadata{
		IdentityType: "agent",
		NodeID:       "node-1",
		IsActive:     true,
	}
	store.SaveIdentity("did:test:1", metadata1)

	metadata2 := IdentityMetadata{
		IdentityType: "agent",
		NodeID:       "node-2",
		IsActive:     true,
	}
	store.SaveIdentity("did:test:2", metadata2)

	metadata3 := IdentityMetadata{
		IdentityType: "agent",
		NodeID:       "node-1",
		IsActive:     true,
	}
	store.SaveIdentity("did:test:3", metadata3)

	node1Identities := store.FindIdentitiesByNode("node-1")
	if len(node1Identities) != 2 {
		t.Errorf("Expected 2 identities for node-1, got %d", len(node1Identities))
	}

	node2Identities := store.FindIdentitiesByNode("node-2")
	if len(node2Identities) != 1 {
		t.Errorf("Expected 1 identity for node-2, got %d", len(node2Identities))
	}
}

func TestIdentityStoreFindByProvider(t *testing.T) {
	dir := t.TempDir()
	store, err := NewIdentityStore(dir)
	if err != nil {
		t.Fatalf("Failed to create identity store: %v", err)
	}

	// Save identities for different providers
	metadata1 := IdentityMetadata{
		IdentityType: "agent",
		NodeID:       "node-1",
		Provider:     "openai",
		IsActive:     true,
	}
	store.SaveIdentity("did:test:1", metadata1)

	metadata2 := IdentityMetadata{
		IdentityType: "agent",
		NodeID:       "node-1",
		Provider:     "anthropic",
		IsActive:     true,
	}
	store.SaveIdentity("did:test:2", metadata2)

	openaiIdentities := store.FindIdentitiesByProvider("openai")
	if len(openaiIdentities) != 1 {
		t.Errorf("Expected 1 OpenAI identity, got %d", len(openaiIdentities))
	}

	anthropicIdentities := store.FindIdentitiesByProvider("anthropic")
	if len(anthropicIdentities) != 1 {
		t.Errorf("Expected 1 Anthropic identity, got %d", len(anthropicIdentities))
	}
}

func TestIdentityStoreFindByAPIKeyHash(t *testing.T) {
	dir := t.TempDir()
	store, err := NewIdentityStore(dir)
	if err != nil {
		t.Fatalf("Failed to create identity store: %v", err)
	}

	apiKey := "test-api-key-123"
	apiKeyHash := HashAPIKey(apiKey)

	metadata := IdentityMetadata{
		IdentityType: "agent",
		NodeID:       "node-1",
		AgentName:    "test-agent",
		APIKeyHash:   apiKeyHash,
		IsActive:     true,
	}
	store.SaveIdentity("did:test:1", metadata)

	// Find by API key hash
	identity := store.FindIdentityByAPIKeyHash(apiKeyHash, "test-agent", "node-1")
	if identity == nil {
		t.Fatal("Expected to find identity by API key hash")
	}

	if identity.DID != "did:test:1" {
		t.Errorf("Expected DID did:test:1, got %s", identity.DID)
	}

	// Should not find with wrong agent name
	identity = store.FindIdentityByAPIKeyHash(apiKeyHash, "wrong-agent", "node-1")
	if identity != nil {
		t.Error("Should not find identity with wrong agent name")
	}
}

func TestIdentityStoreMarkUsed(t *testing.T) {
	dir := t.TempDir()
	store, err := NewIdentityStore(dir)
	if err != nil {
		t.Fatalf("Failed to create identity store: %v", err)
	}

	did := "did:test:789"
	metadata := IdentityMetadata{
		IdentityType: "agent",
		NodeID:       "node-1",
		IsActive:     true,
	}

	err = store.SaveIdentity(did, metadata)
	if err != nil {
		t.Fatalf("Failed to save identity: %v", err)
	}

	// Mark as used
	err = store.MarkIdentityUsed(did)
	if err != nil {
		t.Fatalf("Failed to mark identity as used: %v", err)
	}

	identity, err := store.GetIdentity(did)
	if err != nil {
		t.Fatalf("Failed to get identity: %v", err)
	}

	if identity.Metadata.SessionCount != 2 { // Initial save + mark used
		t.Errorf("Expected session count 2, got %d", identity.Metadata.SessionCount)
	}
	if identity.Metadata.LastUsed.IsZero() {
		t.Error("LastUsed should be updated")
	}
}

func TestIdentityStoreActivateDeactivate(t *testing.T) {
	dir := t.TempDir()
	store, err := NewIdentityStore(dir)
	if err != nil {
		t.Fatalf("Failed to create identity store: %v", err)
	}

	did := "did:test:999"
	metadata := IdentityMetadata{
		IdentityType: "agent",
		NodeID:       "node-1",
		IsActive:     true,
	}

	err = store.SaveIdentity(did, metadata)
	if err != nil {
		t.Fatalf("Failed to save identity: %v", err)
	}

	// Deactivate
	err = store.DeactivateIdentity(did)
	if err != nil {
		t.Fatalf("Failed to deactivate identity: %v", err)
	}

	identity, err := store.GetIdentity(did)
	if err != nil {
		t.Fatalf("Failed to get identity: %v", err)
	}

	if identity.Metadata.IsActive {
		t.Error("Identity should be deactivated")
	}

	// Activate
	err = store.ActivateIdentity(did)
	if err != nil {
		t.Fatalf("Failed to activate identity: %v", err)
	}

	identity, err = store.GetIdentity(did)
	if err != nil {
		t.Fatalf("Failed to get identity: %v", err)
	}

	if !identity.Metadata.IsActive {
		t.Error("Identity should be activated")
	}
}

func TestIdentityStoreDelete(t *testing.T) {
	dir := t.TempDir()
	store, err := NewIdentityStore(dir)
	if err != nil {
		t.Fatalf("Failed to create identity store: %v", err)
	}

	did := "did:test:delete"
	metadata := IdentityMetadata{
		IdentityType: "agent",
		NodeID:       "node-1",
		IsActive:     true,
	}

	err = store.SaveIdentity(did, metadata)
	if err != nil {
		t.Fatalf("Failed to save identity: %v", err)
	}

	// Delete
	err = store.DeleteIdentity(did)
	if err != nil {
		t.Fatalf("Failed to delete identity: %v", err)
	}

	// Should not exist
	_, err = store.GetIdentity(did)
	if err == nil {
		t.Error("Identity should be deleted")
	}

	// File should be deleted
	safeDID := strings.ReplaceAll(did, ":", "_")
	filePath := filepath.Join(dir, safeDID+".json")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("Identity file should be deleted")
	}
}

func TestIdentityStoreListAll(t *testing.T) {
	dir := t.TempDir()
	store, err := NewIdentityStore(dir)
	if err != nil {
		t.Fatalf("Failed to create identity store: %v", err)
	}

	// Save multiple identities
	for i := 0; i < 5; i++ {
		metadata := IdentityMetadata{
			IdentityType: "agent",
			NodeID:       "node-1",
			IsActive:     true,
		}
		store.SaveIdentity("did:test:list-"+string(rune('a'+i)), metadata)
	}

	allIdentities := store.ListAllIdentities()
	if len(allIdentities) != 5 {
		t.Errorf("Expected 5 identities, got %d", len(allIdentities))
	}
}

func TestIdentityStoreGetActiveCount(t *testing.T) {
	dir := t.TempDir()
	store, err := NewIdentityStore(dir)
	if err != nil {
		t.Fatalf("Failed to create identity store: %v", err)
	}

	// Save active and inactive identities
	for i := 0; i < 3; i++ {
		metadata := IdentityMetadata{
			IdentityType: "agent",
			NodeID:       "node-1",
			IsActive:     true,
		}
		store.SaveIdentity("did:test:active-"+string(rune('a'+i)), metadata)
	}

	for i := 0; i < 2; i++ {
		metadata := IdentityMetadata{
			IdentityType: "agent",
			NodeID:       "node-1",
			IsActive:     false,
		}
		store.SaveIdentity("did:test:inactive-"+string(rune('a'+i)), metadata)
	}

	activeCount := store.GetActiveIdentitiesCount()
	if activeCount != 3 {
		t.Errorf("Expected 3 active identities, got %d", activeCount)
	}
}

func TestHashAPIKey(t *testing.T) {
	apiKey := "test-api-key"
	hash1 := HashAPIKey(apiKey)
	hash2 := HashAPIKey(apiKey)

	if hash1 != hash2 {
		t.Error("Hash should be consistent")
	}

	hash3 := HashAPIKey("different-api-key")
	if hash1 == hash3 {
		t.Error("Different API keys should have different hashes")
	}
}
