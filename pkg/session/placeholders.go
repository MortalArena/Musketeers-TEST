package session

import (
	"github.com/dgraph-io/badger/v4"
)

// RolesManager مدير الأدوار - placeholder
type RolesManager struct {
	SessionID string
}

// NewRolesManager ينشئ مدير أدوار جديد
func NewRolesManager(sessionID string) *RolesManager {
	return &RolesManager{
		SessionID: sessionID,
	}
}

// ChatHistory تاريخ المحادثات - placeholder
type ChatHistory struct {
	SessionID string
}

// NewChatHistory ينشئ تاريخ محادثات جديد
func NewChatHistory(sessionID string) *ChatHistory {
	return &ChatHistory{
		SessionID: sessionID,
	}
}

// ArtifactsStore مخزن القطع الأثرية - placeholder
type ArtifactsStore struct {
	SessionID string
	DB        *badger.DB
}

// NewArtifactsStore ينشئ مخزن قطع أثرية جديد
func NewArtifactsStore(sessionID string, db *badger.DB) *ArtifactsStore {
	return &ArtifactsStore{
		SessionID: sessionID,
		DB:        db,
	}
}

// TaskManager مدير المهام - placeholder
type TaskManager struct {
	SessionID string
}

// NewTaskManager ينشئ مدير مهام جديد
func NewTaskManager(sessionID string) *TaskManager {
	return &TaskManager{
		SessionID: sessionID,
	}
}
