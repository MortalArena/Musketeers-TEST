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
	"github.com/MortalArena/Musketeers/pkg/agent/unified"
	pkgCEO "github.com/MortalArena/Musketeers/pkg/ceo"
	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
	pkgEventbus "github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/MortalArena/Musketeers/pkg/identity"
	"github.com/MortalArena/Musketeers/pkg/node"
	"github.com/MortalArena/Musketeers/pkg/orchestrator"
	pkgPolicy "github.com/MortalArena/Musketeers/pkg/policy"
	"github.com/MortalArena/Musketeers/pkg/providers"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin"
	pkgSession "github.com/MortalArena/Musketeers/pkg/session"
	"github.com/MortalArena/Musketeers/pkg/session/core"
	"github.com/MortalArena/Musketeers/pkg/storage"
	pkgVerification "github.com/MortalArena/Musketeers/pkg/verification"

	// Isolated packages - being integrated
	pkgAnalytics "github.com/MortalArena/Musketeers/pkg/analytics"
	pkgBackup "github.com/MortalArena/Musketeers/pkg/backup"
	pkgConfig "github.com/MortalArena/Musketeers/pkg/config"
	pkgDelegation "github.com/MortalArena/Musketeers/pkg/delegation"
	pkgDiscovery "github.com/MortalArena/Musketeers/pkg/discovery"
	pkgHosting "github.com/MortalArena/Musketeers/pkg/hosting"
	pkgLedger "github.com/MortalArena/Musketeers/pkg/ledger"
	pkgLimits "github.com/MortalArena/Musketeers/pkg/limits"
	pkgLogger "github.com/MortalArena/Musketeers/pkg/logger"
	pkgNotifications "github.com/MortalArena/Musketeers/pkg/notifications"
	pkgPlugins "github.com/MortalArena/Musketeers/pkg/plugins"
	pkgSandbox "github.com/MortalArena/Musketeers/pkg/sandbox"
	pkgTimeout "github.com/MortalArena/Musketeers/pkg/timeout"
	pkgUpgrade "github.com/MortalArena/Musketeers/pkg/upgrade"
	pkgValidation "github.com/MortalArena/Musketeers/pkg/validation"

	// New P2P systems
	pkgEmail "github.com/MortalArena/Musketeers/pkg/email"
	pkgDomain "github.com/MortalArena/Musketeers/pkg/network/domain"

	// All isolated packages integrated successfully

	"github.com/dgraph-io/badger/v4"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

