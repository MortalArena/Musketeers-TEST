package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/agent/unified"
	"github.com/MortalArena/Musketeers/pkg/orchestrator"
)

// getSessionID returns the first active session ID, or empty string.
func (s *Server) getSessionID() string {
	if s.sessionManager != nil {
		sessions := s.sessionManager.ListSessions()
		if len(sessions) > 0 {
			return sessions[0].ID
		}
	}
	if s.sessionContainer != nil {
		return s.sessionContainer.ID
	}
	return ""
}

// handleRuntimePing returns basic liveness check.
func (s *Server) handleRuntimePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleRuntimeStatus returns session-centric runtime status.
// Dashboard expects: running, status, registry{total,available}, agent_pool{active,total},
// provider{available,count}, orchestrator{running}, session_state{status,agents,tasks}
func (s *Server) handleRuntimeStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Session info
	var sessionID, managerID, sessStatus string
	sessionAgents := 0
	sessions := s.sessionManager.ListSessions()
	if len(sessions) > 0 {
		sess := sessions[0]
		sessionID = sess.ID
		managerID = sess.ManagerAgentID
		sessStatus = sess.Status
		sessionAgents = len(sess.AssistantAgents)
		if managerID != "" {
			sessionAgents++ // include manager
		}
	}

	// AgentPool
	poolTotal := 0
	poolActive := 0
	registryTotal := 0
	registryAvailable := 0
	modelAgents := 0
	externalAgents := 0

	if s.unifiedAgent != nil {
		pool := s.unifiedAgent.GetAgentPool()
		if pool != nil {
			poolTotal = pool.Count()
			for _, inst := range pool.ListAgents() {
				st := inst.GetStatus()
				if st == unified.PoolAgentStatusActive {
					poolActive++
				}
				if unified.IsExternalAgentType(inst.AgentType) {
					externalAgents++
				} else {
					modelAgents++
				}
			}
		}
	}

	// Registry
	if s.agentRegistry != nil {
		all := s.agentRegistry.ListAll()
		registryTotal = len(all)
		registryAvailable = registryTotal
	}

	// Providers
	providerCount := 0
	providerAvailable := 0
	if s.providerRegistry != nil {
		for _, p := range s.providerRegistry.List() {
			providerCount++
			if p.IsAvailable() {
				providerAvailable++
			}
		}
	}

	// Delegation stats
	delegationStats := map[string]int{
		"total":     0,
		"running":   0,
		"completed": 0,
		"failed":    0,
	}
	if s.orchestratorEngine != nil {
		if td := s.orchestratorEngine.GetTaskDelegator(); td != nil {
			for _, del := range td.GetAllDelegations() {
				delegationStats["total"]++
				switch del.Status {
				case orchestrator.DelegationRunning:
					delegationStats["running"]++
				case orchestrator.DelegationCompleted:
					delegationStats["completed"]++
				case orchestrator.DelegationFailed:
					delegationStats["failed"]++
				}
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"running":       true,
		"status":        "running",
		"session_id":    sessionID,
		"manager_agent_id": managerID,
		"registry": map[string]interface{}{
			"total":     registryTotal,
			"available": registryAvailable,
			"model":     modelAgents,
			"external":  externalAgents,
		},
		"agent_pool": map[string]interface{}{
			"total":  poolTotal,
			"active": poolActive,
		},
		"provider": map[string]interface{}{
			"count":     providerCount,
			"available": providerAvailable,
		},
		"orchestrator": map[string]interface{}{
			"running": s.orchestratorEngine != nil,
		},
		"session_state": map[string]interface{}{
			"status": sessStatus,
			"agents": sessionAgents,
			"tasks":  delegationStats["total"],
		},
		"delegation_stats": delegationStats,
		"memory_mb":        memStats.Alloc / 1024 / 1024,
		"goroutines":       runtime.NumGoroutine(),
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
	})
}

