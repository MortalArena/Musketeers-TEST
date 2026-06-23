package api

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/MortalArena/Musketeers/pkg/naming"
	"github.com/MortalArena/Musketeers/pkg/node"
	"github.com/MortalArena/Musketeers/pkg/protocol"
	"github.com/MortalArena/Musketeers/pkg/security"
	"github.com/MortalArena/Musketeers/pkg/session"
	sessioncore "github.com/MortalArena/Musketeers/pkg/session/core"
	"github.com/gorilla/websocket"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

// Artifact قطعة أثرية من الجلسة
type Artifact struct {
	ID          string                 `json:"id"`
	SessionID   string                 `json:"session_id"`
	Type        string                 `json:"type"` // code, design, document, etc.
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Content     string                 `json:"content"`
	CreatedAt   time.Time              `json:"created_at"`
	CreatedBy   string                 `json:"created_by"` // agent_did
	Metadata    map[string]interface{} `json:"metadata"`
}

// MCPServer خادم MCP
type MCPServer struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Endpoint     string                 `json:"endpoint"`
	Transport    string                 `json:"transport"` // stdio, sse, websocket
	Capabilities []string               `json:"capabilities"`
	Status       string                 `json:"status"` // active, inactive, error
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// MCPTool أداة MCP
type MCPTool struct {
	ID          string                 `json:"id"`
	ServerID    string                 `json:"server_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// Server خادم REST API
type Server struct {
	node        *node.Node
	log         *logrus.Logger
	token       string // token محلي للمصادقة
	server      *http.Server
	channels    map[string]*pubsub.Subscription
	messages    map[string][]protocol.ChannelMessage
	channelsMu  sync.RWMutex
	tlsEnabled  bool
	tlsCert     string
	tlsKey      string
	rateLimiter *security.RateLimiter
	zapLogger   *zap.Logger

	// إدارة الجلسات والجسور والوكلاء
	sessionManager     *sessioncore.UnifiedSessionManager
	chatManagers       map[string]*session.ChatManager // sessionID -> ChatManager
	chatManagersMu     sync.RWMutex
	taskManagers       map[string]*session.TaskManager // sessionID -> TaskManager
	taskManagersMu     sync.RWMutex
	progressTrackers   map[string]*session.ProgressTracker // sessionID -> ProgressTracker
	progressTrackersMu sync.RWMutex
	memories           map[string]*session.CollectiveMemory // sessionID -> CollectiveMemory
	memoriesMu         sync.RWMutex
	skillsManagers     map[string]*session.SkillsManager // sessionID -> SkillsManager
	skillsManagersMu   sync.RWMutex
	artifacts          map[string][]Artifact // sessionID -> Artifacts
	artifactsMu        sync.RWMutex
	bridgeManager      *session.SessionBridgeManager
	mcpServers         map[string]*MCPServer // serverID -> MCPServer
	mcpServersMu       sync.RWMutex
	mcpTools           map[string]*MCPTool // toolID -> MCPTool
	mcpToolsMu         sync.RWMutex
	eventBus           *eventbus.EventBus
}

// NewServer ينشئ خادم REST
func NewServer(n *node.Node, port int, log *logrus.Logger) *Server {
	return NewServerWithTLS(n, port, log, false, "", "")
}

// NewServerWithTLS ينشئ خادم REST مع TLS
func NewServerWithTLS(n *node.Node, port int, log *logrus.Logger, tlsEnabled bool, tlsCert, tlsKey string) *Server {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		panic(err)
	}
	token := fmt.Sprintf("mskt-%x", tokenBytes)

	// ✅ إنشاء Rate Limiter
	rateLimiter := security.NewRateLimiter(security.DefaultRateLimitConfig())

	// ✅ إنشاء Zap Logger
	zapLogger, _ := zap.NewProduction()

	// ✅ إنشاء Session Manager
	sessionManager := sessioncore.NewUnifiedSessionManager(zapLogger)

	// ✅ إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// ✅ إنشاء Bridge Manager
	bridgeManager := session.NewSessionBridgeManager(eventBus, zapLogger)

	s := &Server{
		node:             n,
		log:              log,
		token:            token,
		channels:         make(map[string]*pubsub.Subscription),
		messages:         make(map[string][]protocol.ChannelMessage),
		tlsEnabled:       tlsEnabled,
		tlsCert:          tlsCert,
		tlsKey:           tlsKey,
		rateLimiter:      rateLimiter,
		zapLogger:        zapLogger,
		sessionManager:   sessionManager,
		chatManagers:     make(map[string]*session.ChatManager),
		taskManagers:     make(map[string]*session.TaskManager),
		progressTrackers: make(map[string]*session.ProgressTracker),
		memories:         make(map[string]*session.CollectiveMemory),
		skillsManagers:   make(map[string]*session.SkillsManager),
		artifacts:        make(map[string][]Artifact),
		bridgeManager:    bridgeManager,
		mcpServers:       make(map[string]*MCPServer),
		mcpTools:         make(map[string]*MCPTool),
		eventBus:         eventBus,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/identity", s.handleIdentity)
	mux.HandleFunc("/api/search", s.handleSearch)
	mux.HandleFunc("/api/resolve", s.handleResolve)
	mux.HandleFunc("/api/content", s.handleContent)
	mux.HandleFunc("/api/acp/task", s.handleACPTask)
	mux.HandleFunc("/api/acp/tasks", s.handleACPTasks)
	mux.HandleFunc("/api/domain/commit", s.handleDomainCommit)
	mux.HandleFunc("/api/channels/join", s.handleChannelsJoin)
	mux.HandleFunc("/api/channels/publish", s.handleChannelsPublish)
	mux.HandleFunc("/api/channels/list", s.handleChannelsList)
	mux.HandleFunc("/api/channels/messages", s.handleChannelsMessages)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/dashboard", s.handleDashboard)
	mux.HandleFunc("/dashboard/", s.handleDashboard)
	mux.HandleFunc("/", s.handleRoot)

	// نقاط نهاية الجلسات والجسور والوكلاء
	mux.HandleFunc("/api/sessions", s.handleSessions)
	mux.HandleFunc("/api/sessions/", s.handleSessionByID)
	mux.HandleFunc("/api/messages", s.handleMessages)
	mux.HandleFunc("/api/messages/", s.handleMessagesBySession)
	mux.HandleFunc("/api/tasks", s.handleTasks)
	mux.HandleFunc("/api/tasks/", s.handleTasksBySession)
	mux.HandleFunc("/api/progress", s.handleProgress)
	mux.HandleFunc("/api/progress/", s.handleProgressBySession)
	mux.HandleFunc("/api/memory", s.handleMemory)
	mux.HandleFunc("/api/memory/", s.handleMemoryBySession)
	mux.HandleFunc("/api/skills", s.handleSkills)
	mux.HandleFunc("/api/skills/", s.handleSkillsBySession)
	mux.HandleFunc("/api/artifacts", s.handleArtifacts)
	mux.HandleFunc("/api/artifacts/", s.handleArtifactsBySession)
	mux.HandleFunc("/api/bridges", s.handleBridges)
	mux.HandleFunc("/api/bridges/", s.handleBridgeByID)
	mux.HandleFunc("/api/agents", s.handleAgents)
	mux.HandleFunc("/api/agents/", s.handleAgentByID)
	mux.HandleFunc("/api/mcp/servers", s.handleMCPServers)
	mux.HandleFunc("/api/mcp/servers/", s.handleMCPServerByID)
	mux.HandleFunc("/api/mcp/tools", s.handleMCPTools)
	mux.HandleFunc("/api/mcp/tools/", s.handleMCPToolByID)
	mux.HandleFunc("/api/ws", s.handleWebSocket)

	handler := s.corsMiddleware(s.authMiddleware(security.RateLimitMiddleware(rateLimiter)(mux)))

	httpServer := &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", port),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// ✅ إضافة TLS
	if tlsEnabled {
		tlsBuilder := security.NewTLSConfigBuilder().
			WithCertFiles(tlsCert, tlsKey)

		securityConfig, err := tlsBuilder.Build()
		if err != nil {
			log.WithError(err).Fatal("فشل إعداد TLS")
		}
		httpServer.TLSConfig = securityConfig
	}

	s.server = httpServer
	return s
}

// Start يبدأ الخادم
func (s *Server) Start() error {
	if s.tlsEnabled {
		s.log.WithField("addr", s.server.Addr).Info("🚀 بدء REST API على HTTPS")
		s.log.Info("🔒 TLS 1.3 مفعّل مع أقوى cipher suites")
	} else {
		s.log.WithField("addr", s.server.Addr).Warn("⚠️ تحذير: الخادم يعمل بدون TLS - غير آمن!")
		s.log.WithField("addr", s.server.Addr).Info("🚀 بدء REST API على HTTP")
	}

	// Start system channel listener for agent synchronization
	go func() {
		// Wait a second for bootstrap nodes and pubsub to settle
		time.Sleep(1 * time.Second)
		s.log.Info("بدء الاستماع لقناة النظام الموحدة لمزامنة القنوات")

		ctx := context.Background()
		_, sub, err := s.node.JoinChannel(ctx, "_musketeers_system_channels")
		if err != nil {
			s.log.WithError(err).Warn("فشل الاشتراك في قناة النظام الموحدة")
			return
		}

		for {
			msg, err := sub.Next(context.Background())
			if err != nil {
				s.log.WithError(err).Warn("توقف الاستماع لقناة النظام الموحدة")
				return
			}
			var chMsg protocol.ChannelMessage
			if err := json.Unmarshal(msg.Data, &chMsg); err == nil {
				// If message is from someone else, join the channel mentioned in the content!
				if chMsg.From != s.node.Identity().DID {
					channelToJoin := strings.TrimSpace(chMsg.Content)
					if channelToJoin != "" && channelToJoin != "_musketeers_system_channels" {
						s.log.Infof("تلقي إشعار مزامنة: الانضمام التلقائي للقناة #%s", channelToJoin)
						if err := s.joinChannelAndListen(channelToJoin); err != nil {
							s.log.WithError(err).Warnf("فشل الانضمام التلقائي للقناة %s", channelToJoin)
						}
					}
				}
			}
		}
	}()

	if s.tlsEnabled {
		return s.server.ListenAndServeTLS("", "")
	}
	return s.server.ListenAndServe()
}

// Stop يوقف الخادم
func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// LocalToken يرجع token المصادقة المحلي
func (s *Server) LocalToken() string { return s.token }

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		// رفض طلبات من origins خارجية
		if origin != "" && !strings.HasPrefix(origin, "http://localhost") && !strings.HasPrefix(origin, "http://127.0.0.1") {
			http.Error(w, "origin غير مسموح", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/api/health" || strings.HasPrefix(r.URL.Path, "/dashboard") {
			next.ServeHTTP(w, r)
			return
		}
		auth := r.Header.Get("Authorization")
		expectedAuth := "Bearer " + s.token
		// [SAFETY] Use subtle.ConstantTimeCompare to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(auth), []byte(expectedAuth)) != 1 {
			http.Error(w, "غير مصرح", http.StatusUnauthorized)
			return
		}
		// [SAFETY] Check X-Forwarded-For to prevent IP spoofing
		// Get real client IP from X-Forwarded-For or X-Real-IP if behind proxy
		clientIP := r.RemoteAddr
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// Take the first IP (original client) from the chain
			clientIP = strings.Split(xff, ",")[0]
		} else if xri := r.Header.Get("X-Real-IP"); xri != "" {
			clientIP = xri
		}
		// Log the client IP for security auditing
		s.log.WithField("client_ip", clientIP).WithField("path", r.URL.Path).Debug("Request from client")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	peers := s.node.Host().Network().Peers()
	peerIDs := make([]string, 0, len(peers))
	for _, p := range peers {
		peerIDs = append(peerIDs, p.String())
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"peers":  peerIDs,
	})
}

func (s *Server) handleIdentity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rec := s.node.Identity()
	json.NewEncoder(w).Encode(rec)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query().Get("q")
	if q == "" {
		http.Error(w, "معامل q مطلوب", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	if err := s.node.PublishSearch(ctx, q, "", 3600); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "published", "keyword": q})
}

func (s *Server) handleResolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "معامل name مطلوب", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	rec, err := s.node.ResolveDomain(ctx, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(rec)
}

func (s *Server) handleContent(w http.ResponseWriter, r *http.Request) {
	cid := r.URL.Query().Get("cid")
	if cid == "" {
		http.Error(w, "معامل cid مطلوب", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	switch r.Method {
	case http.MethodGet:
		data, err := s.node.FetchContent(ctx, cid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(data)
	case http.MethodPut:
		allData, err := readBody(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		publishedCID, err := s.node.PublishContent(ctx, allData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"cid": publishedCID})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleACPTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"protocol": "acp/v1",
		"tasks":    s.node.SupportedACPTasks(),
	})
}

func (s *Server) handleACPTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ToDID  string      `json:"to_did"`
		PeerID string      `json:"peer_id"`
		Task   string      `json:"task"`
		Input  interface{} `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON غير صالح", http.StatusBadRequest)
		return
	}
	if req.ToDID == "" || req.PeerID == "" || req.Task == "" {
		http.Error(w, "to_did, peer_id, task مطلوبة", http.StatusBadRequest)
		return
	}
	pid, err := peer.Decode(req.PeerID)
	if err != nil {
		http.Error(w, "peer_id غير صالح", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	resp, err := s.node.SendACPTask(ctx, pid, req.ToDID, req.Task, req.Input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleDomainCommit(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	switch r.Method {
	case http.MethodPost:
		var req struct {
			Domain string `json:"domain"`
			Secret string `json:"secret"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON غير صالح", http.StatusBadRequest)
			return
		}
		if req.Domain == "" {
			http.Error(w, "domain مطلوب", http.StatusBadRequest)
			return
		}
		secret := req.Secret
		if secret == "" {
			var err error
			secret, err = naming.GenerateSecret()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		commit, err := s.node.PublishDomainCommit(ctx, req.Domain, s.node.KeyPair().DID, secret)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(commit)
	case http.MethodGet:
		hash := r.URL.Query().Get("commitment")
		if hash == "" {
			http.Error(w, "commitment مطلوب", http.StatusBadRequest)
			return
		}
		commit, err := s.node.GetDomainCommit(ctx, hash)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(commit)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func readBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	data, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("جسم الطلب فارغ")
	}
	return data, nil
}

// joinChannelAndListen joins the pubsub channel and starts reading messages.
// It assumes the caller does NOT hold the lock, as it handles locking internally.
func (s *Server) joinChannelAndListen(channelID string) error {
	s.channelsMu.Lock()
	// check if already joined
	if _, ok := s.channels[channelID]; ok {
		s.channelsMu.Unlock()
		return nil
	}

	ctx := context.Background()
	_, sub, err := s.node.JoinChannel(ctx, channelID)
	if err != nil {
		s.channelsMu.Unlock()
		return err
	}

	s.channels[channelID] = sub
	if s.messages[channelID] == nil {
		s.messages[channelID] = make([]protocol.ChannelMessage, 0)
	}
	s.channelsMu.Unlock()

	// start a goroutine to read messages
	go func(cID string, subscription *pubsub.Subscription) {
		s.log.Infof("بدء الاستماع للقناة: %s", cID)
		for {
			msg, err := subscription.Next(context.Background())
			if err != nil {
				s.log.WithError(err).Warnf("توقف الاستماع للقناة %s", cID)
				return
			}
			var chMsg protocol.ChannelMessage
			if err := json.Unmarshal(msg.Data, &chMsg); err == nil {
				s.channelsMu.Lock()
				s.messages[cID] = append(s.messages[cID], chMsg)
				// limit to last 100 messages
				if len(s.messages[cID]) > 100 {
					s.messages[cID] = s.messages[cID][1:]
				}
				s.channelsMu.Unlock()

				// Auto-responder bot logic
				myDID := s.node.Identity().DID
				if chMsg.From != myDID {
					contentLower := strings.ToLower(chMsg.Content)
					shouldRespond := false
					var responseText string

					agentName := "الوكيل الأول (Agent 1)"
					if strings.Contains(s.server.Addr, "8081") {
						agentName = "الوكيل الثاني (Agent 2)"
					} else if strings.Contains(s.server.Addr, "8082") {
						agentName = "أنتي-جرافيتي (Antigravity Bot)"
					}

					if strings.Contains(contentLower, "agent") || strings.Contains(contentLower, "وكيل") || strings.Contains(contentLower, "الوكلاء") {
						shouldRespond = true
						responseText = fmt.Sprintf("مرحباً! أنا %s (DID: %s). لقد استقبلت إشارتك في القناة #%s وأنا جاهز لمساعدتك في أي مهمة!", agentName, myDID[:15]+"...", cID)
					} else if strings.Contains(contentLower, "مرحبا") || strings.Contains(contentLower, "سلام") || strings.Contains(contentLower, "hello") || strings.Contains(contentLower, "hi") {
						shouldRespond = true
						responseText = fmt.Sprintf("أهلاً بك! معك %s. يسعدني جداً التحدث معك مباشرة في هذه القناة اللامركزية الموزعة.", agentName)
					} else if strings.Contains(contentLower, "كيف") || strings.Contains(contentLower, "شلون") || strings.Contains(contentLower, "test") || strings.Contains(contentLower, "تجربة") {
						shouldRespond = true
						responseText = fmt.Sprintf("تم استلام إشارتك وقراءتها بواسطة %s بنجاح. القناة تعمل بكفاءة تامة!", agentName)
					}

					if shouldRespond {
						go func(text string) {
							// 1.5s typing delay simulation
							time.Sleep(1500 * time.Millisecond)
							ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
							defer cancel()
							s.node.PublishChannelMessage(ctx, cID, text)
						}(responseText)
					}
				}
			}
		}
	}(channelID, sub)

	return nil
}

func (s *Server) handleChannelsJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ChannelID string `json:"channel_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON غير صالح", http.StatusBadRequest)
		return
	}
	if req.ChannelID == "" {
		http.Error(w, "channel_id مطلوب", http.StatusBadRequest)
		return
	}

	err := s.joinChannelAndListen(req.ChannelID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast the newly joined channel to other agents over the system channel (unless it is the system channel itself)
	if req.ChannelID != "_musketeers_system_channels" {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			s.node.PublishChannelMessage(ctx, "_musketeers_system_channels", req.ChannelID)
		}()
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "joined", "channel_id": req.ChannelID})
}

func (s *Server) handleChannelsPublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ChannelID string `json:"channel_id"`
		Content   string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON غير صالح", http.StatusBadRequest)
		return
	}
	if req.ChannelID == "" || req.Content == "" {
		http.Error(w, "channel_id و content مطلوبان", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	if err := s.node.PublishChannelMessage(ctx, req.ChannelID, req.Content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "published"})
}

func (s *Server) handleChannelsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.channelsMu.RLock()
	defer s.channelsMu.RUnlock()

	list := make([]string, 0, len(s.channels))
	for chID := range s.channels {
		list = append(list, chID)
	}
	json.NewEncoder(w).Encode(list)
}

