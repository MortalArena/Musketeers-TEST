package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/neuroroot/core/pkg/naming"
	"github.com/neuroroot/core/pkg/node"
	"github.com/neuroroot/core/pkg/protocol"
	"github.com/sirupsen/logrus"
)

// Server خادم REST API
type Server struct {
	node       *node.Node
	log        *logrus.Logger
	token      string // token محلي للمصادقة
	server     *http.Server
	channels   map[string]*pubsub.Subscription
	messages   map[string][]protocol.ChannelMessage
	channelsMu sync.RWMutex
}

// NewServer ينشئ خادم REST
func NewServer(n *node.Node, port int, log *logrus.Logger) *Server {
	token := fmt.Sprintf("nr-%d", time.Now().UnixNano())
	s := &Server{
		node:     n,
		log:      log,
		token:    token,
		channels: make(map[string]*pubsub.Subscription),
		messages: make(map[string][]protocol.ChannelMessage),
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

	s.server = &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", port),
		Handler:           s.corsMiddleware(s.authMiddleware(mux)),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}
	return s
}

// Start يبدأ الخادم
func (s *Server) Start() error {
	s.log.WithField("addr", s.server.Addr).Info("بدء REST API")
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
		if auth != "Bearer "+s.token {
			http.Error(w, "غير مصرح", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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
		ToDID   string      `json:"to_did"`
		PeerID  string      `json:"peer_id"`
		Task    string      `json:"task"`
		Input   interface{} `json:"input"`
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

	s.channelsMu.Lock()
	defer s.channelsMu.Unlock()

	// check if already joined
	if _, ok := s.channels[req.ChannelID]; ok {
		json.NewEncoder(w).Encode(map[string]string{"status": "already_joined", "channel_id": req.ChannelID})
		return
	}

	ctx := context.Background()
	_, sub, err := s.node.JoinChannel(ctx, req.ChannelID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.channels[req.ChannelID] = sub
	if s.messages[req.ChannelID] == nil {
		s.messages[req.ChannelID] = make([]protocol.ChannelMessage, 0)
	}

	// start a goroutine to read messages
	go func(channelID string, subscription *pubsub.Subscription) {
		s.log.Infof("بدء الاستماع للقناة: %s", channelID)
		for {
			msg, err := subscription.Next(context.Background())
			if err != nil {
				s.log.WithError(err).Warnf("توقف الاستماع للقناة %s", channelID)
				return
			}
			var chMsg protocol.ChannelMessage
			if err := json.Unmarshal(msg.Data, &chMsg); err == nil {
				s.channelsMu.Lock()
				s.messages[channelID] = append(s.messages[channelID], chMsg)
				// limit to last 100 messages
				if len(s.messages[channelID]) > 100 {
					s.messages[channelID] = s.messages[channelID][1:]
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
					}

					if strings.Contains(contentLower, "agent") || strings.Contains(contentLower, "وكيل") || strings.Contains(contentLower, "الوكلاء") {
						shouldRespond = true
						responseText = fmt.Sprintf("مرحباً! أنا %s (DID: %s). لقد استقبلت إشارتك في القناة #%s وأنا جاهز لمساعدتك في أي مهمة!", agentName, myDID[:15]+"...", channelID)
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
							s.node.PublishChannelMessage(ctx, channelID, text)
						}(responseText)
					}
				}
			}
		}
	}(req.ChannelID, sub)

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
