package identity

import (
	"testing"
)

func TestIdentityManagerCreateOrUpdate(t *testing.T) {
	dir := t.TempDir()
	manager, err := NewIdentityManager(dir)
	if err != nil {
		t.Fatalf("Failed to create identity manager: %v", err)
	}

	did := "did:test:manager-1"
	nodeID := "node-1"

	// Create new identity
	identity, err := manager.CreateOrUpdateIdentity(did, nodeID, "agent", "test-agent", "1.0.0", "openai", "test-api-key")
	if err != nil {
		t.Fatalf("Failed to create identity: %v", err)
	}

	if identity.DID != did {
		t.Errorf("Expected DID %s, got %s", did, identity.DID)
	}
	if identity.Metadata.AgentName != "test-agent" {
		t.Errorf("Expected agent name test-agent, got %s", identity.Metadata.AgentName)
	}
	if identity.Metadata.APIKeyHash != HashAPIKey("test-api-key") {
		t.Error("API key hash mismatch")
	}

	// Update existing identity
	identity, err = manager.CreateOrUpdateIdentity(did, nodeID, "agent", "test-agent", "2.0.0", "openai", "new-api-key")
	if err != nil {
		t.Fatalf("Failed to update identity: %v", err)
	}

	if identity.Metadata.AgentVersion != "2.0.0" {
		t.Errorf("Expected version 2.0.0, got %s", identity.Metadata.AgentVersion)
	}
	if identity.Metadata.SessionCount != 2 {
		t.Errorf("Expected session count 2, got %d", identity.Metadata.SessionCount)
	}
}

func TestIdentityManagerFindByIdentityType(t *testing.T) {
	dir := t.TempDir()
	manager, err := NewIdentityManager(dir)
	if err != nil {
		t.Fatalf("Failed to create identity manager: %v", err)
	}

	// Create multiple identities from different nodes
	for i := 0; i < 3; i++ {
		did := "did:test:human-" + string(rune('a'+i))
		nodeID := "node-" + string(rune('a'+i))
		_, err := manager.CreateOrUpdateIdentity(did, nodeID, "human", "", "", "", "")
		if err != nil {
			t.Fatalf("Failed to create human identity: %v", err)
		}
	}

	for i := 0; i < 5; i++ {
		did := "did:test:agent-" + string(rune('a'+i))
		nodeID := "agent-node-" + string(rune('a'+i))
		_, err := manager.CreateOrUpdateIdentity(did, nodeID, "agent", "agent-"+string(rune('a'+i)), "1.0.0", "openai", "api-key")
		if err != nil {
			t.Fatalf("Failed to create agent identity: %v", err)
		}
	}

	humans := manager.FindIdentitiesByType("human")
	if len(humans) != 3 {
		t.Errorf("Expected 3 human identities, got %d", len(humans))
	}

	agents := manager.FindIdentitiesByType("agent")
	if len(agents) != 5 {
		t.Errorf("Expected 5 agent identities, got %d", len(agents))
	}
}

func TestIdentityManagerFindIdentityForAgent(t *testing.T) {
	dir := t.TempDir()
	manager, err := NewIdentityManager(dir)
	if err != nil {
		t.Fatalf("Failed to create identity manager: %v", err)
	}

	did := "did:test:agent-find"
	_, err = manager.CreateOrUpdateIdentity(did, "node-1", "agent", "my-agent", "1.0.0", "openai", "api-key")
	if err != nil {
		t.Fatalf("Failed to create identity: %v", err)
	}

	// Find by agent name and provider
	identity := manager.FindIdentityForAgent("node-1", "my-agent", "openai")
	if identity == nil {
		t.Fatal("Expected to find identity for agent")
	}

	if identity.DID != did {
		t.Errorf("Expected DID %s, got %s", did, identity.DID)
	}

	// Should not find with wrong provider
	identity = manager.FindIdentityForAgent("node-1", "my-agent", "anthropic")
	if identity != nil {
		t.Error("Should not find identity with wrong provider")
	}
}