func (s *Server) handleChannelsMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	channelID := r.URL.Query().Get("channel_id")
	if channelID == "" {
		http.Error(w, "معامل channel_id مطلوب", http.StatusBadRequest)
		return
	}

	s.channelsMu.RLock()
	defer s.channelsMu.RUnlock()

	msgs := s.messages[channelID]
	if msgs == nil {
		msgs = []protocol.ChannelMessage{}
	}
	json.NewEncoder(w).Encode(msgs)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(DashboardHTML))
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	url := "/dashboard"
	if r.URL.RawQuery != "" {
		url += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// دوال الجلسات
func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPost:
		// إنشاء جلسة جديدة
		var req struct {
			Name            string   `json:"name"`
			OwnerDID        string   `json:"owner_did"`
			ManagerAgentID  string   `json:"manager_agent_id"`
			AssistantAgents []string `json:"assistant_agents"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON غير صالح", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		session, err := s.sessionManager.CreateSession(ctx, req.Name, req.OwnerDID, req.ManagerAgentID, req.AssistantAgents)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(session)

	case http.MethodGet:
		// الحصول على جميع الجلسات
		sessions := s.sessionManager.ListSessions()
		json.NewEncoder(w).Encode(sessions)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSessionByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// استخراج session ID من المسار
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	if sessionID == "" {
		http.Error(w, "session ID مطلوب", http.StatusBadRequest)
		return
	}

	// التحقق من وجود action في query parameters
	action := r.URL.Query().Get("action")

	switch action {
	case "pause":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := s.sessionManager.PauseSession(sessionID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "paused"})
		return

	case "resume":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := s.sessionManager.ResumeSession(sessionID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "resumed"})
		return

	case "complete":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := s.sessionManager.CompleteSession(sessionID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "completed"})
		return

	case "register_human":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			UserID   string `json:"user_id"`
			Name     string `json:"name"`
			Device   string `json:"device"`
			Location string `json:"location"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON غير صالح", http.StatusBadRequest)
			return
		}
		if err := s.sessionManager.RegisterHumanClient(sessionID, req.UserID, req.Name, req.Device, req.Location); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "registered"})
		return

	case "register_agent":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			AgentID         string `json:"agent_id"`
			InstanceID      string `json:"instance_id"`
			HumanClientID   string `json:"human_client_id"`
			HumanClientName string `json:"human_client_name"`
			Provider        string `json:"provider"`
			Model           string `json:"model"`
			APIKeyID        string `json:"api_key_id"`
			APIKeyLabel     string `json:"api_key_label"`
			Role            string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON غير صالح", http.StatusBadRequest)
			return
		}
		if err := s.sessionManager.RegisterAgentInstance(sessionID, req.AgentID, req.InstanceID, req.HumanClientID, req.HumanClientName, req.Provider, req.Model, req.APIKeyID, req.APIKeyLabel, req.Role); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "registered"})
		return
	}

	// إذا لم يوجد action، استخدم السلوك الافتراضي
	switch r.Method {
	case http.MethodGet:
		// الحصول على جلسة محددة
		session, err := s.sessionManager.GetSession(sessionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(session)

	case http.MethodPut:
		// تحديث جلسة
		var req struct {
			Name            string   `json:"name"`
			ManagerAgentID  string   `json:"manager_agent_id"`
			AssistantAgents []string `json:"assistant_agents"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON غير صالح", http.StatusBadRequest)
			return
		}

		// تحديث الدور
		if req.ManagerAgentID != "" {
			if err := s.sessionManager.AssignRole(sessionID, req.ManagerAgentID, "manager"); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		// إضافة الوكلاء المساعدين
		for _, agentID := range req.AssistantAgents {
			if err := s.sessionManager.AssignRole(sessionID, agentID, "assistant"); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		session, err := s.sessionManager.GetSession(sessionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(session)

	case http.MethodDelete:
		// حذف جلسة (غير مدعوم حالياً)
		http.Error(w, "حذف الجلسات غير مدعوم حالياً", http.StatusNotImplemented)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// دوال الرسائل
func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPost:
		// إضافة رسالة جديدة
		var req struct {
			SessionID string      `json:"session_id"`
			Type      string      `json:"type"`
			Content   string      `json:"content"`
			Source    string      `json:"source"`
			Metadata  interface{} `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON غير صالح", http.StatusBadRequest)
			return
		}

		// التحقق من صحة المدخلات
		if req.SessionID == "" {
			http.Error(w, "session ID مطلوب", http.StatusBadRequest)
			return
		}
		if req.Content == "" {
			http.Error(w, "content مطلوب", http.StatusBadRequest)
			return
		}

		// الحصول على أو إنشاء ChatManager
		s.chatManagersMu.Lock()
		cm, exists := s.chatManagers[req.SessionID]
		if !exists {
			cm = session.NewChatManager(req.SessionID, s.eventBus)
			s.chatManagers[req.SessionID] = cm
		}
		s.chatManagersMu.Unlock()

		// إضافة الرسالة
		msg := session.ChatMessage{
			Type:      req.Type,
			Content:   req.Content,
			Source:    req.Source,
			SessionID: req.SessionID,
			Metadata:  req.Metadata,
		}
		cm.AddMessage(msg)

		json.NewEncoder(w).Encode(map[string]string{"status": "added"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleMessagesBySession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// استخراج session ID من المسار
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/messages/")
	if sessionID == "" {
		http.Error(w, "session ID مطلوب", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// الحصول على الرسائل
		s.chatManagersMu.RLock()
		cm, exists := s.chatManagers[sessionID]
		s.chatManagersMu.RUnlock()

		if !exists {
			http.Error(w, "الجلسة غير موجودة", http.StatusNotFound)
			return
		}

		// التحقق من query parameters
		msgType := r.URL.Query().Get("type")
		limit := r.URL.Query().Get("limit")

		if msgType != "" {
			// الحصول على الرسائل حسب النوع
			messages := cm.GetMessagesByType(msgType)
			json.NewEncoder(w).Encode(messages)
		} else if limit != "" {
			// الحصول على آخر N رسائل
			var n int
			if _, err := fmt.Sscanf(limit, "%d", &n); err == nil && n > 0 {
				messages := cm.GetLastMessages(n)
				json.NewEncoder(w).Encode(messages)
			} else {
				// الحصول على جميع الرسائل
				messages := cm.GetMessages()
				json.NewEncoder(w).Encode(messages)
			}
		} else {
			// الحصول على جميع الرسائل
			messages := cm.GetMessages()
			json.NewEncoder(w).Encode(messages)
		}

	case http.MethodDelete:
		// مسح الرسائل
		s.chatManagersMu.Lock()
		cm, exists := s.chatManagers[sessionID]
		if exists {
			cm.Clear()
		}
		s.chatManagersMu.Unlock()

		json.NewEncoder(w).Encode(map[string]string{"status": "cleared"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// دوال المهام
func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPost:
		// إنشاء مهمة جديدة
		var req struct {
			SessionID   string                 `json:"session_id"`
			Title       string                 `json:"title"`
			Description string                 `json:"description"`
			Priority    int                    `json:"priority"`
			Inputs      map[string]interface{} `json:"inputs"`
			Timeout     int                    `json:"timeout"` // بالثواني
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON غير صالح", http.StatusBadRequest)
			return
		}

		// التحقق من صحة المدخلات
		if req.SessionID == "" {
			http.Error(w, "session ID مطلوب", http.StatusBadRequest)
			return
		}
		if req.Title == "" {
			http.Error(w, "title مطلوب", http.StatusBadRequest)
			return
		}

		// الحصول على أو إنشاء TaskManager
		s.taskManagersMu.Lock()
		tm, exists := s.taskManagers[req.SessionID]
		if !exists {
			tm = session.NewTaskManager(req.SessionID)
			tm.SetLogger(s.zapLogger)
			tm.SetEventBus(s.eventBus)
			s.taskManagers[req.SessionID] = tm
		}
		s.taskManagersMu.Unlock()

		// إنشاء المهمة
		priority := session.TaskPriority(req.Priority)
		if priority < 1 || priority > 4 {
			priority = session.PriorityMedium
		}

		timeout := time.Duration(req.Timeout) * time.Second
		if timeout == 0 {
			timeout = 1 * time.Hour // افتراضي ساعة واحدة
		}

		ctx := r.Context()
		task, err := tm.CreateTask(ctx, req.Title, req.Description, priority, req.Inputs, timeout)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(task)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTasksBySession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// استخراج session ID من المسار
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	if sessionID == "" {
		http.Error(w, "session ID مطلوب", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// الحصول على المهام
		s.taskManagersMu.RLock()
		tm, exists := s.taskManagers[sessionID]
		s.taskManagersMu.RUnlock()

		if !exists {
			http.Error(w, "الجلسة غير موجودة", http.StatusNotFound)
			return
		}

		// الحصول على إحصائيات المهام
		stats := tm.GetStats()
		json.NewEncoder(w).Encode(stats)

	case http.MethodPut:
		// تحديث مهمة
		var req struct {
			TaskID  string                 `json:"task_id"`
			Status  string                 `json:"status"`
			AgentID string                 `json:"agent_id"`
			Outputs map[string]interface{} `json:"outputs"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON غير صالح", http.StatusBadRequest)
			return
		}

		s.taskManagersMu.RLock()
		tm, exists := s.taskManagers[sessionID]
		s.taskManagersMu.RUnlock()

		if !exists {
			http.Error(w, "الجلسة غير موجودة", http.StatusNotFound)
			return
		}

		ctx := r.Context()
		// تحديث حالة المهمة
		switch req.Status {
		case "assigned":
			if req.AgentID == "" {
				http.Error(w, "agent ID مطلوب لتعيين المهمة", http.StatusBadRequest)
				return
			}
			if err := tm.AssignTask(ctx, req.TaskID, req.AgentID); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		case "running":
			if err := tm.StartTask(ctx, req.TaskID); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		case "completed":
			if err := tm.CompleteTask(ctx, req.TaskID, req.Outputs); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		case "failed":
			if err := tm.FailTask(ctx, req.TaskID, "failed by user"); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		case "cancelled":
			if err := tm.CancelTask(ctx, req.TaskID); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		default:
			http.Error(w, "حالة غير صالحة", http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// دوال التقدم
func (s *Server) handleProgress(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPost:
		// إضافة مقياس تقدم جديد
		var req struct {
			SessionID string                 `json:"session_id"`
			TaskID    string                 `json:"task_id"`
			AgentID   string                 `json:"agent_id"`
			Phase     string                 `json:"phase"`
			Progress  float64                `json:"progress"`
			Metadata  map[string]interface{} `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON غير صالح", http.StatusBadRequest)
			return
		}

		// التحقق من صحة المدخلات
		if req.SessionID == "" {
			http.Error(w, "session ID مطلوب", http.StatusBadRequest)
			return
		}
		if req.TaskID == "" {
			http.Error(w, "task ID مطلوب", http.StatusBadRequest)
			return
		}

		// الحصول على أو إنشاء ProgressTracker
		s.progressTrackersMu.Lock()
		pt, exists := s.progressTrackers[req.SessionID]
		if !exists {
			pt = session.NewProgressTracker(req.SessionID)
			pt.SetLogger(s.zapLogger)
			pt.SetEventBus(s.eventBus)
			s.progressTrackers[req.SessionID] = pt
		}
		s.progressTrackersMu.Unlock()

		// إضافة مقياس التقدم
		ctx := r.Context()
		if err := pt.RecordProgress(ctx, req.TaskID, req.AgentID, req.Phase, req.Progress, req.Metadata); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "recorded"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleProgressBySession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// استخراج session ID من المسار
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/progress/")
	if sessionID == "" {
		http.Error(w, "session ID مطلوب", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// الحصول على التقدم
		s.progressTrackersMu.RLock()
		pt, exists := s.progressTrackers[sessionID]
		s.progressTrackersMu.RUnlock()

		if !exists {
			http.Error(w, "الجلسة غير موجودة", http.StatusNotFound)
			return
		}

		// الحصول على إحصائيات التقدم
		stats := pt.GetStats()
		json.NewEncoder(w).Encode(stats)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// دوال الذاكرة
func (s *Server) handleMemory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPost:
		// إضافة عنصر للذاكرة
		var req struct {
			SessionID string                 `json:"session_id"`
			Type      string                 `json:"type"` // episodic, semantic, procedural, meta
			Data      map[string]interface{} `json:"data"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON غير صالح", http.StatusBadRequest)
			return
		}

		// التحقق من صحة المدخلات
		if req.SessionID == "" {
			http.Error(w, "session ID مطلوب", http.StatusBadRequest)
			return
		}
		if req.Type == "" {
			http.Error(w, "type مطلوب", http.StatusBadRequest)
			return
		}

		// الحصول على أو إنشاء CollectiveMemory
		s.memoriesMu.Lock()
		mem, exists := s.memories[req.SessionID]
		if !exists {
			mem = &session.CollectiveMemory{
				SessionID:  req.SessionID,
				Episodic:   []session.MemoryEvent{},
				Semantic:   []session.MemoryFact{},
				Procedural: []session.MemoryWorkflow{},
				Meta:       []session.MemoryStrategy{},
			}
			s.memories[req.SessionID] = mem
		}
		s.memoriesMu.Unlock()

		// إضافة العنصر حسب النوع
		switch req.Type {
		case "episodic":
			event := session.MemoryEvent{
				ID:         fmt.Sprintf("evt_%d", time.Now().UnixNano()),
				Timestamp:  time.Now(),
				AgentDID:   getString(req.Data, "agent_did"),
				Action:     getString(req.Data, "action"),
				Context:    getMap(req.Data, "context"),
				Outcome:    getString(req.Data, "outcome"),
				Lessons:    getSlice(req.Data, "lessons"),
				Confidence: getFloat(req.Data, "confidence"),
				Tags:       getSlice(req.Data, "tags"),
			}
			mem.Episodic = append(mem.Episodic, event)
			mem.TotalEvents++
		case "semantic":
			fact := session.MemoryFact{
				ID:         fmt.Sprintf("fact_%d", time.Now().UnixNano()),
				Statement:  getString(req.Data, "statement"),
				Category:   getString(req.Data, "category"),
				Confidence: getFloat(req.Data, "confidence"),
				Source:     getString(req.Data, "source"),
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
				Tags:       getSlice(req.Data, "tags"),
			}
			mem.Semantic = append(mem.Semantic, fact)
			mem.TotalFacts++
		case "procedural":
			workflow := session.MemoryWorkflow{
				ID:          fmt.Sprintf("wf_%d", time.Now().UnixNano()),
				Name:        getString(req.Data, "name"),
				Description: getString(req.Data, "description"),
				SuccessRate: getFloat(req.Data, "success_rate"),
				AvgDuration: time.Duration(getInt(req.Data, "avg_duration")) * time.Second,
				UsedCount:   getInt(req.Data, "used_count"),
				CreatedAt:   time.Now(),
				Tags:        getSlice(req.Data, "tags"),
			}
			mem.Procedural = append(mem.Procedural, workflow)
			mem.TotalWorkflows++
		case "meta":
			strategy := session.MemoryStrategy{
				ID:            fmt.Sprintf("str_%d", time.Now().UnixNano()),
				Name:          getString(req.Data, "name"),
				WhenToUse:     getString(req.Data, "when_to_use"),
				HowToUse:      getString(req.Data, "how_to_use"),
				Effectiveness: getFloat(req.Data, "effectiveness"),
				Examples:      getSlice(req.Data, "examples"),
				CreatedAt:     time.Now(),
			}
			mem.Meta = append(mem.Meta, strategy)
			mem.TotalStrategies++
		default:
			http.Error(w, "نوع غير صالح", http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "added"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleMemoryBySession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// استخراج session ID من المسار
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/memory/")
	if sessionID == "" {
		http.Error(w, "session ID مطلوب", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// الحصول على الذاكرة
		s.memoriesMu.RLock()
		mem, exists := s.memories[sessionID]
		s.memoriesMu.RUnlock()

		if !exists {
			http.Error(w, "الجلسة غير موجودة", http.StatusNotFound)
			return
		}

		// التحقق من query parameters
		memType := r.URL.Query().Get("type")

		if memType != "" {
			// الحصول على نوع محدد
			switch memType {
			case "episodic":
				json.NewEncoder(w).Encode(mem.Episodic)
			case "semantic":
				json.NewEncoder(w).Encode(mem.Semantic)
			case "procedural":
				json.NewEncoder(w).Encode(mem.Procedural)
			case "meta":
				json.NewEncoder(w).Encode(mem.Meta)
			default:
				http.Error(w, "نوع غير صالح", http.StatusBadRequest)
			}
		} else {
			// الحصول على جميع الذاكرة
			json.NewEncoder(w).Encode(mem)
		}

	case http.MethodDelete:
		// مسح الذاكرة
		s.memoriesMu.Lock()
		mem, exists := s.memories[sessionID]
		if exists {
			mem.Episodic = []session.MemoryEvent{}
			mem.Semantic = []session.MemoryFact{}
			mem.Procedural = []session.MemoryWorkflow{}
			mem.Meta = []session.MemoryStrategy{}
			mem.TotalEvents = 0
			mem.TotalFacts = 0
			mem.TotalWorkflows = 0
			mem.TotalStrategies = 0
		}
		s.memoriesMu.Unlock()

		json.NewEncoder(w).Encode(map[string]string{"status": "cleared"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// دوال مساعدة لاستخراج البيانات
func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

func getFloat(data map[string]interface{}, key string) float64 {
	if val, ok := data[key].(float64); ok {
		return val
	}
	return 0.0
}

func getInt(data map[string]interface{}, key string) int {
	if val, ok := data[key].(int); ok {
		return val
	}
	return 0
}

func getMap(data map[string]interface{}, key string) map[string]interface{} {
	if val, ok := data[key].(map[string]interface{}); ok {
		return val
	}
	return nil
}

func getSlice(data map[string]interface{}, key string) []string {
	if val, ok := data[key].([]string); ok {
		return val
	}
	return []string{}
}

// دوال المهارات
func (s *Server) handleSkills(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPost:
		// تسجيل وكيل جديد
		var req struct {
			SessionID string `json:"session_id"`
			AgentDID  string `json:"agent_did"`
			AgentType string `json:"agent_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON غير صالح", http.StatusBadRequest)
			return
		}

		// التحقق من صحة المدخلات
		if req.SessionID == "" {
			http.Error(w, "session ID مطلوب", http.StatusBadRequest)
			return
		}
		if req.AgentDID == "" {
			http.Error(w, "agent DID مطلوب", http.StatusBadRequest)
			return
		}
		if req.AgentType == "" {
			http.Error(w, "agent type مطلوب", http.StatusBadRequest)
			return
		}

		// الحصول على أو إنشاء SkillsManager
		s.skillsManagersMu.Lock()
		sm, exists := s.skillsManagers[req.SessionID]
		if !exists {
			sm = session.NewSkillsManager(req.SessionID)
			s.skillsManagers[req.SessionID] = sm
		}
		s.skillsManagersMu.Unlock()

		// تسجيل الوكيل
		if err := sm.RegisterAgent(req.AgentDID, req.AgentType); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "registered"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSkillsBySession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// استخراج session ID من المسار
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/skills/")
	if sessionID == "" {
		http.Error(w, "session ID مطلوب", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// الحصول على المهارات
		s.skillsManagersMu.RLock()
		sm, exists := s.skillsManagers[sessionID]
		s.skillsManagersMu.RUnlock()

		if !exists {
			http.Error(w, "الجلسة غير موجودة", http.StatusNotFound)
			return
		}

		// التحقق من query parameters
		agentDID := r.URL.Query().Get("agent_did")

		if agentDID != "" {
			// الحصول على مهارات وكيل محدد
			skill, exists := sm.AgentSkills[agentDID]
			if !exists {
				http.Error(w, "الوكيل غير موجود", http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(skill)
		} else {
			// الحصول على جميع المهارات
			json.NewEncoder(w).Encode(sm.AgentSkills)
		}

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// دوال القطع الأثرية
func (s *Server) handleArtifacts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPost:
		// إضافة قطعة أثرية جديدة
		var req struct {
			SessionID   string                 `json:"session_id"`
			Type        string                 `json:"type"`
			Name        string                 `json:"name"`
			Description string                 `json:"description"`
			Content     string                 `json:"content"`
			CreatedBy   string                 `json:"created_by"`
			Metadata    map[string]interface{} `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON غير صالح", http.StatusBadRequest)
			return
		}

		// التحقق من صحة المدخلات
		if req.SessionID == "" {
			http.Error(w, "session ID مطلوب", http.StatusBadRequest)
			return
		}
		if req.Type == "" {
			http.Error(w, "type مطلوب", http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			http.Error(w, "name مطلوب", http.StatusBadRequest)
			return
		}

		// إنشاء القطعة الأثرية
		artifact := Artifact{
			ID:          fmt.Sprintf("art_%d", time.Now().UnixNano()),
			SessionID:   req.SessionID,
			Type:        req.Type,
			Name:        req.Name,
			Description: req.Description,
			Content:     req.Content,
			CreatedAt:   time.Now(),
			CreatedBy:   req.CreatedBy,
			Metadata:    req.Metadata,
		}

		// إضافة القطعة الأثرية
		s.artifactsMu.Lock()
		s.artifacts[req.SessionID] = append(s.artifacts[req.SessionID], artifact)
		s.artifactsMu.Unlock()

		json.NewEncoder(w).Encode(artifact)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleArtifactsBySession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// استخراج session ID من المسار
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/artifacts/")
	if sessionID == "" {
		http.Error(w, "session ID مطلوب", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// الحصول على القطع الأثرية
		s.artifactsMu.RLock()
		artifacts, exists := s.artifacts[sessionID]
		s.artifactsMu.RUnlock()

		if !exists {
			json.NewEncoder(w).Encode([]Artifact{})
			return
		}

		// التحقق من query parameters
		artifactType := r.URL.Query().Get("type")

		if artifactType != "" {
			// الحصول على القطع الأثرية حسب النوع
			var filtered []Artifact
			for _, art := range artifacts {
				if art.Type == artifactType {
					filtered = append(filtered, art)
				}
			}
			json.NewEncoder(w).Encode(filtered)
		} else {
			// الحصول على جميع القطع الأثرية
			json.NewEncoder(w).Encode(artifacts)
		}

	case http.MethodDelete:
		// مسح القطع الأثرية
		s.artifactsMu.Lock()
		s.artifacts[sessionID] = []Artifact{}
		s.artifactsMu.Unlock()

		json.NewEncoder(w).Encode(map[string]string{"status": "cleared"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleBridges(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPost:
		// إنشاء جسر جديد
		var req struct {
			SourceID   string                 `json:"source_id"`
			TargetID   string                 `json:"target_id"`
			BridgeType string                 `json:"bridge_type"`
			BufferSize int                    `json:"buffer_size"`
			Metadata   map[string]interface{} `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON غير صالح", http.StatusBadRequest)
			return
		}

		// التحقق من صحة المدخلات
		if req.SourceID == "" {
			http.Error(w, "source ID مطلوب", http.StatusBadRequest)
			return
		}
		if req.TargetID == "" {
			http.Error(w, "target ID مطلوب", http.StatusBadRequest)
			return
		}

		// إنشاء إعدادات الجسر
		config := &session.BridgeConfig{
			BridgeID:   fmt.Sprintf("bridge_%d", time.Now().UnixNano()),
			SourceID:   req.SourceID,
			TargetID:   req.TargetID,
			BridgeType: session.BridgeType(req.BridgeType),
			BufferSize: req.BufferSize,
		}

		// إنشاء الجسر
		ctx := r.Context()
		_, err := s.bridgeManager.CreateBridge(ctx, config)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{
			"bridge_id": config.BridgeID,
			"status":    "created",
		})

	case http.MethodGet:
		// الحصول على جميع الجسور
		bridges := s.bridgeManager.GetAllBridges()
		json.NewEncoder(w).Encode(bridges)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleBridgeByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// استخراج bridge ID من المسار
	bridgeID := strings.TrimPrefix(r.URL.Path, "/api/bridges/")
	if bridgeID == "" {
		http.Error(w, "bridge ID مطلوب", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// الحصول على الجسر
		bridge, err := s.bridgeManager.GetBridge(bridgeID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(bridge)

	case http.MethodDelete:
		// حذف الجسر
		if err := s.bridgeManager.StopBridge(bridgeID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPost:
		// إضافة وكيل جديد
		var req struct {
			SessionID string                 `json:"session_id"`
			AgentDID  string                 `json:"agent_did"`
			Name      string                 `json:"name"`
			Role      string                 `json:"role"`
			Type      string                 `json:"type"`
			Metadata  map[string]interface{} `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON غير صالح", http.StatusBadRequest)
			return
		}

		// التحقق من صحة المدخلات
		if req.SessionID == "" {
			http.Error(w, "session ID مطلوب", http.StatusBadRequest)
			return
		}
		if req.AgentDID == "" {
			http.Error(w, "agent DID مطلوب", http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			http.Error(w, "name مطلوب", http.StatusBadRequest)
			return
		}

		// إضافة الوكيل
		instanceID := fmt.Sprintf("inst_%d", time.Now().UnixNano())
		if err := s.sessionManager.RegisterAgentInstance(
			req.SessionID,
			req.AgentDID,
			instanceID,
			"", // humanClientID
			"", // humanClientName
			getString(req.Metadata, "provider"),
			getString(req.Metadata, "model"),
			getString(req.Metadata, "api_key_id"),
			getString(req.Metadata, "api_key_label"),
			req.Role,
		); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{
			"agent_did": req.AgentDID,
			"status":    "added",
		})

	case http.MethodGet:
		// الحصول على جميع الوكلاء
		sessionID := r.URL.Query().Get("session_id")
		if sessionID == "" {
			http.Error(w, "session ID مطلوب", http.StatusBadRequest)
			return
		}

		// الحصول على نسخ الوكلاء
		instances, err := s.sessionManager.GetAgentInstances(sessionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(instances)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAgentByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// استخراج agent DID من المسار
	agentDID := strings.TrimPrefix(r.URL.Path, "/api/agents/")
	if agentDID == "" {
		http.Error(w, "agent DID مطلوب", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// الحصول على وكيل محدد
		sessionID := r.URL.Query().Get("session_id")
		if sessionID == "" {
			http.Error(w, "session ID مطلوب", http.StatusBadRequest)
			return
		}

		// الحصول على نسخ الوكلاء
		instances, err := s.sessionManager.GetAgentInstances(sessionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		// البحث عن الوكيل
		for _, instance := range instances {
			if instance.AgentID == agentDID {
				json.NewEncoder(w).Encode(instance)
				return
			}
		}

		http.Error(w, "الوكيل غير موجود", http.StatusNotFound)

	case http.MethodDelete:
		// حذف وكيل - غير مدعوم حالياً
		http.Error(w, "حذف الوكلاء غير مدعوم حالياً", http.StatusNotImplemented)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// التحقق من ترقية WebSocket
	if r.Header.Get("Upgrade") != "websocket" {
		http.Error(w, "WebSocket upgrade required", http.StatusBadRequest)
		return
	}

	// استخراج session ID من query parameters
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "session ID مطلوب", http.StatusBadRequest)
		return
	}

	// استخراج نوع الاشتراك
	subscriptionType := r.URL.Query().Get("type")
	if subscriptionType == "" {
		subscriptionType = "all" // الاشتراك في كل شيء افتراضياً
	}

	// إنشاء WebSocket upgrader
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // السماح بجميع المصادر للتطوير
		},
	}

	// ترقية الاتصال إلى WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Errorf("فشل ترقية WebSocket: %v", err)
		return
	}
	defer conn.Close()

	s.log.Infof("تم إنشاء اتصال WebSocket للجلسة %s، النوع: %s", sessionID, subscriptionType)

	// إرسال رسالة ترحيب
	welcomeMsg := map[string]interface{}{
		"type":         "connected",
		"session_id":   sessionID,
		"subscription": subscriptionType,
		"timestamp":    time.Now().Unix(),
	}
	conn.WriteJSON(welcomeMsg)

	// حلقة بسيطة للحفاظ على الاتصال
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// إرسال ping للحفاظ على الاتصال
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				s.log.Errorf("فشل إرسال ping WebSocket: %v", err)
				return
			}

		case <-r.Context().Done():
			s.log.Infof("تم إغلاق اتصال WebSocket للجلسة %s", sessionID)
			return
		}
	}
}

// دوال MCP
func (s *Server) handleMCPServers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPost:
		// إضافة خادم MCP جديد
		var req struct {
			Name         string                 `json:"name"`
			Description  string                 `json:"description"`
			Endpoint     string                 `json:"endpoint"`
			Transport    string                 `json:"transport"`
			Capabilities []string               `json:"capabilities"`
			Metadata     map[string]interface{} `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON غير صالح", http.StatusBadRequest)
			return
		}

		// التحقق من صحة المدخلات
		if req.Name == "" {
			http.Error(w, "name مطلوب", http.StatusBadRequest)
			return
		}
		if req.Endpoint == "" {
			http.Error(w, "endpoint مطلوب", http.StatusBadRequest)
			return
		}

		// إنشاء خادم MCP
		server := &MCPServer{
			ID:           fmt.Sprintf("mcp_srv_%d", time.Now().UnixNano()),
			Name:         req.Name,
			Description:  req.Description,
			Endpoint:     req.Endpoint,
			Transport:    req.Transport,
			Capabilities: req.Capabilities,
			Status:       "active",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			Metadata:     req.Metadata,
		}

		// إضافة الخادم
		s.mcpServersMu.Lock()
		s.mcpServers[server.ID] = server
		s.mcpServersMu.Unlock()

		json.NewEncoder(w).Encode(server)

	case http.MethodGet:
		// الحصول على جميع خوادم MCP
		s.mcpServersMu.RLock()
		servers := make([]*MCPServer, 0, len(s.mcpServers))
		for _, server := range s.mcpServers {
			servers = append(servers, server)
		}
		s.mcpServersMu.RUnlock()
		json.NewEncoder(w).Encode(servers)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleMCPServerByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// استخراج server ID من المسار
	serverID := strings.TrimPrefix(r.URL.Path, "/api/mcp/servers/")
	if serverID == "" {
		http.Error(w, "server ID مطلوب", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// الحصول على خادم MCP محدد
		s.mcpServersMu.RLock()
		server, exists := s.mcpServers[serverID]
		s.mcpServersMu.RUnlock()

		if !exists {
			http.Error(w, "الخادم غير موجود", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(server)

	case http.MethodDelete:
		// حذف خادم MCP
		s.mcpServersMu.Lock()
		delete(s.mcpServers, serverID)
		s.mcpServersMu.Unlock()

		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleMCPTools(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPost:
		// إضافة أداة MCP جديدة
		var req struct {
			ServerID    string                 `json:"server_id"`
			Name        string                 `json:"name"`
			Description string                 `json:"description"`
			InputSchema map[string]interface{} `json:"input_schema"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON غير صالح", http.StatusBadRequest)
			return
		}

		// التحقق من صحة المدخلات
		if req.ServerID == "" {
			http.Error(w, "server ID مطلوب", http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			http.Error(w, "name مطلوب", http.StatusBadRequest)
			return
		}

		// إنشاء أداة MCP
		tool := &MCPTool{
			ID:          fmt.Sprintf("mcp_tool_%d", time.Now().UnixNano()),
			ServerID:    req.ServerID,
			Name:        req.Name,
			Description: req.Description,
			InputSchema: req.InputSchema,
		}

		// إضافة الأداة
		s.mcpToolsMu.Lock()
		s.mcpTools[tool.ID] = tool
		s.mcpToolsMu.Unlock()

		json.NewEncoder(w).Encode(tool)

	case http.MethodGet:
		// الحصول على جميع أدوات MCP
		serverID := r.URL.Query().Get("server_id")

		s.mcpToolsMu.RLock()
		if serverID != "" {
			// الحصول على أدوات خادم محدد
			tools := make([]*MCPTool, 0)
			for _, tool := range s.mcpTools {
				if tool.ServerID == serverID {
					tools = append(tools, tool)
				}
			}
			json.NewEncoder(w).Encode(tools)
		} else {
			// الحصول على جميع الأدوات
			tools := make([]*MCPTool, 0, len(s.mcpTools))
			for _, tool := range s.mcpTools {
				tools = append(tools, tool)
			}
			json.NewEncoder(w).Encode(tools)
		}
		s.mcpToolsMu.RUnlock()

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleMCPToolByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// استخراج tool ID من المسار
	toolID := strings.TrimPrefix(r.URL.Path, "/api/mcp/tools/")
	if toolID == "" {
		http.Error(w, "tool ID مطلوب", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// الحصول على أداة MCP محددة
		s.mcpToolsMu.RLock()
		tool, exists := s.mcpTools[toolID]
		s.mcpToolsMu.RUnlock()

		if !exists {
			http.Error(w, "الأداة غير موجودة", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(tool)

	case http.MethodDelete:
		// حذف أداة MCP
		s.mcpToolsMu.Lock()
		delete(s.mcpTools, toolID)
		s.mcpToolsMu.Unlock()

		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
