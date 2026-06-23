package orchestrator

import (
	"context"
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/MortalArena/Musketeers/pkg/session"
	"github.com/dgraph-io/badger/v4"
	"go.uber.org/zap"
)

func TestChatConnectorCreation(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()

	// إنشاء AgentRegistry
	agentRegistry := agent.NewAgentRegistry()
	zapLogger := zap.NewNop()
	agentRegistry.SetLogger(zapLogger)

	// إنشاء Session Container
	db, err := badger.Open(badger.DefaultOptions(t.TempDir() + "/badger"))
	if err != nil {
		t.Fatalf("فشل فتح BadgerDB: %v", err)
	}
	defer db.Close()

	sessionConfig := &session.SessionConfig{
		Name:        "Test Session",
		Description: "Test session for chat connector",
		OwnerDID:    "did:test:123",
		MaxAgents:   10,
		ProjectType: "test",
	}

	testCtx := context.Background()
	sessionContainer, err := session.NewSessionContainer(testCtx, db, sessionConfig, eventBus)
	if err != nil {
		t.Fatalf("فشل إنشاء SessionContainer: %v", err)
	}

	// إنشاء مفتاح خاص
	_, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("فشل توليد المفتاح: %v", err)
	}

	// إنشاء ChatConnector
	chatConnector := NewChatConnector(eventBus, agentRegistry, sessionContainer, privateKey, zapLogger)

	if chatConnector == nil {
		t.Fatal("فشل إنشاء ChatConnector")
	}

	t.Log("تم إنشاء ChatConnector بنجاح")
}

func TestChatConnectorStartStop(t *testing.T) {
	// إنشاء المكونات
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()
	agentRegistry := agent.NewAgentRegistry()
	zapLogger := zap.NewNop()
	agentRegistry.SetLogger(zapLogger)

	db, err := badger.Open(badger.DefaultOptions(t.TempDir() + "/badger"))
	if err != nil {
		t.Fatalf("فشل فتح BadgerDB: %v", err)
	}
	defer db.Close()

	sessionConfig := &session.SessionConfig{
		Name:        "Test Session",
		Description: "Test session for chat connector",
		OwnerDID:    "did:test:123",
		MaxAgents:   10,
		ProjectType: "test",
	}
	testCtx := context.Background()
	sessionContainer, err := session.NewSessionContainer(testCtx, db, sessionConfig, eventBus)
	if err != nil {
		t.Fatalf("فشل إنشاء SessionContainer: %v", err)
	}

	_, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("فشل توليد المفتاح: %v", err)
	}

	chatConnector := NewChatConnector(eventBus, agentRegistry, sessionContainer, privateKey, zapLogger)

	// بدء ChatConnector
	err = chatConnector.Start()
	if err != nil {
		t.Fatalf("فشل بدء ChatConnector: %v", err)
	}

	// انتظار قصير
	time.Sleep(100 * time.Millisecond)

	// إيقاف ChatConnector
	err = chatConnector.Stop()
	if err != nil {
		t.Fatalf("فشل إيقاف ChatConnector: %v", err)
	}

	t.Log("تم بدء وإيقاف ChatConnector بنجاح")
}

func TestChatConnectorPrivateChannel(t *testing.T) {
	// إنشاء المكونات
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()
	agentRegistry := agent.NewAgentRegistry()
	zapLogger := zap.NewNop()
	agentRegistry.SetLogger(zapLogger)

	db, err := badger.Open(badger.DefaultOptions(t.TempDir() + "/badger"))
	if err != nil {
		t.Fatalf("فشل فتح BadgerDB: %v", err)
	}
	defer db.Close()

	sessionConfig := &session.SessionConfig{
		Name:        "Test Session",
		Description: "Test session for chat connector",
		OwnerDID:    "did:test:123",
		MaxAgents:   10,
		ProjectType: "test",
	}
	testCtx := context.Background()
	sessionContainer, err := session.NewSessionContainer(testCtx, db, sessionConfig, eventBus)
	if err != nil {
		t.Fatalf("فشل إنشاء SessionContainer: %v", err)
	}

	_, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("فشل توليد المفتاح: %v", err)
	}

	chatConnector := NewChatConnector(eventBus, agentRegistry, sessionContainer, privateKey, zapLogger)
	err = chatConnector.Start()
	if err != nil {
		t.Fatalf("فشل بدء ChatConnector: %v", err)
	}
	defer chatConnector.Stop()

	// إنشاء قناة خاصة
	channelID, err := chatConnector.CreatePrivateChannel("did:test:client", "did:test:agent")
	if err != nil {
		t.Fatalf("فشل إنشاء قناة خاصة: %v", err)
	}

	if channelID == "" {
		t.Fatal("معرف القناة فارغ")
	}

	t.Log("تم إنشاء قناة خاصة بنجاح", channelID)
}

