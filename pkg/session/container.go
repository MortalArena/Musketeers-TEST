package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent/tools"
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

	// [NEW] سجل أحداث الجلسة — تاريخ كامل لكل ما حدث
	Journal *SessionJournal // [WHY] يسجل كل حدث في الجلسة للرجوع إليه عند إعادة الفتح أو الانضمام

	// [NEW] سجل الأدوات المركزي
	ToolRegistry *tools.ToolRegistry // [WHY] يسجل جميع الأدوات ويتحكم بالصلاحيات

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
	Name          string `json:"name"`
	Description   string `json:"description"`
	OwnerDID      string `json:"owner_did"`
	MaxAgents     int    `json:"max_agents"`
	ProjectType   string `json:"project_type"`
	SessionFolder string `json:"session_folder,omitempty"` // فولدر الجلسة المخصص
}

// ============================================================
// [FIX] تنظيم فولدر الجلسة بشكل احترافي
// ============================================================

// SessionFolderStructure هيكلية فولدر الجلسة الاحترافية
type SessionFolderStructure struct {
	BasePath string
	// الفولدرات الفرعية
	Attachments string // ملفات المرفقات (صور، ملفات، إلخ)
	Knowledge   string // المعرفة الجماعية (ملفات مارك داون المحولة)
	Artifacts   string // المنتجات النهائية والنتائج
	WorkFiles   string // ملفات العمل المؤقتة
	Logs        string // سجلات الجلسة
	Backup      string // النسخ الاحتياطية
}

