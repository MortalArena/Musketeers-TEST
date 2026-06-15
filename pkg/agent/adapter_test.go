package agent

import (
	"context"
	"testing"
	"time"
)

// MockAgent وكيل تجريبي للاختبار
type MockAgent struct {
	info      *AgentInfo
	status    *AgentStatus
	available bool
}

func NewMockAgent() *MockAgent {
	return &MockAgent{
		info: &AgentInfo{
			ID:            "mock-agent-1",
			Name:          "Mock Agent",
			Type:          AgentTypeAPI,
			Provider:      "mock",
			Model:         "mock-model",
			Version:       "1.0.0",
			MaxTokens:     4096,
			ContextWindow: 8192,
			CreatedAt:     time.Now(),
		},
		status: &AgentStatus{
			IsAvailable:  true,
			Load:         0,
			LastSeen:     time.Now(),
			ResponseTime: 100 * time.Millisecond,
			SuccessRate:  1.0,
			TotalTasks:   0,
			FailedTasks:  0,
		},
		available: true,
	}
}

func (ma *MockAgent) GetInfo() *AgentInfo {
	return ma.info
}

func (ma *MockAgent) SendMessage(ctx context.Context, prompt string) (*AgentResponse, error) {
	return &AgentResponse{
		Content:  "Mock response",
		Tokens:   10,
		Duration: 50 * time.Millisecond,
	}, nil
}

func (ma *MockAgent) ExecuteTask(ctx context.Context, task *AgentTask) (*TaskExecutionResult, error) {
	return &TaskExecutionResult{
		Success:  true,
		Output:   "Task completed",
		Duration: 100 * time.Millisecond,
	}, nil
}

func (ma *MockAgent) GetCapabilities() []AgentCapability {
	return []AgentCapability{
		CapabilityCodeGeneration,
		CapabilityCodeReview,
	}
}

func (ma *MockAgent) GetStatus() *AgentStatus {
	return ma.status
}

func (ma *MockAgent) IsAvailable() bool {
	return ma.available
}

func (ma *MockAgent) Close() error {
	ma.available = false
	ma.status.IsAvailable = false
	return nil
}

func TestMockAgentGetInfo(t *testing.T) {
	agent := NewMockAgent()
	info := agent.GetInfo()

	if info.ID != "mock-agent-1" {
		t.Errorf("Expected ID 'mock-agent-1', got '%s'", info.ID)
	}
	if info.Type != AgentTypeAPI {
		t.Errorf("Expected Type 'api', got '%s'", info.Type)
	}
}

func TestMockAgentSendMessage(t *testing.T) {
	agent := NewMockAgent()
	ctx := context.Background()

	response, err := agent.SendMessage(ctx, "test prompt")
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if response.Content != "Mock response" {
		t.Errorf("Expected content 'Mock response', got '%s'", response.Content)
	}
}

func TestMockAgentExecuteTask(t *testing.T) {
	agent := NewMockAgent()
	ctx := context.Background()

	task := &AgentTask{
		ID:          "task-1",
		Title:       "Test Task",
		Description: "Test description",
	}

	result, err := agent.ExecuteTask(ctx, task)
	if err != nil {
		t.Fatalf("ExecuteTask failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success to be true")
	}
}

func TestMockAgentGetCapabilities(t *testing.T) {
	agent := NewMockAgent()
	capabilities := agent.GetCapabilities()

	if len(capabilities) != 2 {
		t.Errorf("Expected 2 capabilities, got %d", len(capabilities))
	}
}

func TestMockAgentIsAvailable(t *testing.T) {
	agent := NewMockAgent()

	if !agent.IsAvailable() {
		t.Error("Expected agent to be available")
	}

	agent.Close()
	if agent.IsAvailable() {
		t.Error("Expected agent to be unavailable after Close")
	}
}

func TestAgentTypeConstants(t *testing.T) {
	if AgentTypeAPI != "api" {
		t.Errorf("Expected AgentTypeAPI to be 'api', got '%s'", AgentTypeAPI)
	}
	if AgentTypeCLI != "cli" {
		t.Errorf("Expected AgentTypeCLI to be 'cli', got '%s'", AgentTypeCLI)
	}
	if AgentTypeIDE != "ide" {
		t.Errorf("Expected AgentTypeIDE to be 'ide', got '%s'", AgentTypeIDE)
	}
}

func TestAgentCapabilityConstants(t *testing.T) {
	if CapabilityCodeGeneration != "code_generation" {
		t.Errorf("Expected CapabilityCodeGeneration to be 'code_generation', got '%s'", CapabilityCodeGeneration)
	}
	if CapabilityCodeReview != "code_review" {
		t.Errorf("Expected CapabilityCodeReview to be 'code_review', got '%s'", CapabilityCodeReview)
	}
}