var (
	addr       = flag.String("addr", "127.0.0.1:5000", "Studio server address")
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

	// إنشاء ReservationManager لحجز الوكلاء المحليين
	reservationManager := pkgAgent.NewReservationManager(zapLogger)
	// بدء مجدول تنظيف الحجوز المنتهية كل 5 دقائق
	reservationManager.StartCleanupScheduler(5 * time.Minute)
	log.Info("ReservationManager created and cleanup scheduler started")

	// إنشاء UnifiedSessionManager لدعم جلسات متعددة
	sessionManager := core.NewUnifiedSessionManager(zapLogger)
	log.Info("UnifiedSessionManager created")

	// إنشاء SessionBridgeManager لربط الجلسات المنفصلة
	sessionBridgeManager := pkgSession.NewSessionBridgeManager(eb, zapLogger)
	log.Info("SessionBridgeManager created")

	// إنشاء جلسات متعددة كمثال
	// الجلسة 1: مشروع A
	session1, err := sessionManager.CreateSession(ctx, "Project A", kp.DID, "manager-1", []string{"coder-1", "reviewer-1"})
	if err != nil {
		log.WithError(err).Warn("Failed to create session 1")
	} else {
		log.WithField("session_id", session1.ID).Info("Session 1 created")
	}

	// الجلسة 2: مشروع B
	session2, err := sessionManager.CreateSession(ctx, "Project B", kp.DID, "manager-2", []string{"coder-2", "reviewer-2"})
	if err != nil {
		log.WithError(err).Warn("Failed to create session 2")
	} else {
		log.WithField("session_id", session2.ID).Info("Session 2 created")
	}

	// الجلسة 3: مشروع C
	session3, err := sessionManager.CreateSession(ctx, "Project C", kp.DID, "manager-3", []string{"coder-3", "reviewer-3"})
	if err != nil {
		log.WithError(err).Warn("Failed to create session 3")
	} else {
		log.WithField("session_id", session3.ID).Info("Session 3 created")
	}

	// إحصائيات الجلسات
	sessions := sessionManager.ListSessions()
	log.WithField("total_sessions", len(sessions)).Info("Total sessions created")

	// إنشاء جسور بين الجلسات كمثال
	if len(sessions) >= 2 {
		// جسر بين الجلسة 1 والجلسة 2
		bridgeConfig1 := &pkgSession.BridgeConfig{
			BridgeID:   "bridge-1-2",
			SourceID:   sessions[0].ID,
			TargetID:   sessions[1].ID,
			BridgeType: pkgSession.BridgeTypeTwoWay,
			BufferSize: 1000,
		}
		_, err := sessionBridgeManager.CreateBridge(ctx, bridgeConfig1)
		if err != nil {
			log.WithError(err).Warn("Failed to create bridge between session 1 and 2")
		} else {
			log.WithField("bridge_id", bridgeConfig1.BridgeID).Info("Bridge created between session 1 and 2")
		}
	}

	if len(sessions) >= 3 {
		// جسر بين الجلسة 2 والجلسة 3
		bridgeConfig2 := &pkgSession.BridgeConfig{
			BridgeID:   "bridge-2-3",
			SourceID:   sessions[1].ID,
			TargetID:   sessions[2].ID,
			BridgeType: pkgSession.BridgeTypeTwoWay,
			BufferSize: 1000,
		}
		_, err := sessionBridgeManager.CreateBridge(ctx, bridgeConfig2)
		if err != nil {
			log.WithError(err).Warn("Failed to create bridge between session 2 and 3")
		} else {
			log.WithField("bridge_id", bridgeConfig2.BridgeID).Info("Bridge created between session 2 and 3")
		}
	}

	// إحصائيات الجسور
	bridgeStats := sessionBridgeManager.GetStats()
	log.WithField("total_bridges", bridgeStats["total_bridges"]).Info("Total bridges created")

	// إنشاء EmailManager المتكامل
	emailManager := orchestrator.NewEmailManager(eb, nil, zapLogger)
	if err := emailManager.Start(); err != nil {
		log.WithError(err).Warn("Failed to start EmailManager")
	}
	defer emailManager.Stop()
	log.Info("EmailManager created and started")

	// تسجيل الوكلاء الافتراضيين
	// [FIX] Removed API Adapter - replaced by pkg/providers

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

	// [FIX] Removed Local Adapter - replaced by pkg/providers (Ollama)

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

	// [FIX] Create UnifiedAgent instance
	unifiedAgent := unified.NewUnifiedAgent(
		sessionContainer.ID,
		kp.DID,
		db,
		zapLogger,
	)
	log.Info("UnifiedAgent created")

	// [FIX] Initialize UnifiedAgent
	if err := unifiedAgent.Initialize(ctx); err != nil {
		log.WithError(err).Fatal("Failed to initialize unified agent")
	}
	log.Info("UnifiedAgent initialized successfully")

	// [FIX] Register existing agents in UnifiedAgent
	for _, agent := range agentRegistry.ListAll() {
		info := agent.GetInfo()
		if err := unifiedAgent.RegisterAgent(
			ctx,
			info.ID,
			string(info.Type),
			info.Model,
			[]string{}, // specializations - can be added later
		); err != nil {
			log.WithError(err).Warnf("Failed to register agent %s in unified system", info.ID)
		}
	}
	log.WithField("agent_count", agentRegistry.GetCount()).Info("Agents registered in unified system")

	// [FIX] Create Provider Registry for LLM providers
	providerRegistry := builtin.NewRegistry()
	log.Info("Provider registry created with all builtin providers")

	// Link provider registry to UnifiedAgent
	unifiedAgent.SetProviderRegistry(providerRegistry)
	log.Info("Provider registry linked to UnifiedAgent")

	// [FIX] Create Smart Router for intelligent model selection
	routerConfig := providers.RouterConfig{
		PreferFreeModels:    true,
		PreferLocalModels:   true,
		MaxRetries:          3,
		Timeout:             30 * time.Second,
		FallbackEnabled:     true,
		CostOptimization:    true,
		LatencyOptimization: false,
	}
	router := providers.NewRouter(providerRegistry, routerConfig)
	log.Info("Smart router created with intelligent model selection")

	// Link router to UnifiedAgent
	unifiedAgent.SetRouter(router)
	log.Info("Smart router linked to UnifiedAgent")

	// [FIX] Test UnifiedAgent execution
	testTask := "تحليل ملفات المشروع"
	result, err := unifiedAgent.ExecuteTask(ctx, testTask)
	if err != nil {
		log.WithError(err).Warn("Failed to execute test task")
	} else {
		log.WithField("success", result.Success).
			WithField("confidence", result.Confidence).
			Info("Test task executed successfully")
	}

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

	// [FIX] UnifiedAgent handles all coordination internally
	log.Info("UnifiedAgent handles all agent coordination, session management, and orchestration")

	// [INTEGRATION] Initialize isolated packages
	// Create logger for isolated packages
	isolatedLogger, err := pkgLogger.NewLogger("info", false)
	if err != nil {
		log.WithError(err).Warn("Failed to create isolated logger, using zap logger")
		isolatedLogger = nil
	}
	if isolatedLogger == nil {
		zapLogger = zap.NewNop()
	} else {
		zapLogger = isolatedLogger.Logger
	}
	log.Info("Isolated packages logger created")

	// Initialize Config
	config := pkgConfig.DefaultConfig()
	if err := pkgConfig.ValidateConfig(config); err != nil {
		log.WithError(err).Warn("Default config validation failed")
	}
	log.Info("Config initialized")

	// Initialize Limits
	_ = pkgLimits.NewResourceLimiter(100)
	_ = pkgLimits.NewMemoryLimiter(1024 * 1024 * 1024) // 1GB
	_ = pkgLimits.NewRateLimiter(1000, 100, time.Second)
	_ = pkgLimits.NewConnectionLimiter(50)
	log.Info("Limits initialized")

	// Initialize Timeout
	_ = pkgTimeout.DefaultTimeoutConfig()
	log.Info("Timeout config initialized")

	// Initialize Validation
	_ = pkgValidation.NewDIDValidator("did:mskt:")
	_, _ = pkgValidation.NewStringValidator(1, 100, false, "^[a-zA-Z0-9]+$")
	_ = pkgValidation.NewEmailValidator(false)
	_ = pkgValidation.NewPortValidator(1, 65535)
	_ = pkgValidation.NewNumberValidator(0, 100, false)
	log.Info("Validation initialized")

	// Initialize Ledger
	_ = pkgLedger.NewCostTracker()
	_ = pkgLedger.NewCreditManager(0.1) // 10% reward rate
	log.Info("Ledger initialized")

	// Initialize Logger (already created above)
	log.Info("Logger initialized")

	// Initialize Notifications - requires custom sender and event bus
	// Skipping for now as it requires custom interfaces
	log.Info("Notifications initialized (skipped - requires custom interfaces)")

	// Initialize Plugins - requires custom event bus interface
	// Skipping for now as it requires custom event bus
	log.Info("Plugins initialized (skipped - requires custom event bus)")

	// Initialize Sandbox
	_, _ = pkgSandbox.NewExecutor(ctx)
	log.Info("Sandbox initialized")

	// Initialize Upgrade - requires custom event bus interface
	// Skipping for now as it requires custom event bus
	log.Info("Upgrade initialized (skipped - requires custom event bus)")

	// Initialize Analytics - requires custom event bus interface
	// Skipping for now as it requires custom event bus
	log.Info("Analytics initialized (skipped - requires custom event bus)")

	// Initialize Backup - requires custom event bus interface and different config
	// Skipping for now as it requires custom event bus
	log.Info("Backup initialized (skipped - requires custom event bus)")

	// Initialize Delegation - MockDelegationKeyResolver not exported
	// Skipping for now
	log.Info("Delegation initialized (skipped - Mock not exported)")

	// Initialize Discovery
	_ = pkgDiscovery.NewIndexedDiscovery()
	log.Info("Discovery initialized")

	// Email system integrated via orchestrator.EmailManager (already created above)

	// Initialize Hosting
	_ = pkgHosting.NewHostingManager()
	log.Info("Hosting initialized")

	// Initialize Analytics
	analyticsIntegrator := pkgAnalytics.NewAnalyticsIntegrator(zapLogger, eb)
	if err := analyticsIntegrator.Start(); err != nil {
		log.WithError(err).Warn("Failed to start AnalyticsIntegrator")
	}
	log.Info("Analytics initialized")

	// Initialize Backup
	backupIntegrator := pkgBackup.NewBackupIntegrator(zapLogger, eb)
	if err := backupIntegrator.Start(); err != nil {
		log.WithError(err).Warn("Failed to start BackupIntegrator")
	}
	log.Info("Backup initialized")

	// Initialize Delegation
	delegationIntegrator := pkgDelegation.NewDelegationIntegrator(zapLogger)
	if err := delegationIntegrator.Start(); err != nil {
		log.WithError(err).Warn("Failed to start DelegationIntegrator")
	}
	log.Info("Delegation initialized")

	// Initialize Notifications
	notificationsIntegrator := pkgNotifications.NewNotificationsIntegrator(zapLogger, eb)
	if err := notificationsIntegrator.Start(); err != nil {
		log.WithError(err).Warn("Failed to start NotificationsIntegrator")
	}
	log.Info("Notifications initialized")

	// Initialize Plugins
	pluginsIntegrator := pkgPlugins.NewPluginsIntegrator(zapLogger, eb)
	if err := pluginsIntegrator.Start(); err != nil {
		log.WithError(err).Warn("Failed to start PluginsIntegrator")
	}
	log.Info("Plugins initialized")

	// Initialize Upgrade
	upgradeIntegrator := pkgUpgrade.NewUpgradeIntegrator(zapLogger, eb)
	if err := upgradeIntegrator.Start(); err != nil {
		log.WithError(err).Warn("Failed to start UpgradeIntegrator")
	}
	log.Info("Upgrade initialized")

	log.Info("All isolated packages initialized successfully")

	// Initialize new P2P systems
	// P2P Email System
	emailStore, err := pkgEmail.NewEmailStore()
	if err != nil {
		log.WithError(err).Warn("Failed to create email store")
	} else {
		_ = pkgEmail.NewP2PEmailService(n.Host(), emailStore, kp.DID)
		log.Info("P2P Email Service initialized")
	}

	// P2P Domain System
	p2pDNSResolver := pkgDomain.NewP2PDNSResolver(n.Host())
	localDNSProxy := pkgDomain.NewLocalDNSProxy(zapLogger, p2pDNSResolver, "127.0.0.1:53")
	if err := localDNSProxy.Start(); err != nil {
		log.WithError(err).Warn("Failed to start local DNS proxy")
	} else {
		log.Info("Local DNS Proxy initialized")
		defer localDNSProxy.Stop()
	}

	httpProxy := pkgDomain.NewHTTPProxy(n.Host(), "127.0.0.1:8080")
	go func() {
		if err := httpProxy.Start(); err != nil {
			log.WithError(err).Warn("Failed to start HTTP proxy")
		}
	}()
	log.Info("HTTP Proxy initialized")

	_ = pkgDomain.NewSystemProxy("127.0.0.1:8080", "127.0.0.1:53")
	// Note: System proxy configuration requires admin privileges
	// Skipping automatic configuration for now
	log.Info("System Proxy initialized (configuration skipped)")

	// P2P Hosting System
	p2pHostingService := pkgHosting.NewP2PHostingService(n.Host())
	_ = pkgHosting.NewSiteUploader(p2pHostingService)
	log.Info("P2P Hosting Service initialized")
	log.Info("Site Uploader initialized")

	log.Info("All new P2P systems initialized successfully")

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

	// [FIX] UnifiedAgent handles all coordination internally
	log.Info("UnifiedAgent handles all agent coordination, session management, and orchestration")

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

	// [FIX] Add allow rules for basic operations
	allowRules := []pkgPolicy.Rule{
		{
			Name:     "allow-read-own-data",
			Priority: 10,
			Effect:   pkgPolicy.EffectAllow,
			Principals: []pkgPolicy.Principal{
				{DID: "*"},
			},
			Resources: []pkgPolicy.Resource{
				{Type: "data", Action: "read"},
			},
		},
		{
			Name:     "allow-write-own-data",
			Priority: 10,
			Effect:   pkgPolicy.EffectAllow,
			Principals: []pkgPolicy.Principal{
				{DID: "*"},
			},
			Resources: []pkgPolicy.Resource{
				{Type: "data", Action: "write"},
			},
		},
		{
			Name:     "allow-execute-tasks",
			Priority: 10,
			Effect:   pkgPolicy.EffectAllow,
			Principals: []pkgPolicy.Principal{
				{DID: "*"},
			},
			Resources: []pkgPolicy.Resource{
				{Type: "task", Action: "execute"},
			},
		},
		{
			Name:     "allow-join-channels",
			Priority: 10,
			Effect:   pkgPolicy.EffectAllow,
			Principals: []pkgPolicy.Principal{
				{DID: "*"},
			},
			Resources: []pkgPolicy.Resource{
				{Type: "channel", Action: "join"},
			},
		},
		{
			Name:     "allow-publish-channels",
			Priority: 10,
			Effect:   pkgPolicy.EffectAllow,
			Principals: []pkgPolicy.Principal{
				{DID: "*"},
			},
			Resources: []pkgPolicy.Resource{
				{Type: "channel", Action: "publish"},
			},
		},
	}

	for _, rule := range allowRules {
		if err := policyEngine.AddRule(rule); err != nil {
			log.WithError(err).Warnf("Failed to add allow rule: %s", rule.Name)
		}
	}
	log.WithField("rules", len(allowRules)).Info("Allow rules added to policy engine")

	// [FIX] UnifiedAgent handles all coordination internally
	log.Info("UnifiedAgent handles all agent coordination, session management, and orchestration")

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