// SetupSessionFolderStructure ينشئ هيكلية فولدر الجلسة الاحترافية
func SetupSessionFolderStructure(sessionID, basePath string) (*SessionFolderStructure, error) {
	if basePath == "" {
		basePath = "./sessions/default"
	}

	// إنشاء الفولدر الأساسي
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("فشل إنشاء الفولدر الأساسي: %w", err)
	}

	structure := &SessionFolderStructure{
		BasePath:    basePath,
		Attachments: filepath.Join(basePath, "attachments"),
		Knowledge:   filepath.Join(basePath, "knowledge"),
		Artifacts:   filepath.Join(basePath, "artifacts"),
		WorkFiles:   filepath.Join(basePath, "work_files"),
		Logs:        filepath.Join(basePath, "logs"),
		Backup:      filepath.Join(basePath, "backup"),
	}

	// إنشاء جميع الفولدرات الفرعية
	folders := []string{
		structure.Attachments,
		structure.Knowledge,
		structure.Artifacts,
		structure.WorkFiles,
		structure.Logs,
		structure.Backup,
	}

	for _, folder := range folders {
		if err := os.MkdirAll(folder, 0755); err != nil {
			return nil, fmt.Errorf("فشل إنشاء الفولدر %s: %w", folder, err)
		}
	}

	// إنشاء ملف README في الفولدر الأساسي
	readmePath := filepath.Join(basePath, "README.md")
	readmeContent := fmt.Sprintf(`# Session Folder: %s

## هيكلية الفولدر الاحترافية

### 📁 attachments/
ملفات المرفقات من العميل البشري (صور، ملفات، لينكات)

### 📁 knowledge/
المعرفة الجماعية المحولة إلى مارك داون (متاحة لجميع الوكلاء)

### 📁 artifacts/
المنتجات النهائية والنتائج (للاستلام من قبل العميل)

### 📁 work_files/
ملفات العمل المؤقتة (للاستخدام الداخلي للوكلاء)

### 📁 logs/
سجلات الجلسة والأنشطة

### 📁 backup/
النسخ الاحتياطية والبيانات التاريخية

## معلومات الجلسة
- Session ID: %s
- Created: %s
`, sessionID, sessionID, time.Now().Format("2006-01-02 15:04:05"))

	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return nil, fmt.Errorf("فشل إنشاء ملف README: %w", err)
	}

	return structure, nil
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
	// ربط WorkflowEngine بـ SessionContainer للتكامل
	session.Workflow.SetSessionContainer(session)
	session.Artifacts = NewArtifactsStore(session.ID, db)
	session.Tasks = NewTaskManager(session.ID)
	session.Progress = NewProgressTracker(session.ID)
	session.Handoff = NewHandoffManager(session.ID, "")
	session.Aggregator = NewAggregator(session.ID)
	session.Reviewer = NewFinalReviewer()

	// تهيئة ChatManager مع فولدر الجلسة
	if config.SessionFolder != "" {
		session.ChatManager = NewChatManagerWithFolder(session.ID, eb, config.SessionFolder)
	} else {
		session.ChatManager = NewChatManager(session.ID, eb)
	}

	// [NEW] تهيئة سجل أحداث الجلسة
	session.Journal = NewSessionJournal(session.ID)
	session.Journal.Append(JournalSessionCreated, config.OwnerDID, "human", "تم إنشاء الجلسة", map[string]interface{}{
		"name":        config.Name,
		"description": config.Description,
		"owner":       config.OwnerDID,
		"max_agents":  config.MaxAgents,
	})

	// [NEW] تهيئة سجل الأدوات المركزي
	session.ToolRegistry = tools.NewToolRegistry()
	RegisterSessionTools(session.ToolRegistry, session)

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

	s.Journal.Append(JournalSessionPaused, "system", "system", "تم إيقاف الجلسة مؤقتاً", nil)

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

	s.Journal.Append(JournalSessionResumed, "system", "system", "تم استئناف الجلسة", nil)

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

	// تسجيل في سجل الأحداث
	entryType := JournalTaskUpdated
	if status == "completed" {
		entryType = JournalTaskCompleted
	} else if status == "failed" {
		entryType = JournalTaskFailed
	}
	s.Journal.Append(entryType, "system", "system", "تحديث حالة المهمة: "+taskID+" → "+status, map[string]interface{}{
		"task_id": taskID,
		"status":  status,
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

	// تسجيل في سجل الأحداث
	s.Journal.Append(JournalTaskCreated, assignedTo, "agent", "تم إنشاء مهمة: "+title, map[string]interface{}{
		"task_id":  taskID,
		"title":    title,
		"assigned": assignedTo,
		"priority": priority,
	})

	return nil
}

// VerifyOwner يتحقق من أن المتصل هو مالك الجلسة
func (s *SessionContainer) VerifyOwner(callerDID string) error {
	if callerDID == "" {
		return fmt.Errorf("caller DID cannot be empty")
	}
	if callerDID != s.OwnerDID {
		return fmt.Errorf("caller %s is not the session owner %s", callerDID, s.OwnerDID)
	}
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

	// تسجيل في سجل الأحداث
	s.Journal.Append(JournalAgentAdded, did, "agent", "تم إضافة وكيل: "+name, map[string]interface{}{
		"agent_did":  did,
		"agent_name": name,
		"role":       role,
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

// ReplaceRemoteState يستبدل الحالة المحلية بحالة من مصدر بعيد
// [WHY] يُستخدم عند استقبال session.state.changed من جهاز آخر
// [HOW] يدمج الحالة البعيدة مع المحلية: يضيف العناصر الجديدة فقط
// [SAFETY] لا يحذف العناصر المحلية لتجنب فقدان عمل المستخدمين المحليين
func (s *SessionContainer) ReplaceRemoteState(remote UnifiedSessionState) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if remote.SessionID != s.ID {
		return
	}

	remoteTime := remote.UpdatedAt
	localTime := s.state.UpdatedAt

	// فقط إذا كانت الحالة البعيدة أحدث
	if remoteTime.Before(localTime) && !remoteTime.Equal(localTime) {
		return
	}

	// دمج الوكلاء: أضف الوكلاء الجدد من البعيد
	existingAgents := make(map[string]bool)
	for _, a := range s.state.Agents {
		existingAgents[a.DID] = true
	}
	for _, a := range remote.Agents {
		if !existingAgents[a.DID] {
			s.state.Agents = append(s.state.Agents, a)
		}
	}

	// دمج المهام: أضف المهام الجديدة من البعيد
	existingTasks := make(map[string]bool)
	for _, t := range s.state.Tasks {
		existingTasks[t.ID] = true
	}
	for _, t := range remote.Tasks {
		if !existingTasks[t.ID] {
			s.state.Tasks = append(s.state.Tasks, t)
		} else {
			// تحديث حالة المهمة إذا كانت موجودة
			for i := range s.state.Tasks {
				if s.state.Tasks[i].ID == t.ID {
					s.state.Tasks[i].Status = t.Status
					break
				}
			}
		}
	}

	s.state.Status = remote.Status
	s.state.UpdatedAt = remote.UpdatedAt
	s.updateProgress()

	s.Journal.Append(JournalStateChanged, "remote", "node", "تم تحديث الحالة من جهاز بعيد", map[string]interface{}{
		"remote_agents":   len(remote.Agents),
		"remote_tasks":    len(remote.Tasks),
		"remote_progress": remote.Progress.Percentage,
	})
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

// SessionExportData يحتوي على جميع بيانات الجلسة للتصدير
type SessionExportData struct {
	SessionContainer *SessionContainer   `json:"session_container"`
	State            UnifiedSessionState `json:"state"`
	JournalEntries   []JournalEntry      `json:"journal_entries,omitempty"` // [NEW] سجل الأحداث الكامل
	ExportedAt       time.Time           `json:"exported_at"`
	ExporterDID      string              `json:"exporter_did"`
	Delegation       string              `json:"delegation,omitempty"` // توقيع التفويض للاستيراد
}

// Export يُصدّر الجلسة كاملة (للنقل بين الأجهزة)
// [WHY] يسمح بنسخ الجلسة بالكامل إلى جهاز آخر
func (s *SessionContainer) Export() (*SessionExportData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.stateMu.RLock()
	stateCopy := s.state
	s.stateMu.RUnlock()

	journalCopy := s.Journal.Export()

	data := &SessionExportData{
		SessionContainer: s,
		State:            stateCopy,
		JournalEntries:   journalCopy,
		ExportedAt:       time.Now(),
	}

	return data, nil
}

// Import يحمّل بيانات جلسة من تصدير سابق
// [SAFETY] يتحقق من صحة البيانات وتطابق session ID
func (s *SessionContainer) Import(data *SessionExportData) error {
	if data == nil || data.SessionContainer == nil {
		return fmt.Errorf("بيانات التصدير فارغة")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// التحقق من صحة البيانات
	if data.SessionContainer.ID == "" {
		return fmt.Errorf("معرف الجلسة فارغ في بيانات التصدير")
	}

	// [SAFETY] التحقق من صحة OwnerDID
	if data.SessionContainer.OwnerDID == "" {
		return fmt.Errorf("معرف المالك فارغ في بيانات التصدير")
	}

	// [SAFETY] التحقق من صحة الحالة
	if len(data.State.Agents) == 0 && len(data.State.Tasks) == 0 {
		return fmt.Errorf("الحالة فارغة في بيانات التصدير")
	}

	// [SAFETY] التحقق من حدود الموارد
	if len(data.State.Agents) > MaxAgentsInState {
		return fmt.Errorf("عدد الوكلاء يتجاوز الحد الأقصى: %d > %d", len(data.State.Agents), MaxAgentsInState)
	}
	if len(data.State.Tasks) > MaxTasksInState {
		return fmt.Errorf("عدد المهام يتجاوز الحد الأقصى: %d > %d", len(data.State.Tasks), MaxTasksInState)
	}

	// نسخ بيانات الجلسة المستوردة
	if s.ID == "" {
		s.ID = data.SessionContainer.ID
	} else if s.ID != data.SessionContainer.ID {
		return fmt.Errorf("لا يمكن استيراد جلسة بمعرف مختلف: %s ≠ %s", data.SessionContainer.ID, s.ID)
	}

	s.Name = data.SessionContainer.Name
	s.Description = data.SessionContainer.Description
	s.OwnerDID = data.SessionContainer.OwnerDID
	s.Status = data.SessionContainer.Status
	s.Version = data.SessionContainer.Version
	s.UpdatedAt = time.Now()

	// استيراد الحالة الموحدة
	s.stateMu.Lock()
	s.state = data.State
	if s.state.Agents == nil {
		s.state.Agents = make([]AgentInfo, 0)
	}
	if s.state.Tasks == nil {
		s.state.Tasks = make([]TaskInfo, 0)
	}
	s.state.SessionID = s.ID
	s.state.UpdatedAt = time.Now()
	s.updateProgress()
	s.stateMu.Unlock()

	// استيراد سجل الأحداث
	if len(data.JournalEntries) > 0 && s.Journal != nil {
		s.Journal.Import(data.JournalEntries)
		s.Journal.Append(JournalImported, data.ExporterDID, "human", "تم استيراد الجلسة من جهاز آخر", map[string]interface{}{
			"exported_at": data.ExportedAt,
		})
	}

	return nil
}

// ToJSON يحول بيانات التصدير إلى JSON
func (data *SessionExportData) ToJSON() ([]byte, error) {
	return json.Marshal(data)
}

// FromJSONSession يحمّل بيانات التصدير من JSON
func FromJSONSession(data []byte) (*SessionExportData, error) {
	var export SessionExportData
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, err
	}
	return &export, nil
}
