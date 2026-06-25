package wiring

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// WiringLayer الطبقة التي تربط جميع Adapters تلقائياً
type WiringLayer struct {
	sessionID string
	agentID   string
	logger    *zap.Logger
	mu        sync.RWMutex

	// Adapters المسجلة
	adapters map[string]Adapter
	// Connections بين Adapters
	connections map[string][]Connection
	// حالة التوصيل
	connected bool
}

// Adapter واجهة عامة للـ Adapters
type Adapter interface {
	// Connect يربط الـ Adapter بمكون آخر
	Connect(ctx context.Context, target interface{}) error
	// Disconnect يفصل الـ Adapter
	Disconnect(ctx context.Context) error
	// IsConnected يرجع حالة الاتصال
	IsConnected() bool
	// GetName يرجع اسم الـ Adapter
	GetName() string
}

// Connection يمثل اتصال بين Adapterين
type Connection struct {
	From     string
	To       string
	Priority int // 1-10، 10 هو الأعلى
	Status   string // "pending", "active", "failed"
}

// NewWiringLayer ينشئ طبقة توصيل جديدة
func NewWiringLayer(sessionID, agentID string, logger *zap.Logger) *WiringLayer {
	return &WiringLayer{
		sessionID:  sessionID,
		agentID:    agentID,
		logger:     logger,
		adapters:   make(map[string]Adapter),
		connections: make(map[string][]Connection),
		connected:  false,
	}
}

// RegisterAdapter يسجل Adapter جديد
func (wl *WiringLayer) RegisterAdapter(adapter Adapter) error {
	wl.mu.Lock()
	defer wl.mu.Unlock()

	name := adapter.GetName()
	if _, exists := wl.adapters[name]; exists {
		return fmt.Errorf("adapter '%s' already registered", name)
	}

	wl.adapters[name] = adapter
	wl.logger.Info("تم تسجيل Adapter",
		zap.String("adapter", name),
		zap.String("session_id", wl.sessionID),
		zap.String("agent_id", wl.agentID),
	)

	return nil
}

// UnregisterAdapter يلغي تسجيل Adapter
func (wl *WiringLayer) UnregisterAdapter(name string) error {
	wl.mu.Lock()
	defer wl.mu.Unlock()

	if _, exists := wl.adapters[name]; !exists {
		return fmt.Errorf("adapter '%s' not found", name)
	}

	// فصل جميع الاتصالات المتعلقة بهذا Adapter
	if connections, exists := wl.connections[name]; exists {
		for _, conn := range connections {
			if adapter, exists := wl.adapters[conn.To]; exists {
				adapter.Disconnect(context.Background())
			}
		}
		delete(wl.connections, name)
	}

	delete(wl.adapters, name)
	wl.logger.Info("تم إلغاء تسجيل Adapter",
		zap.String("adapter", name),
		zap.String("session_id", wl.sessionID),
	)

	return nil
}

// AddConnection يضيف اتصال بين Adapterين
func (wl *WiringLayer) AddConnection(from, to string, priority int) error {
	wl.mu.Lock()
	defer wl.mu.Unlock()

	// التحقق من وجود Adapters
	if _, exists := wl.adapters[from]; !exists {
		return fmt.Errorf("adapter '%s' not found", from)
	}

	if _, exists := wl.adapters[to]; !exists {
		return fmt.Errorf("adapter '%s' not found", to)
	}

	// إضافة الاتصال
	connection := Connection{
		From:     from,
		To:       to,
		Priority: priority,
		Status:   "pending",
	}

	wl.connections[from] = append(wl.connections[from], connection)
	wl.logger.Info("تم إضافة اتصال",
		zap.String("from", from),
		zap.String("to", to),
		zap.Int("priority", priority),
	)

	return nil
}

// ConnectAll يربط جميع Adapters تلقائياً
func (wl *WiringLayer) ConnectAll(ctx context.Context) error {
	wl.mu.Lock()
	defer wl.mu.Unlock()

	if wl.connected {
		return fmt.Errorf("already connected")
	}

	wl.logger.Info("بدء ربط جميع Adapters تلقائياً",
		zap.String("session_id", wl.sessionID),
		zap.String("agent_id", wl.agentID),
	)

	// ترتيب الاتصالات حسب الأولوية
	sortedConnections := wl.sortConnectionsByPriority()

	// تنفيذ الاتصالات بالترتيب
	for _, conn := range sortedConnections {
		fromAdapter, exists := wl.adapters[conn.From]
		if !exists {
			wl.logger.Warn("Adapter المصدر غير موجود",
				zap.String("from", conn.From),
			)
			continue
		}

		toAdapter, exists := wl.adapters[conn.To]
		if !exists {
			wl.logger.Warn("Adapter الهدف غير موجود",
				zap.String("to", conn.To),
			)
			continue
		}

		// تنفيذ الاتصال
		if err := fromAdapter.Connect(ctx, toAdapter); err != nil {
			wl.logger.Error("فشل ربط Adapter",
				zap.String("from", conn.From),
				zap.String("to", conn.To),
				zap.Error(err),
			)
			conn.Status = "failed"
		} else {
			conn.Status = "active"
			wl.logger.Info("تم ربط Adapter بنجاح",
				zap.String("from", conn.From),
				zap.String("to", conn.To),
			)
		}
	}

	wl.connected = true
	wl.logger.Info("تم ربط جميع Adapters بنجاح",
		zap.String("session_id", wl.sessionID),
		zap.String("agent_id", wl.agentID),
	)

	return nil
}

