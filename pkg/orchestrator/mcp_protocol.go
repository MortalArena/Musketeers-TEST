package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/capability"
	capgithub "github.com/MortalArena/Musketeers/pkg/capability/github"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/MortalArena/Musketeers/pkg/policy"
	"go.uber.org/zap"
)

// ============================================================
// MCP Protocol - Model Context Protocol
// ============================================================

// MCPManager يدير بروتوكول MCP للتواصل مع الأدوات الخارجية
type MCPManager struct {
	// المكونات الأساسية
	eventBus *eventbus.EventBus

	// MCP Servers
	servers map[string]*MCPServer
	mu      sync.RWMutex

	// MCP Clients
	clients map[string]*MCPClient

	// Channels للتواصل الداخلي
	mcpToEventBus chan *MCPMessage
	eventBusToMCP chan eventbus.Event

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Logger
	logger *zap.Logger

	// Metrics
	metrics *MCPMetrics

	// ربط بـ capability system للتنفيذ الحقيقي
	capabilityManager *capability.Manager
}

// MCPMetrics مقاييس MCP
type MCPMetrics struct {
	ToolsInvoked  int64
	ResourcesRead int64
	PromptsUsed   int64
	Errors        int64
	LastActivity  time.Time
	ServersCount  int
	ClientsCount  int
}

// MCPServer يمثل MCP Server
type MCPServer struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"` // github, postgres, slack, etc.
	Tools     []*MCPTool             `json:"tools"`
	Resources []*MCPResource         `json:"resources"`
	Prompts   []*MCPPrompt           `json:"prompts"`
	Config    map[string]interface{} `json:"config"`
	Enabled   bool                   `json:"enabled"`
	LastUsed  time.Time              `json:"last_used"`
}

// MCPTool أداة MCP
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// MCPResource مورد MCP
type MCPResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mime_type"`
}

// MCPPrompt موجه MCP
type MCPPrompt struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Arguments   []MCPArgument `json:"arguments"`
}

// MCPArgument وسيط MCP
type MCPArgument struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// MCPClient عميل MCP
type MCPClient struct {
	ID        string                 `json:"id"`
	ServerID  string                 `json:"server_id"`
	Config    map[string]interface{} `json:"config"`
	Connected bool                   `json:"connected"`
	LastUsed  time.Time              `json:"last_used"`
}

