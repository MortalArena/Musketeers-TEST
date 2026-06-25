package adapters

import (
	"context"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/sdk/interfaces"
)

type AgentAdapter struct {
	agent agent.UnifiedAgent
}

func NewAgentAdapter(a agent.UnifiedAgent) *AgentAdapter {
	return &AgentAdapter{agent: a}
}

func (a *AgentAdapter) Info() *interfaces.AgentInfo {
	info := a.agent.GetInfo()
	return &interfaces.AgentInfo{
		ID:        info.ID,
		Name:      info.Name,
		Type:      string(info.Type),
		Provider:  info.Provider,
		Model:     info.Model,
		Version:   info.Version,
		CreatedAt: info.CreatedAt,
	}
}

func (a *AgentAdapter) SendMessage(ctx context.Context, prompt string) (*interfaces.AgentResponse, error) {
	resp, err := a.agent.SendMessage(ctx, prompt)
	if err != nil {
		return nil, err
	}
	return &interfaces.AgentResponse{
		Content:  resp.Content,
		Metadata: resp.Metadata,
	}, nil
}

func (a *AgentAdapter) ExecuteTask(ctx context.Context, task *interfaces.Task) (*interfaces.TaskResult, error) {
	agtTask := &agent.AgentTask{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Context:     task.Context,
		Inputs:      task.Inputs,
		Timeout:     task.Timeout,
	}
	result, err := a.agent.ExecuteTask(ctx, agtTask)
	if err != nil {
		return nil, err
	}
	return &interfaces.TaskResult{
		Success:  result.Success,
		Output:   result.Output,
		Error:    result.Error,
		Duration: result.Duration,
	}, nil
}

func (a *AgentAdapter) Capabilities() []string {
	caps := a.agent.GetCapabilities()
	strs := make([]string, len(caps))
	for i, c := range caps {
		strs[i] = string(c)
	}
	return strs
}

func (a *AgentAdapter) Status() *interfaces.AgentStatus {
	s := a.agent.GetStatus()
	return &interfaces.AgentStatus{
		IsAvailable: s.IsAvailable,
		CurrentTask: s.CurrentTask,
		Load:        s.Load,
		LastSeen:    s.LastSeen,
		SuccessRate: s.SuccessRate,
	}
}

func (a *AgentAdapter) IsAvailable() bool {
	return a.agent.IsAvailable()
}

func (a *AgentAdapter) Close() error {
	return a.agent.Close()
}

var _ interfaces.AgentInterface = (*AgentAdapter)(nil)