func TestChatConnectorPublicChannel(t *testing.T) {
	// إنشاء المكونات
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()
	agentRegistry := agent.NewAgentRegistry()
	zapLogger := zap.NewNop()
	agentRegistry.SetLogger(zapLogger)

	db, err := badger.Open(badger.DefaultOptions(t.TempDir() + "/badger"))
	if err != nil {
		t.Fatalf("فشل فتح BadgerDB: %v", err)
	}
	defer db.Close()

	sessionConfig := &session.SessionConfig{
		Name:        "Test Session",
		Description: "Test session for chat connector",
		OwnerDID:    "did:test:123",
		MaxAgents:   10,
		ProjectType: "test",
	}
	testCtx := context.Background()
	sessionContainer, err := session.NewSessionContainer(testCtx, db, sessionConfig, eventBus)
	if err != nil {
		t.Fatalf("فشل إنشاء SessionContainer: %v", err)
	}

	_, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("فشل توليد المفتاح: %v", err)
	}

	chatConnector := NewChatConnector(eventBus, agentRegistry, sessionContainer, privateKey, zapLogger)
	err = chatConnector.Start()
	if err != nil {
		t.Fatalf("فشل بدء ChatConnector: %v", err)
	}
	defer chatConnector.Stop()

	// إنشاء قناة عامة
	err = chatConnector.CreatePublicChannel("general-chat")
	if err != nil {
		t.Fatalf("فشل إنشاء قناة عامة: %v", err)
	}

	t.Log("تم إنشاء قناة عامة بنجاح")
}

func TestChatConnectorSessionChannel(t *testing.T) {
	// إنشاء المكونات
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()
	agentRegistry := agent.NewAgentRegistry()
	zapLogger := zap.NewNop()
	agentRegistry.SetLogger(zapLogger)

	db, err := badger.Open(badger.DefaultOptions(t.TempDir() + "/badger"))
	if err != nil {
		t.Fatalf("فشل فتح BadgerDB: %v", err)
	}
	defer db.Close()

	sessionConfig := &session.SessionConfig{
		Name:        "Test Session",
		Description: "Test session for chat connector",
		OwnerDID:    "did:test:123",
		MaxAgents:   10,
		ProjectType: "test",
	}
	testCtx := context.Background()
	sessionContainer, err := session.NewSessionContainer(testCtx, db, sessionConfig, eventBus)
	if err != nil {
		t.Fatalf("فشل إنشاء SessionContainer: %v", err)
	}

	_, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("فشل توليد المفتاح: %v", err)
	}

	chatConnector := NewChatConnector(eventBus, agentRegistry, sessionContainer, privateKey, zapLogger)
	err = chatConnector.Start()
	if err != nil {
		t.Fatalf("فشل بدء ChatConnector: %v", err)
	}
	defer chatConnector.Stop()

	// إنشاء قناة جلسة
	channelID, err := chatConnector.CreateSessionChannel("session-123")
	if err != nil {
		t.Fatalf("فشل إنشاء قناة جلسة: %v", err)
	}

	if channelID == "" {
		t.Fatal("معرف القناة فارغ")
	}

	t.Log("تم إنشاء قناة جلسة بنجاح", channelID)
}

func TestChatConnectorMetrics(t *testing.T) {
	// إنشاء المكونات
	eventBus := eventbus.NewEventBus()
	defer eventBus.Stop()
	agentRegistry := agent.NewAgentRegistry()
	zapLogger := zap.NewNop()
	agentRegistry.SetLogger(zapLogger)

	db, err := badger.Open(badger.DefaultOptions(t.TempDir() + "/badger"))
	if err != nil {
		t.Fatalf("فشل فتح BadgerDB: %v", err)
	}
	defer db.Close()

	sessionConfig := &session.SessionConfig{
		Name:        "Test Session",
		Description: "Test session for chat connector",
		OwnerDID:    "did:test:123",
		MaxAgents:   10,
		ProjectType: "test",
	}
	testCtx := context.Background()
	sessionContainer, err := session.NewSessionContainer(testCtx, db, sessionConfig, eventBus)
	if err != nil {
		t.Fatalf("فشل إنشاء SessionContainer: %v", err)
	}

	_, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("فشل توليد المفتاح: %v", err)
	}

	chatConnector := NewChatConnector(eventBus, agentRegistry, sessionContainer, privateKey, zapLogger)
	err = chatConnector.Start()
	if err != nil {
		t.Fatalf("فشل بدء ChatConnector: %v", err)
	}
	defer chatConnector.Stop()

	// الحصول على المقاييس
	metrics := chatConnector.GetMetrics()
	if metrics == nil {
		t.Fatal("المقاييس nil")
	}

	t.Logf("المقاييس: MessagesSent=%d, MessagesReceived=%d, PrivateChannels=%d, PublicChannels=%d, SessionChannels=%d",
		metrics.MessagesSent, metrics.MessagesReceived, metrics.PrivateChannels, metrics.PublicChannels, metrics.SessionChannels)
}
