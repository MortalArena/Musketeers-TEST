package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/agent/tools"
	"github.com/MortalArena/Musketeers/pkg/agent/unified"
	"go.uber.org/zap"
)

type mockUnifiedAgent struct {
	info agent.AgentInfo
}

func (m *mockUnifiedAgent) GetInfo() *agent.AgentInfo { return &m.info }
func (m *mockUnifiedAgent) SendMessage(ctx context.Context, prompt string) (*agent.AgentResponse, error) {
	return &agent.AgentResponse{Content: "mock response"}, nil
}
func (m *mockUnifiedAgent) ExecuteTask(ctx context.Context, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	return &agent.TaskExecutionResult{Success: true, Output: "mock task result"}, nil
}
func (m *mockUnifiedAgent) GetCapabilities() []agent.AgentCapability { return nil }
func (m *mockUnifiedAgent) GetStatus() *agent.AgentStatus {
	return &agent.AgentStatus{IsAvailable: true}
}
func (m *mockUnifiedAgent) IsAvailable() bool { return true }
func (m *mockUnifiedAgent) Close() error      { return nil }

func TestTaskDelegator_SelectAgent_ModelPreferred(t *testing.T) {
	pool := unified.NewAgentPool("test-session",
		unified.AgentPoolConfig{MaxAgents: 10},
		tools.NewToolRegistry(),
		zap.NewNop())
	defer pool.Close()

	_, err := pool.RegisterAgent(&mockUnifiedAgent{
		info: agent.AgentInfo{
			ID:       "model-1",
			Name:     "Model Agent 1",
			Type:     agent.AgentTypeAPI,
			Provider: "test",
			Model:    "test-model",
		},
	}, tools.RoleRegular)
	if err != nil {
		t.Fatalf("failed to register model agent: %v", err)
	}

	_, err = pool.RegisterAgent(&mockUnifiedAgent{
		info: agent.AgentInfo{
			ID:       "cli-1",
			Name:     "CLI Agent",
			Type:     agent.AgentTypeCLI,
			Provider: "external",
			Model:    "",
		},
	}, tools.RoleRegular)
	if err != nil {
		t.Fatalf("failed to register external agent: %v", err)
	}

	td := NewTaskDelegator(pool, nil, zap.NewNop())

	task := &agent.AgentTask{ID: "task-1", Title: "test task"}
	agentID, err := td.SelectAgent(task)
	if err != nil {
		t.Fatalf("SelectAgent failed: %v", err)
	}
	if agentID != "model-1" {
		t.Fatalf("expected model-1, got %s", agentID)
	}
}

func TestTaskDelegator_SelectAgent_ExternalFallback(t *testing.T) {
	pool := unified.NewAgentPool("test-session",
		unified.AgentPoolConfig{MaxAgents: 10},
		tools.NewToolRegistry(),
		zap.NewNop())
	defer pool.Close()

	_, err := pool.RegisterAgent(&mockUnifiedAgent{
		info: agent.AgentInfo{
			ID:       "cli-1",
			Name:     "CLI Agent",
			Type:     agent.AgentTypeCLI,
			Provider: "external",
			Model:    "",
		},
	}, tools.RoleRegular)
	if err != nil {
		t.Fatalf("failed to register external agent: %v", err)
	}

	td := NewTaskDelegator(pool, nil, zap.NewNop())

	task := &agent.AgentTask{ID: "task-2", Title: "test"}
	agentID, err := td.SelectAgent(task)
	if err != nil {
		t.Fatalf("SelectAgent failed: %v", err)
	}
	if agentID != "cli-1" {
		t.Fatalf("expected cli-1, got %s", agentID)
	}
}

func TestTaskDelegator_SelectAgent_AssignedAgentPrefersExplicit(t *testing.T) {
	pool := unified.NewAgentPool("test-session",
		unified.AgentPoolConfig{MaxAgents: 10},
		tools.NewToolRegistry(),
		zap.NewNop())
	defer pool.Close()

	_, err := pool.RegisterAgent(&mockUnifiedAgent{
		info: agent.AgentInfo{
			ID:       "cli-1",
			Name:     "CLI Agent",
			Type:     agent.AgentTypeCLI,
			Provider: "external",
			Model:    "",
		},
	}, tools.RoleRegular)
	if err != nil {
		t.Fatalf("failed to register cli agent: %v", err)
	}

	_, err = pool.RegisterAgent(&mockUnifiedAgent{
		info: agent.AgentInfo{
			ID:       "model-1",
			Name:     "Model Agent",
			Type:     agent.AgentTypeAPI,
			Provider: "test",
			Model:    "test-model",
		},
	}, tools.RoleRegular)
	if err != nil {
		t.Fatalf("failed to register model agent: %v", err)
	}

	td := NewTaskDelegator(pool, nil, zap.NewNop())

	task := &agent.AgentTask{
		ID:     "task-3",
		Title:  "test",
		Inputs: map[string]interface{}{"assigned_agent": "cli-1"},
	}
	agentID, err := td.SelectAgent(task)
	if err != nil {
		t.Fatalf("SelectAgent failed: %v", err)
	}
	if agentID != "cli-1" {
		t.Fatalf("expected cli-1 (by assignment), got %s", agentID)
	}
}