// DisconnectAll يفصل جميع Adapters
func (wl *WiringLayer) DisconnectAll(ctx context.Context) error {
	wl.mu.Lock()
	defer wl.mu.Unlock()

	if !wl.connected {
		return nil
	}

	wl.logger.Info("بدء فصل جميع Adapters",
		zap.String("session_id", wl.sessionID),
	)

	// فصل جميع Adapters
	for name, adapter := range wl.adapters {
		if err := adapter.Disconnect(ctx); err != nil {
			wl.logger.Error("فشل فصل Adapter",
				zap.String("adapter", name),
				zap.Error(err),
			)
		}
	}

	// إعادة تعيين حالة الاتصالات
	for from, connections := range wl.connections {
		for i := range connections {
			connections[i].Status = "pending"
		}
		wl.connections[from] = connections
	}

	wl.connected = false
	wl.logger.Info("تم فصل جميع Adapters بنجاح",
		zap.String("session_id", wl.sessionID),
	)

	return nil
}

// GetConnectionStatus يرجع حالة جميع الاتصالات
func (wl *WiringLayer) GetConnectionStatus() map[string]interface{} {
	wl.mu.RLock()
	defer wl.mu.RUnlock()

	status := make(map[string]interface{})
	status["connected"] = wl.connected
	status["adapters_count"] = len(wl.adapters)
	status["connections_count"] = len(wl.connections)

	connectionsStatus := make(map[string][]map[string]interface{})
	for from, connections := range wl.connections {
		for _, conn := range connections {
			connectionsStatus[from] = append(connectionsStatus[from], map[string]interface{}{
				"to":       conn.To,
				"priority": conn.Priority,
				"status":   conn.Status,
			})
		}
	}
	status["connections"] = connectionsStatus

	return status
}

// sortConnectionsByPriority يرتب الاتصالات حسب الأولوية
func (wl *WiringLayer) sortConnectionsByPriority() []Connection {
	var allConnections []Connection

	for _, connections := range wl.connections {
		allConnections = append(allConnections, connections...)
	}

	// ترتيب حسب الأولوية (تنازلي)
	for i := 0; i < len(allConnections); i++ {
		for j := i + 1; j < len(allConnections); j++ {
			if allConnections[i].Priority < allConnections[j].Priority {
				allConnections[i], allConnections[j] = allConnections[j], allConnections[i]
			}
		}
	}

	return allConnections
}

// AutoWire يربط Adapters تلقائياً بناءً على القواعد المحددة
func (wl *WiringLayer) AutoWire(ctx context.Context) error {
	wl.logger.Info("بدء التوصيل التلقائي",
		zap.String("session_id", wl.sessionID),
	)

	// قواعد التوصيل التلقائي
	rules := []struct {
		from     string
		to       string
		priority int
	}{
		// ThinkingEngine يجب أن يتصل بـ SessionManager أولاً
		{"ThinkingEngine", "SessionManager", 10},
		// SessionManager يجب أن يتصل بـ WorkflowEngine
		{"SessionManager", "WorkflowEngine", 9},
		// WorkflowEngine يجب أن يتصل بـ TaskManager
		{"WorkflowEngine", "TaskManager", 8},
		// TaskManager يجب أن يتصل بـ ToolExecutor
		{"TaskManager", "ToolExecutor", 7},
		// ToolExecutor يجب أن يتصل بـ RuntimeIntegration
		{"ToolExecutor", "RuntimeIntegration", 6},
		// RuntimeIntegration يجب أن يتصل بـ ProviderRegistry
		{"RuntimeIntegration", "ProviderRegistry", 5},
		// ProviderRegistry يجب أن يتصل بـ Router
		{"ProviderRegistry", "Router", 4},
		// Router يجب أن يتصل بـ EventBus
		{"Router", "EventBus", 3},
		// EventBus يجب أن يتصل بـ SyncManager
		{"EventBus", "SyncManager", 2},
		// SyncManager يجب أن يتصل بـ CollectiveSystem
		{"SyncManager", "CollectiveSystem", 1},
	}

	// إضافة الاتصالات بناءً على القواعد
	for _, rule := range rules {
		if _, exists := wl.adapters[rule.from]; exists {
			if _, exists := wl.adapters[rule.to]; exists {
				if err := wl.AddConnection(rule.from, rule.to, rule.priority); err != nil {
					wl.logger.Warn("فشل إضافة اتصال تلقائي",
						zap.String("from", rule.from),
						zap.String("to", rule.to),
						zap.Error(err),
					)
				}
			}
		}
	}

	// تنفيذ الاتصالات
	return wl.ConnectAll(ctx)
}

// IsConnected يرجع حالة التوصيل العامة
func (wl *WiringLayer) IsConnected() bool {
	wl.mu.RLock()
	defer wl.mu.RUnlock()
	return wl.connected
}

// GetAdapter يرجع Adapter معين
func (wl *WiringLayer) GetAdapter(name string) (Adapter, error) {
	wl.mu.RLock()
	defer wl.mu.RUnlock()

	adapter, exists := wl.adapters[name]
	if !exists {
		return nil, fmt.Errorf("adapter '%s' not found", name)
	}

	return adapter, nil
}

// GetAllAdapters يرجع جميع Adapters المسجلة
func (wl *WiringLayer) GetAllAdapters() map[string]Adapter {
	wl.mu.RLock()
	defer wl.mu.RUnlock()

	// نسخة للقراءة فقط
	adapters := make(map[string]Adapter)
	for name, adapter := range wl.adapters {
		adapters[name] = adapter
	}

	return adapters
}
