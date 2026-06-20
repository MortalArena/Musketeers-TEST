package main

import (
	"context"
	"flag"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MortalArena/Musketeers/pkg/acp"
	pkgAgent "github.com/MortalArena/Musketeers/pkg/agent"
	pkgAdapters "github.com/MortalArena/Musketeers/pkg/agent/adapters"
	agentTools "github.com/MortalArena/Musketeers/pkg/agent/tools"
	"github.com/MortalArena/Musketeers/pkg/agent_bridge"
	pkgCapability "github.com/MortalArena/Musketeers/pkg/capability"
	pkgCEO "github.com/MortalArena/Musketeers/pkg/ceo"
	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
	pkgEventbus "github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/MortalArena/Musketeers/pkg/identity"
	"github.com/MortalArena/Musketeers/pkg/node"
	pkgOrchestrator "github.com/MortalArena/Musketeers/pkg/orchestrator"
	pkgPolicy "github.com/MortalArena/Musketeers/pkg/policy"
	pkgSession "github.com/MortalArena/Musketeers/pkg/session"
	"github.com/MortalArena/Musketeers/pkg/storage"
	pkgVerification "github.com/MortalArena/Musketeers/pkg/verification"
	"github.com/dgraph-io/badger/v4"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

var (
	addr       = flag.String("addr", "127.0.0.1:5000", "Studio server address")
	agentAddr  = flag.String("agent-addr", "127.0.0.1:5001", "Agent bridge address")
	dataDir    = flag.String("data-dir", "./studio-data", "Data directory")
	bootstrap  = flag.String("bootstrap", "", "Bootstrap peer multiaddr")
	founderPub = flag.String("founder-pub", "", "Founder public key hex")
	verbose    = flag.Bool("verbose", false, "Verbose logging")
)

func main() {
	flag.Parse()

	log := logrus.New()
	if *verbose {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}
	log.SetFormatter(&logrus.JSONFormatter{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// إنشاء مفاتيح
	kp, err := nrcrypto.GenerateKeyPair()
	if err != nil {
		log.WithError(err).Fatal("Failed to generate key pair")
	}

	// إنشاء سجل هوية
	idRec, err := identity.NewIdentityRecord(ctx, kp, []string{"studio"}, 86400*365) // سنة واحدة
	if err != nil {
		log.WithError(err).Fatal("Failed to create identity record")
	}

	// إنشاء عقدة
	cfg := &node.Config{
		DataDir:        *dataDir,
		ListenPort:     4001,
		StorageQuotaMB: 2048, // 2GB
		FounderPubHex:  *founderPub,
		BootstrapPeers: parseBootstrap(*bootstrap),
	}

	n, err := node.New(ctx, cfg, kp, idRec)
	if err != nil {
		log.WithError(err).Fatal("Failed to create node")
	}
	defer n.Close()

	log.WithField("did", kp.DID).Info("Studio node created")

	// نشر الهوية على DHT
	if err := n.PublishIdentity(ctx); err != nil {
		log.WithError(err).Warn("Failed to publish identity")
	}

	// إنشاء QuotaManager
	qm := storage.NewQuotaManager()
	qm.SetLimit(kp.DID, 2*1024*1024*1024) // 2GB لـ Studio

	// إنشاء Event Bus
	eb := pkgEventbus.NewEventBus()
	log.Info("Event Bus created")

	// إنشاء BadgerDB
	db, err := badger.Open(badger.DefaultOptions(*dataDir + "/badger"))
	if err != nil {
		log.WithError(err).Fatal("Failed to open BadgerDB")
	}
	defer db.Close()
	log.Info("BadgerDB created")

	// إنشاء Agent Registry
	agentRegistry := pkgAgent.NewAgentRegistry()
	zapLogger := zap.NewNop()
	agentRegistry.SetLogger(zapLogger)
	log.Info("Agent Registry created")

	// تسجيل الوكلاء الافتراضيين
	// API Adapter
	apiConfig := &pkgAdapters.APIConfig{
		APIKey:    "sk-test",
		BaseURL:   "https://api.anthropic.com",
		Model:     "claude-3-opus",
		MaxTokens: 4096,
		Timeout:   30 * time.Second,
	}
	apiAdapter := pkgAdapters.NewAPIAdapter(apiConfig)
	agentRegistry.Register(apiAdapter, nil)

	// CLI Adapter
	cliConfig := &pkgAdapters.CLIConfig{
		Name:    "claude-code",
		Command: "claude",
		Args:    []string{},
	}
	cliAdapter := pkgAdapters.NewCLIAdapter(cliConfig)
	agentRegistry.Register(cliAdapter, nil)

	// IDE Adapter
	ideConfig := &pkgAdapters.IDEConfig{
		Name:    "cursor",
		IDEType: "cursor",
	}
	ideAdapter := pkgAdapters.NewIDEAdapter(ideConfig)
	agentRegistry.Register(ideAdapter, nil)

	// Local Adapter
	localConfig := &pkgAdapters.LocalConfig{
		Name:    "ollama",
		Model:   "llama2",
		BaseURL: "http://localhost:11434",
	}
	localAdapter := pkgAdapters.NewLocalAdapter(localConfig)
	agentRegistry.Register(localAdapter, nil)

	// Browser Adapter
	browserAdapter := pkgAdapters.NewComputerUseAdapter("sk-test")
	agentRegistry.Register(browserAdapter, nil)

	// Custom Adapter
	customAdapter := pkgAdapters.NewCustomAgent("custom", "custom", "custom-model", func(ctx context.Context, task *pkgAgent.AgentTask) (*pkgAgent.TaskExecutionResult, error) {
		return &pkgAgent.TaskExecutionResult{
			Success: true,
			Output:  "Custom agent executed task",
		}, nil
	})
	customAdapter.Initialize(map[string]interface{}{})
	agentRegistry.Register(customAdapter, nil)

	log.WithField("agent_count", agentRegistry.GetCount()).Info("Agents registered")

	// إنشاء Session Container
	sessionConfig := &pkgSession.SessionConfig{
		Name:        "Default Session",
		Description: "Default Musketeers session",
		OwnerDID:    kp.DID,
		MaxAgents:   10,
		ProjectType: "general",
	}
	sessionContainer, err := pkgSession.NewSessionContainer(ctx, db, sessionConfig, eb)
	if err != nil {
		log.WithError(err).Fatal("Failed to create session container")
	}
	log.WithField("session_id", sessionContainer.ID).Info("Session Container created")

	// [WHY] تهيئة ToolExecutor لتنفيذ الأدوات مع حدود أمان
	// [HOW] يفرض حدود على استدعاءات الأدوات وحجم الملفات والمسارات
	// [SAFETY] يمنع الحلقات اللانهائية والوصول غير المصرح به
	toolExecutor := agentTools.NewToolExecutor(*dataDir, zapLogger)
	log.Info("Tool Executor created")

	// [WHY] تهيئة CEOSupervisor لمراقبة صحة الشبكة
	// [HOW] يسجل نفسه كوكيل admin ويشغل HealthCheck دوري
	// [SAFETY] يراقب النظام وينشر تنبيهات عند المشاكل
	ceoLogger := stdlog.New(os.Stdout, "[CEO] ", stdlog.LstdFlags)
	ceoSupervisor := pkgCEO.NewCEOSupervisor(eb, agentRegistry, ceoLogger)
	if err := ceoSupervisor.Start(); err != nil {
		log.WithError(err).Fatal("Failed to start CEO supervisor")
	}
	defer ceoSupervisor.Stop()
	log.Info("CEO Supervisor started")

	// إنشاء Orchestrator components
	sessionManager := pkgOrchestrator.NewSessionManager(zapLogger)
	sessionManager.SetAgentRegistry(agentRegistry)
	sessionManager.SetEventBus(eb)
	// [WHY] ربط ToolExecutor مع SessionManager
	// [HOW] يمرر ToolExecutor لاستخدامه في تنفيذ المهام
	// [SAFETY] يضمن أن الوكلاء يستخدمون حدود الأمان
	sessionManager.SetToolExecutor(toolExecutor)

	delegationManager := pkgOrchestrator.NewDelegationManager(sessionContainer.ID, zapLogger)
	delegationManager.SetAgentRegistry(agentRegistry)
	delegationManager.SetSessionManager(sessionManager)
	delegationManager.SetEventBus(eb)

	log.Info("Orchestrator components created")

	// إنشاء Verification components
	verifier := pkgVerification.NewMultiStageVerifier()
	verifier.SetLogger(zapLogger)
	verifier.RegisterVerifier(pkgVerification.NewDefaultSyntaxVerifier())
	verifier.RegisterVerifier(pkgVerification.NewDefaultSemanticsVerifier())
	verifier.RegisterVerifier(pkgVerification.NewDefaultSecurityVerifier())
	verifier.RegisterVerifier(pkgVerification.NewDefaultPerformanceVerifier())
	verifier.RegisterVerifier(pkgVerification.NewDefaultIntegrationVerifier())
	log.Info("Verification components created")

	// إنشاء ACP Handler (يسجل تلقائياً المهام المدمجة)
	_ = acp.NewRouter()
	log.Info("ACP handlers registered")

	// إنشاء مدير الجلسات
	sessionMgr := agent_bridge.NewSessionManager(log)

	// إنشاء الجسر المتعدد
	multiplexedBrg := agent_bridge.NewMultiplexedBridge(log)

	// إنشاء Connector لربط Bridge و Event Bus و Adapters
	connector := pkgOrchestrator.NewConnector(eb, multiplexedBrg, agentRegistry, zapLogger)
	if err := connector.Start(); err != nil {
		log.WithError(err).Fatal("Failed to start connector")
	}
	defer connector.Stop()
	log.Info("Connector started")

	// إنشاء ChatConnector لربط الشات والقنوات
	// ملاحظة: ChatConnector يتطلب مفتاح ed25519.PrivateKey
	// نستخدم kp.Private للمصادقة والتشفير
	chatConnector := pkgOrchestrator.NewChatConnector(eb, agentRegistry, sessionContainer, kp.Private, zapLogger)
	if err := chatConnector.Start(); err != nil {
		log.WithError(err).Fatal("Failed to start chat connector")
	}
	defer chatConnector.Stop()
	log.Info("Chat connector started")

	// إنشاء ExternalPlatformManager لإدارة المنصات الخارجية
	// ملاحظة: ExternalPlatformManager يتطلب capability.Manager
	// [SAFETY] إنشاء policy.Engine حقيقي بدلاً من nil
	policyEngine := pkgPolicy.NewEngine()
	// إضافة قاعدة افتراضية للسماح بالعمليات الأساسية
	defaultRule := pkgPolicy.Rule{
		Name:     "default-deny",
		Priority: 0,
		Effect:   pkgPolicy.EffectDeny,
		Principals: []pkgPolicy.Principal{
			{DID: "*"},
		},
		Resources: []pkgPolicy.Resource{
			{Type: "*", Action: "*"},
		},
	}
	if err := policyEngine.AddRule(defaultRule); err != nil {
		log.WithError(err).Warn("Failed to add default policy rule")
	}
	capabilityManager := pkgCapability.NewManager(policyEngine)
	platformManager := pkgOrchestrator.NewExternalPlatformManager(eb, capabilityManager, zapLogger)
	if err := platformManager.Start(); err != nil {
		log.WithError(err).Fatal("Failed to start external platform manager")
	}
	defer platformManager.Stop()
	log.Info("External platform manager started")

	// إنشاء خادم الجسر
	bridgeServer := agent_bridge.NewServer(n, *agentAddr, sessionMgr, multiplexedBrg, log)
	if err := bridgeServer.Start(ctx); err != nil {
		log.WithError(err).Fatal("Failed to start bridge server")
	}
	defer bridgeServer.Stop()

	log.WithField("addr", *agentAddr).Info("Agent bridge server started")

	// بدء واجهة Studio
	log.WithField("addr", *addr).Info("Studio starting...")

	// بدء خادم HTTP
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Musketeers Studio is running"))
	})

	server := &http.Server{
		Addr:    *addr,
		Handler: mux,
	}

	go func() {
		log.WithField("addr", *addr).Info("HTTP server starting...")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("HTTP server failed")
		}
	}()

	// انتظار إشارة الإنهاء
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Info("Studio shutting down...")

	// إيقاف خادم HTTP
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.WithError(err).Error("HTTP server shutdown error")
	}
}

// parseBootstrap يحلل عناوين bootstrap
func parseBootstrap(bootstrap string) []string {
	if bootstrap == "" {
		return nil
	}
	return []string{bootstrap}
}
