package connection

import (
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Session يمثل جلسة اتصال مع وكيل
type Session struct {
	id           string
	agentID      string
	conn         net.Conn
	lastActivity time.Time
	mu           sync.RWMutex
	log          *logrus.Logger
}

// NewSession ينشئ جلسة جديدة
func NewSession(id string, conn net.Conn, agentID string, log *logrus.Logger) *Session {
	return &Session{
		id:           id,
		agentID:      agentID,
		conn:         conn,
		lastActivity: time.Now(),
		log:          log,
	}
}

// ID يرجع معرف الجلسة
func (s *Session) ID() string {
	return s.id
}

// AgentID يرجع معرف الوكيل
func (s *Session) AgentID() string {
	return s.agentID
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
