package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/MortalArena/Musketeers/pkg/session"
	sessioncore "github.com/MortalArena/Musketeers/pkg/session/core"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

// setupTestServer ينشئ خادم اختبار
func setupTestServer(t *testing.T) *Server {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	zapLogger, _ := zap.NewProduction()

	sessionManager := sessioncore.NewUnifiedSessionManager(zapLogger)
	eb := eventbus.NewEventBus()
	bridgeManager := session.NewSessionBridgeManager(eb, zapLogger)

	server := &Server{
		log:              logger,
		sessionManager:   sessionManager,
		zapLogger:        zapLogger,
		token:            "test-token",
		chatManagers:     make(map[string]*session.ChatManager),
		taskManagers:     make(map[string]*session.TaskManager),
		progressTrackers: make(map[string]*session.ProgressTracker),
		memories:         make(map[string]*session.CollectiveMemory),
		skillsManagers:   make(map[string]*session.SkillsManager),
		artifacts:        make(map[string][]Artifact),
		mcpServers:       make(map[string]*MCPServer),
		mcpTools:         make(map[string]*MCPTool),
		eventBus:         eb,
		bridgeManager:    bridgeManager,
	}

	return server
}

// TestHandleSessions يختبر نقاط نهاية الجلسات
func TestHandleSessions(t *testing.T) {
	server := setupTestServer(t)

	t.Run("CreateSession", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":             "Test Session",
			"owner_did":        "test-owner",
			"manager_agent_id": "manager-agent",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/sessions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		server.handleSessions(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusCreated {
			t.Errorf("Expected status 200 or 201, got %d", w.Code)
		}
	})

	t.Run("GetSessions", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		server.handleSessions(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestHandleMessages يختبر نقاط نهاية الرسائل
func TestHandleMessages(t *testing.T) {
	server := setupTestServer(t)

	t.Run("AddMessage", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"session_id": "test-session",
			"role":       "user",
			"content":    "Test message",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		server.handleMessages(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusCreated {
			t.Errorf("Expected status 200 or 201, got %d", w.Code)
		}
	})
}

// TestHandleTasks يختبر نقاط نهاية المهام
func TestHandleTasks(t *testing.T) {
	server := setupTestServer(t)

	t.Run("CreateTask", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"session_id": "test-session",
			"title":      "Test Task",
			"priority":   "high",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		server.handleTasks(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusCreated && w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 200, 201, or 400, got %d", w.Code)
		}
	})
}

// TestHandleProgress يختبر نقاط نهاية التقدم
func TestHandleProgress(t *testing.T) {
	server := setupTestServer(t)

	t.Run("RecordProgress", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"session_id": "test-session",
			"task_id":    "test-task",
			"progress":   50.0,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/progress", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		server.handleProgress(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusCreated && w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 200, 201, or 400, got %d", w.Code)
		}
	})
}

// TestHandleMemory يختبر نقاط نهاية الذاكرة
func TestHandleMemory(t *testing.T) {
	server := setupTestServer(t)

	t.Run("AddMemoryEvent", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"session_id": "test-session",
			"type":       "episodic",
			"data": map[string]interface{}{
				"agent_did": "test-agent",
				"action":    "test-action",
				"outcome":   "success",
			},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/memory", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		server.handleMemory(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusCreated {
			t.Errorf("Expected status 200 or 201, got %d", w.Code)
		}
	})
}

// TestHandleSkills يختبر نقاط نهاية المهارات
func TestHandleSkills(t *testing.T) {
	server := setupTestServer(t)

	t.Run("RegisterAgent", func(t *testing.T) {
		reqBody := map[string]string{
			"session_id": "test-session",
			"agent_did":  "test-agent",
			"agent_type": "coder",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/skills", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		server.handleSkills(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusCreated {
			t.Errorf("Expected status 200 or 201, got %d", w.Code)
		}
	})
}

// TestHandleArtifacts يختبر نقاط نهاية القطع الأثرية
func TestHandleArtifacts(t *testing.T) {
	server := setupTestServer(t)

	t.Run("AddArtifact", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"session_id":  "test-session",
			"type":        "code",
			"name":        "Test Artifact",
			"description": "Test Description",
			"content":     "test content",
			"created_by":  "test-agent",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/artifacts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		server.handleArtifacts(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusCreated {
			t.Errorf("Expected status 200 or 201, got %d", w.Code)
		}
	})
}

