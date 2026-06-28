package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/agent/tools"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
	"go.uber.org/zap"
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
	EventBus *eventbus.EventBus `json:"-"` // [FIX] غير قابل للتسلسل

	// Storage
	DB *badger.DB `json:"-"` // [FIX] غير قابل للتسلسل

	mu         sync.RWMutex
	ctx        context.Context
	cancelFunc context.CancelFunc

	// Hybrid Persistence
	dirty       bool
	dirtyMu     sync.Mutex
	flushTicker *time.Ticker
	flushDone   chan struct{}

	// [NEW] التحقق من قدرات الوكلاء
	CapabilityVerifier *AgentCapabilityVerifier // [WHY] يتحقق من القدرات المعلنة للوكلاء

	// [NEW] Context Reranker — فهرسة وبحث سياقي ذكي (مثل Cursor @)
	// [FIX] مخزن كـ interface{} لتجنب دوائر الاستيراد مع pkg/agent/thinking
	ContextReranker interface{} `json:"-"` // [WHY] فهرسة جميع ملفات المشروع والبحث الذكي
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
// يحتوي على بيانات الهوية، القدرات المعلنة والمحققة، ومقاييس الأداء
type AgentInfo struct {
	DID    string `json:"did"`    // [WHY] معرف الوكيل
	Name   string `json:"name"`   // [WHY] اسم الوكيل
	Status string `json:"status"` // [WHY] حالة الوكيل
	Role   string `json:"role"`   // [WHY] دور الوكيل

	// [WHY] هوية الوكيل من agent.GetInfo()
	Provider      string `json:"provider,omitempty"`       // [WHY] المزود (claude, openai, ollama)
	Model         string `json:"model,omitempty"`          // [WHY] النموذج (claude-4, gpt-4o)
	ContextWindow int    `json:"context_window,omitempty"` // [WHY] حجم نافذة السياق
	MaxTokens     int    `json:"max_tokens,omitempty"`     // [WHY] الحد الأقصى للتوكنز
	AgentType     string `json:"agent_type,omitempty"`     // [WHY] نوع الوكيل

	// [WHY] القدرات — المعلنة مقابل المحققة
	ClaimedCapabilities  []string `json:"claimed_capabilities,omitempty"`  // [WHY] القدرات التي أعلنها الوكيل
	VerifiedCapabilities []string `json:"verified_capabilities,omitempty"` // [WHY] القدرات التي تم التحقق منها
	FailedCapabilities   []string `json:"failed_capabilities,omitempty"`   // [WHY] القدرات التي فشل التحقق منها
	VerificationStatus   string   `json:"verification_status"`             // [WHY] unverified, verified, partial, failed
	VerifiedAt           int64    `json:"verified_at,omitempty"`           // [WHY] وقت آخر تحقق

	// [WHY] تتبع الأداء في الجلسة
	JoinedAt    int64   `json:"joined_at"`             // [WHY] وقت الانضمام
	LastActive  int64   `json:"last_active,omitempty"` // [WHY] آخر نشاط
	TotalTasks  int     `json:"total_tasks"`           // [WHY] إجمالي المهام
	SuccessRate float64 `json:"success_rate"`          // [WHY] معدل النجاح
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

	// [NEW] تهيئة مدقق قدرات الوكلاء
	session.CapabilityVerifier = NewAgentCapabilityVerifier()

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
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(s)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("session:%s", s.ID)
	err = s.DB.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
	if err == nil {
		s.dirtyMu.Lock()
		s.dirty = false
		s.dirtyMu.Unlock()
	}
	return err
}

// MarkDirty يعلن أن الجلسة بحاجة للحفظ (لنظام Hybrid Persistence)
func (s *SessionContainer) MarkDirty() {
	s.dirtyMu.Lock()
	s.dirty = true
	s.dirtyMu.Unlock()
}

// StartFlushWorker يبدأ عامل دوري يحفظ الجلسة كل 30 ثانية إذا كانت متسخة
func (s *SessionContainer) StartFlushWorker(ctx context.Context) {
	s.flushTicker = time.NewTicker(30 * time.Second)
	s.flushDone = make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				_ = r
			}
		}()
		defer close(s.flushDone)
		for {
			select {
			case <-s.flushTicker.C:
				s.dirtyMu.Lock()
				isDirty := s.dirty
				s.dirtyMu.Unlock()
				if isDirty {
					if err := s.Save(); err != nil {
						// log would go through EventBus if needed
					}
				}
			case <-ctx.Done():
				// Flush one last time before stopping
				s.dirtyMu.Lock()
				isDirty := s.dirty
				s.dirtyMu.Unlock()
				if isDirty {
					_ = s.Save()
				}
				return
			}
		}
	}()
}