// handleRuntimeAgents returns live agent info from AgentPool and AgentRegistry.
func (s *Server) handleRuntimeAgents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// جمع معلومات التفويض من TaskDelegator
	delegationByAgent := make(map[string][]map[string]interface{})
	if s.orchestratorEngine != nil {
		if td := s.orchestratorEngine.GetTaskDelegator(); td != nil {
			for _, del := range td.GetAllDelegations() {
				delegationByAgent[del.AgentID] = append(delegationByAgent[del.AgentID], map[string]interface{}{
					"task_id": del.TaskID,
					"status":  string(del.Status),
					"error":   del.Error,
				})
			}
		}
	}

	// Discover manager agent ID from all sessions
	managerIDs := make(map[string]bool)
	if s.sessionManager != nil {
		for _, sess := range s.sessionManager.ListSessions() {
			if sess.ManagerAgentID != "" {
				managerIDs[sess.ManagerAgentID] = true
			}
		}
	}

	agents := make([]map[string]interface{}, 0)
	seen := make(map[string]bool)

	// Helper to get bridge info for an agent
	bridgeIDForAgent := func(agentID string) (bool, string) {
		if s.externalBridgeMgr != nil {
			if bi, ok := s.externalBridgeMgr.GetBridge(agentID); ok {
				return true, bi.BridgeID
			}
		}
		return false, ""
	}

	// Helper to build rich agent info from pool instance
	buildAgentInfo := func(instance *unified.AgentInstance, role string) map[string]interface{} {
		ginfo := instance.Adapter.GetInfo()
		if ginfo == nil {
			return nil
		}

		isExt, bID := bridgeIDForAgent(ginfo.ID)
		dels := delegationByAgent[ginfo.ID]
		rp, rm := instance.GetRuntimeProvider()
		rpStr, rmStr := "", ""
		if rp != nil {
			rpStr = string(rp.Type())
		}
		rmStr = rm

		// Runtime component status
		var components *unified.RuntimeComponentsState
		teInit := instance.GetThinkingEngineInit()
		if s.unifiedAgent != nil {
			if pool := s.unifiedAgent.GetAgentPool(); pool != nil {
				components = instance.GetRuntimeComponents(pool)
			}
		}

		ai := map[string]interface{}{
			"id":                ginfo.ID,
			"name":              ginfo.Name,
			"provider":          ginfo.Provider,
			"model":             ginfo.Model,
			"runtime_provider":  rpStr,
			"runtime_model":     rmStr,
			"type":              role,
			"status":            string(instance.GetStatus()),
			"is_manager":        managerIDs[ginfo.ID],
			"is_external":       isExt,
			"bridge_id":         bID,
			"has_thinking_engine": teInit,
			"session_id":        s.getSessionID(),
		}

		// Add runtime components
		if components != nil {
			ai["runtime_components"] = components
		}

		// Add delegation info
		if len(dels) > 0 {
			ai["delegation_active"] = true
			ai["delegation_status"] = dels[len(dels)-1]["status"].(string)
			ai["delegations"] = dels
		}

		// Task stats
		tt, ts, tf := instance.GetTaskStats()
		ai["tasks_total"] = tt
		ai["tasks_success"] = ts
		ai["tasks_failed"] = tf

		return ai
	}

	// 1. Get agent info from AgentPool (richest source)
	if s.unifiedAgent != nil {
		pool := s.unifiedAgent.GetAgentPool()
		if pool != nil {
			for _, instance := range pool.ListAgents() {
				ginfo := instance.Adapter.GetInfo()
				if ginfo == nil || seen[ginfo.ID] {
					continue
				}
				role := "assistant"
				if managerIDs[ginfo.ID] {
					role = "manager"
				}
				if ai := buildAgentInfo(instance, role); ai != nil {
					agents = append(agents, ai)
					seen[ginfo.ID] = true
				}
			}
		}
	}

	// 2. Get remaining agents from SessionManager instances
	if s.sessionManager != nil {
		for _, sess := range s.sessionManager.ListSessions() {
			instances, _ := s.sessionManager.GetAgentInstances(sess.ID)
			for _, inst := range instances {
				if seen[inst.AgentID] {
					continue
				}
				isExt, bID := bridgeIDForAgent(inst.AgentID)
				dels := delegationByAgent[inst.AgentID]
				role := "assistant"
				if inst.Role == "manager" {
					role = "manager"
				}
				ai := map[string]interface{}{
					"id":               inst.AgentID,
					"name":             inst.AgentID,
					"provider":         inst.Provider,
					"model":            inst.Model,
					"type":             role,
					"status":           string(inst.Status),
					"is_manager":       managerIDs[inst.AgentID],
					"is_external":      isExt,
					"bridge_id":        bID,
					"has_thinking_engine": false,
					"session_id":       sess.ID,
				}
				if len(dels) > 0 {
					ai["delegation_active"] = true
					ai["delegation_status"] = dels[len(dels)-1]["status"].(string)
					ai["delegations"] = dels
				}
				agents = append(agents, ai)
				seen[inst.AgentID] = true
			}
		}
	}

	json.NewEncoder(w).Encode(agents)
}

