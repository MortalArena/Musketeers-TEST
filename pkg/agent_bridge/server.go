package agent_bridge

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/agent_bridge/protocol"
	"github.com/MortalArena/Musketeers/pkg/node"
	"github.com/sirupsen/logrus"
)

// Server خادم جسر الوكلاء
type Server struct {
	node           *node.Node
	addr           string
	listener       net.Listener
	sessionMgr     *SessionManager
	multiplexedBrg *MultiplexedBridge
	log            *logrus.Logger
	mu             sync.RWMutex
	running        bool
	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc
}

// NewServer ينشئ خادم جسر جديد
func NewServer(n *node.Node, addr string, sessionMgr *SessionManager, multiplexedBrg *MultiplexedBridge, log *logrus.Logger) *Server {
	return &Server{
		node:           n,
		addr:           addr,
		sessionMgr:     sessionMgr,
		multiplexedBrg: multiplexedBrg,
		log:            log,
	}
}

// Start يبدأ الخادم
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}
	s.running = true
	s.shutdownCtx, s.shutdownCancel = context.WithCancel(ctx)
	s.mu.Unlock()

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.addr, err)
	}
	s.listener = listener

	s.log.WithField("addr", s.addr).Info("Agent Bridge Server started")

	go s.acceptConnections()

	return nil
}

// acceptConnections يقبل الاتصالات الواردة
func (s *Server) acceptConnections() {
	for {
		select {
		case <-s.shutdownCtx.Done():
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				s.log.WithError(err).Error("Failed to accept connection")
				continue
			}

			go s.handleConnection(conn)
		}
	}
}

// handleConnection يعالج اتصال جديد
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	// ✅ استخدام GetOrCreate لإعادة استخدام الجلسات الموجودة
	// في التنفيذ الحالي، نستخدم sessionID كـ agentID مؤقتاً
	// في المستقبل، سيتم استخراج agentID من المصادقة
	agentID := generateSessionID()
	session := s.sessionMgr.GetOrCreate(agentID, conn)

	s.log.WithField("session_id", session.ID()).WithField("agent_id", agentID).Info("Session established")

	// بدء معالجة الرسائل
	for {
		select {
		case <-s.shutdownCtx.Done():
			return
		default:
			msg, err := protocol.ReadMessage(conn)
			if err != nil {
				s.log.WithError(err).WithField("session_id", session.ID()).Error("Failed to read message")
				return
			}

			if err := s.handleMessage(session, msg); err != nil {
				s.log.WithError(err).WithField("session_id", session.ID()).Error("Failed to handle message")
				return
			}
		}
	}
}

// handleMessage يعالج رسالة من جلسة
func (s *Server) handleMessage(session *Session, msg *protocol.Message) error {
	switch msg.Type {
	case protocol.MessageTypeTaskRequest:
		return s.multiplexedBrg.HandleTaskRequest(session, msg)
	case protocol.MessageTypeTaskResponse:
		return s.multiplexedBrg.HandleTaskResponse(session, msg)
	case protocol.MessageTypeHeartbeat:
		return s.handleHeartbeat(session, msg)
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

// handleHeartbeat يعالج نبض القلب
func (s *Server) handleHeartbeat(session *Session, msg *protocol.Message) error {
	session.UpdateLastActivity()
	return protocol.WriteMessage(session.Conn(), protocol.NewMessage(protocol.MessageTypeHeartbeatAck, nil))
}

// Stop يوقف الخادم
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false
	s.shutdownCancel()

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			s.log.WithError(err).Error("Failed to close listener")
		}
	}

	s.sessionMgr.CloseAll()

	s.log.Info("Agent Bridge Server stopped")
	return nil
}
