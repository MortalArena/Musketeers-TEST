package unified

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/agent/tools"
	"go.uber.org/zap"
)

type BridgeAgentStatus string

const (
	BridgeAgentConnected    BridgeAgentStatus = "connected"
	BridgeAgentDisconnected BridgeAgentStatus = "disconnected"
	BridgeAgentError        BridgeAgentStatus = "error"
)

type BridgeAgentInfo struct {
	AgentID     string            `json:"agent_id"`
	AgentType   string            `json:"agent_type"`
	BridgeID    string            `json:"bridge_id"`
	SessionID   string            `json:"session_id"`
	Status      BridgeAgentStatus `json:"status"`
	ConnectedAt time.Time         `json:"connected_at"`
	External    bool              `json:"external"`
}

type ExternalBridgeManager struct {
	mu           sync.RWMutex
	bridges      map[string]*BridgeAgentInfo
	agentPool    *AgentPool
	eventBus     *SessionEventBus
	sessionEvent *SessionEventBus
	logger       *zap.Logger
}

func NewExternalBridgeManager(agentPool *AgentPool, eventBus *SessionEventBus, logger *zap.Logger) *ExternalBridgeManager {
	return &ExternalBridgeManager{
		bridges:   make(map[string]*BridgeAgentInfo),
		agentPool: agentPool,
		eventBus:  eventBus,
		logger:    logger,
	}
}

func (ebm *ExternalBridgeManager) SetSessionEventBus(seb *SessionEventBus) {
	ebm.mu.Lock()
	defer ebm.mu.Unlock()
	ebm.sessionEvent = seb
}

func (ebm *ExternalBridgeManager) RegisterExternalAgent(adapter agent.UnifiedAgent, roleStr string) (*AgentInstance, error) {
	if ebm.agentPool == nil {
		return nil, fmt.Errorf("agentPool not set on ExternalBridgeManager")
	}

	agentRole := tools.AgentRole(roleStr)
	instance, err := ebm.agentPool.RegisterAgent(adapter, agentRole)
	if err != nil {
		return nil, fmt.Errorf("failed to register external agent in pool: %w", err)
	}

	info := adapter.GetInfo()
	bridgeInfo := &BridgeAgentInfo{
		AgentID:     info.ID,
		AgentType:   string(info.Type),
		BridgeID:    fmt.Sprintf("bridge-%s-%d", info.ID, time.Now().UnixNano()),
		SessionID:   ebm.agentPool.GetSessionID(),
		Status:      BridgeAgentConnected,
		ConnectedAt: time.Now(),
		External:    true,
	}

	ebm.mu.Lock()
	ebm.bridges[info.ID] = bridgeInfo
	ebm.mu.Unlock()

	ebm.logger.Info("External bridge agent registered",
		zap.String("agent_id", info.ID),
		zap.String("type", string(info.Type)),
		zap.String("bridge_id", bridgeInfo.BridgeID),
		zap.String("session_id", bridgeInfo.SessionID))

	return instance, nil
}

func (ebm *ExternalBridgeManager) WireToEventBus(agentID string) error {
	if ebm.eventBus == nil {
		return fmt.Errorf("eventBus not set on ExternalBridgeManager")
	}

	ch := ebm.eventBus.SubscribeAgent(agentID)
	if ch == nil {
		return fmt.Errorf("failed to subscribe agent %s to event bus", agentID)
	}

	go ebm.processExternalAgentEvents(agentID, ch)

	ebm.logger.Info("External bridge agent wired to session event bus",
		zap.String("agent_id", agentID))
	return nil
}

func (ebm *ExternalBridgeManager) UnwireFromEventBus(agentID string) error {
	if ebm.eventBus == nil {
		return nil
	}
	ebm.eventBus.UnsubscribeAgent(agentID)

	ebm.mu.Lock()
	if bi, ok := ebm.bridges[agentID]; ok {
		bi.Status = BridgeAgentDisconnected
	}
	ebm.mu.Unlock()

	ebm.logger.Info("External bridge agent unwired from session event bus",
		zap.String("agent_id", agentID))
	return nil
}

func (ebm *ExternalBridgeManager) processExternalAgentEvents(agentID string, ch chan *SessionEvent) {
	defer func() {
		if r := recover(); r != nil {
			ebm.logger.Warn("External agent event processor panicked",
				zap.String("agent_id", agentID),
				zap.Any("panic", r))
		}
	}()

	for {
		select {
		case event, ok := <-ch:
			if !ok {
				return
			}

			instance, err := ebm.agentPool.GetAgent(agentID)
			if err != nil {
				continue
			}

			if event.EventType == TaskAssigned || event.EventType == AgentMessage {
				taskStr := ""
				if data, ok := event.Data.(string); ok {
					taskStr = data
				}
				if taskStr == "" {
					continue
				}

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				resp, err := instance.Adapter.SendMessage(ctx, taskStr)
				cancel()

				if err != nil {
					ebm.logger.Warn("External bridge agent send failed",
						zap.String("agent_id", agentID),
						zap.Error(err))
					continue
				}

				ebm.logger.Info("External bridge agent responded",
					zap.String("agent_id", agentID),
					zap.Int("response_len", len(resp.Content)))
			}
		}
	}
}

func (ebm *ExternalBridgeManager) GetBridges() []*BridgeAgentInfo {
	ebm.mu.RLock()
	defer ebm.mu.RUnlock()

	result := make([]*BridgeAgentInfo, 0, len(ebm.bridges))
	for _, bi := range ebm.bridges {
		result = append(result, bi)
	}
	return result
}

func (ebm *ExternalBridgeManager) GetBridge(agentID string) (*BridgeAgentInfo, bool) {
	ebm.mu.RLock()
	defer ebm.mu.RUnlock()
	bi, ok := ebm.bridges[agentID]
	return bi, ok
}

func (ebm *ExternalBridgeManager) IsExternalAgent(agentID string) bool {
	ebm.mu.RLock()
	defer ebm.mu.RUnlock()
	_, ok := ebm.bridges[agentID]
	return ok
}
