package api

import (
	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/agent/unified"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/MortalArena/Musketeers/pkg/orchestrator"
	"github.com/MortalArena/Musketeers/pkg/providers"
	pkgRuntime "github.com/MortalArena/Musketeers/pkg/runtime"
	"github.com/MortalArena/Musketeers/pkg/session"
)

// ServerRuntime shares live studio/runtime components with the REST API layer.
// Wiring only — does not replace existing services.
type ServerRuntime struct {
	EventBus           *eventbus.EventBus
	SessionManager     *orchestrator.SessionManager
	BridgeManager      *session.SessionBridgeManager
	ProviderRegistry   *providers.ProviderRegistry
	APIKeyManager      *providers.APIKeyManager
	OwnerDID           string
	AgentRegistry      *agent.AgentRegistry
	UnifiedAgent       *unified.UnifiedAgent
	ExternalBridgeMgr  *unified.ExternalBridgeManager
	OrchestratorEngine *orchestrator.OrchestratorEngine
	SessionContainer   *session.SessionContainer
	ApplicationRuntime *pkgRuntime.ApplicationRuntime
}

// UseRuntime attaches shared runtime dependencies created by cmd/studio.
func (s *Server) UseRuntime(rt *ServerRuntime) {
	if s == nil || rt == nil {
		return
	}
	if rt.EventBus != nil {
		s.eventBus = rt.EventBus
	}
	if rt.SessionManager != nil {
		s.sessionManager = rt.SessionManager
	}
	if rt.BridgeManager != nil {
		s.bridgeManager = rt.BridgeManager
	}
	if rt.ProviderRegistry != nil {
		s.providerRegistry = rt.ProviderRegistry
	}
	if rt.APIKeyManager != nil {
		s.apiKeyManager = rt.APIKeyManager
	}
	if rt.OwnerDID != "" {
		s.ownerDID = rt.OwnerDID
	}
	if rt.AgentRegistry != nil {
		s.agentRegistry = rt.AgentRegistry
	}
	if rt.UnifiedAgent != nil {
		s.unifiedAgent = rt.UnifiedAgent
	}
	if rt.ExternalBridgeMgr != nil {
		s.externalBridgeMgr = rt.ExternalBridgeMgr
	}
	if rt.OrchestratorEngine != nil {
		s.orchestratorEngine = rt.OrchestratorEngine
	}
	if rt.SessionContainer != nil {
		s.sessionContainer = rt.SessionContainer
	}
	if rt.ApplicationRuntime != nil {
		s.applicationRuntime = rt.ApplicationRuntime
	}
}