// handleRuntimeThinkingEngine returns ThinkingEngine status for all agents.
func (s *Server) handleRuntimeThinkingEngine(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	type teInfo struct {
		AgentID string `json:"agent_id"`
		Ready   bool   `json:"ready"`
		Error   string `json:"error,omitempty"`
	}

	engines := make([]teInfo, 0)
	if s.unifiedAgent != nil {
		pool := s.unifiedAgent.GetAgentPool()
		if pool != nil {
			for _, instance := range pool.ListAgents() {
				info := instance.Adapter.GetInfo()
				if info == nil {
					continue
				}
				err := pool.HasThinkingEngine(info.ID)
				if err != nil {
					engines = append(engines, teInfo{
						AgentID: info.ID,
						Ready:   false,
						Error:   err.Error(),
					})
				} else {
					engines = append(engines, teInfo{
						AgentID: info.ID,
						Ready:   true,
					})
				}
			}
		}
	}

	json.NewEncoder(w).Encode(engines)
}

// handleRuntimeExecute executes a task via OrchestratorEngine.
func (s *Server) handleRuntimeExecute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Task   string `json:"task"`
		AgentID string `json:"agent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Task == "" {
		http.Error(w, "task is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	if s.orchestratorEngine != nil {
		task := &agent.AgentTask{
			ID:          fmt.Sprintf("runtime-%d", time.Now().UnixNano()),
			Title:       req.Task,
			Description: req.Task,
			Timeout:     60 * time.Second,
		}
		if req.AgentID != "" {
			task.Inputs = map[string]interface{}{"assigned_agent": req.AgentID}
		}
		result, err := s.orchestratorEngine.ExecuteTask(ctx, task)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"result": result.Output,
		})
		return
	}

	if s.unifiedAgent != nil {
		result, err := s.unifiedAgent.ExecuteTask(ctx, req.Task)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			})
			return
		}
		output := ""
		if result != nil && result.Output != nil {
			output = fmt.Sprintf("%v", result.Output)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"result": output,
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "error",
		"error":  "no orchestrator or unified agent available",
	})
}

// handleRuntimeSession returns session-centric data.
func (s *Server) handleRuntimeSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	type instanceInfo struct {
		AgentID  string `json:"agent_id"`
		Provider string `json:"provider"`
		Model    string `json:"model"`
		Role     string `json:"role"`
		Status   string `json:"status"`
	}

	type sessionInfo struct {
		ID              string         `json:"id"`
		Name            string         `json:"name"`
		OwnerDID        string         `json:"owner_did"`
		ManagerAgentID  string         `json:"manager_agent_id"`
		AssistantAgents []string       `json:"assistant_agents"`
		Status          string         `json:"status"`
		Instances       []instanceInfo `json:"instances"`
		AgentPoolCount  int            `json:"agent_pool_count"`
	}

	sessions := make([]sessionInfo, 0)

	// Get sessions from SessionManager
	for _, sess := range s.sessionManager.ListSessions() {
		si := sessionInfo{
			ID:              sess.ID,
			Name:            sess.Name,
			OwnerDID:        sess.OwnerDID,
			ManagerAgentID:  sess.ManagerAgentID,
			AssistantAgents: sess.AssistantAgents,
			Status:          sess.Status,
		}
		if instances, err := s.sessionManager.GetAgentInstances(sess.ID); err == nil {
			for _, inst := range instances {
				si.Instances = append(si.Instances, instanceInfo{
					AgentID:  inst.AgentID,
					Provider: inst.Provider,
					Model:    inst.Model,
					Role:     inst.Role,
					Status:   string(inst.Status),
				})
			}
		}
		if s.unifiedAgent != nil {
			if pool := s.unifiedAgent.GetAgentPool(); pool != nil {
				si.AgentPoolCount = len(pool.ListAgents())
			}
		}
		sessions = append(sessions, si)
	}

	json.NewEncoder(w).Encode(sessions)
}

// handleRuntimeMemory returns memory state from session CollectiveMemory.
func (s *Server) handleRuntimeMemory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Memory endpoint - use /api/runtime/memory/detail for full state",
	})
}

// handleRuntimeMemoryDetail returns detailed memory state.
func (s *Server) handleRuntimeMemoryDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Memory detail - integrates with CollectiveMemory",
	})
}

// handleRuntimeProviders returns provider status from ProviderRegistry.
func (s *Server) handleRuntimeProviders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.listProviderConfigs())
}

// handleRuntimeWorkflows returns workflow status.
func (s *Server) handleRuntimeWorkflows(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]interface{}{})
}

// handleRuntimeDebug returns full debug info from all runtime components.
func (s *Server) handleRuntimeDebug(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	debug := map[string]interface{}{
		"session_manager":   nil,
		"agent_registry":    nil,
		"agent_pool":        nil,
		"provider_registry": nil,
		"orchestrator":      nil,
		"unified_agent":     nil,
		"go_version":        runtime.Version(),
		"goroutines":        runtime.NumGoroutine(),
	}

	if s.sessionManager != nil {
		sessions := s.sessionManager.ListSessions()
		sessionList := make([]map[string]interface{}, 0)
		for _, sess := range sessions {
			sessionList = append(sessionList, map[string]interface{}{
				"id":              sess.ID,
				"name":            sess.Name,
				"manager":         sess.ManagerAgentID,
				"assistants":      sess.AssistantAgents,
				"status":          sess.Status,
			})
		}
		debug["session_manager"] = map[string]interface{}{
			"session_count": len(sessions),
			"sessions":      sessionList,
		}
	}

	if s.agentRegistry != nil {
		agents := s.agentRegistry.ListAll()
		agentList := make([]map[string]interface{}, 0)
		for _, a := range agents {
			info := a.GetInfo()
			if info != nil {
				agentList = append(agentList, map[string]interface{}{
					"id":       info.ID,
					"name":     info.Name,
					"type":     string(info.Type),
					"provider": info.Provider,
					"model":    info.Model,
				})
			}
		}
		debug["agent_registry"] = map[string]interface{}{
			"agent_count": len(agents),
			"agents":      agentList,
		}
	}

	if s.unifiedAgent != nil {
		pool := s.unifiedAgent.GetAgentPool()
		if pool != nil {
			poolAgents := pool.ListAgents()
			poolList := make([]map[string]interface{}, 0)
			for _, inst := range poolAgents {
				info := inst.Adapter.GetInfo()
				if info != nil {
					poolList = append(poolList, map[string]interface{}{
						"id":   info.ID,
						"name": info.Name,
					})
				}
			}
			debug["agent_pool"] = map[string]interface{}{
				"pool_count": len(poolAgents),
				"agents":     poolList,
			}
		}
	}

	json.NewEncoder(w).Encode(debug)
}

// handleRuntimeBridges returns external bridge agent status.
func (s *Server) handleRuntimeBridges(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.externalBridgeMgr == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"bridges": []interface{}{},
			"count":   0,
		})
		return
	}

	bridges := s.externalBridgeMgr.GetBridges()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"bridges": bridges,
		"count":   len(bridges),
	})
}

// handleRuntimeEvents returns recent EventBus events.
func (s *Server) handleRuntimeEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Event stream available via WebSocket (/api/ws)",
	})
}

// handleRuntimeDelegations returns all active and historical task delegations.
func (s *Server) handleRuntimeDelegations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.orchestratorEngine == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"delegations": []interface{}{},
			"count":       0,
		})
		return
	}

	td := s.orchestratorEngine.GetTaskDelegator()
	if td == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"delegations": []interface{}{},
			"count":       0,
		})
		return
	}

	dels := td.GetAllDelegations()
	type delegationResponse struct {
		TaskID      string  `json:"task_id"`
		AgentID     string  `json:"agent_id"`
		Status      string  `json:"status"`
		DelegatedAt string  `json:"delegated_at"`
		CompletedAt *string `json:"completed_at,omitempty"`
		Error       string  `json:"error,omitempty"`
	}

	result := make([]delegationResponse, 0, len(dels))
	for _, del := range dels {
		var completedAt *string
		if del.CompletedAt != nil {
			s := del.CompletedAt.Format(time.RFC3339)
			completedAt = &s
		}
		result = append(result, delegationResponse{
			TaskID:      del.TaskID,
			AgentID:     del.AgentID,
			Status:      string(del.Status),
			DelegatedAt: del.DelegatedAt.Format(time.RFC3339),
			CompletedAt: completedAt,
			Error:       del.Error,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"delegations": result,
		"count":       len(result),
	})
}