// StopFlushWorker يوقف عامل الحفظ الدوري
func (s *SessionContainer) StopFlushWorker() {
	if s.flushTicker != nil {
		s.flushTicker.Stop()
	}
}

// FlushNow يحفظ فوراً إذا كانت الجلسة متسخة
func (s *SessionContainer) FlushNow() error {
	s.dirtyMu.Lock()
	isDirty := s.dirty
	s.dirtyMu.Unlock()
	if !isDirty {
		return nil
	}
	return s.Save()
}

// Load يحمل الجلسة من BadgerDB
// [SAFETY] بعد فك التسلسل، يعيد تهيئة EventBus و DB (لأنهما json:"-")
func (s *SessionContainer) Load(id string, db *badger.DB, eb *eventbus.EventBus) error {
	if db == nil {
		return fmt.Errorf("DB cannot be nil")
	}
	if eb == nil {
		return fmt.Errorf("EventBus cannot be nil")
	}

	key := fmt.Sprintf("session:%s", id)

	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, s)
		})
	})
	if err != nil {
		return err
	}

	// [FIX] إعادة تهيئة الحقول غير القابلة للتسلسل
	s.DB = db
	s.EventBus = eb

	// [FIX] إعادة تهيئة المراجع الداخلية إذا كانت nil بعد Load
	if s.Memory == nil {
		s.Memory = NewCollectiveMemory(s.ID, db)
	}
	if s.Skills == nil {
		s.Skills = NewSkillsManager(s.ID)
	}
	if s.Workflow == nil {
		s.Workflow = NewWorkflowEngine(s.ID)
	}
	if s.Artifacts == nil {
		s.Artifacts = NewArtifactsStore(s.ID, db)
	}
	if s.Tasks == nil {
		s.Tasks = NewTaskManager(s.ID)
	}
	if s.Progress == nil {
		s.Progress = NewProgressTracker(s.ID)
	}
	if s.Handoff == nil {
		s.Handoff = NewHandoffManager(s.ID, "")
	}
	if s.Aggregator == nil {
		s.Aggregator = NewAggregator(s.ID)
	}
	if s.Reviewer == nil {
		s.Reviewer = NewFinalReviewer()
	}
	if s.ChatManager == nil {
		s.ChatManager = NewChatManager(s.ID, eb)
	}
	if s.Journal == nil {
		s.Journal = NewSessionJournal(s.ID)
	}
	if s.ToolRegistry == nil {
		s.ToolRegistry = tools.NewToolRegistry()
		RegisterSessionTools(s.ToolRegistry, s)
	}
	if s.CapabilityVerifier == nil {
		s.CapabilityVerifier = NewAgentCapabilityVerifier()
	}
	if s.ContextReranker == nil {
		// [FIX] إعادة تهيئة ContextReranker بعد Load — يستخدم zap.NewNop() مؤقتاً
		s.InitContextReranker(zap.NewNop())
	}
	if s.ctx == nil {
		s.ctx = context.Background()
		// [FIX] إعادة إنشاء cancelFunc مع context الجديد — يمنع nil pointer في Stop()
		s.ctx, s.cancelFunc = context.WithCancel(s.ctx)
	}
	if s.cancelFunc == nil {
		// [FIX] حتى لو ctx كان موجوداً، ننشئ cancelFunc إذا كانت nil
		s.ctx, s.cancelFunc = context.WithCancel(context.Background())
	}
	if s.flushTicker == nil {
		// [FIX] إعادة إنشاء flushTicker — يمنع nil pointer في StopFlushWorker()
		s.flushTicker = time.NewTicker(30 * time.Second)
	}
	if s.flushDone == nil {
		s.flushDone = make(chan struct{})
	}
	if s.state.Agents == nil {
		s.state.Agents = make([]AgentInfo, 0)
	}
	if s.state.Tasks == nil {
		s.state.Tasks = make([]TaskInfo, 0)
	}

	return nil
}

