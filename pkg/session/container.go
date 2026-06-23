package session

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
)

// SessionContainer الحاوية الكاملة للجلسة - القلب النابض
// [WHY] يدير جميع مكونات الجلسة ويوفر حالة موحدة
// [HOW] يستخدم stateMu لحماية الحالة الموحدة ويفك القفل قبل النشر
// [SAFETY] يفك القفل دائماً قبل استدعاء eventBus.Publish لمنع Deadlock
type SessionContainer struct {
	// Metadata
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	OwnerDID    string    `json:"owner_did"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Version     int       `json:"version"`
	Status      string    `json:"status"` // active, paused, completed, failed

	// المكونات الجديدة
	Memory     *CollectiveMemory
	Skills     *SkillsManager
	Workflow   *WorkflowEngine
	Artifacts  *ArtifactsStore
	Tasks      *TaskManager
	Progress   *ProgressTracker
	Handoff    *HandoffManager
	Aggregator *Aggregator
	Reviewer   *FinalReviewer

	// [WHY] ChatManager لإدارة الرسائل
	ChatManager *ChatManager

	// [WHY] UnifiedSessionState الحالة الموحدة للجلسة
	// [HOW] يحتوي على ملخص الحالة للعميل
	// [SAFETY] محمي بـ stateMu
	state   UnifiedSessionState
	stateMu sync.RWMutex

	// Event Bus
	EventBus *eventbus.EventBus

	// Storage
	DB *badger.DB

	mu         sync.RWMutex
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// [WHY] UnifiedSessionState الحالة الموحدة للجلسة
// [HOW] يحتوي على ملخص الحالة للعميل
type UnifiedSessionState struct {
	SessionID string       `json:"session_id"` // [WHY] معرف الجلسة
	Status    string       `json:"status"`     // [WHY] حالة الجلسة
	Agents    []AgentInfo  `json:"agents"`     // [WHY] قائمة الوكلاء
	Tasks     []TaskInfo   `json:"tasks"`      // [WHY] قائمة المهام
	Progress  ProgressInfo `json:"progress"`   // [WHY] تقدم الجلسة
	UpdatedAt time.Time    `json:"updated_at"` // [WHY] وقت التحديث
}

// [WHY] AgentInfo معلومات الوكيل
type AgentInfo struct {
	DID    string `json:"did"`    // [WHY] معرف الوكيل
	Name   string `json:"name"`   // [WHY] اسم الوكيل
	Status string `json:"status"` // [WHY] حالة الوكيل
	Role   string `json:"role"`   // [WHY] دور الوكيل
}

// [WHY] TaskInfo معلومات المهمة
type TaskInfo struct {
	ID         string `json:"id"`          // [WHY] معرف المهمة
	Title      string `json:"title"`       // [WHY] عنوان المهمة
	Status     string `json:"status"`      // [WHY] حالة المهمة
	AssignedTo string `json:"assigned_to"` // [WHY] الوكيل المسؤول
	Priority   string `json:"priority"`    // [WHY] أولوية المهمة
}

// [WHY] ProgressInfo معلومات التقدم
type ProgressInfo struct {
	TotalTasks     int     `json:"total_tasks"`     // [WHY] إجمالي المهام
	CompletedTasks int     `json:"completed_tasks"` // [WHY] المهام المكتملة
	Percentage     float64 `json:"percentage"`      // [WHY] نسبة الإنجاز
}

// SessionConfig إعدادات الجلسة
type SessionConfig struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	OwnerDID    string `json:"owner_did"`
	MaxAgents   int    `json:"max_agents"`
	ProjectType string `json:"project_type"`
}

// [SAFETY] حدود الموارد لمنع استهلاك غير محدود
const (
	// [SAFETY] الحد الأقصى لعدد الوكلاء في الحالة الموحدة
	MaxAgentsInState = 20
	// [SAFETY] الحد الأقصى لعدد المهام في الحالة الموحدة
	MaxTasksInState = 100
	// [SAFETY] الحد الأقصى لاسم الجلسة
	MaxSessionNameLength = 200
	// [SAFETY] الحد الأقصى لوصف الجلسة
	MaxSessionDescriptionLength = 2000
)

// NewSessionContainer ينشئ حاوية جلسة جديدة
// [WHY] يهيئ جميع المكونات بما فيها ChatManager والحالة الموحدة
// [HOW] ينشئ ChatManager ويهيئ UnifiedSessionState
// [SAFETY] يتحقق من أن eventBus ليس nil
func NewSessionContainer(ctx context.Context, db *badger.DB, config *SessionConfig, eb *eventbus.EventBus) (*SessionContainer, error) {
	if eb == nil {
		return nil, fmt.Errorf("eventBus cannot be nil") // [SAFETY] منع nil pointer
	}

	// [SAFETY] التحقق من صحة الإعدادات
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if config.Name == "" {
		return nil, fmt.Errorf("session name cannot be empty")
	}
	if len(config.Name) > MaxSessionNameLength {
		return nil, fmt.Errorf("session name too long (max %d characters)", MaxSessionNameLength)
	}
	if len(config.Description) > MaxSessionDescriptionLength {
		return nil, fmt.Errorf("session description too long (max %d characters)", MaxSessionDescriptionLength)
	}
	if config.OwnerDID == "" {
		return nil, fmt.Errorf("owner DID cannot be empty")
	}
	if config.MaxAgents <= 0 || config.MaxAgents > MaxAgentsInState {
		return nil, fmt.Errorf("max agents must be between 1 and %d", MaxAgentsInState)
	}

	sessionCtx, cancel := context.WithCancel(ctx)

	session := &SessionContainer{
		ID:          fmt.Sprintf("sess_%s", uuid.New().String()),
		Name:        config.Name,
		Description: config.Description,
		OwnerDID:    config.OwnerDID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     1,
		Status:      "active",
		DB:          db,
		EventBus:    eb,
		ctx:         sessionCtx,
		cancelFunc:  cancel,
	}

	// تهيئة المكونات
	session.Memory = NewCollectiveMemory(session.ID, db)
	session.Skills = NewSkillsManager(session.ID)
	session.Workflow = NewWorkflowEngine(session.ID)
	session.Artifacts = NewArtifactsStore(session.ID, db)
	session.Tasks = NewTaskManager(session.ID)
	session.Progress = NewProgressTracker(session.ID)
	session.Handoff = NewHandoffManager(session.ID, "")
	session.Aggregator = NewAggregator(session.ID)
	session.Reviewer = NewFinalReviewer()

	// [WHY] تهيئة ChatManager
	session.ChatManager = NewChatManager(session.ID, eb)

	// [WHY] تهيئة الحالة الموحدة
	session.state = UnifiedSessionState{
		SessionID: session.ID,
		Status:    "active",
		Agents:    make([]AgentInfo, 0),
		Tasks:     make([]TaskInfo, 0),
		Progress: ProgressInfo{
			TotalTasks:     0,
			CompletedTasks: 0,
			Percentage:     0.0,
		},
		UpdatedAt: time.Now(),
	}

	// نشر حدث الإنشاء
	eb.Publish(eventbus.Event{
		Type:      "session.created",
		Payload:   session,
		Source:    "session_container",
		SessionID: session.ID,
	})

	return session, nil
}

// Save يحفظ الجلسة في BadgerDB
func (s *SessionContainer) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.Marshal(s)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("session:%s", s.ID)
	return s.DB.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

// Load يحمل الجلسة من BadgerDB
func (s *SessionContainer) Load(id string) error {
	key := fmt.Sprintf("session:%s", id)

	return s.DB.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, s)
		})
	})
}

// Stop يوقف الجلسة
func (s *SessionContainer) Stop() error {
	s.cancelFunc()
	s.Status = "paused"
	s.UpdatedAt = time.Now()

	s.EventBus.Publish(eventbus.Event{
		Type:      "session.paused",
		Payload:   s.ID,
		Source:    "session_container",
		SessionID: s.ID,
	})

	return s.Save()
}

// Resume يستأنف الجلسة
func (s *SessionContainer) Resume() error {
	s.ctx, s.cancelFunc = context.WithCancel(context.Background())
	s.Status = "active"
	s.UpdatedAt = time.Now()

	// [SAFETY] تحديث الحالة الموحدة
	s.stateMu.Lock()
	s.state.Status = "active"
	s.state.UpdatedAt = time.Now()
	stateCopy := s.state
	s.stateMu.Unlock()

	// [HOW] نشر حدث session.resumed بعد فك القفل
	s.EventBus.Publish(eventbus.Event{
		Type:      "session.resumed",
		Payload:   s.ID,
		Source:    "session_container",
		SessionID: s.ID,
	})

	// [HOW] نشر حدث session.state.changed بعد فك القفل
	s.EventBus.Publish(eventbus.Event{
		Type:      "session.state.changed",
		Payload:   stateCopy,
		Source:    "session_container",
		SessionID: s.ID,
	})

	return nil
}

// [WHY] UpdateTaskStatus يحدث حالة مهمة
// [HOW] يحدث الحالة الموحدة وينشر حدث session.state.changed
// [SAFETY] يفك القفل قبل استدعاء eventBus.Publish لمنع Deadlock
func (s *SessionContainer) UpdateTaskStatus(taskID, status string) error {
	// [SAFETY] قفل للكتابة على الحالة الموحدة
	s.stateMu.Lock()

	// [HOW] تحديث المهمة في الحالة الموحدة
	for i := range s.state.Tasks {
		if s.state.Tasks[i].ID == taskID {
			s.state.Tasks[i].Status = status
			break
		}
	}

	// [HOW] تحديث التقدم
	s.updateProgress()

	// [HOW] نسخ الحالة للنشر
	stateCopy := s.state

	// [SAFETY] فك القفل فوراً قبل النشر لمنع Deadlock
	s.stateMu.Unlock()

	// [HOW] نشر حدث session.state.changed بعد فك القفل
	s.EventBus.Publish(eventbus.Event{
		Type:      "session.state.changed",
		Payload:   stateCopy,
		Source:    "session_container",
		SessionID: s.ID,
	})

	return nil
}

// [WHY] AddTask يضيف مهمة جديدة
// [HOW] يضيف المهمة للحالة الموحدة وينشر حدث session.state.changed
// [SAFETY] يفك القفل قبل استدعاء eventBus.Publish لمنع Deadlock
func (s *SessionContainer) AddTask(taskID, title, assignedTo, priority string) error {
	// [SAFETY] التحقق من صحة المدخلات
	if taskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}
	if title == "" {
		return fmt.Errorf("task title cannot be empty")
	}
	if assignedTo == "" {
		return fmt.Errorf("task assigned to cannot be empty")
	}
	if priority == "" {
		return fmt.Errorf("task priority cannot be empty")
	}

	// [SAFETY] قفل للكتابة على الحالة الموحدة
	s.stateMu.Lock()

	// [SAFETY] التحقق من الحد الأقصى للمهام
	if len(s.state.Tasks) >= MaxTasksInState {
		s.stateMu.Unlock()
		return fmt.Errorf("maximum tasks limit reached (%d)", MaxTasksInState)
	}

	// [HOW] إضافة المهمة للحالة الموحدة
	s.state.Tasks = append(s.state.Tasks, TaskInfo{
		ID:         taskID,
		Title:      title,
		Status:     "pending",
		AssignedTo: assignedTo,
		Priority:   priority,
	})

	// [HOW] تحديث التقدم
	s.updateProgress()

	// [HOW] نسخ الحالة للنشر
	stateCopy := s.state

	// [SAFETY] فك القفل فوراً قبل النشر لمنع Deadlock
	s.stateMu.Unlock()

	// [HOW] نشر حدث session.state.changed بعد فك القفل
	s.EventBus.Publish(eventbus.Event{
		Type:      "session.state.changed",
		Payload:   stateCopy,
		Source:    "session_container",
		SessionID: s.ID,
	})

	return nil
}

// [WHY] AddAgent يضيف وكيل جديد
// [HOW] يضيف الوكيل للحالة الموحدة وينشر حدث session.state.changed
// [SAFETY] يفك القفل قبل استدعاء eventBus.Publish لمنع Deadlock
func (s *SessionContainer) AddAgent(did, name, role string) error {
	// [SAFETY] التحقق من صحة المدخلات
	if did == "" {
		return fmt.Errorf("agent DID cannot be empty")
	}
	if name == "" {
		return fmt.Errorf("agent name cannot be empty")
	}
	if role == "" {
		return fmt.Errorf("agent role cannot be empty")
	}

	// [SAFETY] قفل للكتابة على الحالة الموحدة
	s.stateMu.Lock()

	// [SAFETY] التحقق من الحد الأقصى للوكلاء
	if len(s.state.Agents) >= MaxAgentsInState {
		s.stateMu.Unlock()
		return fmt.Errorf("maximum agents limit reached (%d)", MaxAgentsInState)
	}

	// [HOW] إضافة الوكيل للحالة الموحدة مع معالجة الأخطاء
	defer func() {
		if r := recover(); r != nil {
			s.stateMu.Unlock()
			panic(r) // إعادة إطلاق panic بعد فك القفل
		}
	}()

	s.state.Agents = append(s.state.Agents, AgentInfo{
		DID:    did,
		Name:   name,
		Status: "active",
		Role:   role,
	})

	// [HOW] نسخ الحالة للنشر
	stateCopy := s.state

	// [SAFETY] فك القفل فوراً قبل النشر لمنع Deadlock
	s.stateMu.Unlock()

	// [HOW] نشر حدث session.state.changed بعد فك القفل
	s.EventBus.Publish(eventbus.Event{
		Type:      "session.state.changed",
		Payload:   stateCopy,
		Source:    "session_container",
		SessionID: s.ID,
	})

	return nil
}

// [WHY] GetUnifiedState يحصل على الحالة الموحدة
// [HOW] ينسخ الحالة ويعيدها
// [SAFETY] يستخدم RLock للقراءة فقط
func (s *SessionContainer) GetUnifiedState() UnifiedSessionState {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()

	// [WHY] نسخ الحالة لمنع تعديلها من الخارج
	stateCopy := s.state
	return stateCopy
}

// [WHY] updateProgress يحدث التقدم
// [HOW] يحسب نسبة الإنجاز بناءً على المهام المكتملة
// [SAFETY] يجب استدعاؤه داخل stateMu.Lock()
func (s *SessionContainer) updateProgress() {
	total := len(s.state.Tasks)
	completed := 0

	for _, task := range s.state.Tasks {
		if task.Status == "completed" {
			completed++
		}
	}

	s.state.Progress.TotalTasks = total
	s.state.Progress.CompletedTasks = completed

	if total > 0 {
		s.state.Progress.Percentage = float64(completed) / float64(total) * 100.0
	} else {
		s.state.Progress.Percentage = 0.0
	}

	s.state.UpdatedAt = time.Now()
}
