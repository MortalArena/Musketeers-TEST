package connection

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ConnectionManager مدير جلسات الاتصال
type ConnectionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	log      *logrus.Logger
	cleanup  *time.Ticker
}

// NewConnectionManager ينشئ مدير اتصالات جديد
func NewConnectionManager(log *logrus.Logger) *ConnectionManager {
	cm := &ConnectionManager{
		sessions: make(map[string]*Session),
		log:      log,
		cleanup:  time.NewTicker(5 * time.Minute),
	}
	go cm.cleanupRoutine()
	return cm
}

// Register يسجل جلسة جديدة
func (cm *ConnectionManager) Register(session *Session) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.sessions[session.ID()]; exists {
		return fmt.Errorf("session already exists: %s", session.ID())
	}

	cm.sessions[session.ID()] = session
	cm.log.WithField("session_id", session.ID()).Info("Session registered")
	return nil
}

// GetOrCreate يجلب جلسة موجودة أو ينشئ واحدة جديدة
func (cm *ConnectionManager) GetOrCreate(agentID string, conn net.Conn) *Session {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// ابحث عن جلسة موجودة لنفس الوكيل
	for _, session := range cm.sessions {
		if session.AgentID() == agentID {
			session.UpdateLastActivity()
			cm.log.WithField("session_id", session.ID()).WithField("agent_id", agentID).Info("Reusing existing session")
			return session
		}
	}

	// أنشئ جلسة جديدة
	sessionID := generateSessionID()
	session := NewSession(sessionID, conn, agentID, cm.log)
	cm.sessions[sessionID] = session
	cm.log.WithField("session_id", sessionID).WithField("agent_id", agentID).Info("Created new session")
	return session
}

// Unregister يلغي تسجيل جلسة
func (cm *ConnectionManager) Unregister(sessionID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if session, exists := cm.sessions[sessionID]; exists {
		if session.Conn() != nil {
			session.Conn().Close()
		}
		delete(cm.sessions, sessionID)
		cm.log.WithField("session_id", sessionID).Info("Session unregistered")
	}
}

// Get يجلب جلسة بالمعرف
func (cm *ConnectionManager) Get(sessionID string) (*Session, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	session, exists := cm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return session, nil
}

// GetAll يرجع جميع الجلسات
func (cm *ConnectionManager) GetAll() []*Session {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	sessions := make([]*Session, 0, len(cm.sessions))
	for _, session := range cm.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// Count يرجع عدد الجلسات النشطة
func (cm *ConnectionManager) Count() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.sessions)
}

// CloseAll يغلق جميع الجلسات
func (cm *ConnectionManager) CloseAll() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for sessionID, session := range cm.sessions {
		if session.Conn() != nil {
			session.Conn().Close()
		}
		delete(cm.sessions, sessionID)
	}
	cm.log.Info("All sessions closed")
}

// cleanupRoutine ينظف الجلسات الخاملة
func (cm *ConnectionManager) cleanupRoutine() {
	for range cm.cleanup.C {
		cm.cleanupInactiveSessions()
	}
}

// cleanupInactiveSessions ينظف الجلسات الخاملة لأكثر من 30 دقيقة
func (cm *ConnectionManager) cleanupInactiveSessions() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	inactiveThreshold := 30 * time.Minute

	for sessionID, session := range cm.sessions {
		if now.Sub(session.LastActivity()) > inactiveThreshold {
			cm.log.WithField("session_id", sessionID).Info("Closing inactive session")
			if session.Conn() != nil {
				session.Conn().Close()
			}
			delete(cm.sessions, sessionID)
		}
	}
}

// Stop يوقف مدير الاتصالات
func (cm *ConnectionManager) Stop() {
	cm.cleanup.Stop()
	cm.CloseAll()
}

// generateSessionID يولد معرف جلسة فريد
func generateSessionID() string {
	return fmt.Sprintf("conn_%d", time.Now().UnixNano())
}
