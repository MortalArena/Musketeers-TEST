package runtime

import (
	"context"
	"fmt"

	"github.com/MortalArena/Musketeers/pkg/capability"
	"github.com/MortalArena/Musketeers/pkg/capability/pipeline"
	"github.com/MortalArena/Musketeers/pkg/policy"
)

type AgentMetadata struct {
	DID          string   `json:"did"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
	Tasks        []string `json:"tasks"`
}

type AgentContext interface {
	DID() string
	Metadata() *AgentMetadata
	Runtime() AgentRuntime
	Execute(cmd capability.Command) (*capability.Result, error)
	Notify(title, message string) error
	Context() context.Context
}

type AgentContextImpl struct {
	did      string
	metadata *AgentMetadata
	runtime  AgentRuntime
	pipeline *pipeline.Pipeline
	notifier Notifier
	ctx      context.Context
}

type Notifier interface {
	Notify(title, message string) error
}

func NewAgentContext(
	did string,
	metadata *AgentMetadata,
	runtime AgentRuntime,
	pipeline *pipeline.Pipeline,
	notifier Notifier,
	ctx context.Context,
) *AgentContextImpl {
	return &AgentContextImpl{
		did:      did,
		metadata: metadata,
		runtime:  runtime,
		pipeline: pipeline,
		notifier: notifier,
		ctx:      ctx,
	}
}

func (c *AgentContextImpl) DID() string { return c.did }

func (c *AgentContextImpl) Metadata() *AgentMetadata { return c.metadata }

func (c *AgentContextImpl) Runtime() AgentRuntime { return c.runtime }

func (c *AgentContextImpl) Execute(cmd capability.Command) (*capability.Result, error) {
	if c.pipeline == nil {
		return nil, fmt.Errorf("capability pipeline is not configured")
	}
	return c.pipeline.Execute(c.ctx, policy.Principal{DID: c.did}, cmd, func(context.Context, policy.Principal, capability.Command) (*capability.Result, error) {
		return nil, fmt.Errorf("capability executor is not configured")
	})
}

func (c *AgentContextImpl) Notify(title, message string) error {
	if c.notifier == nil {
		return nil
	}
	return c.notifier.Notify(title, message)
}

func (c *AgentContextImpl) Context() context.Context { return c.ctx }
