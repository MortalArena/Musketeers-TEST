package adapters

import (
	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/channel"
	"github.com/MortalArena/Musketeers/pkg/content"
	"github.com/MortalArena/Musketeers/pkg/discovery"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/MortalArena/Musketeers/pkg/node"
	"github.com/MortalArena/Musketeers/pkg/policy"
	"github.com/MortalArena/Musketeers/pkg/providers"
	"github.com/MortalArena/Musketeers/pkg/registry"
	sdkpkg "github.com/MortalArena/Musketeers/pkg/sdk"
	"github.com/MortalArena/Musketeers/pkg/sdk/interfaces"
	sdkadapters "github.com/MortalArena/Musketeers/pkg/sdk/interfaces/adapters"
	"github.com/MortalArena/Musketeers/pkg/session"
	sessioncore "github.com/MortalArena/Musketeers/pkg/session/core"
	"github.com/MortalArena/Musketeers/pkg/workflow"
)

// SDKIntegration يربط جميع 15 محول SDK في نظام واحد
type SDKIntegration struct {
	client    *sdkpkg.NeuroClient
	adapters  *SDKAdapterSet
}

// SDKAdapterSet مجموعة جميع محولات SDK
type SDKAdapterSet struct {
	Node       interfaces.NodeInterface
	Session    interfaces.SessionInterface
	Identity   interfaces.IdentityInterface
	Comm       interfaces.CommunicationInterface
	Agent      interfaces.AgentInterface
	Workflow   interfaces.WorkflowInterface
	Storage    interfaces.StorageInterface
	Security   interfaces.SecurityInterface
	A2A        interfaces.A2AInterface
	EventBus   interfaces.EventBus
	AI         interfaces.AIInterface
	UIBridge   interfaces.UIBridgeInterface
	Journal    interfaces.JournalInterface
	Sync       interfaces.SyncInterface
	Discovery  interfaces.DiscoveryInterface
}

// NewSDKIntegration ينشئ تكامل SDK بالكامل مع جميع المحولات
func NewSDKIntegration(
	n *node.Node,
	sc *session.SessionContainer,
	usm *sessioncore.UnifiedSessionManager,
	disc discovery.Discovery,
	reg registry.Registry,
	bus *eventbus.EventBus,
	store content.BlockStore,
	pe *policy.Engine,
	chMgr channel.ChannelManager,
	provider providers.Provider,
	journal *session.SessionJournal,
	crdtMgr *sdkpkg.CRDTSyncManager,
	wfEngine workflow.WorkflowEngine,
	unifiedAgent agent.UnifiedAgent,
) *SDKIntegration {
	set := &SDKAdapterSet{
		Node:      sdkadapters.NewNodeAdapter(n),
		Session:   sdkadapters.NewSessionAdapter(sc, usm),
		Identity:  sdkadapters.NewIdentityAdapter(n),
		Comm:      sdkadapters.NewCommAdapter(chMgr),
		Agent:     sdkadapters.NewAgentAdapter(unifiedAgent),
		Workflow:  sdkadapters.NewWorkflowAdapter(wfEngine),
		Storage:   sdkadapters.NewStorageAdapter(store),
		Security:  sdkadapters.NewSecurityAdapter(n, pe),
		A2A:       sdkadapters.NewA2AAdapter(n),
		EventBus:  sdkadapters.NewEventBusAdapter(bus),
		AI:        sdkadapters.NewAIAdapter(provider),
		UIBridge:  sdkadapters.NewUIBridgeAdapter(bus),
		Journal:   sdkadapters.NewJournalAdapter(journal),
		Sync:      sdkadapters.NewSyncAdapter(crdtMgr),
		Discovery: sdkadapters.NewDiscoveryAdapter(disc, reg),
	}

	client := sdkpkg.New(nil, nil, nil, nil, reg)

	return &SDKIntegration{
		client:   client,
		adapters: set,
	}
}

// Adapters يعيد مجموعة جميع المحولات
func (si *SDKIntegration) Adapters() *SDKAdapterSet {
	return si.adapters
}

// Client يعيد NeuroClient
func (si *SDKIntegration) Client() *sdkpkg.NeuroClient {
	return si.client
}
