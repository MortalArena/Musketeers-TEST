package orchestrator

import (
	"testing"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/agent_bridge"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

func TestConnectorCreation(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء MultiplexedBridge
	log := logrus.New()
	bridge := agent_bridge.NewMultiplexedBridge(log)

	// إنشاء AgentRegistry
	agentRegistry := agent.NewAgentRegistry()
	zapLogger := zap.NewNop()
	agentRegistry.SetLogger(zapLogger)

	// إنشاء Connector
	connector := NewConnector(eventBus, bridge, agentRegistry, zapLogger)

	if connector == nil {
		t.Fatal("فشل إنشاء Connector")
	}

	t.Log("تم إنشاء Connector بنجاح")
}

func TestConnectorStartStop(t *testing.T) {
	// إنشاء المكونات
	eventBus := eventbus.NewEventBus()
	log := logrus.New()
	bridge := agent_bridge.NewMultiplexedBridge(log)
	agentRegistry := agent.NewAgentRegistry()
	zapLogger := zap.NewNop()
	agentRegistry.SetLogger(zapLogger)

	// إنشاء Connector
	connector := NewConnector(eventBus, bridge, agentRegistry, zapLogger)

	// بدء Connector
	err := connector.Start()
	if err != nil {
		t.Fatalf("فشل بدء Connector: %v", err)
	}

	// انتظار قصير
	time.Sleep(100 * time.Millisecond)

	// إيقاف Connector
	err = connector.Stop()
	if err != nil {
		t.Fatalf("فشل إيقاف Connector: %v", err)
	}

	t.Log("تم بدء وإيقاف Connector بنجاح")
}

func TestConnectorMetrics(t *testing.T) {
	// إنشاء المكونات
	eventBus := eventbus.NewEventBus()
	log := logrus.New()
	bridge := agent_bridge.NewMultiplexedBridge(log)
	agentRegistry := agent.NewAgentRegistry()
	zapLogger := zap.NewNop()
	agentRegistry.SetLogger(zapLogger)

	// إنشاء Connector
	connector := NewConnector(eventBus, bridge, agentRegistry, zapLogger)

	// بدء Connector
	err := connector.Start()
	if err != nil {
		t.Fatalf("فشل بدء Connector: %v", err)
	}
	defer connector.Stop()

	// انتظار قصير
	time.Sleep(100 * time.Millisecond)

	// الحصول على المقاييس
	metrics := connector.GetMetrics()
	if metrics == nil {
		t.Fatal("المقاييس nil")
	}

	t.Logf("المقاييس: MessagesReceived=%d, MessagesSent=%d, EventsPublished=%d",
		metrics.MessagesReceived, metrics.MessagesSent, metrics.EventsPublished)
}

func TestConnectorAdapterRegistration(t *testing.T) {
	// إنشاء المكونات
	eventBus := eventbus.NewEventBus()
	log := logrus.New()
	bridge := agent_bridge.NewMultiplexedBridge(log)
	agentRegistry := agent.NewAgentRegistry()
	zapLogger := zap.NewNop()
	agentRegistry.SetLogger(zapLogger)

	// إنشاء Connector
	connector := NewConnector(eventBus, bridge, agentRegistry, zapLogger)

	// تسجيل محول مخصص
	customAdapter := &TestAdapter{name: "custom"}
	err := connector.RegisterAdapter("custom", customAdapter)
	if err != nil {
		t.Fatalf("فشل تسجيل المحول: %v", err)
	}

	// الحصول على المحول
	retrievedAdapter, err := connector.GetAdapter("custom")
	if err != nil {
		t.Fatalf("فشل الحصول على المحول: %v", err)
	}

	if retrievedAdapter.Name() != "custom" {
		t.Fatalf("اسم المحول غير متطابق")
	}

	t.Log("تم تسجيل واسترجاع المحول بنجاح")
}

// TestAdapter محول اختباري
type TestAdapter struct {
	name string
}

func (ta *TestAdapter) Name() string {
	return ta.name
}

func (ta *TestAdapter) Convert(data interface{}) (interface{}, error) {
	return data, nil
}
