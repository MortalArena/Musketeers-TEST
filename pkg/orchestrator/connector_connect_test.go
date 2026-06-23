package orchestrator

import (
	"testing"
	"time"

	agent "github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

func TestConnectorGetOnlineAgents(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()

	// إنشاء AgentRegistry
	agentRegistry := agent.NewAgentRegistry()
	agentRegistry.SetLogger(zap.NewNop())

	// إنشاء Connector
	connector := NewConnector(eventBus, nil, agentRegistry, zap.NewNop())

	// بدء Connector
	if err := connector.Start(); err != nil {
		t.Fatalf("فشل بدء Connector: %v", err)
	}
	defer connector.Stop()

	// الحصول على الوكلاء المتصلين
	onlineAgents := connector.GetOnlineAgents()

	if onlineAgents == nil {
		t.Fatal("يجب أن يكون هناك قائمة وكلاء متصلين")
	}

	t.Logf("عدد الوكلاء المتصلين: %d", len(onlineAgents))
}

func TestConnectorGetAllAgents(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()

	// إنشاء AgentRegistry
	agentRegistry := agent.NewAgentRegistry()
	agentRegistry.SetLogger(zap.NewNop())

	// إنشاء Connector
	connector := NewConnector(eventBus, nil, agentRegistry, zap.NewNop())

	// بدء Connector
	if err := connector.Start(); err != nil {
		t.Fatalf("فشل بدء Connector: %v", err)
	}
	defer connector.Stop()

	// الحصول على جميع الوكلاء
	allAgents := connector.GetAllAgents()

	if allAgents == nil {
		t.Fatal("يجب أن يكون هناك قائمة جميع الوكلاء")
	}

	t.Logf("عدد جميع الوكلاء: %d", len(allAgents))
}

func TestConnectorGetAgentHealthReport(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()

	// إنشاء AgentRegistry
	agentRegistry := agent.NewAgentRegistry()
	agentRegistry.SetLogger(zap.NewNop())

	// إنشاء Connector
	connector := NewConnector(eventBus, nil, agentRegistry, zap.NewNop())

	// بدء Connector
	if err := connector.Start(); err != nil {
		t.Fatalf("فشل بدء Connector: %v", err)
	}
	defer connector.Stop()

	// الحصول على تقرير صحة الوكلاء
	healthReport := connector.GetAgentHealthReport()

	if healthReport == nil {
		t.Fatal("يجب أن يكون هناك تقرير صحة")
	}

	t.Logf("تقرير الصحة: %+v", healthReport)
}

func TestConnectorCleanupInactiveAgents(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()

	// إنشاء AgentRegistry
	agentRegistry := agent.NewAgentRegistry()
	agentRegistry.SetLogger(zap.NewNop())

	// إنشاء Connector
	connector := NewConnector(eventBus, nil, agentRegistry, zap.NewNop())

	// بدء Connector
	if err := connector.Start(); err != nil {
		t.Fatalf("فشل بدء Connector: %v", err)
	}
	defer connector.Stop()

	// تنظيف الوكلاء غير النشطين
	removed := connector.CleanupInactiveAgents(1 * time.Hour)

	if removed == nil {
		t.Fatal("يجب أن يكون هناك قائمة الوكلاء الذين تمت إزالتهم")
	}

	t.Logf("عدد الوكلاء الذين تمت إزالتهم: %d", len(removed))
}

func TestConnectorGetAgentMetadata(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()

	// إنشاء AgentRegistry
	agentRegistry := agent.NewAgentRegistry()
	agentRegistry.SetLogger(zap.NewNop())

	// إنشاء Connector
	connector := NewConnector(eventBus, nil, agentRegistry, zap.NewNop())

	// بدء Connector
	if err := connector.Start(); err != nil {
		t.Fatalf("فشل بدء Connector: %v", err)
	}
	defer connector.Stop()

	// الحصول على بيانات وصفية للوكيل
	metadata, err := connector.GetAgentMetadata("test-agent-id")

	if err != nil {
		t.Logf("الوكيل غير موجود (متوقع): %v", err)
	} else {
		t.Logf("البيانات الوصفية: %+v", metadata)
	}
}

func TestConnectorGetAgentStats(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()

	// إنشاء AgentRegistry
	agentRegistry := agent.NewAgentRegistry()
	agentRegistry.SetLogger(zap.NewNop())

	// إنشاء Connector
	connector := NewConnector(eventBus, nil, agentRegistry, zap.NewNop())

	// بدء Connector
	if err := connector.Start(); err != nil {
		t.Fatalf("فشل بدء Connector: %v", err)
	}
	defer connector.Stop()

	// الحصول على إحصائيات الوكيل
	stats, err := connector.GetAgentStats("test-agent-id")

	if err != nil {
		t.Logf("الوكيل غير موجود (متوقع): %v", err)
	} else {
		t.Logf("الإحصائيات: %+v", stats)
	}
}