// MCPMessage رسالة MCP
type MCPMessage struct {
	Type        string                 `json:"type"` // tools/list, tools/call, resources/read, prompts/get
	ServerID    string                 `json:"server_id"`
	ToolName    string                 `json:"tool_name,omitempty"`
	ResourceURI string                 `json:"resource_uri,omitempty"`
	PromptName  string                 `json:"prompt_name,omitempty"`
	Arguments   map[string]interface{} `json:"arguments,omitempty"`
	Result      interface{}            `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// NewMCPManager ينشئ MCPManager جديد
func NewMCPManager(eventBus *eventbus.EventBus, logger *zap.Logger) *MCPManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &MCPManager{
		eventBus:      eventBus,
		servers:       make(map[string]*MCPServer),
		clients:       make(map[string]*MCPClient),
		mcpToEventBus: make(chan *MCPMessage, 1000),
		eventBusToMCP: make(chan eventbus.Event, 1000),
		ctx:           ctx,
		cancel:        cancel,
		logger:        logger,
		metrics:       &MCPMetrics{},
	}
}

// Start يبدأ MCPManager
func (m *MCPManager) Start() error {
	m.logger.Info("بدء MCPManager")

	// تسجيل MCP Servers الافتراضية
	m.registerDefaultServers()

	// الاشتراك في أحداث Event Bus
	m.subscribeToEventBus()

	// بدء معالج MCP
	m.wg.Add(1)
	go m.mcpHandler()

	// بدء معالج Event Bus
	m.wg.Add(1)
	go m.eventBusHandler()

	m.logger.Info("تم بدء MCPManager بنجاح")
	return nil
}

// Stop يوقف MCPManager
func (m *MCPManager) Stop() error {
	m.logger.Info("إيقاف MCPManager")

	m.cancel()
	m.wg.Wait()

	close(m.mcpToEventBus)
	close(m.eventBusToMCP)

	m.logger.Info("تم إيقاف MCPManager بنجاح")
	return nil
}

// ============================================================
// تسجيل MCP Servers
// ============================================================

// registerDefaultServers يسجل MCP Servers الافتراضية
func (m *MCPManager) registerDefaultServers() {
	// GitHub MCP Server
	githubServer := &MCPServer{
		ID:   "github",
		Name: "GitHub",
		Type: "github",
		Tools: []*MCPTool{
			{
				Name:        "create_issue",
				Description: "Create a new GitHub issue",
				InputSchema: map[string]interface{}{
					"repo":  map[string]string{"type": "string", "description": "Repository name"},
					"title": map[string]string{"type": "string", "description": "Issue title"},
					"body":  map[string]string{"type": "string", "description": "Issue body"},
				},
			},
			{
				Name:        "create_pull_request",
				Description: "Create a new pull request",
				InputSchema: map[string]interface{}{
					"repo":   map[string]string{"type": "string", "description": "Repository name"},
					"branch": map[string]string{"type": "string", "description": "Branch name"},
					"title":  map[string]string{"type": "string", "description": "PR title"},
					"body":   map[string]string{"type": "string", "description": "PR body"},
				},
			},
			{
				Name:        "read_file",
				Description: "Read a file from repository",
				InputSchema: map[string]interface{}{
					"repo": map[string]string{"type": "string", "description": "Repository name"},
					"path": map[string]string{"type": "string", "description": "File path"},
				},
			},
		},
		Resources: []*MCPResource{
			{
				URI:         "repo://source_code",
				Name:        "Source Code",
				Description: "Repository source code",
				MimeType:    "text/plain",
			},
		},
		Enabled: true,
		Config:  map[string]interface{}{},
	}
	m.RegisterServer(githubServer)

	// Postgres MCP Server
	postgresServer := &MCPServer{
		ID:   "postgres",
		Name: "PostgreSQL",
		Type: "database",
		Tools: []*MCPTool{
			{
				Name:        "execute_sql",
				Description: "Execute SQL query",
				InputSchema: map[string]interface{}{
					"query": map[string]string{"type": "string", "description": "SQL query"},
				},
			},
			{
				Name:        "insert",
				Description: "Insert data into table",
				InputSchema: map[string]interface{}{
					"table": map[string]string{"type": "string", "description": "Table name"},
					"data":  map[string]string{"type": "object", "description": "Data to insert"},
				},
			},
		},
		Resources: []*MCPResource{
			{
				URI:         "db://schema",
				Name:        "Database Schema",
				Description: "Database schema information",
				MimeType:    "application/json",
			},
		},
		Enabled: true,
		Config:  map[string]interface{}{},
	}
	m.RegisterServer(postgresServer)

	// Slack MCP Server
	slackServer := &MCPServer{
		ID:   "slack",
		Name: "Slack",
		Type: "messaging",
		Tools: []*MCPTool{
			{
				Name:        "send_message",
				Description: "Send message to Slack channel",
				InputSchema: map[string]interface{}{
					"channel": map[string]string{"type": "string", "description": "Channel name"},
					"text":    map[string]string{"type": "string", "description": "Message text"},
				},
			},
		},
		Enabled: true,
		Config:  map[string]interface{}{},
	}
	m.RegisterServer(slackServer)

	m.logger.Info("تم تسجيل MCP Servers الافتراضية",
		zap.Int("count", len(m.servers)),
	)
}

// RegisterServer يسجل MCP Server جديد
func (m *MCPManager) RegisterServer(server *MCPServer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.servers[server.ID]; exists {
		return fmt.Errorf("السيرفر %s مسجل بالفعل", server.ID)
	}

	m.servers[server.ID] = server
	m.metrics.ServersCount++

	m.logger.Info("تم تسجيل MCP Server جديد",
		zap.String("server_id", server.ID),
		zap.String("name", server.Name),
		zap.String("type", server.Type),
	)

	return nil
}

// GetServer يحصل على MCP Server بالمعرف
func (m *MCPManager) GetServer(serverID string) (*MCPServer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	server, exists := m.servers[serverID]
	if !exists {
		return nil, fmt.Errorf("السيرفر %s غير موجود", serverID)
	}

	return server, nil
}

// ListServers يرجع قائمة جميع MCP Servers
func (m *MCPManager) ListServers() []*MCPServer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	servers := make([]*MCPServer, 0, len(m.servers))
	for _, server := range m.servers {
		servers = append(servers, server)
	}

	return servers
}

// ============================================================
// أدوات MCP
// ============================================================

// SetCapabilityManager يضبط مدير القدرات للتنفيذ الحقيقي لأدوات MCP
func (m *MCPManager) SetCapabilityManager(cm *capability.Manager) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.capabilityManager = cm
	m.logger.Info("تم ربط MCPManager بـ capability manager")
}

// ListTools يرجع قائمة الأدوات من سيرفر معين
func (m *MCPManager) ListTools(serverID string) ([]*MCPTool, error) {
	server, err := m.GetServer(serverID)
	if err != nil {
		return nil, err
	}

	return server.Tools, nil
}

// CallTool يستدعي أداة من سيرفر معين مع دعم التنفيذ الحقيقي عبر capabilities
func (m *MCPManager) CallTool(serverID, toolName string, arguments map[string]interface{}) (interface{}, error) {
	server, err := m.GetServer(serverID)
	if err != nil {
		return nil, err
	}

	if !server.Enabled {
		return nil, fmt.Errorf("السيرفر %s غير مفعّل", serverID)
	}

	// تحديث آخر استخدام
	server.LastUsed = time.Now()

	// إنشاء رسالة MCP للإرسال إلى Event Bus
	msg := &MCPMessage{
		Type:      "tools/call",
		ServerID:  serverID,
		ToolName:  toolName,
		Arguments: arguments,
		Timestamp: time.Now(),
	}

	m.mcpToEventBus <- msg

	m.mu.Lock()
	m.metrics.ToolsInvoked++
	m.metrics.LastActivity = time.Now()
	m.mu.Unlock()

	// التنفيذ الحقيقي عبر capabilities
	if m.capabilityManager != nil {
		principal := policy.Principal{DID: "mcp_manager"}
		switch serverID {
		case "github":
			return m.executeGitHubTool(m.ctx, principal, toolName, arguments)
		case "postgres":
			return m.executePostgresTool(m.ctx, principal, toolName, arguments)
		case "slack":
			return m.executeSlackTool(m.ctx, principal, toolName, arguments)
		}
	}

	// fallback: إرسال إلى Event Bus مع رسالة انتظار
	return map[string]interface{}{
		"success": true,
		"tool":    toolName,
		"result":  fmt.Sprintf("تم إرسال الأداة %s إلى Event Bus للتنفيذ", toolName),
	}, nil
}

// executeGitHubTool ينفذ أدوات GitHub عبر capability/github الحقيقي
func (m *MCPManager) executeGitHubTool(ctx context.Context, principal policy.Principal, toolName string, arguments map[string]interface{}) (interface{}, error) {
	if m.capabilityManager == nil {
		return nil, fmt.Errorf("مدير القدرات غير مهيأ")
	}

	switch toolName {
	case "create_issue":
		repo, _ := arguments["repo"].(string)
		title, _ := arguments["title"].(string)
		body, _ := arguments["body"].(string)
		if repo == "" || title == "" {
			return nil, fmt.Errorf("المعلمات repo و title مطلوبتان")
		}
		// repo قد يكون "owner/repo" أو اسم المستودع فقط
		owner, repoName := repo, repo
		if parts := strings.SplitN(repo, "/", 2); len(parts) == 2 {
			owner, repoName = parts[0], parts[1]
		}
		cmd := capgithub.CreateIssueCommand{Owner: owner, Repo: repoName, Title: title, Body: body}
		return m.capabilityManager.Execute(ctx, principal, cmd)

	case "create_pull_request":
		repo, _ := arguments["repo"].(string)
		title, _ := arguments["title"].(string)
		body, _ := arguments["body"].(string)
		owner, repoName := repo, repo
		if parts := strings.SplitN(repo, "/", 2); len(parts) == 2 {
			owner, repoName = parts[0], parts[1]
		}
		cmd := capgithub.CreateIssueCommand{Owner: owner, Repo: repoName, Title: title, Body: body}
		return m.capabilityManager.Execute(ctx, principal, cmd)

	case "read_file":
		repo, _ := arguments["repo"].(string)
		filePath, _ := arguments["path"].(string)
		owner, repoName := repo, repo
		if parts := strings.SplitN(repo, "/", 2); len(parts) == 2 {
			owner, repoName = parts[0], parts[1]
		}
		cmd := capgithub.ReadFileCommand{Owner: owner, Repo: repoName, Path: filePath}
		return m.capabilityManager.Execute(ctx, principal, cmd)
	}

	return nil, fmt.Errorf("أداة GitHub غير مدعومة: %s", toolName)
}

// executePostgresTool ينفذ أدوات PostgreSQL (محاكي حالياً)
func (m *MCPManager) executePostgresTool(ctx context.Context, principal policy.Principal, toolName string, arguments map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{
		"success": true,
		"tool":    toolName,
		"note":    "PostgreSQL يتطلب تكوين اتصال بقاعدة البيانات",
	}, nil
}

// executeSlackTool ينفذ أدوات Slack (محاكي حالياً)
func (m *MCPManager) executeSlackTool(ctx context.Context, principal policy.Principal, toolName string, arguments map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{
		"success": true,
		"tool":    toolName,
		"note":    "Slack يتطلب تكوين Webhook أو API token",
	}, nil
}

// ============================================================
// موارد MCP
// ============================================================

// ListResources يرجع قائمة الموارد من سيرفر معين
func (m *MCPManager) ListResources(serverID string) ([]*MCPResource, error) {
	server, err := m.GetServer(serverID)
	if err != nil {
		return nil, err
	}

	return server.Resources, nil
}

// ReadResource يقرأ مورد من سيرفر معين
func (m *MCPManager) ReadResource(serverID, resourceURI string) (interface{}, error) {
	server, err := m.GetServer(serverID)
	if err != nil {
		return nil, err
	}

	if !server.Enabled {
		return nil, fmt.Errorf("السيرفر %s غير مفعّل", serverID)
	}

	// تحديث آخر استخدام
	server.LastUsed = time.Now()

	// إنشاء رسالة MCP
	msg := &MCPMessage{
		Type:        "resources/read",
		ServerID:    serverID,
		ResourceURI: resourceURI,
		Timestamp:   time.Now(),
	}

	m.mcpToEventBus <- msg

	m.mu.Lock()
	m.metrics.ResourcesRead++
	m.metrics.LastActivity = time.Now()
	m.mu.Unlock()

	// محاكاة النتيجة
	result := map[string]interface{}{
		"uri":     resourceURI,
		"content": fmt.Sprintf("محتوى المورد %s", resourceURI),
	}

	return result, nil
}

// ============================================================
// موجهات MCP
// ============================================================

// ListPrompts يرجع قائمة الموجهات من سيرفر معين
func (m *MCPManager) ListPrompts(serverID string) ([]*MCPPrompt, error) {
	server, err := m.GetServer(serverID)
	if err != nil {
		return nil, err
	}

	return server.Prompts, nil
}

// GetPrompt يحصل على موجه من سيرفر معين
func (m *MCPManager) GetPrompt(serverID, promptName string) (interface{}, error) {
	server, err := m.GetServer(serverID)
	if err != nil {
		return nil, err
	}

	if !server.Enabled {
		return nil, fmt.Errorf("السيرفر %s غير مفعّل", serverID)
	}

	// تحديث آخر استخدام
	server.LastUsed = time.Now()

	// إنشاء رسالة MCP
	msg := &MCPMessage{
		Type:       "prompts/get",
		ServerID:   serverID,
		PromptName: promptName,
		Timestamp:  time.Now(),
	}

	m.mcpToEventBus <- msg

	m.mu.Lock()
	m.metrics.PromptsUsed++
	m.metrics.LastActivity = time.Now()
	m.mu.Unlock()

	// محاكاة النتيجة
	result := map[string]interface{}{
		"prompt": promptName,
		"text":   fmt.Sprintf("نص الموجه %s", promptName),
	}

	return result, nil
}

// ============================================================
// معالجة الرسائل
// ============================================================

// subscribeToEventBus يرتبط بأحداث Event Bus
func (m *MCPManager) subscribeToEventBus() {
	m.eventBus.Subscribe("mcp.request", m.handleMCPRequest)
	m.eventBus.Subscribe("mcp.response", m.handleMCPResponse)
}

// mcpHandler يعالج رسائل MCP
func (m *MCPManager) mcpHandler() {
	defer m.wg.Done()

	for {
		select {
		case <-m.ctx.Done():
			return
		case msg := <-m.mcpToEventBus:
			m.processMCPMessage(msg)
		}
	}
}

// processMCPMessage يعالج رسالة MCP
func (m *MCPManager) processMCPMessage(msg *MCPMessage) {
	// تحويل الرسالة إلى حدث Event Bus
	event := eventbus.Event{
		Type:      "mcp.message",
		Payload:   msg,
		Source:    msg.ServerID,
		Timestamp: msg.Timestamp,
	}

	// نشر الحدث
	m.eventBus.Publish(event)

	m.logger.Debug("تم معالجة رسالة MCP",
		zap.String("type", msg.Type),
		zap.String("server_id", msg.ServerID),
	)
}

// eventBusHandler يعالج أحداث Event Bus
func (m *MCPManager) eventBusHandler() {
	defer m.wg.Done()

	for {
		select {
		case <-m.ctx.Done():
			return
		case event := <-m.eventBusToMCP:
			m.processEventBusEvent(event)
		}
	}
}

// processEventBusEvent يعالج حدث Event Bus
func (m *MCPManager) processEventBusEvent(event eventbus.Event) {
	m.logger.Debug("تم معالجة حدث Event Bus",
		zap.String("event_type", event.Type),
	)
}

// handleMCPRequest يعالج طلب MCP
func (m *MCPManager) handleMCPRequest(event eventbus.Event) {
	m.logger.Debug("استقبال طلب MCP",
		zap.String("server_id", event.Source),
	)
}

// handleMCPResponse يعالج رد MCP
func (m *MCPManager) handleMCPResponse(event eventbus.Event) {
	m.logger.Debug("استقبال رد MCP",
		zap.String("server_id", event.Source),
	)
}

// ============================================================
// المقاييس
// ============================================================

// GetMetrics يحصل على المقاييس
func (m *MCPManager) GetMetrics() *MCPMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &MCPMetrics{
		ToolsInvoked:  m.metrics.ToolsInvoked,
		ResourcesRead: m.metrics.ResourcesRead,
		PromptsUsed:   m.metrics.PromptsUsed,
		Errors:        m.metrics.Errors,
		LastActivity:  m.metrics.LastActivity,
		ServersCount:  m.metrics.ServersCount,
		ClientsCount:  m.metrics.ClientsCount,
	}
}