func TestTaskDelegator_DelegateTask_TracksDelegation(t *testing.T) {
	pool := unified.NewAgentPool("test-session",
		unified.AgentPoolConfig{MaxAgents: 10},
		tools.NewToolRegistry(),
		zap.NewNop())
	defer pool.Close()

	_, err := pool.RegisterAgent(&mockUnifiedAgent{
		info: agent.AgentInfo{
			ID:       "cli-1",
			Name:     "CLI Agent",
			Type:     agent.AgentTypeCLI,
			Provider: "external",
			Model:    "",
		},
	}, tools.RoleRegular)
	if err != nil {
		t.Fatalf("failed to register external agent: %v", err)
	}

	eventBus := unified.NewSessionEventBus("test-session", zap.NewNop())
	td := NewTaskDelegator(pool, eventBus, zap.NewNop())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	task := &agent.AgentTask{ID: "task-event-1", Title: "test events"}

	agentID, err := td.SelectAgent(task)
	if err != nil {
		t.Fatalf("SelectAgent failed: %v", err)
	}

	oe := NewOrchestratorEngine(nil)
	td.DelegateTask(ctx, task, agentID, oe)

	dels := td.GetAllDelegations()
	if len(dels) != 1 {
		t.Fatalf("expected 1 delegation tracked, got %d", len(dels))
	}
	if dels[0].TaskID != "task-event-1" {
		t.Fatalf("expected task-event-1, got %s", dels[0].TaskID)
	}
	if dels[0].AgentID != agentID {
		t.Fatalf("expected agent %s, got %s", agentID, dels[0].AgentID)
	}
	t.Logf("Delegation tracked: task=%s agent=%s status=%s", dels[0].TaskID, dels[0].AgentID, dels[0].Status)
}

func TestTaskDelegator_GetDelegationsByAgent(t *testing.T) {
	td := NewTaskDelegator(nil, nil, zap.NewNop())

	td.mu.Lock()
	td.delegations["task-a"] = &AgentDelegation{
		TaskID:  "task-a",
		AgentID: "agent-1",
		Status:  DelegationCompleted,
	}
	td.delegations["task-b"] = &AgentDelegation{
		TaskID:  "task-b",
		AgentID: "agent-2",
		Status:  DelegationRunning,
	}
	td.delegations["task-c"] = &AgentDelegation{
		TaskID:  "task-c",
		AgentID: "agent-1",
		Status:  DelegationRunning,
	}
	td.mu.Unlock()

	dels := td.GetDelegationsByAgent("agent-1")
	if len(dels) != 2 {
		t.Fatalf("expected 2 delegations for agent-1, got %d", len(dels))
	}

	dels = td.GetDelegationsByAgent("agent-2")
	if len(dels) != 1 {
		t.Fatalf("expected 1 delegation for agent-2, got %d", len(dels))
	}

	all := td.GetAllDelegations()
	if len(all) != 3 {
		t.Fatalf("expected 3 total delegations, got %d", len(all))
	}
}

func TestTaskDelegator_SelectAgent_NoAgents(t *testing.T) {
	pool := unified.NewAgentPool("test-session",
		unified.AgentPoolConfig{MaxAgents: 10},
		tools.NewToolRegistry(),
		zap.NewNop())
	defer pool.Close()

	td := NewTaskDelegator(pool, nil, zap.NewNop())
	_, err := td.SelectAgent(&agent.AgentTask{ID: "task-empty", Title: "test"})
	if err == nil {
		t.Fatal("expected error when no agents in pool")
	}
}

func TestTaskDelegator_SelectAgent_NilPool(t *testing.T) {
	td := NewTaskDelegator(nil, nil, zap.NewNop())
	_, err := td.SelectAgent(&agent.AgentTask{ID: "task-nil", Title: "test"})
	if err == nil {
		t.Fatal("expected error when pool is nil")
	}
}