// Stop يوقف الجلسة
func (s *SessionContainer) Stop() error {
	s.cancelFunc()
	s.Status = "paused"
	s.UpdatedAt = time.Now()
	s.MarkDirty()

	if s.EventBus != nil {
		s.EventBus.Publish(eventbus.Event{
			Type:      "session.paused",
			Payload:   s.ID,
			Source:    "session_container",
			SessionID: s.ID,
		})
	}

	if s.Journal != nil {
		s.Journal.Append(JournalSessionPaused, "system", "system", "تم إيقاف الجلسة مؤقتاً", nil)
	}

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

	if s.EventBus != nil {
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
	}

	if s.Journal != nil {
		s.Journal.Append(JournalSessionResumed, "system", "system", "تم استئناف الجلسة", nil)
	}

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
	if s.Journal != nil {
		s.Journal.Append(entryType, "system", "system", "تحديث حالة المهمة: "+taskID+" → "+status, map[string]interface{}{
			"task_id": taskID,
			"status":  status,
		})
	}

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

	// [HOW] إضافة الوكيل للحالة الموحدة
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

	if s.Journal != nil {
		s.Journal.Append(JournalAgentAdded, did, "agent", "تم إضافة وكيل: "+name, map[string]interface{}{
			"agent_did":  did,
			"agent_name": name,
			"role":       role,
		})
	}

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

	// Hybrid Persistence: أي تغيير في الحالة = علامة للحفظ
	s.dirtyMu.Lock()
	s.dirty = true
	s.dirtyMu.Unlock()
}

// ============================================================
// Agent Capability Verification — التحقق من قدرات الوكلاء
// ============================================================

// RegisterAgentFromUnified يسجل وكيلاً في الجلسة مع بياناته الكاملة
// [WHY] يلتقط بيانات الهوية، القدرات المعلنة، ويجري التحقق
// [HOW] يستخدم AgentInfo من الوكيل لتعبئة الحقول، ثم يشغل التحقق
func (s *SessionContainer) RegisterAgentFromUnified(ua agent.UnifiedAgent, role string) (*AgentInfo, error) {
	if ua == nil {
		return nil, fmt.Errorf("unified agent cannot be nil")
	}

	info := ua.GetInfo()
	if info == nil {
		return nil, fmt.Errorf("agent info cannot be nil")
	}

	now := time.Now().UnixMilli()

	// [WHY] جمع القدرات المعلنة
	claimedCap := ua.GetCapabilities()
	claimedStrs := make([]string, len(claimedCap))
	for i, c := range claimedCap {
		claimedStrs[i] = string(c)
	}

	// [WHY] إضافة الوكيل إلى الحالة الموحدة
	err := s.AddAgent(info.ID, info.Name, role)
	if err != nil {
		return nil, fmt.Errorf("failed to add agent to session: %w", err)
	}

	// [WHY] تحديث بيانات الوكيل بالبيانات الكاملة
	s.stateMu.Lock()
	for i := range s.state.Agents {
		if s.state.Agents[i].DID == info.ID {
			s.state.Agents[i].Provider = info.Provider
			s.state.Agents[i].Model = info.Model
			s.state.Agents[i].ContextWindow = info.ContextWindow
			s.state.Agents[i].MaxTokens = info.MaxTokens
			s.state.Agents[i].AgentType = string(info.Type)
			s.state.Agents[i].ClaimedCapabilities = claimedStrs
			s.state.Agents[i].VerificationStatus = string(VerificationUnverified)
			s.state.Agents[i].JoinedAt = now
			s.state.Agents[i].LastActive = now
		}
	}
	stateCopy := s.state
	s.stateMu.Unlock()

	// [WHY] تسجيل في سجل الأحداث
	s.Journal.Append(JournalAgentCapabilities, info.ID, "agent",
		"تم تسجيل الوكيل مع "+fmt.Sprintf("%d", len(claimedStrs))+" قدرة معلنة",
		map[string]interface{}{
			"agent_id":     info.ID,
			"agent_name":   info.Name,
			"role":         role,
			"model":        info.Model,
			"provider":     info.Provider,
			"capabilities": claimedStrs,
		})

	// [WHY] نشر حدث تحديث الحالة
	s.EventBus.Publish(eventbus.Event{
		Type:      "session.state.changed",
		Payload:   stateCopy,
		Source:    "session_container",
		SessionID: s.ID,
	})

	// [WHY] تشغيل التحقق من القدرات
	report, err := s.CapabilityVerifier.VerifyAll(s.ctx, ua)
	if err == nil {
		_ = s.SetAgentVerification(info.ID, report)
	}

	// إرجاع معلومات الوكيل المحدثة
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	for i := range s.state.Agents {
		if s.state.Agents[i].DID == info.ID {
			agentCopy := s.state.Agents[i]
			return &agentCopy, nil
		}
	}
	return nil, fmt.Errorf("agent not found after registration")
}

