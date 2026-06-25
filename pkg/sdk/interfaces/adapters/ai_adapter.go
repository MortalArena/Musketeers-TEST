package adapters

import (
	"context"

	"github.com/MortalArena/Musketeers/pkg/providers"
	"github.com/MortalArena/Musketeers/pkg/sdk/interfaces"
)

type AIAdapter struct {
	provider providers.Provider
}

func NewAIAdapter(p providers.Provider) *AIAdapter {
	return &AIAdapter{provider: p}
}

func (a *AIAdapter) Complete(ctx context.Context, req *interfaces.AIRequest) (*interfaces.AIResponse, error) {
	messages := make([]providers.Message, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = providers.Message{Role: providers.MessageRole(m.Role), Content: m.Content}
	}
	pReq := &providers.CompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      req.Stream,
	}
	pResp, err := a.provider.Complete(ctx, pReq)
	if err != nil {
		return nil, err
	}
	resp := &interfaces.AIResponse{
		ID:      pResp.ID,
		Model:   pResp.Model,
		Content: pResp.Content,
		Usage: &interfaces.AIUsage{
			PromptTokens:     pResp.Usage.PromptTokens,
			CompletionTokens: pResp.Usage.CompletionTokens,
			TotalTokens:      pResp.Usage.TotalTokens,
		},
	}
	return resp, nil
}

func (a *AIAdapter) StreamComplete(ctx context.Context, req *interfaces.AIRequest, cb func(chunk *interfaces.AIStreamChunk)) error {
	messages := make([]providers.Message, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = providers.Message{Role: providers.MessageRole(m.Role), Content: m.Content}
	}
	pReq := &providers.CompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      true,
	}
	return a.provider.StreamComplete(ctx, pReq, func(chunk providers.StreamChunk) error {
		cb(&interfaces.AIStreamChunk{
			Delta:        chunk.Delta,
			FinishReason: chunk.FinishReason,
		})
		return nil
	})
}

func (a *AIAdapter) ListModels(ctx context.Context) ([]*interfaces.AIModel, error) {
	models, err := a.provider.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*interfaces.AIModel, len(models))
	for i, m := range models {
		result[i] = &interfaces.AIModel{
			ID:            m.ID,
			Name:          m.Name,
			ContextLength: m.ContextLength,
			IsAvailable:   m.IsAvailable,
		}
	}
	return result, nil
}

func (a *AIAdapter) Name() string {
	return a.provider.Name()
}

func (a *AIAdapter) IsAvailable() bool {
	return a.provider.IsAvailable()
}

var _ interfaces.AIInterface = (*AIAdapter)(nil)
