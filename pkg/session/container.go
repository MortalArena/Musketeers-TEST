package session

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
)

// SessionContainer الحاوية الكاملة للجلسة - القلب النابض
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
	Roles      *RolesManager
	Chat       *ChatHistory
	Artifacts  *ArtifactsStore
	Tasks      *TaskManager

	// Event Bus
	EventBus *eventbus.EventBus

	// Storage
	DB *badger.DB

	mu         sync.RWMutex
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// SessionConfig إعدادات الجلسة
type SessionConfig struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	OwnerDID    string `json:"owner_did"`
	MaxAgents   int    `json:"max_agents"`
	ProjectType string `json:"project_type"`
}

// NewSessionContainer ينشئ حاوية جلسة جديدة
func NewSessionContainer(ctx context.Context, db *badger.DB, config *SessionConfig, eb *eventbus.EventBus) (*SessionContainer, error) {
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
	session.Roles = NewRolesManager(session.ID)
	session.Chat = NewChatHistory(session.ID)
	session.Artifacts = NewArtifactsStore(session.ID, db)
	session.Tasks = NewTaskManager(session.ID)

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

	s.EventBus.Publish(eventbus.Event{
		Type:      "session.resumed",
		Payload:   s.ID,
		Source:    "session_container",
		SessionID: s.ID,
	})

	return nil
}