// SetAgentVerification يحدث حالة التحقق لقدرات وكيل
// [WHY] يخزن نتائج التحقق في AgentInfo للجلسة
func (s *SessionContainer) SetAgentVerification(agentID string, report *VerificationReport) error {
	if report == nil {
		return fmt.Errorf("verification report cannot be nil")
	}

	s.stateMu.Lock()
	found := false
	for i := range s.state.Agents {
		if s.state.Agents[i].DID == agentID {
			s.state.Agents[i].VerifiedCapabilities = report.Verified
			s.state.Agents[i].FailedCapabilities = report.Failed
			s.state.Agents[i].VerificationStatus = report.OverallStatus
			s.state.Agents[i].VerifiedAt = report.VerifiedAt.UnixMilli()
			found = true
			break
		}
	}
	stateCopy := s.state
	s.stateMu.Unlock()

	if !found {
		return fmt.Errorf("agent %s not found in session", agentID)
	}

	// [WHY] تسجيل في سجل الأحداث
	statusEmoji := report.OverallStatus
	s.Journal.Append(JournalCapabilityVerification, agentID, "system",
		"تم التحقق من قدرات الوكيل: "+statusEmoji,
		map[string]interface{}{
			"agent_id": agentID,
			"verified": report.Verified,
			"failed":   report.Failed,
			"status":   report.OverallStatus,
			"probes":   len(report.Probes),
		})

	// [WHY] نشر حدث تحديث الحالة
	s.EventBus.Publish(eventbus.Event{
		Type:      "session.state.changed",
		Payload:   stateCopy,
		Source:    "session_container",
		SessionID: s.ID,
	})

	return nil
}

// GetVerifiedCapabilities يرجع القدرات المحققة فقط لوكيل
// [WHY] يوفر مصدر موثوق للقدرات لتوزيع المهام
func (s *SessionContainer) GetVerifiedCapabilities(agentID string) []string {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()

	for i := range s.state.Agents {
		if s.state.Agents[i].DID == agentID {
			return s.state.Agents[i].VerifiedCapabilities
		}
	}
	return nil
}

// UpdateAgentTaskResult يحدث إحصائيات أداء الوكيل بعد تنفيذ مهمة
// [WHY] يحتفظ بسجل أداء حقيقي لكل وكيل داخل الجلسة
func (s *SessionContainer) UpdateAgentTaskResult(agentID string, success bool) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	for i := range s.state.Agents {
		if s.state.Agents[i].DID == agentID {
			now := time.Now().UnixMilli()
			s.state.Agents[i].LastActive = now
			s.state.Agents[i].TotalTasks++

			total := float64(s.state.Agents[i].TotalTasks)
			if total > 0 {
				// Simple moving average
				if success {
					s.state.Agents[i].SuccessRate = (s.state.Agents[i].SuccessRate*(total-1) + 1.0) / total
				} else {
					s.state.Agents[i].SuccessRate = (s.state.Agents[i].SuccessRate * (total - 1)) / total
				}
			}

			// Hybrid Persistence: علامة للحفظ
			s.dirtyMu.Lock()
			s.dirty = true
			s.dirtyMu.Unlock()
			break
		}
	}
}

// AgentRecord يرجع سجل كامل لوكيل في الجلسة
// [WHY] يعطي صورة كاملة عن الوكيل: هويته، قدراته، أداؤه
func (s *SessionContainer) AgentRecord(agentID string) *AgentInfo {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()

	for i := range s.state.Agents {
		if s.state.Agents[i].DID == agentID {
			record := s.state.Agents[i]
			return &record
		}
	}
	return nil
}

