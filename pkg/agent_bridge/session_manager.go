package agent_bridge

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Session يمثل جلسة اتصال مع وكيل
type Session struct {
	id            string
	conn          net.Conn
	lastActivity  time.Time
	mu            sync.RWMutex
	log           *logrus.Logger
}

// NewSession ينشئ جلسة جديدة
func NewSession(id string, conn net.Conn, log *logrus.Logger) *Session {
	return &Session{
		id:           id,
		conn:         conn,
		lastActivity: time.Now(),
		log:          log,
	}
}

// ID يرجع معرف الجلسة
func (s *Session) ID() string {
	return s.id
}

// Conn يرجع اتصال الجلسة
func (s *Session) Conn() net.Conn {
	return s.conn
}

// LastActivity يرجع وقت آخر نشاط
func (s *Session) LastActivity() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastActivity
}

// UpdateLastActivity يحدث وقت آخر نشاط
func (s *Session) UpdateLastActivity() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastActivity = time.Now()
}

// SessionManager يدير جلسات الاتصال
type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	log      *logrus.Logger
	cleanup  *time.Ticker
}

// NewSessionManager ينشئ مدير جلسات جديد
func NewSessionManager(log *logrus.Logger) *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
		log:      log,
		cleanup:  time.NewTicker(5 * time.Minute),
	}
	go sm.cleanupRoutine()
	return sm
}

// Register يسجل جلسة جديدة
func (sm *SessionManager) Register(session *Session) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.sessions[session.ID()]; exists {
		return fmt.Errorf("session already exists: %s", session.ID())
	}

	sm.sessions[session.ID()] = session
	sm.log.WithField("session_id", session.ID()).Info("Session registered")
	return nil
}

// Unregister يلغي تسجيل جلسة
func (sm *SessionManager) Unregister(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		if session.Conn() != nil {
			session.Conn().Close()
		}
		delete(sm.sessions, sessionID)
		sm.log.WithField("session_id", sessionID).Info("Session unregistered")
	}
}

// Get يجلب جلسة بالمعرف
func (sm *SessionManager) Get(sessionID string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return session, nil
}

// GetAll يرجع جميع الجلسات
func (sm *SessionManager) GetAll() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*Session, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// Count يرجع عدد الجلسات النشطة
func (sm *SessionManager) Count() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}

// CloseAll يغلق جميع الجلسات
func (sm *SessionManager) CloseAll() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for sessionID, session := range sm.sessions {
		if session.Conn() != nil {
			session.Conn().Close()
		}
		delete(sm.sessions, sessionID)
	}
	sm.log.Info("All sessions closed")
}

// cleanupRoutine ينظف الجلسات الخاملة
func (sm *SessionManager) cleanupRoutine() {
	for range sm.cleanup.C {
		sm.cleanupInactiveSessions()
	}
}

// cleanupInactiveSessions ينظف الجلسات الخاملة لأكثر من 30 دقيقة
func (sm *SessionManager) cleanupInactiveSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	inactiveThreshold := 30 * time.Minute

	for sessionID, session := range sm.sessions {
		if now.Sub(session.LastActivity()) > inactiveThreshold {
			sm.log.WithField("session_id", sessionID).Info("Closing inactive session")
			if session.Conn() != nil {
				session.Conn().Close()
			}
			delete(sm.sessions, sessionID)
		}
	}
}

// Stop يوقف مدير الجلسات
func (sm *SessionManager) Stop() {
	sm.cleanup.Stop()
	sm.CloseAll()
}
