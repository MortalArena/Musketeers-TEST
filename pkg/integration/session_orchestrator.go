package integration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/session/core"
	"go.uber.org/zap"
)

// SessionOrchestrator ينسق الجلسات والوكلاء
type SessionOrchestrator struct {
	sessionManager     *core.UnifiedSessionManager
	agentRegistry      *agent.AgentRegistry
	agentIntegration   *AgentSessionIntegration
	instanceManager    *InstanceSessionIntegration
	roleAssignment     *RoleAssignment
	taskRouting        *TaskRouting
	agentCommunication *AgentCommunication
	logger             *zap.Logger
	mu                 sync.RWMutex
}

// NewSessionOrchestrator ينشئ منسق جلسات جديد
func NewSessionOrchestrator(
	sessionManager *core.UnifiedSessionManager,
	agentRegistry *agent.AgentRegistry,
	agentIntegration *AgentSessionIntegration,
	instanceManager *InstanceSessionIntegration,
	roleAssignment *RoleAssignment,
	taskRouting *TaskRouting,
	agentCommunication *AgentCommunication,
	logger *zap.Logger,
) *SessionOrchestrator {
	return &SessionOrchestrator{
		sessionManager:     sessionManager,
		agentRegistry:      agentRegistry,
		agentIntegration:   agentIntegration,
		instanceManager:    instanceManager,
		roleAssignment:     roleAssignment,
		taskRouting:        taskRouting,
		agentCommunication: agentCommunication,
		logger:             logger,
	}
}

// OrchestrateSession ينسق جلسة
func (so *SessionOrchestrator) OrchestrateSession(ctx context.Context, sessionID string) error {
	so.mu.Lock()
	defer so.mu.Unlock()

	// الحصول على الجلسة
	session, err := so.sessionManager.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// التحقق من حالة الجلسة
	if session.Status != core.SessionStatusActive {
		return fmt.Errorf("session is not active: %s", session.Status)
	}

	so.logger.Info("Session orchestration started",
		zap.String("session_id", sessionID),
		zap.String("session_name", session.Name),
	)

	return nil
}

// ManageSessionLifecycle يدير دورة حياة الجلسة
func (so *SessionOrchestrator) ManageSessionLifecycle(ctx context.Context, sessionID string) error {
	so.mu.Lock()
	defer so.mu.Unlock()

	// الحصول على الجلسة
	session, err := so.sessionManager.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// إدارة دورة حياة الجلسة
	switch session.Status {
	case core.SessionStatusInitializing:
		// تهيئة الجلسة
		err = so.sessionManager.ResumeSession(sessionID)
		if err != nil {
			return fmt.Errorf("failed to initialize session: %w", err)
		}
	case core.SessionStatusActive:
		// الجلسة نشطة
		so.logger.Info("Session is active",
			zap.String("session_id", sessionID),
		)
	case core.SessionStatusPaused:
		// الجلسة متوقفة
		so.logger.Info("Session is paused",
			zap.String("session_id", sessionID),
		)
	case core.SessionStatusCompleted:
		// الجلسة مكتملة
		so.logger.Info("Session is completed",
			zap.String("session_id", sessionID),
		)
	case core.SessionStatusFailed:
		// الجلسة فشلت
		so.logger.Error("Session failed",
			zap.String("session_id", sessionID),
		)
	}

	return nil
}

// ManageSessionTasks يدير المهام داخل الجلسة
func (so *SessionOrchestrator) ManageSessionTasks(ctx context.Context, sessionID string, task *agent.AgentTask) (map[string]*agent.TaskExecutionResult, error) {
	so.mu.Lock()
	defer so.mu.Unlock()

	// تنفيذ المهمة على جميع وكلاء الجلسة
	results, err := so.agentIntegration.ExecuteTaskOnSessionAgents(ctx, sessionID, task)
	if err != nil {
		return nil, fmt.Errorf("failed to execute task on session agents: %w", err)
	}

	so.logger.Info("Session task managed",
		zap.String("session_id", sessionID),
		zap.String("task_id", task.ID),
		zap.Int("results", len(results)),
	)

	return results, nil
}

// ManageSessionCommunication يدير التواصل داخل الجلسة
func (so *SessionOrchestrator) ManageSessionCommunication(sessionID string) error {
	so.mu.Lock()
	defer so.mu.Unlock()

	// الحصول على الوكلاء في الجلسة
	agents, err := so.agentIntegration.GetAgentsInSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get agents in session: %w", err)
	}

	// تمكين التواصل بين الوكلاء
	for _, agentInstance := range agents {
		agentID := agentInstance.GetInfo().ID
		messages, err := so.agentCommunication.GetAgentMessages(agentID)
		if err != nil {
			so.logger.Warn("Failed to get agent messages",
				zap.String("agent_id", agentID),
				zap.Error(err),
			)
			continue
		}

		if len(messages) > 0 {
			so.logger.Info("Agent has pending messages",
				zap.String("agent_id", agentID),
				zap.Int("message_count", len(messages)),
			)
		}
	}

	so.logger.Info("Session communication managed",
		zap.String("session_id", sessionID),
		zap.Int("total_agents", len(agents)),
	)

	return nil
}

