package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

// ============================================================
// A2A Protocol - Agent-to-Agent Protocol
// ============================================================

// A2AManager يدير بروتوكول A2A للتواصل بين الوكلاء
type A2AManager struct {
	// المكونات الأساسية
	eventBus *eventbus.EventBus

	// الوكلاء المسجلين
	agents map[string]*A2AAgent
	mu     sync.RWMutex

	// Sessions
	sessions map[string]*A2ASession

	// Channels للتواصل الداخلي
	a2aToEventBus chan *A2AMessage
	eventBusToA2A chan eventbus.Event

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Logger
	logger *zap.Logger

	// Metrics
	metrics *A2AMetrics
}

// A2AMetrics مقاييس A2A
type A2AMetrics struct {
	MessagesSent     int64
	MessagesReceived int64
	TasksAssigned    int64
	TasksCompleted   int64
	ArtifactsShared  int64
	Errors           int64
	LastActivity     time.Time
	AgentsCount      int
	SessionsCount    int
}

// A2AAgent يمثل وكيل في نظام A2A
type A2AAgent struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // planner, coder, tester, reviewer, etc.
	Skills      []string               `json:"skills"`
	Status      string                 `json:"status"` // idle, busy, offline
	Config      map[string]interface{} `json:"config"`
	CurrentTask string                 `json:"current_task,omitempty"`
	LastActive  time.Time              `json:"last_active"`
}

// A2ASession يمثل جلسة تعاون بين الوكلاء
type A2ASession struct {
	mu sync.Mutex

	ID           string                 `json:"id"`
	TaskID       string                 `json:"task_id"`
	Goal         string                 `json:"goal"`
	Participants []string               `json:"participants"`
	Status       string                 `json:"status"` // active, completed, failed
	Artifacts    []*A2AArtifact         `json:"artifacts"`
	Events       []*A2AEvent            `json:"events"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// A2AArtifact يمثل ناتج عمل الوكلاء
type A2AArtifact struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"` // code, document, report, test_result
	Name      string                 `json:"name"`
	Content   interface{}            `json:"content"`
	CreatedBy string                 `json:"created_by"`
	CreatedAt time.Time              `json:"created_at"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// A2AEvent يمثل حدث في الجلسة
type A2AEvent struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // task_assigned, task_completed, artifact_shared, error
	AgentID     string                 `json:"agent_id"`
	Description string                 `json:"description"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
}

// A2AMessage رسالة بين الوكلاء
type A2AMessage struct {
	MessageID string                 `json:"message_id"`
	SessionID string                 `json:"session_id"`
	Sender    string                 `json:"sender"`
	Receiver  string                 `json:"receiver"`
	Type      string                 `json:"type"` // task, result, artifact, status_update
	Goal      string                 `json:"goal,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Artifacts []*A2AArtifact         `json:"artifacts,omitempty"`
	Status    string                 `json:"status,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewA2AManager ينشئ A2AManager جديد
func NewA2AManager(eventBus *eventbus.EventBus, logger *zap.Logger) *A2AManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &A2AManager{
		eventBus:      eventBus,
		agents:        make(map[string]*A2AAgent),
		sessions:      make(map[string]*A2ASession),
		a2aToEventBus: make(chan *A2AMessage, 1000),
		eventBusToA2A: make(chan eventbus.Event, 1000),
		ctx:           ctx,
		cancel:        cancel,
		logger:        logger,
		metrics:       &A2AMetrics{},
	}
}

// Start يبدأ A2AManager
func (a *A2AManager) Start() error {
	a.logger.Info("بدء A2AManager")

	// تسجيل الوكلاء الافتراضيين
	a.registerDefaultAgents()

	// الاشتراك في أحداث Event Bus
	a.subscribeToEventBus()

	// بدء معالج A2A
	a.wg.Add(1)
	go a.a2aHandler()

	// بدء معالج Event Bus
	a.wg.Add(1)
	go a.eventBusHandler()

	a.logger.Info("تم بدء A2AManager بنجاح")
	return nil
}

// Stop يوقف A2AManager
func (a *A2AManager) Stop() error {
	a.logger.Info("إيقاف A2AManager")

	a.cancel()
	a.wg.Wait()

	close(a.a2aToEventBus)
	close(a.eventBusToA2A)

	a.logger.Info("تم إيقاف A2AManager بنجاح")
	return nil
}

// ============================================================
// تسجيل الوكلاء
// ============================================================

