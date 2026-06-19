package integration

import (
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"go.uber.org/zap"
)

// AgentCommunication يدير التواصل بين الوكلاء
type AgentCommunication struct {
	registry *agent.AgentRegistry
	logger   *zap.Logger
	messages map[string][]*AgentMessage // agentID -> messages
	mu       sync.RWMutex
}

// AgentMessage رسالة بين الوكلاء
type AgentMessage struct {
	ID        string    `json:"id"`
	FromAgent string    `json:"from_agent"`
	ToAgent   string    `json:"to_agent"`
	Content   string    `json:"content"`
	Type      string    `json:"type"` // task, result, info, error
	Timestamp time.Time `json:"timestamp"`
	TaskID    string    `json:"task_id,omitempty"`
}

// NewAgentCommunication ينشئ نظام تواصل جديد
func NewAgentCommunication(registry *agent.AgentRegistry, logger *zap.Logger) *AgentCommunication {
	return &AgentCommunication{
		registry: registry,
		logger:   logger,
		messages: make(map[string][]*AgentMessage),
	}
}

// SendMessageBetweenAgents يرسل رسالة بين وكيلين
func (ac *AgentCommunication) SendMessageBetweenAgents(fromAgentID, toAgentID, content, messageType string) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	// التحقق من وجود الوكيل المرسل
	_, err := ac.registry.Get(fromAgentID)
	if err != nil {
		return fmt.Errorf("failed to get from agent: %w", err)
	}

	// التحقق من وجود الوكيل المستقبل
	_, err = ac.registry.Get(toAgentID)
	if err != nil {
		return fmt.Errorf("failed to get to agent: %w", err)
	}

	// إنشاء الرسالة
	message := &AgentMessage{
		ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		FromAgent: fromAgentID,
		ToAgent:   toAgentID,
		Content:   content,
		Type:      messageType,
		Timestamp: time.Now(),
	}

	// إضافة الرسالة إلى الوكيل المستقبل
	ac.messages[toAgentID] = append(ac.messages[toAgentID], message)

	ac.logger.Info("Message sent between agents",
		zap.String("from_agent", fromAgentID),
		zap.String("to_agent", toAgentID),
		zap.String("message_type", messageType),
		zap.String("message_id", message.ID),
	)

	return nil
}

// BroadcastMessage يبث رسالة إلى جميع الوكلاء
func (ac *AgentCommunication) BroadcastMessage(fromAgentID, content, messageType string) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	// التحقق من وجود الوكيل المرسل
	_, err := ac.registry.Get(fromAgentID)
	if err != nil {
		return fmt.Errorf("failed to get from agent: %w", err)
	}

	// الحصول على جميع الوكلاء
	agents := ac.registry.ListAll()

	// إرسال الرسالة إلى جميع الوكلاء
	for _, agentInstance := range agents {
		toAgentID := agentInstance.GetInfo().ID
		if toAgentID == fromAgentID {
			continue // لا ترسل إلى نفسك
		}

		message := &AgentMessage{
			ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
			FromAgent: fromAgentID,
			ToAgent:   toAgentID,
			Content:   content,
			Type:      messageType,
			Timestamp: time.Now(),
		}

		ac.messages[toAgentID] = append(ac.messages[toAgentID], message)
	}

	ac.logger.Info("Message broadcasted",
		zap.String("from_agent", fromAgentID),
		zap.String("message_type", messageType),
		zap.Int("recipients", len(agents)-1),
	)

	return nil
}

// ShareTaskResult يشارك نتيجة مهمة مع وكلاء آخرين
func (ac *AgentCommunication) ShareTaskResult(fromAgentID, taskID string, result *agent.TaskExecutionResult, targetAgentIDs []string) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	// التحقق من وجود الوكيل المرسل
	_, err := ac.registry.Get(fromAgentID)
	if err != nil {
		return fmt.Errorf("failed to get from agent: %w", err)
	}

	// تحويل النتيجة إلى نص
	content := fmt.Sprintf("Task Result for %s:\nSuccess: %v\nOutput: %s\nDuration: %v",
		taskID, result.Success, result.Output, result.Duration)

	// إرسال النتيجة إلى الوكلاء المستهدفين
	for _, toAgentID := range targetAgentIDs {
		// التحقق من وجود الوكيل المستقبل
		_, err := ac.registry.Get(toAgentID)
		if err != nil {
			ac.logger.Warn("Failed to get target agent",
				zap.String("target_agent", toAgentID),
				zap.Error(err),
			)
			continue
		}

		message := &AgentMessage{
			ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
			FromAgent: fromAgentID,
			ToAgent:   toAgentID,
			Content:   content,
			Type:      "result",
			Timestamp: time.Now(),
			TaskID:    taskID,
		}

		ac.messages[toAgentID] = append(ac.messages[toAgentID], message)
	}

	ac.logger.Info("Task result shared",
		zap.String("from_agent", fromAgentID),
		zap.String("task_id", taskID),
		zap.Int("recipients", len(targetAgentIDs)),
	)

	return nil
}

// GetAgentMessages يحصل على رسائل وكيل
func (ac *AgentCommunication) GetAgentMessages(agentID string) ([]*AgentMessage, error) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	// التحقق من وجود الوكيل
	_, err := ac.registry.Get(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	messages, exists := ac.messages[agentID]
	if !exists {
		return []*AgentMessage{}, nil
	}

	return messages, nil
}

// GetAgentMessagesByType يحصل على رسائل وكيل حسب النوع
func (ac *AgentCommunication) GetAgentMessagesByType(agentID, messageType string) ([]*AgentMessage, error) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	// التحقق من وجود الوكيل
	_, err := ac.registry.Get(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	messages, exists := ac.messages[agentID]
	if !exists {
		return []*AgentMessage{}, nil
	}

	// تصفية الرسائل حسب النوع
	filtered := make([]*AgentMessage, 0)
	for _, message := range messages {
		if message.Type == messageType {
			filtered = append(filtered, message)
		}
	}

	return filtered, nil
}

// GetAgentMessagesByTask يحصل على رسائل وكيل حسب المهمة
func (ac *AgentCommunication) GetAgentMessagesByTask(agentID, taskID string) ([]*AgentMessage, error) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	// التحقق من وجود الوكيل
	_, err := ac.registry.Get(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	messages, exists := ac.messages[agentID]
	if !exists {
		return []*AgentMessage{}, nil
	}

	// تصفية الرسائل حسب المهمة
	filtered := make([]*AgentMessage, 0)
	for _, message := range messages {
		if message.TaskID == taskID {
			filtered = append(filtered, message)
		}
	}

	return filtered, nil
}

// ClearAgentMessages يمسح رسائل وكيل
func (ac *AgentCommunication) ClearAgentMessages(agentID string) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	// التحقق من وجود الوكيل
	_, err := ac.registry.Get(agentID)
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	delete(ac.messages, agentID)

	ac.logger.Info("Agent messages cleared",
		zap.String("agent_id", agentID),
	)

	return nil
}

// GetCommunicationSummary يحصل على ملخص التواصل
func (ac *AgentCommunication) GetCommunicationSummary() map[string]interface{} {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	totalMessages := 0
	messagesByType := make(map[string]int)
	messagesByAgent := make(map[string]int)

	for agentID, messages := range ac.messages {
		messagesByAgent[agentID] = len(messages)
		totalMessages += len(messages)

		for _, message := range messages {
			messagesByType[message.Type]++
		}
	}

	return map[string]interface{}{
		"total_messages":    totalMessages,
		"messages_by_type":  messagesByType,
		"messages_by_agent": messagesByAgent,
	}
}