func TestIdentityManagerFindIdentityForAPIKey(t *testing.T) {
	dir := t.TempDir()
	manager, err := NewIdentityManager(dir)
	if err != nil {
		t.Fatalf("Failed to create identity manager: %v", err)
	}

	apiKey := "test-api-key-123"
	did := "did:test:api-key-find"
	_, err = manager.CreateOrUpdateIdentity(did, "node-1", "agent", "my-agent", "1.0.0", "openai", apiKey)
	if err != nil {
		t.Fatalf("Failed to create identity: %v", err)
	}

	// Find by API key
	identity := manager.FindIdentityForAPIKey(apiKey, "my-agent", "node-1")
	if identity == nil {
		t.Fatal("Expected to find identity by API key")
	}

	if identity.DID != did {
		t.Errorf("Expected DID %s, got %s", did, identity.DID)
	}

	// Should not find with wrong API key
	identity = manager.FindIdentityForAPIKey("wrong-api-key", "my-agent", "node-1")
	if identity != nil {
		t.Error("Should not find identity with wrong API key")
	}
}

func TestIdentityManagerUpdateAPIKey(t *testing.T) {
	dir := t.TempDir()
	manager, err := NewIdentityManager(dir)
	if err != nil {
		t.Fatalf("Failed to create identity manager: %v", err)
	}

	did := "did:test:update-api"
	_, err = manager.CreateOrUpdateIdentity(did, "node-1", "agent", "my-agent", "1.0.0", "openai", "old-api-key")
	if err != nil {
		t.Fatalf("Failed to create identity: %v", err)
	}

	// Update API key
	err = manager.UpdateAPIKey(did, "new-api-key")
	if err != nil {
		t.Fatalf("Failed to update API key: %v", err)
	}

	identity, err := manager.GetIdentity(did)
	if err != nil {
		t.Fatalf("Failed to get identity: %v", err)
	}

	if identity.Metadata.APIKeyHash != HashAPIKey("new-api-key") {
		t.Error("API key hash should be updated")
	}
}

func TestIdentityManagerUpdateAgentVersion(t *testing.T) {
	dir := t.TempDir()
	manager, err := NewIdentityManager(dir)
	if err != nil {
		t.Fatalf("Failed to create identity manager: %v", err)
	}

	did := "did:test:update-version"
	_, err = manager.CreateOrUpdateIdentity(did, "node-1", "agent", "my-agent", "1.0.0", "openai", "api-key")
	if err != nil {
		t.Fatalf("Failed to create identity: %v", err)
	}

	// Update version
	err = manager.UpdateAgentVersion(did, "2.0.0")
	if err != nil {
		t.Fatalf("Failed to update version: %v", err)
	}

	identity, err := manager.GetIdentity(did)
	if err != nil {
		t.Fatalf("Failed to get identity: %v", err)
	}

	if identity.Metadata.AgentVersion != "2.0.0" {
		t.Error("Agent version should be updated")
	}
}

func TestIdentityManagerLimits(t *testing.T) {
	dir := t.TempDir()
	manager, err := NewIdentityManager(dir)
	if err != nil {
		t.Fatalf("Failed to create identity manager: %v", err)
	}

	// Get default limits
	maxHuman, maxAgent := manager.GetIdentityLimits()
	if maxHuman != 8 {
		t.Errorf("Expected max human limit 8, got %d", maxHuman)
	}
	if maxAgent != 128 {
		t.Errorf("Expected max agent limit 128, got %d", maxAgent)
	}

	// Set new limits
	manager.SetIdentityLimits(16, 256)

	maxHuman, maxAgent = manager.GetIdentityLimits()
	if maxHuman != 16 {
		t.Errorf("Expected max human limit 16, got %d", maxHuman)
	}
	if maxAgent != 256 {
		t.Errorf("Expected max agent limit 256, got %d", maxAgent)
	}
}

func TestIdentityManagerIdentityReuse(t *testing.T) {
	dir := t.TempDir()
	manager, err := NewIdentityManager(dir)
	if err != nil {
		t.Fatalf("Failed to create identity manager: %v", err)
	}

	apiKey := "test-api-key-reuse"
	did1 := "did:test:reuse-1"

	// Create first identity
	_, err = manager.CreateOrUpdateIdentity(did1, "node-1", "agent", "my-agent", "1.0.0", "openai", apiKey)
	if err != nil {
		t.Fatalf("Failed to create first identity: %v", err)
	}

	// Try to create second identity with same API key from different node (should create new identity)
	did2 := "did:test:reuse-2"
	identity, err := manager.CreateOrUpdateIdentity(did2, "node-2", "agent", "my-agent", "1.0.0", "openai", apiKey)
	if err != nil {
		t.Fatalf("Failed to create second identity: %v", err)
	}

	// Should have created a new identity (different node)
	if identity.DID != did2 {
		t.Errorf("Expected DID %s, got %s", did2, identity.DID)
	}
	if identity.Metadata.APIKeyHash != HashAPIKey(apiKey) {
		t.Error("API key hash should match")
	}
}