// registerDefaultAgents يسجل الوكلاء الافتراضيين
func (a *A2AManager) registerDefaultAgents() {
	// Planner Agent
	planner := &A2AAgent{
		ID:     "planner",
		Name:   "Planner Agent",
		Type:   "planner",
		Skills: []string{"planning", "architecture", "task_distribution"},
		Status: "idle",
		Config: map[string]interface{}{},
	}
	a.RegisterAgent(planner)

	// Coder Agent
	coder := &A2AAgent{
		ID:     "coder",
		Name:   "Coder Agent",
		Type:   "coder",
		Skills: []string{"coding", "debugging", "code_review"},
		Status: "idle",
		Config: map[string]interface{}{},
	}
	a.RegisterAgent(coder)

	// Tester Agent
	tester := &A2AAgent{
		ID:     "tester",
		Name:   "Tester Agent",
		Type:   "tester",
		Skills: []string{"testing", "quality_assurance", "integration_testing"},
		Status: "idle",
		Config: map[string]interface{}{},
	}
	a.RegisterAgent(tester)

	// Reviewer Agent
	reviewer := &A2AAgent{
		ID:     "reviewer",
		Name:   "Reviewer Agent",
		Type:   "reviewer",
		Skills: []string{"code_review", "quality_check", "security_review"},
		Status: "idle",
		Config: map[string]interface{}{},
	}
	a.RegisterAgent(reviewer)

	// Research Agent
	research := &A2AAgent{
		ID:     "research",
		Name:   "Research Agent",
		Type:   "research",
		Skills: []string{"research", "documentation", "analysis"},
		Status: "idle",
		Config: map[string]interface{}{},
	}
	a.RegisterAgent(research)

	a.logger.Info("تم تسجيل الوكلاء الافتراضيين",
		zap.Int("count", len(a.agents)),
	)
}

// RegisterAgent يسجل وكيل جديد
func (a *A2AManager) RegisterAgent(agent *A2AAgent) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.agents[agent.ID]; exists {
		return fmt.Errorf("الوكيل %s مسجل بالفعل", agent.ID)
	}

	a.agents[agent.ID] = agent
	a.metrics.AgentsCount++

	a.logger.Info("تم تسجيل وكيل جديد",
		zap.String("agent_id", agent.ID),
		zap.String("name", agent.Name),
		zap.String("type", agent.Type),
	)

	return nil
}

// GetAgent يحصل على وكيل بالمعرف
func (a *A2AManager) GetAgent(agentID string) (*A2AAgent, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	agent, exists := a.agents[agentID]
	if !exists {
		return nil, fmt.Errorf("الوكيل %s غير موجود", agentID)
	}

	return agent, nil
}

// ListAgents يرجع قائمة جميع الوكلاء
func (a *A2AManager) ListAgents() []*A2AAgent {
	a.mu.RLock()
	defer a.mu.RUnlock()

	agents := make([]*A2AAgent, 0, len(a.agents))
	for _, agent := range a.agents {
		agents = append(agents, agent)
	}

	return agents
}

// FindAgentsBySkill يبحث عن وكلاء حسب المهارة
func (a *A2AManager) FindAgentsBySkill(skill string) []*A2AAgent {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var agents []*A2AAgent
	for _, agent := range a.agents {
		for _, s := range agent.Skills {
			if s == skill {
				agents = append(agents, agent)
				break
			}
		}
	}

	return agents
}

// ============================================================
// إدارة الجلسات
// ============================================================

