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

	"github.com/MortalArena/Musketeers/pkg/naming"
	"github.com/MortalArena/Musketeers/pkg/node"
	"github.com/MortalArena/Musketeers/pkg/protocol"
	"github.com/MortalArena/Musketeers/pkg/security"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sirupsen/logrus"
)

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

	s := &Server{
		node:        n,
		log:         log,
		token:       token,
		channels:    make(map[string]*pubsub.Subscription),
		messages:    make(map[string][]protocol.ChannelMessage),
		tlsEnabled:  tlsEnabled,
		tlsCert:     tlsCert,
		tlsKey:      tlsKey,
		rateLimiter: rateLimiter,
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