// AllAgentRecords يرجع سجلات جميع الوكلاء في الجلسة
func (s *SessionContainer) AllAgentRecords() []AgentInfo {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()

	records := make([]AgentInfo, len(s.state.Agents))
	copy(records, s.state.Agents)
	return records
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
// [FIX] يستعيد الحقول غير القابلة للتسلسل (EventBus, DB)
func (s *SessionContainer) Import(data *SessionExportData, db *badger.DB, eb *eventbus.EventBus) error {
	if data == nil || data.SessionContainer == nil {
		return fmt.Errorf("بيانات التصدير فارغة")
	}
	if db == nil {
		return fmt.Errorf("DB cannot be nil")
	}
	if eb == nil {
		return fmt.Errorf("EventBus cannot be nil")
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

	// [SAFETY] التحقق من تطابق معرف الجلسة قبل أي تحقق آخر من الحالة
	if s.ID != "" && s.ID != data.SessionContainer.ID {
		return fmt.Errorf("لا يمكن استيراد جلسة بمعرف مختلف: %s ≠ %s", data.SessionContainer.ID, s.ID)
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
	}

	s.Name = data.SessionContainer.Name
	s.Description = data.SessionContainer.Description
	s.OwnerDID = data.SessionContainer.OwnerDID
	s.Status = data.SessionContainer.Status
	s.Version = data.SessionContainer.Version
	s.UpdatedAt = time.Now()

	// [FIX] استعادة الحقول غير القابلة للتسلسل
	s.DB = db
	s.EventBus = eb

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
	if len(data.JournalEntries) > 0 {
		if s.Journal == nil {
			s.Journal = NewSessionJournal(s.ID)
		}
		s.Journal.Import(data.JournalEntries)
		if data.ExporterDID != "" {
			s.Journal.Append(JournalImported, data.ExporterDID, "human", "تم استيراد الجلسة من جهاز آخر", map[string]interface{}{
				"exported_at": data.ExportedAt,
			})
		}
	}

	// [FIX] إعادة تهيئة المكونات المفقودة بعد الاستيراد
	if s.Memory == nil {
		s.Memory = NewCollectiveMemory(s.ID, db)
	}
	if s.Skills == nil {
		s.Skills = NewSkillsManager(s.ID)
	}
	if s.Workflow == nil {
		s.Workflow = NewWorkflowEngine(s.ID)
	}
	if s.Artifacts == nil {
		s.Artifacts = NewArtifactsStore(s.ID, db)
	}
	if s.Tasks == nil {
		s.Tasks = NewTaskManager(s.ID)
	}
	if s.Progress == nil {
		s.Progress = NewProgressTracker(s.ID)
	}
	if s.Handoff == nil {
		s.Handoff = NewHandoffManager(s.ID, "")
	}
	if s.Aggregator == nil {
		s.Aggregator = NewAggregator(s.ID)
	}
	if s.Reviewer == nil {
		s.Reviewer = NewFinalReviewer()
	}
	if s.ChatManager == nil {
		s.ChatManager = NewChatManager(s.ID, eb)
	}
	if s.ToolRegistry == nil {
		s.ToolRegistry = tools.NewToolRegistry()
		RegisterSessionTools(s.ToolRegistry, s)
	}
	if s.CapabilityVerifier == nil {
		s.CapabilityVerifier = NewAgentCapabilityVerifier()
	}
	if s.ContextReranker == nil {
		s.initContextRerankerUnsafe(zap.NewNop())
	}
	if s.ctx == nil {
		s.ctx = context.Background()
		// [FIX] إعادة إنشاء cancelFunc يمنع nil pointer في Stop()
		s.ctx, s.cancelFunc = context.WithCancel(s.ctx)
	}
	if s.cancelFunc == nil {
		s.ctx, s.cancelFunc = context.WithCancel(context.Background())
	}
	if s.flushTicker == nil {
		s.flushTicker = time.NewTicker(30 * time.Second)
	}
	if s.flushDone == nil {
		s.flushDone = make(chan struct{})
	}

	return nil
}

// InitContextReranker يهيئ محرك البحث السياقي — يُستدعى بعد إنشاء الحاوية
func (s *SessionContainer) InitContextReranker(logger *zap.Logger) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.initContextRerankerUnsafe(logger)
}

// initContextRerankerUnsafe يهيئ محرك البحث السياقي بدون قفل — يُستدعى فقط مع المُتصل الذي يحمل القفل
func (s *SessionContainer) initContextRerankerUnsafe(logger *zap.Logger) {
	if s.ContextReranker != nil {
		return
	}

	projectRoot := "."
	candidates := []string{".", "..", "../..", "../../..", "../../../.."}
	for _, p := range candidates {
		fullPath := filepath.Join(p, "go.mod")
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			abs, _ := filepath.Abs(p)
			projectRoot = abs
			break
		}
	}
	if projectRoot == "." && s.Name != "" {
		projectRoot = filepath.Join(".", "sessions", s.ID)
	}

	logger.Info("ContextReranker يجب أن يتم ضبطه من UnifiedAgent لتجنب import cycle",
		zap.String("session_id", s.ID),
		zap.String("project_root", projectRoot))
}

// GetContextReranker يرجع ContextReranker — يهيئه تلقائياً إذا لم يكن موجوداً
// [FIX] يرجع interface{} لتجنب import cycle
func (s *SessionContainer) GetContextReranker(logger *zap.Logger) interface{} {
	if s.ContextReranker == nil {
		s.InitContextReranker(logger)
	}
	return s.ContextReranker
}

// SetContextReranker يضبط ContextReranker من الخارج (من UnifiedAgent)
// [FIX] يقبل interface{} لتجنب import cycle
func (s *SessionContainer) SetContextReranker(cr interface{}) {
	s.ContextReranker = cr
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