// CreateSession ينشئ جلسة جديدة
func (a *A2AManager) CreateSession(taskID, goal string, participants []string) (*A2ASession, error) {
	sessionID := fmt.Sprintf("session_%s_%d", taskID, time.Now().UnixNano())

	session := &A2ASession{
		ID:           sessionID,
		TaskID:       taskID,
		Goal:         goal,
		Participants: participants,
		Status:       "active",
		Artifacts:    []*A2AArtifact{},
		Events:       []*A2AEvent{},
		Metadata:     map[string]interface{}{},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	a.mu.Lock()
	a.sessions[sessionID] = session
	a.metrics.SessionsCount++
	a.mu.Unlock()

	// تسجيل حدث إنشاء الجلسة
	event := &A2AEvent{
		ID:          generateChatID(),
		Type:        "session_created",
		AgentID:     "system",
		Description: fmt.Sprintf("تم إنشاء جلسة %s", sessionID),
		Data:        map[string]interface{}{"session_id": sessionID},
		Timestamp:   time.Now(),
	}
	session.Events = append(session.Events, event)

	a.logger.Info("تم إنشاء جلسة جديدة",
		zap.String("session_id", sessionID),
		zap.String("task_id", taskID),
		zap.String("goal", goal),
	)

	return session, nil
}

// GetSession يحصل على جلسة بالمعرف
func (a *A2AManager) GetSession(sessionID string) (*A2ASession, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	session, exists := a.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	return session, nil
}

// UpdateSessionStatus يحدث حالة الجلسة
func (a *A2AManager) UpdateSessionStatus(sessionID, status string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	session, exists := a.sessions[sessionID]
	if !exists {
		return fmt.Errorf("الجلسة %s غير موجودة", sessionID)
	}

	session.Status = status
	session.UpdatedAt = time.Now()

	// تسجيل حدث تحديث الحالة
	event := &A2AEvent{
		ID:          generateChatID(),
		Type:        "status_updated",
		AgentID:     "system",
		Description: fmt.Sprintf("تم تحديث حالة الجلسة إلى %s", status),
		Data:        map[string]interface{}{"status": status},
		Timestamp:   time.Now(),
	}
	session.Events = append(session.Events, event)

	return nil
}

// ============================================================
// إرسال الرسائل
// ============================================================

// SendMessage يرسل رسالة بين الوكلاء
func (a *A2AManager) SendMessage(msg *A2AMessage) error {
	// التحقق من وجود المرسل والمستقبل
	sender, err := a.GetAgent(msg.Sender)
	if err != nil {
		return fmt.Errorf("المرسل %s غير موجود", msg.Sender)
	}

	receiver, err := a.GetAgent(msg.Receiver)
	if err != nil {
		return fmt.Errorf("المستقبل %s غير موجود", msg.Receiver)
	}

	// تحديث حالة المرسل
	sender.LastActive = time.Now()

	// تحديث حالة المستقبل
	receiver.LastActive = time.Now()

	// إضافة الرسالة إلى الجلسة إذا كانت مرتبطة بجلسة
	if msg.SessionID != "" {
		session, err := a.GetSession(msg.SessionID)
		if err == nil {
			// تسجيل حدث الرسالة
			event := &A2AEvent{
				ID:          generateChatID(),
				Type:        "message_sent",
				AgentID:     msg.Sender,
				Description: fmt.Sprintf("رسالة من %s إلى %s", msg.Sender, msg.Receiver),
				Data:        map[string]interface{}{"message_type": msg.Type, "goal": msg.Goal},
				Timestamp:   time.Now(),
			}
			session.Events = append(session.Events, event)
			session.UpdatedAt = time.Now()
		}
	}

	a.a2aToEventBus <- msg

	a.mu.Lock()
	a.metrics.MessagesSent++
	a.metrics.LastActivity = time.Now()
	a.mu.Unlock()

	return nil
}

// BroadcastMessage يبث رسالة لجميع الوكلاء في جلسة
func (a *A2AManager) BroadcastMessage(sessionID, sender string, msgType string, data map[string]interface{}) error {
	session, err := a.GetSession(sessionID)
	if err != nil {
		return err
	}

	for _, participantID := range session.Participants {
		if participantID == sender {
			continue // لا ترسل لنفسك
		}

		msg := &A2AMessage{
			MessageID: generateChatID(),
			SessionID: sessionID,
			Sender:    sender,
			Receiver:  participantID,
			Type:      msgType,
			Context:   data,
			Timestamp: time.Now(),
		}

		if err := a.SendMessage(msg); err != nil {
			a.logger.Error("فشل إرسال رسالة",
				zap.String("sender", sender),
				zap.String("receiver", participantID),
				zap.Error(err),
			)
		}
	}

	return nil
}

// ============================================================
// إدارة المهام
// ============================================================

// AssignTask يوزع مهمة على وكيل
func (a *A2AManager) AssignTask(sessionID, agentID, task string, context map[string]interface{}) error {
	agent, err := a.GetAgent(agentID)
	if err != nil {
		return err
	}

	// تحديث حالة الوكيل
	agent.Status = "busy"
	agent.CurrentTask = task

	// إرسال رسالة المهمة
	msg := &A2AMessage{
		MessageID: generateChatID(),
		SessionID: sessionID,
		Sender:    "planner", // افتراضياً من المخطط
		Receiver:  agentID,
		Type:      "task",
		Goal:      task,
		Context:   context,
		Timestamp: time.Now(),
	}

	if err := a.SendMessage(msg); err != nil {
		return err
	}

	a.mu.Lock()
	a.metrics.TasksAssigned++
	a.mu.Unlock()

	// تسجيل حدث توزيع المهمة
	if sessionID != "" {
		session, err := a.GetSession(sessionID)
		if err == nil {
			event := &A2AEvent{
				ID:          generateChatID(),
				Type:        "task_assigned",
				AgentID:     agentID,
				Description: fmt.Sprintf("تم توزيع مهمة على %s", agentID),
				Data:        map[string]interface{}{"task": task},
				Timestamp:   time.Now(),
			}
			session.Events = append(session.Events, event)
			session.UpdatedAt = time.Now()
		}
	}

	return nil
}

// CompleteTask يكمل مهمة
func (a *A2AManager) CompleteTask(sessionID, agentID string, artifacts []*A2AArtifact) error {
	agent, err := a.GetAgent(agentID)
	if err != nil {
		return err
	}

	// تحديث حالة الوكيل
	agent.Status = "idle"
	agent.CurrentTask = ""

	// إضافة Artifacts إلى الجلسة
	if sessionID != "" {
		session, err := a.GetSession(sessionID)
		if err == nil {
			session.Artifacts = append(session.Artifacts, artifacts...)

			// تسجيل حدث إكمال المهمة
			event := &A2AEvent{
				ID:          generateChatID(),
				Type:        "task_completed",
				AgentID:     agentID,
				Description: fmt.Sprintf("أكمل %s مهمته", agentID),
				Data:        map[string]interface{}{"artifacts_count": len(artifacts)},
				Timestamp:   time.Now(),
			}
			session.Events = append(session.Events, event)
			session.UpdatedAt = time.Now()
		}
	}

	a.mu.Lock()
	a.metrics.TasksCompleted++
	a.mu.Unlock()

	return nil
}

// ============================================================
// معالجة الرسائل
// ============================================================

// subscribeToEventBus يرتبط بأحداث Event Bus
func (a *A2AManager) subscribeToEventBus() {
	a.eventBus.Subscribe("a2a.message", a.handleA2AMessage)
	a.eventBus.Subscribe("a2a.broadcast", a.handleA2ABroadcast)
	a.eventBus.Subscribe("agent.status", a.handleAgentStatus)
}

// a2aHandler يعالج رسائل A2A
func (a *A2AManager) a2aHandler() {
	defer a.wg.Done()

	for {
		select {
		case <-a.ctx.Done():
			return
		case msg := <-a.a2aToEventBus:
			a.processA2AMessage(msg)
		}
	}
}

// processA2AMessage يعالج رسالة A2A
func (a *A2AManager) processA2AMessage(msg *A2AMessage) {
	// تحويل الرسالة إلى حدث Event Bus
	event := eventbus.Event{
		Type:      "a2a.message",
		Payload:   msg,
		Source:    msg.Sender,
		SessionID: msg.SessionID,
		Timestamp: msg.Timestamp,
	}

	// نشر الحدث
	a.eventBus.Publish(event)

	a.mu.Lock()
	a.metrics.MessagesReceived++
	a.mu.Unlock()

	a.logger.Debug("تم معالجة رسالة A2A",
		zap.String("sender", msg.Sender),
		zap.String("receiver", msg.Receiver),
		zap.String("type", msg.Type),
	)
}

// eventBusHandler يعالج أحداث Event Bus
func (a *A2AManager) eventBusHandler() {
	defer a.wg.Done()

	for {
		select {
		case <-a.ctx.Done():
			return
		case event := <-a.eventBusToA2A:
			a.processEventBusEvent(event)
		}
	}
}

// processEventBusEvent يعالج حدث Event Bus
func (a *A2AManager) processEventBusEvent(event eventbus.Event) {
	a.logger.Debug("تم معالجة حدث Event Bus",
		zap.String("event_type", event.Type),
	)
}

// handleA2AMessage يعالج رسالة A2A
func (a *A2AManager) handleA2AMessage(event eventbus.Event) {
	a.logger.Debug("استقبال رسالة A2A",
		zap.String("source", event.Source),
	)
}

// handleA2ABroadcast يعالج بث A2A
func (a *A2AManager) handleA2ABroadcast(event eventbus.Event) {
	a.logger.Debug("استقبال بث A2A",
		zap.String("source", event.Source),
	)
}

// handleAgentStatus يعالج حالة الوكيل
func (a *A2AManager) handleAgentStatus(event eventbus.Event) {
	a.logger.Debug("استقبال حالة وكيل",
		zap.String("agent_id", event.Source),
	)
}

// ============================================================
// المقاييس
// ============================================================

// GetMetrics يحصل على المقاييس
func (a *A2AManager) GetMetrics() *A2AMetrics {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return &A2AMetrics{
		MessagesSent:     a.metrics.MessagesSent,
		MessagesReceived: a.metrics.MessagesReceived,
		TasksAssigned:    a.metrics.TasksAssigned,
		TasksCompleted:   a.metrics.TasksCompleted,
		ArtifactsShared:  a.metrics.ArtifactsShared,
		Errors:           a.metrics.Errors,
		LastActivity:     a.metrics.LastActivity,
		AgentsCount:      a.metrics.AgentsCount,
		SessionsCount:    a.metrics.SessionsCount,
	}
}