// ExecuteTaskWithOrchestration ينفذ مهمة مع تنسيق كامل
func (so *SessionOrchestrator) ExecuteTaskWithOrchestration(ctx context.Context, sessionID string, task *agent.AgentTask, strategy string) (*agent.TaskExecutionResult, error) {
	so.mu.Lock()
	defer so.mu.Unlock()

	// التحقق من حالة الجلسة
	session, err := so.sessionManager.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session.Status != core.SessionStatusActive {
		return nil, fmt.Errorf("session is not active: %s", session.Status)
	}

	// تنفيذ المهمة حسب الاستراتيجية
	var result *agent.TaskExecutionResult

	switch strategy {
	case "manager":
		// تنفيذ على مدير الجلسة
		result, err = so.agentIntegration.ExecuteTaskOnManager(ctx, sessionID, task)
		if err != nil {
			return nil, fmt.Errorf("failed to execute task on manager: %w", err)
		}
	case "all":
		// تنفيذ على جميع الوكلاء
		results, err := so.agentIntegration.ExecuteTaskOnSessionAgents(ctx, sessionID, task)
		if err != nil {
			return nil, fmt.Errorf("failed to execute task on all agents: %w", err)
		}
		result = so.taskRouting.MergeResults(results)
	case "routing":
		// استخدام توجيه المهام
		results, err := so.taskRouting.RouteTask(ctx, task)
		if err != nil {
			return nil, fmt.Errorf("failed to route task: %w", err)
		}
		result = so.taskRouting.MergeResults(results)
	default:
		// تنفيذ على جميع الوكلاء
		results, err := so.agentIntegration.ExecuteTaskOnSessionAgents(ctx, sessionID, task)
		if err != nil {
			return nil, fmt.Errorf("failed to execute task on all agents: %w", err)
		}
		result = so.taskRouting.MergeResults(results)
	}

	so.logger.Info("Task executed with orchestration",
		zap.String("session_id", sessionID),
		zap.String("task_id", task.ID),
		zap.String("strategy", strategy),
		zap.Bool("success", result.Success),
	)

	return result, nil
}

// GetSessionOrchestrationStatus يحصل على حالة تنسيق الجلسة
func (so *SessionOrchestrator) GetSessionOrchestrationStatus(sessionID string) (map[string]interface{}, error) {
	so.mu.RLock()
	defer so.mu.RUnlock()

	// الحصول على الجلسة
	session, err := so.sessionManager.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// الحصول على الوكلاء في الجلسة
	agents, err := so.agentIntegration.GetAgentsInSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agents in session: %w", err)
	}

	// الحصول على ملخص التواصل
	commSummary := so.agentCommunication.GetCommunicationSummary()

	return map[string]interface{}{
		"session_id":       sessionID,
		"session_status":   session.Status,
		"total_agents":     len(agents),
		"manager_agent_id": session.ManagerAgentID,
		"assistant_agents": session.AssistantAgents,
		"communication":    commSummary,
		"orchestrated_at":  time.Now(),
	}, nil
}

// StartSessionOrchestration يبدأ تنسيق الجلسة
func (so *SessionOrchestrator) StartSessionOrchestration(ctx context.Context, sessionID string) error {
	so.mu.Lock()
	defer so.mu.Unlock()

	// تنسيق الجلسة
	err := so.OrchestrateSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to orchestrate session: %w", err)
	}

	// إدارة دورة حياة الجلسة
	err = so.ManageSessionLifecycle(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to manage session lifecycle: %w", err)
	}

	// إدارة التواصل
	err = so.ManageSessionCommunication(sessionID)
	if err != nil {
		return fmt.Errorf("failed to manage session communication: %w", err)
	}

	so.logger.Info("Session orchestration started successfully",
		zap.String("session_id", sessionID),
	)

	return nil
}

// StopSessionOrchestration يوقف تنسيق الجلسة
func (so *SessionOrchestrator) StopSessionOrchestration(sessionID string) error {
	so.mu.Lock()
	defer so.mu.Unlock()

	// إيقاف الجلسة
	err := so.sessionManager.PauseSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to pause session: %w", err)
	}

	so.logger.Info("Session orchestration stopped",
		zap.String("session_id", sessionID),
	)

	return nil
}