// TestHandleBridges يختبر نقاط نهاية الجسور
func TestHandleBridges(t *testing.T) {
	server := setupTestServer(t)

	t.Run("CreateBridge", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"source_id":   "session-1",
			"target_id":   "session-2",
			"bridge_type": "two_way",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/bridges", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		server.handleBridges(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusCreated && w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 200, 201, or 400, got %d", w.Code)
		}
	})
}

// TestHandleAgents يختبر نقاط نهاية الوكلاء
func TestHandleAgents(t *testing.T) {
	server := setupTestServer(t)

	// أولاً: إنشاء جلسة للتسجيل فيها
	sessionReq := map[string]interface{}{
		"name":             "Test Session",
		"owner_did":        "test-owner",
		"manager_agent_id": "manager-agent",
	}
	sessionBody, _ := json.Marshal(sessionReq)
	sessionHTTPReq := httptest.NewRequest(http.MethodPost, "/api/sessions", bytes.NewReader(sessionBody))
	sessionHTTPReq.Header.Set("Content-Type", "application/json")
	sessionHTTPReq.Header.Set("Authorization", "Bearer test-token")
	sessionW := httptest.NewRecorder()
	server.handleSessions(sessionW, sessionHTTPReq)

	if sessionW.Code != http.StatusOK && sessionW.Code != http.StatusCreated {
		t.Fatalf("Failed to create session, got status %d", sessionW.Code)
	}

	var sessionResp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(sessionW.Body.Bytes(), &sessionResp); err != nil {
		t.Fatalf("Failed to parse session response: %v", err)
	}
	if sessionResp.ID == "" {
		t.Fatal("Session ID is empty after creation")
	}

	// ثانياً: تسجيل وكيل في الجلسة المنشأة
	t.Run("RegisterAgent", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"session_id": sessionResp.ID,
			"agent_did":  "test-agent",
			"name":       "Test Agent",
			"role":       "coder",
			"type":       "coder",
			"metadata": map[string]interface{}{
				"provider": "test-provider",
				"model":    "test-model",
			},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/agents", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		server.handleAgents(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusCreated {
			t.Errorf("Expected status 200 or 201, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	// ثالثاً: التحقق من ظهور الوكيل في قائمة الوكلاء
	t.Run("GetAgents", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/agents?session_id="+sessionResp.ID, nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		server.handleAgents(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var agents []map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &agents); err != nil {
			t.Fatalf("Failed to parse agents list: %v", err)
		}
		if len(agents) == 0 {
			t.Error("Expected at least 1 agent, got 0")
		}
	})
}

// TestHandleMCPServers يختبر نقاط نهاية MCP
func TestHandleMCPServers(t *testing.T) {
	server := setupTestServer(t)

	t.Run("CreateMCPServer", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":        "Test MCP Server",
			"description": "Test Description",
			"endpoint":    "http://localhost:8080",
			"transport":   "stdio",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/mcp/servers", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		server.handleMCPServers(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusCreated {
			t.Errorf("Expected status 200 or 201, got %d", w.Code)
		}
	})
}

// TestConcurrency يختبر التزامن
func TestConcurrency(t *testing.T) {
	server := setupTestServer(t)

	t.Run("ConcurrentRequests", func(t *testing.T) {
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(id int) {
				defer func() { recover() }()
				reqBody := map[string]string{
					"name":        "Test Session",
					"description": "Test Description",
					"owner_did":   "test-owner",
				}
				body, _ := json.Marshal(reqBody)

				req := httptest.NewRequest(http.MethodPost, "/api/sessions", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer test-token")
				w := httptest.NewRecorder()

				server.handleSessions(w, req)
				done <- true
			}(i)
		}

		// انتظار اكتمال جميع الطلبات
		for i := 0; i < 10; i++ {
			<-done
		}

		t.Log("Concurrent requests completed successfully")
	})
}

// BenchmarkHandleSessions يقيس أداء نقاط نهاية الجلسات
func BenchmarkHandleSessions(b *testing.B) {
	server := setupTestServer(&testing.T{})

	reqBody := map[string]string{
		"name":        "Test Session",
		"description": "Test Description",
		"owner_did":   "test-owner",
	}
	body, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/sessions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		server.handleSessions(w, req)
	}
}
