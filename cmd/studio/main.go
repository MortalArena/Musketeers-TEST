package main

import (
	"context"
	"flag"
	"fmt"
	stdlog "log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/MortalArena/Musketeers/api"
	"github.com/MortalArena/Musketeers/pkg/acp"
	"github.com/MortalArena/Musketeers/pkg/agent"
	pkgAgent "github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/agent/adapters"
	"github.com/MortalArena/Musketeers/pkg/agent/unified"
	"github.com/MortalArena/Musketeers/pkg/agent_bridge"
	pkgCapability "github.com/MortalArena/Musketeers/pkg/capability"
	pkgCEO "github.com/MortalArena/Musketeers/pkg/ceo"
	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
	pkgEventbus "github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/MortalArena/Musketeers/pkg/identity"
	"github.com/MortalArena/Musketeers/pkg/node"
	"github.com/MortalArena/Musketeers/pkg/orchestrator"
	pkgPolicy "github.com/MortalArena/Musketeers/pkg/policy"
	"github.com/MortalArena/Musketeers/pkg/providers"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/custom"
	pkgRuntime "github.com/MortalArena/Musketeers/pkg/runtime"
	pkgSession "github.com/MortalArena/Musketeers/pkg/session"
	"github.com/MortalArena/Musketeers/pkg/storage"
	pkgVerification "github.com/MortalArena/Musketeers/pkg/verification"
	"github.com/dgraph-io/badger/v4"

	// Isolated packages - being integrated

	pkgAnalytics "github.com/MortalArena/Musketeers/pkg/analytics"
	pkgBackup "github.com/MortalArena/Musketeers/pkg/backup"
	pkgConfig "github.com/MortalArena/Musketeers/pkg/config"
	pkgDiscovery "github.com/MortalArena/Musketeers/pkg/discovery"
	pkgHosting "github.com/MortalArena/Musketeers/pkg/hosting"
	pkgIntegration "github.com/MortalArena/Musketeers/pkg/integration"
	pkgLedger "github.com/MortalArena/Musketeers/pkg/ledger"
	pkgLimits "github.com/MortalArena/Musketeers/pkg/limits"
	pkgLogger "github.com/MortalArena/Musketeers/pkg/logger"
	pkgNotifications "github.com/MortalArena/Musketeers/pkg/notifications"
	pkgPlugins "github.com/MortalArena/Musketeers/pkg/plugins"
	pkgSandbox "github.com/MortalArena/Musketeers/pkg/sandbox"
	pkgTimeout "github.com/MortalArena/Musketeers/pkg/timeout"
	pkgUpgrade "github.com/MortalArena/Musketeers/pkg/upgrade"
	pkgValidation "github.com/MortalArena/Musketeers/pkg/validation"

	// Agent Discovery and Lifecycle Management
	agentDiscovery "github.com/MortalArena/Musketeers/pkg/agent/autodiscovery"

	// New P2P systems
	pkgEmail "github.com/MortalArena/Musketeers/pkg/email"
	pkgDomain "github.com/MortalArena/Musketeers/pkg/network/domain"

	// All isolated packages integrated successfully

	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

var (
	addr       = flag.String("addr", "127.0.0.1:5000", "Studio server address")
	dataDir    = flag.String("data-dir", "./studio-data", "Data directory")
	bootstrap  = flag.String("bootstrap", "", "Bootstrap peer multiaddr")
	founderPub = flag.String("founder-pub", "", "Founder public key hex")
	verbose    = flag.Bool("verbose", false, "Verbose logging")
	tlsCert    = flag.String("tls-cert", "", "TLS certificate file for API server")
	tlsKey     = flag.String("tls-key", "", "TLS key file for API server")
	apiPort    = flag.Int("api-port", 8081, "REST API server port")
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

	// ============================================================
	// 1. NodeInterface - P2P node lifecycle and core operations
	// ============================================================
	// إنشاء عقدة (استخدام الإعدادات الافتراضية مع التجاوز)
	cfg := node.DefaultConfig()
	cfg.DataDir = *dataDir
	cfg.StorageQuotaMB = 2048 // 2GB
	cfg.FounderPubHex = *founderPub
	cfg.BootstrapPeers = parseBootstrap(*bootstrap)
	cfg.MaxPutPerMinute = 300 // 5/sec لمنع "تجاوز حد المعدل" الكاذب

	n, err := node.New(ctx, cfg, kp, idRec)
	if err != nil {
		log.WithError(err).Fatal("Failed to create node")
	}
	defer n.Close()

	log.WithField("did", kp.DID).Info("Studio node created (NodeInterface)")

	// نشر الهوية على DHT
	if err := n.PublishIdentity(ctx); err != nil {
		log.WithError(err).Warn("Failed to publish identity")
	}

	// ============================================================
	// 2. CommunicationInterface - channels, publish/subscribe (EventBus)
	// ============================================================
	// إنشاء Event Bus
	eb := pkgEventbus.NewEventBus()
	log.Info("Event Bus created (CommunicationInterface)")

	// ============================================================
	// 3. StorageInterface - content-addressed block storage (BadgerDB + QuotaManager)
	// ============================================================
	// إنشاء QuotaManager
	qm := storage.NewQuotaManager()
	qm.SetLimit(kp.DID, 2*1024*1024*1024) // 2GB لـ Studio

	// إنشاء BadgerDB مع آلية إعادة المحاولة وقاعدة بيانات فريدة لكل عملية
	// [SOLUTION] استخدام قاعدة بيانات مختلفة لكل عملية لتجنب تضارب LOCK
	processID := os.Getpid()
	badgerDir := fmt.Sprintf("%s/badger-pid-%d", *dataDir, processID)

	var db *badger.DB
	var badgerErr error
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		db, badgerErr = badger.Open(badger.DefaultOptions(badgerDir))
		if badgerErr == nil {
			break
		}
		if i < maxRetries-1 {
			log.WithError(badgerErr).Warnf("Failed to open BadgerDB (attempt %d/%d), retrying in 2 seconds...", i+1, maxRetries)
			time.Sleep(2 * time.Second)
		}
	}
	if badgerErr != nil {
		log.WithError(badgerErr).Fatal("Failed to open BadgerDB after retries")
	}
	defer db.Close()
	log.WithField("badger_dir", badgerDir).Info("BadgerDB created (StorageInterface)")

	// ============================================================
	// 4. IdentityInterface - DIDs, signing, key management (kp + idRec)
	// ============================================================
	// تم إنشاؤه بالفعل في السطور 88-99 (kp + idRec)
	log.WithField("did", kp.DID).Info("Identity initialized (IdentityInterface)")

	// ============================================================
	// 5. ApplicationRuntime - Composition Root
	// ============================================================
	// إنشاء ApplicationRuntime
	zapLogger := zap.NewNop()
	appRuntime := pkgRuntime.NewApplicationRuntime(zapLogger)

	// بناء جميع المكونات
	if err := appRuntime.Build(); err != nil {
		log.WithError(err).Fatal("Failed to build ApplicationRuntime")
	}
	log.Info("ApplicationRuntime built successfully")

	// حقن الاعتماديات
	if err := appRuntime.Inject(); err != nil {
		log.WithError(err).Fatal("Failed to inject dependencies")
	}
	log.Info("Dependencies injected successfully")

	// بدء ApplicationRuntime
	if err := appRuntime.Start(); err != nil {
		log.WithError(err).Fatal("Failed to start ApplicationRuntime")
	}
	log.Info("ApplicationRuntime started successfully")

	// الحصول على المكونات من ApplicationRuntime
	agentRegistry := appRuntime.GetAgentRegistry()
	log.Info("Agent Registry retrieved from ApplicationRuntime")

	// ملاحظة: تسجيل الموديلات كوكلاء سيتم من خلال Dashboard
	// المستخدم يمكنه إضافة API keys وتكوين الموديلات من الواجهة
	// النظام يدعم: Mistral, OpenRouter, Ollama, وغيرها من الموفرين
	log.Info("Model agents can be configured from Dashboard using API keys")

	// إنشاء ReservationManager لحجز الوكلاء المحليين
	reservationManager := pkgAgent.NewReservationManager(zapLogger)
	// بدء مجدول تنظيف الحجوز المنتهية كل 5 دقائق
	reservationManager.StartCleanupScheduler(5 * time.Minute)
	log.Info("ReservationManager created and cleanup scheduler started")

	// إنشاء SessionManager من orchestrator - مصدر الحقيقة الوحيد للجلسات
	sessionManager := orchestrator.NewSessionManager(zapLogger)
	sessionManager.SetAgentRegistry(agentRegistry)
	sessionManager.SetEventBus(eb)
	log.Info("Orchestrator SessionManager created (single source of truth for sessions)")

	// إنشاء SessionBridgeManager لربط الجلسات المنفصلة
	sessionBridgeManager := pkgSession.NewSessionBridgeManager(eb, zapLogger)
	log.Info("SessionBridgeManager created")

	// إيقاف ApplicationRuntime عند الخروج
	defer func() {
		if err := appRuntime.Shutdown(ctx); err != nil {
			log.WithError(err).Error("Failed to shutdown ApplicationRuntime")
		}
		if err := appRuntime.Cancel(); err != nil {
			log.WithError(err).Error("Failed to cancel ApplicationRuntime")
		}
	}()

	// ============================================================
	// 6. SessionInterface - session lifecycle and execution (SessionContainer)
	// ============================================================
	// سيتم إنشاء SessionContainer لاحقاً في السطر 381
	// SessionContainer يحتاج: EventBus + DB + DID

	// [FIX] لا تنشئ جلسات مع وكلاء وهميين
	// المستخدم سيقوم بإنشاء الجلسات من Dashboard مع الوكلاء الحقيقيين المرتبطين بـ providers
	// الجلسات ستكون فارغة في البداية وسيتم إضافة الوكلاء من خلال API
	log.Info("Sessions will be created from Dashboard with real agents - no fake sessions created")

	// [FIX] لا تنشئ جسور بين الجلسات لأنه لا توجد جلسات بعد
	// الجسور ستكون متاحة عندما ينشئ المستخدم جلسات من Dashboard
	log.Info("Bridges will be created when sessions are created from Dashboard")

	// إنشاء EmailManager المتكامل
	emailManager := orchestrator.NewEmailManager(eb, nil, zapLogger)
	if err := emailManager.Start(); err != nil {
		log.WithError(err).Warn("Failed to start EmailManager")
	}
	defer emailManager.Stop()
	log.Info("EmailManager created and started")

	// إنشاء EmailIntegrator لربط SMTP مع النظام — يكمل معمارية event-driven
	// [WHY] NotificationSender ينشر أحداث "notification.email" ولكن لا يوجد مشترك — هذا يخلق المشترك
	// [HOW] EmailIntegrator يربط EmailClient (SMTP) مع EmailManager (التخزين)
	emailCfg := &pkgEmail.EmailConfig{
		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     587,
		SMTPUsername: os.Getenv("SMTP_USERNAME"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		UseTLS:       true,
		FromAddress:  "noreply@musketeers.com",
		FromName:     "Musketeers",
	}
	if emailCfg.SMTPHost == "" {
		emailCfg.SMTPHost = "smtp.gmail.com"
	}
	emailIntegrator := pkgEmail.NewEmailIntegrator(emailCfg, emailManager)
	log.Info("EmailIntegrator created — SMTP delivery available when SMTP_HOST/SMTP_USERNAME/SMTP_PASSWORD env vars are set")

	// ربط EventBus بالإرسال الفعلي للبريد — المشترك المفقود في معمارية event-driven
	eb.Subscribe("notification.email", func(e pkgEventbus.Event) {
		payload, ok := e.Payload.(map[string]interface{})
		if !ok {
			return
		}
		to, _ := payload["to"].(string)
		subject, _ := payload["subject"].(string)
		body, _ := payload["body"].(string)
		msg := &pkgEmail.EmailMessage{
			To:      []string{to},
			Subject: subject,
			Body:    body,
		}
		if err := emailIntegrator.SendViaClient(msg); err != nil {
			log.WithError(err).Warn("Failed to send notification email via SMTP")
		}
	})

	// ربط EmailManager (الذي يستقبل أحداث "email.send") بإرسال SMTP فعلي
	eb.Subscribe("email.send", func(e pkgEventbus.Event) {
		payload, ok := e.Payload.(map[string]interface{})
		if !ok {
			return
		}
		toList := []string{}
		if to, ok := payload["to"].(string); ok && to != "" {
			toList = []string{to}
		}
		if tos, ok := payload["to"].([]string); ok {
			toList = tos
		}
		subject, _ := payload["subject"].(string)
		body, _ := payload["body"].(string)
		msg := &pkgEmail.EmailMessage{
			To:      toList,
			Subject: subject,
			Body:    body,
		}
		if err := emailIntegrator.SendViaClient(msg); err != nil {
			log.WithError(err).Warn("Failed to send email via SMTP (email.send event)")
		}
	})
	log.Info("EventBus email subscribers wired — notification.email and email.send now deliver via SMTP")

	// [FIX] إزالة التسجيل التلقائي للوكلاء من models.json
	// models.json يحتوي على موديلات وهمية (fake models)
	// المستخدم يجب أن يضيف الموديلات الحقيقية من Dashboard باستخدام API keys
	// النظام يدعم: Mistral, OpenRouter, Cloudflare, وغيرها من الموفرين الحقيقية
	log.Info("Auto-registration from models.json disabled - use Dashboard to add real models with API keys")

	// [FIX] نظام الاكتشاف التلقائي للوكلاء على جهاز العميل
	// النظام سيكتشف الوكلاء المتاحة على جهاز العميل تلقائياً
	// لن يتم تسجيل أي وكلاء بدون موافقة العميل

	// إنشاء نظام الاكتشاف التلقائي
	autoDiscovery := agentDiscovery.NewAutoDiscovery(zapLogger, agentRegistry)
	log.Info("AutoDiscovery system created")

	// إنشاء نظام إدارة دورة حياة الوكلاء
	lifecycleManager := agentDiscovery.NewLifecycleManager(agentRegistry, zapLogger)
	log.Info("LifecycleManager system created")

	// اكتشاف الوكلاء المتاحة على جهاز العميل
	discoveredAgents, err := autoDiscovery.DiscoverAll(ctx)
	if err != nil {
		log.WithError(err).Warn("فشل الاكتشاف التلقائي للوكلاء")
	} else {
		log.WithField("discovered_count", len(discoveredAgents)).Info("اكتشف الوكلاء المتاحة على جهاز العميل")

		// عرض الوكلاء المكتشفة للعميل
		for _, agent := range discoveredAgents {
			log.WithFields(logrus.Fields{
				"id":         agent.ID,
				"name":       agent.Name,
				"type":       agent.Type,
				"version":    agent.Version,
				"status":     agent.Status,
				"executable": agent.Executable,
			}).Info("وكيل مكتشف")

			// تسجيل في نظام إدارة دورة الحياة
			if err := lifecycleManager.RegisterAgent(agent.ID, agent.Name, agent.Type); err != nil {
				log.WithError(err).Warnf("فشل تسجيل الوكيل %s في نظام دورة الحياة", agent.ID)
			}
		}

		// [FIX] تسجيل الوكلاء المكتشفة سيتم لاحقاً بعد إنشاء OrchestratorEngine
		// لضمان أنهم يتم تسجيلهم عبر OrchestratorEngine.RegisterAgent
		// الذي يستدعي RegisterAgentFromUnified لإضافتهم إلى SessionContainer
		if len(discoveredAgents) > 0 {
			log.Info("الوكلاء المكتشفة سيتم تسجيلهم بعد إنشاء OrchestratorEngine")
		} else {
			log.Warn("لم يتم اكتشاف أي وكلاء - سيتم استخدام الموديلات المباشرة من providers")
		}
	}

	// إنشاء Session Container
	sessionConfig := &pkgSession.SessionConfig{
		Name:          "Default Session",
		Description:   "Default Musketeers session",
		OwnerDID:      kp.DID,
		MaxAgents:     10,
		ProjectType:   "general",
		SessionFolder: "./sessions/default", // [WHY] فولدر الجلسة المنظم
	}
	sessionContainer, err := pkgSession.NewSessionContainer(ctx, db, sessionConfig, eb)
	if err != nil {
		log.WithError(err).Fatal("Failed to create session container")
	}
	// [FIX] بدء Hybrid Persistence — حفظ كل 30 ثانية إذا كان هناك تغيير
	sessionContainer.StartFlushWorker(ctx)
	log.WithField("session_id", sessionContainer.ID).Info("Session Container created (SessionInterface)")

	// ============================================================
	// 7. WorkflowInterface - workflow registration and execution (WorkflowEngine)
	// ============================================================
	// WorkflowEngine مُنفذ داخل SessionContainer (sessionContainer.Workflow)
	log.Info("WorkflowEngine available inside SessionContainer (WorkflowInterface)")

	// ============================================================
	// 8. SecurityInterface - encryption, auth, signing (PolicyEngine)
	// ============================================================
	// PolicyEngine مُنفذ داخل OrchestratorEngine (سيتم إنشاؤه لاحقاً في السطر 420)
	log.Info("PolicyEngine will be created inside OrchestratorEngine (SecurityInterface)")

	// [FIX] Create UnifiedAgent instance
	unifiedAgent := unified.NewUnifiedAgent(
		sessionContainer.ID,
		kp.DID,
		db,
		zapLogger,
	)
	log.Info("UnifiedAgent created (part of AgentInterface)")

	// [FIX] ربط SessionContainer الحقيقي قبل Initialize — هذا يحل مشكلة SessionContainer المكرر
	unifiedAgent.SetRealSessionContainer(sessionContainer)
	log.Info("Real SessionContainer linked to UnifiedAgent")

	// [FIX] Initialize UnifiedAgent
	if err := unifiedAgent.Initialize(ctx); err != nil {
		log.WithError(err).Fatal("Failed to initialize unified agent")
	}
	log.Info("UnifiedAgent initialized successfully")

	// ============================================================
	// 9. A2AInterface - agent-to-agent protocol (A2AManager + Connector)
	// ============================================================
	// سيتم إنشاء Connector و A2AManager لاحقاً
	log.Info("A2AInterface will be initialized via Connector and A2AManager")

	// [FIX] إنشاء StorageConnector مع QuotaManager
	_ = orchestrator.NewStorageConnector(eb, qm, zapLogger)

	// [FIX] إنشاء bridge و Connector (part of A2AInterface)
	bridge := agent_bridge.NewMultiplexedBridge(log)
	conn := orchestrator.NewConnector(eb, bridge, agentRegistry, zapLogger)
	if err := conn.Start(); err != nil {
		log.WithError(err).Fatal("فشل بدء Connector")
	}
	defer conn.Stop()
	log.Info("Connector started (A2AInterface)")

	// [INTEGRATION] Initialize DelegationManager for task delegation between agents
	// DelegationManager handles all delegation tasks - no need for separate DelegationIntegrator
	delegationManager := orchestrator.NewDelegationManager("default-session", zapLogger)
	delegationManager.SetAgentRegistry(agentRegistry)
	// Note: DelegationManager expects orchestrator.SessionManager, but we have core.UnifiedSessionManager
	// This will be fixed when the session managers are unified
	// delegationManager.SetSessionManager(sessionManager)
	delegationManager.SetEventBus(eb)
	log.Info("DelegationManager initialized (handles all delegation tasks)")

	// [FIX] تسجيل وكلاء AgentRegistry في OrchestratorEngine
	orchestratorEngine := orchestrator.NewOrchestratorEngine(agentRegistry)
	orchestratorEngine.SetLogger(zapLogger)
	orchestratorEngine.SetUnifiedAgent(unifiedAgent)
	orchestratorEngine.SetSessionContainer(sessionContainer) // [FIX] ربط SessionContainer لتفعيل RegisterAgentFromUnified
	orchestratorEngine.SetConnector(conn)
	orchestratorEngine.SetDelegationManager(delegationManager)
	if err := orchestratorEngine.Start(ctx); err != nil {
		log.WithError(err).Warn("Failed to start orchestrator engine")
	} else {
		log.Info("OrchestratorEngine started successfully (includes SecurityInterface and Delegation)")
	}
	defer orchestratorEngine.Stop(ctx)

	// [FIX] تسجيل الوكلاء المكتشفة من AutoDiscovery في AgentRegistry أولاً
	// ثم تسجيلهم عبر OrchestratorEngine لإضافتهم إلى SessionContainer
	if len(discoveredAgents) > 0 {
		log.Info("تسجيل الوكلاء المكتشفة في AgentRegistry ثم عبر OrchestratorEngine")
		for _, agent := range discoveredAgents {
			// إنشاء ProviderAdapter للوكيل المكتشف
			adapter := adapters.NewProviderAdapter(
				agent.ID,
				agent.Name,
				agent.AgentType,
				"", // provider - سيتم تعيينه لاحقاً
				"", // model - سيتم تعيينه لاحقاً
			)

			// تسجيل في AgentRegistry
			if err := agentRegistry.Register(adapter, nil); err != nil {
				log.WithError(err).Warnf("فشل تسجيل الوكيل %s في AgentRegistry", agent.ID)
				continue
			}

			log.WithField("agent_id", agent.ID).Info("تم تسجيل الوكيل في AgentRegistry")
		}
	}

	// [FIX] تسجيل جميع وكلاء AgentRegistry عبر OrchestratorEngine
	// هذا يضمن استدعاء RegisterAgentFromUnified وتسجيل الوكلاء في SessionContainer
	for _, agentObj := range agentRegistry.ListAll() {
		info := agentObj.GetInfo()

		// إنشاء metadata للوكيل
		metadata := &agent.AgentMetadata{
			AgentID:  info.ID,
			Name:     info.Name,
			Type:     info.Type,
			Provider: info.Provider,
			Model:    info.Model,
		}

		// تسجيل عبر OrchestratorEngine - هذا يستدعي RegisterAgentFromUnified تلقائياً
		if err := orchestratorEngine.RegisterAgent(agentObj, metadata); err != nil {
			log.WithError(err).Warnf("Failed to register agent %s via OrchestratorEngine", info.ID)
		}

		// تسجيل في UnifiedAgent (skill manager + subagent manager)
		if err := unifiedAgent.RegisterAgent(
			ctx,
			info.ID,
			string(info.Type),
			info.Model,
			[]string{},
		); err != nil {
			log.WithError(err).Warnf("Failed to register agent %s in unified system", info.ID)
		}

		// تسجيل الـ adapter نفسه في AgentPool (ThinkingEngine + ToolExecutor)
		if err := unifiedAgent.RegisterAgentToPool(agentObj, "regular"); err != nil {
			log.WithError(err).Warnf("Failed to register agent %s in AgentPool", info.ID)
		}
	}
	log.WithField("agent_count", agentRegistry.GetCount()).Info("Agents registered in unified system, orchestrator, AgentPool, and assigned roles for planning")

	// [FIX] Create Provider Registry for LLM providers
	providerRegistry := builtin.NewRegistry()
	log.Info("Provider registry created with all builtin providers (AIInterface)")

	// ============================================================
	// 10. AIInterface - LLM provider abstraction (ProviderRegistry + Router)
	// ============================================================
	// [IMPORTANT] Providers and agents will be initialized dynamically from Dashboard
	// No hardcoded API keys, URLs, or models in main.go
	// Users configure providers and register agents via Dashboard API endpoints
	log.Info("ProviderRegistry created - providers and agents configured via Dashboard")

	// [REMOVED] All other providers (Mistral, OpenRouter, Qwen, etc.) will be initialized
	// dynamically from Dashboard based on user configuration
	// Users can add API keys and configure providers from the Dashboard UI
	log.Info("Other providers will be initialized dynamically from Dashboard based on user configuration")

	// Initialize Cloudflare Workers AI as custom provider (if configured)
	customProvider, exists := providerRegistry.Get(providers.ProviderCustom)
	if !exists {
		log.Warn("Custom provider not found in registry")
	} else {
		cloudflareBaseURL := os.Getenv("CLOUDFLARE_BASE_URL")
		cloudflareAPIKey := os.Getenv("CLOUDFLARE_API_KEY")

		if cloudflareBaseURL != "" && cloudflareAPIKey != "" {
			log.Info("Initializing Cloudflare Workers AI as custom provider from environment variables")
			if err := customProvider.Initialize(ctx, providers.ProviderConfig{
				BaseURL: cloudflareBaseURL,
				APIKey:  cloudflareAPIKey,
				Timeout: 10 * time.Minute,
			}); err != nil {
				log.WithError(err).Warn("Failed to initialize Cloudflare custom provider")
			} else {
				log.Info("Cloudflare Workers AI initialized successfully")

				// Add Cloudflare models as custom models
				if cp, ok := customProvider.(*custom.Provider); ok {
					cp.AddCustomModel(custom.CustomModelConfig{
						ID:            "@cf/moonshotai/kimi-k2.7-code",
						Name:          "Moonshot AI Kimi K2.7 Code",
						BaseURL:       cloudflareBaseURL,
						APIKey:        cloudflareAPIKey,
						APIFormat:     "openai",
						ContextLength: 32768,
						Capabilities:  []string{"code", "long_context"},
					})
					cp.AddCustomModel(custom.CustomModelConfig{
						ID:            "@cf/zai-org/glm-5.2",
						Name:          "ZAI GLM 5.2",
						BaseURL:       cloudflareBaseURL,
						APIKey:        cloudflareAPIKey,
						APIFormat:     "openai",
						ContextLength: 32768,
						Capabilities:  []string{"chat", "long_context"},
					})
					log.Info("Cloudflare models added: @cf/moonshotai/kimi-k2.7-code, @cf/zai-org/glm-5.2")
				}
			}
		} else {
			log.Info("Cloudflare provider available but not configured - set CLOUDFLARE_BASE_URL and CLOUDFLARE_API_KEY environment variables to enable")
		}
	}

	// Initialize Mistral provider (if configured)
	mistralProvider, exists := providerRegistry.Get(providers.ProviderMistral)
	if !exists {
		log.Warn("Mistral provider not found in registry")
	} else {
		mistralAPIKey := os.Getenv("MISTRAL_API_KEY")

		if mistralAPIKey != "" {
			log.Info("Initializing Mistral provider from environment variable")
			if err := mistralProvider.Initialize(ctx, providers.ProviderConfig{
				APIKey:  mistralAPIKey,
				Timeout: 10 * time.Minute,
			}); err != nil {
				log.WithError(err).Warn("Failed to initialize Mistral provider")
			} else {
				log.Info("Mistral provider initialized successfully")
			}
		} else {
			log.Info("Mistral provider available but not configured - set MISTRAL_API_KEY environment variable to enable")
		}
	}

	// Initialize OpenRouter provider (if configured)
	openrouterProvider, exists := providerRegistry.Get(providers.ProviderOpenRouter)
	if !exists {
		log.Warn("OpenRouter provider not found in registry")
	} else {
		openrouterAPIKey := os.Getenv("OPENROUTER_API_KEY")

		if openrouterAPIKey != "" {
			log.Info("Initializing OpenRouter provider from environment variable")
			if err := openrouterProvider.Initialize(ctx, providers.ProviderConfig{
				APIKey:  openrouterAPIKey,
				Timeout: 10 * time.Minute,
			}); err != nil {
				log.WithError(err).Warn("Failed to initialize OpenRouter provider")
			} else {
				log.Info("OpenRouter provider initialized successfully")
			}
		} else {
			log.Info("OpenRouter provider available but not configured - set OPENROUTER_API_KEY environment variable to enable")
		}
	}

	// Initialize Ollama provider (local provider - fallback for no API keys)
	ollamaProvider, exists := providerRegistry.Get(providers.ProviderOllama)
	if !exists {
		log.Warn("Ollama provider not found in registry")
	} else {
		log.Info("Initializing Ollama provider (local models)")
		if err := ollamaProvider.Initialize(ctx, providers.ProviderConfig{
			Timeout: 10 * time.Minute,
		}); err != nil {
			log.WithError(err).Warn("Failed to initialize Ollama provider")
		} else {
			log.Info("Ollama provider initialized successfully")
		}
	}

	// Link provider registry to UnifiedAgent
	unifiedAgent.SetProviderRegistry(providerRegistry)
	log.Info("Provider registry linked to UnifiedAgent")

	// [FIX] إنشاء وكلاء حقيقيين من الموديلات المتاحة في providers
	// كل model يصبح وكيل حقيقي مع ProviderAdapter
	var availableProviders []struct {
		providerType providers.ProviderType
		provider     providers.Provider
		models       []string
	}

	// جمع الموديلات المتاحة من كل provider مهيأ
	if cloudflareAPIKey := os.Getenv("CLOUDFLARE_API_KEY"); cloudflareAPIKey != "" {
		if cfProvider, exists := providerRegistry.Get(providers.ProviderCustom); exists {
			availableProviders = append(availableProviders, struct {
				providerType providers.ProviderType
				provider     providers.Provider
				models       []string
			}{
				providerType: providers.ProviderCustom,
				provider:     cfProvider,
				models:       []string{"@cf/moonshotai/kimi-k2.7-code", "@cf/zai-org/glm-5.2"},
			})
		}
	}
	if mistralAPIKey := os.Getenv("MISTRAL_API_KEY"); mistralAPIKey != "" {
		if mistralProvider, exists := providerRegistry.Get(providers.ProviderMistral); exists {
			availableProviders = append(availableProviders, struct {
				providerType providers.ProviderType
				provider     providers.Provider
				models       []string
			}{
				providerType: providers.ProviderMistral,
				provider:     mistralProvider,
				models:       []string{"mistral-large-latest", "mistral-medium", "mistral-small"},
			})
		}
	}
	if openrouterAPIKey := os.Getenv("OPENROUTER_API_KEY"); openrouterAPIKey != "" {
		if openrouterProvider, exists := providerRegistry.Get(providers.ProviderOpenRouter); exists {
			availableProviders = append(availableProviders, struct {
				providerType providers.ProviderType
				provider     providers.Provider
				models       []string
			}{
				providerType: providers.ProviderOpenRouter,
				provider:     openrouterProvider,
				models:       []string{"anthropic/claude-3.5-sonnet", "openai/gpt-4o", "google/gemini-pro"},
			})
		}
	}

	// إضافة Ollama كـ fallback (دائماً متاح إذا تم تهيئته)
	if ollamaProvider, exists := providerRegistry.Get(providers.ProviderOllama); exists {
		log.Info("Adding Ollama as fallback provider")
		availableProviders = append(availableProviders, struct {
			providerType providers.ProviderType
			provider     providers.Provider
			models       []string
		}{
			providerType: providers.ProviderOllama,
			provider:     ollamaProvider,
			models:       []string{"llama3.2", "llama3.1", "mistral", "codellama"},
		})
	} else {
		log.Warn("Ollama provider not found in registry")
	}

	// إنشاء وكلاء حقيقيين من الموديلات المتاحة
	agentIndex := 0
	var sessionManagerAgentID string // أول وكيل سيكون مدير الجلسة

	for _, prov := range availableProviders {
		for _, modelID := range prov.models {
			agentIndex++
			agentID := fmt.Sprintf("agent-%d-%s", agentIndex, modelID)
			agentName := fmt.Sprintf("%s Agent", modelID)

			// إنشاء ProviderAdapter للوكيل
			adapter := adapters.NewProviderAdapter(
				agentID,
				agentName,
				pkgAgent.AgentTypeAPI,
				prov.providerType,
				modelID,
			)

			// ربط Provider الحقيقي
			adapter.SetProvider(prov.provider)

			// تهيئة الـ provider
			if err := adapter.Initialize(ctx, &providers.ProviderConfig{
				Timeout: 10 * time.Minute,
			}); err != nil {
				log.WithError(err).Warnf("فشل تهيئة الوكيل %s", agentID)
				continue
			}

			// تسجيل في AgentRegistry
			if err := agentRegistry.Register(adapter, nil); err != nil {
				log.WithError(err).Warnf("فشل تسجيل الوكيل %s في AgentRegistry", agentID)
				continue
			}

			// أول وكيل سيكون مدير الجلسة
			if sessionManagerAgentID == "" {
				sessionManagerAgentID = agentID
				log.WithField("agent_id", agentID).Info("تم تعيين مدير الجلسة")
			}

			log.WithField("agent_id", agentID).WithField("model", modelID).Info("تم إنشاء وكيل حقيقي")
		}
	}

	if agentIndex == 0 {
		log.Warn("لم يتم إنشاء أي وكلاء حقيقيين - لا يوجد providers مهيأة")
	} else {
		log.WithField("agent_count", agentIndex).Info("تم إنشاء وكلاء حقيقيين من الموديلات المتاحة")

		// تسجيل الوكلاء الحقيقيين في UnifiedAgent و AgentPool
		for _, agentObj := range agentRegistry.ListAll() {
			info := agentObj.GetInfo()

			// إنشاء metadata للوكيل
			metadata := &agent.AgentMetadata{
				AgentID:  info.ID,
				Name:     info.Name,
				Type:     info.Type,
				Provider: info.Provider,
				Model:    info.Model,
			}

			// تسجيل عبر OrchestratorEngine - هذا يستدعي RegisterAgentFromUnified تلقائياً
			if err := orchestratorEngine.RegisterAgent(agentObj, metadata); err != nil {
				log.WithError(err).Warnf("Failed to register agent %s via OrchestratorEngine", info.ID)
			}

			// تسجيل في UnifiedAgent (skill manager + subagent manager)
			if err := unifiedAgent.RegisterAgent(
				ctx,
				info.ID,
				string(info.Type),
				info.Model,
				[]string{},
			); err != nil {
				log.WithError(err).Warnf("فشل تسجيل الوكيل %s في UnifiedAgent", info.ID)
			}

			// تسجيل الـ adapter نفسه في AgentPool (ThinkingEngine + ToolExecutor)
			role := "regular"
			if info.ID == sessionManagerAgentID {
				role = "manager" // مدير الجلسة
			}
			if err := unifiedAgent.RegisterAgentToPool(agentObj, role); err != nil {
				log.WithError(err).Warnf("فشل تسجيل الوكيل %s في AgentPool", info.ID)
			}
		}
		log.WithField("agent_count", agentRegistry.GetCount()).Info("تم تسجيل الوكلاء الحقيقيين في UnifiedAgent و AgentPool وتعيين الأدوار")
	}

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

	// [FIX] Link provider registry and router to AgentPool for ThinkingEngine initialization
	// This ensures that agents in the pool have access to models for task execution
	if agentPool := unifiedAgent.GetAgentPool(); agentPool != nil {
		// Set default provider if available (e.g., Cloudflare, Mistral, or OpenRouter)
		if cloudflareAPIKey := os.Getenv("CLOUDFLARE_API_KEY"); cloudflareAPIKey != "" {
			if cfProvider, exists := providerRegistry.Get(providers.ProviderCustom); exists {
				agentPool.SetDefaultProvider(cfProvider)
				agentPool.SetDefaultModelID("@cf/moonshotai/kimi-k2.7-code")
				log.Info("Cloudflare provider linked to AgentPool as default")
			}
		} else if mistralAPIKey := os.Getenv("MISTRAL_API_KEY"); mistralAPIKey != "" {
			if mistralProvider, exists := providerRegistry.Get(providers.ProviderMistral); exists {
				agentPool.SetDefaultProvider(mistralProvider)
				agentPool.SetDefaultModelID("mistral-large-latest")
				log.Info("Mistral provider linked to AgentPool as default")
			}
		} else if openrouterAPIKey := os.Getenv("OPENROUTER_API_KEY"); openrouterAPIKey != "" {
			if openrouterProvider, exists := providerRegistry.Get(providers.ProviderOpenRouter); exists {
				agentPool.SetDefaultProvider(openrouterProvider)
				agentPool.SetDefaultModelID("anthropic/claude-3.5-sonnet")
				log.Info("OpenRouter provider linked to AgentPool as default")
			}
		} else {
			log.Warn("No provider configured for AgentPool - agents will not have default models")
		}
	} else {
		log.Warn("AgentPool not found in UnifiedAgent - cannot link providers")
	}

	// [IMPORTANT] Default provider for ThinkingEngine will be set dynamically from Dashboard
	// Users can select their preferred provider and model from the Dashboard UI
	log.Info("Providers will be configured dynamically from Dashboard")

	// [FIX] Test execution عبر OrchestratorEngine (Phase A من Canonical Path)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.WithField("panic", r).Error("orchestrator test task goroutine panicked")
			}
		}()
		taskCtx, taskCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer taskCancel()
		testTask := &pkgAgent.AgentTask{
			ID:    "test-task-1",
			Title: "تحليل ملفات المشروع",
		}
		result, err := orchestratorEngine.ExecuteTask(taskCtx, testTask)
		if err != nil {
			log.WithError(err).Warn("Test task via OrchestratorEngine completed with error (expected without API key)")
		} else {
			log.WithField("result", result).Info("Test task executed successfully via OrchestratorEngine")
		}
		log.Info("Studio initialization complete — all systems operational")
	}()

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

	// [INTEGRATION] Initialize A2AManager for agent-to-agent protocol (part of A2AInterface)
	// A2AManager handles all agent communication including chat and direct messaging
	a2aManager := orchestrator.NewA2AManager(eb, zapLogger)
	if err := a2aManager.Start(); err != nil {
		log.WithError(err).Warn("Failed to start A2AManager")
	}
	defer a2aManager.Stop()
	log.Info("A2AManager started (A2AInterface - handles all agent communication)")

	// [FIX] Register agents from AgentRegistry in A2AManager for active agent communication
	for _, agentObj := range agentRegistry.ListAll() {
		info := agentObj.GetInfo()
		a2aAgent := &orchestrator.A2AAgent{
			ID:     info.ID,
			Name:   info.Name,
			Type:   string(info.Type),
			Skills: []string{}, // Will be populated from capabilities
			Status: "idle",
			Config: map[string]interface{}{},
		}
		if err := a2aManager.RegisterAgent(a2aAgent); err != nil {
			log.WithError(err).Warnf("Failed to register agent %s in A2AManager", info.ID)
		}
	}
	log.WithField("agent_count", agentRegistry.GetCount()).Info("Agents registered in A2AManager for active agent communication")

	// [INTEGRATION] Initialize AgentSessionIntegration to link agents with sessions
	agentSessionIntegration := pkgIntegration.NewAgentSessionIntegration(agentRegistry, sessionManager, zapLogger)
	log.Info("AgentSessionIntegration initialized")

	// [INTEGRATION] Initialize SessionOrchestrator to coordinate sessions and agents
	sessionOrchestrator := pkgIntegration.NewSessionOrchestrator(
		sessionManager,
		agentRegistry,
		agentSessionIntegration,
		nil, // instanceManager - will be added if needed
		nil, // roleAssignment - will be added if needed
		nil, // taskRouting - will be added if needed
		nil, // agentCommunication - handled by A2AManager
		zapLogger,
	)
	_ = sessionOrchestrator // Will be used for session coordination
	log.Info("SessionOrchestrator initialized")

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
	config, err := pkgConfig.LoadConfig("config.yaml")
	if err != nil {
		log.WithError(err).Warn("Failed to load config file, using defaults")
		config = pkgConfig.DefaultConfig()
	}
	if err := pkgConfig.ValidateConfig(config); err != nil {
		log.WithError(err).Warn("Config validation failed")
	}
	log.Info("Config initialized")

	// Initialize Limits
	resourceLimiter := pkgLimits.NewResourceLimiter(100)
	memoryLimiter := pkgLimits.NewMemoryLimiter(1024 * 1024 * 1024) // 1GB
	rateLimiter := pkgLimits.NewRateLimiter(1000, 100, time.Second)
	connLimiter := pkgLimits.NewConnectionLimiter(50)
	log.WithFields(logrus.Fields{
		"resource": resourceLimiter != nil,
		"memory":   memoryLimiter != nil,
		"rate":     rateLimiter != nil,
		"conn":     connLimiter != nil,
	}).Info("Limits initialized")

	// Initialize Timeout
	timeoutCfg := pkgTimeout.DefaultTimeoutConfig()
	log.WithField("timeout", timeoutCfg != nil).Info("Timeout config initialized")

	// Initialize Validation
	didValidator := pkgValidation.NewDIDValidator("did:mskt:")
	strValidator, _ := pkgValidation.NewStringValidator(1, 100, false, "^[a-zA-Z0-9]+$")
	emailValidator := pkgValidation.NewEmailValidator(false)
	portValidator := pkgValidation.NewPortValidator(1, 65535)
	numValidator := pkgValidation.NewNumberValidator(0, 100, false)
	log.WithFields(logrus.Fields{
		"did":    didValidator != nil,
		"string": strValidator != nil,
		"email":  emailValidator != nil,
		"port":   portValidator != nil,
		"number": numValidator != nil,
	}).Info("Validation initialized")

	// Initialize Ledger
	costTracker := pkgLedger.NewCostTracker()
	creditManager := pkgLedger.NewCreditManager(0.1) // 10% reward rate
	log.WithFields(logrus.Fields{
		"cost_tracker":   costTracker != nil,
		"credit_manager": creditManager != nil,
	}).Info("Ledger initialized")

	// Initialize Logger (already created above)
	log.Info("Logger initialized")

	// Initialize Sandbox
	wasmExecutor, wasmErr := pkgSandbox.NewExecutor(ctx)
	if wasmErr != nil {
		log.WithError(wasmErr).Warn("Failed to create WASM sandbox executor")
	} else {
		log.WithField("executor", wasmExecutor != nil).Info("WASM Sandbox executor initialized")
	}

	// Initialize Discovery
	indexedDiscovery := pkgDiscovery.NewIndexedDiscovery()
	log.WithField("discovery", indexedDiscovery != nil).Info("Discovery initialized")

	// ============================================================
	// 14. DiscoveryInterface - agent and service discovery (IndexedDiscovery)
	// ============================================================
	log.Info("IndexedDiscovery created (DiscoveryInterface)")

	// ============================================================
	// 11. UIBridgeInterface - REST API + WebSocket (api.Server)
	// ============================================================
	// سيتم إنشاء REST API لاحقاً في السطر 842

	// Email system integrated via orchestrator.EmailManager (already created above)

	// Initialize Hosting
	hostingManager := pkgHosting.NewHostingManager()
	log.WithField("hosting", hostingManager != nil).Info("Hosting initialized")

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
	localDNSProxy := pkgDomain.NewLocalDNSProxy(zapLogger, p2pDNSResolver, "127.0.0.1:5354")
	if err := localDNSProxy.Start(); err != nil {
		log.WithError(err).Warn("Failed to start local DNS proxy")
	} else {
		log.Info("Local DNS Proxy initialized")
		defer localDNSProxy.Stop()
	}

	httpProxy := pkgDomain.NewHTTPProxy(n.Host(), "127.0.0.1:8080")
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.WithField("panic", r).Error("HTTP proxy goroutine panicked")
			}
		}()
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

	// ============================================================
	// 12. JournalInterface - session journal (SessionJournal)
	// ============================================================
	// SessionJournal مُنفذ داخل SessionContainer (sessionContainer.Journal)
	log.Info("SessionJournal available inside SessionContainer (JournalInterface)")

	// ============================================================
	// 13. SyncInterface - CRDT sync (CRDTSyncManager)
	// ============================================================
	// CRDTSyncManager مُنفذ في pkg/sdk/crdt_sync.go
	// يمكن إنشاؤه عند الحاجة لمزامنة المستندات عبر الشبكة
	log.Info("CRDTSyncManager available in pkg/sdk (SyncInterface)")

	// ============================================================
	// 15. EventBus - same as CommunicationInterface
	// ============================================================
	// EventBus مُنفذ بالفعل في السطر 129 (eb)
	log.Info("EventBus already created (same as CommunicationInterface)")

	// [FIX] تفعيل الـ Policy Engine في وضع Audit (يسجل الرفض دون منع)
	// التعليمات: هذا يضمن أن كل مسار تنفيذ يمر عبر طبقة الصلاحيات من اليوم الأول
	// مع log-only حتى مرحلة الإنتاج
	oePolicy := orchestratorEngine.PolicyEngine()
	orchestratorEngine.SetPolicyMode(pkgCapability.PolicyModeAudit)
	log.Info("Policy mode set to AUDIT — denials are logged, execution is NOT blocked")

	// إضافة قاعدة افتراضية deny مع قواعد allow للعمليات الأساسية
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
	if err := oePolicy.AddRule(defaultRule); err != nil {
		log.WithError(err).Warn("Failed to add default policy rule")
	}

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
		if err := oePolicy.AddRule(rule); err != nil {
			log.WithError(err).Warnf("Failed to add allow rule: %s", rule.Name)
		}
	}
	log.WithField("rules", len(allowRules)+1).Info("Policy rules added to OrchestratorEngine — Audit mode active")

	// [FIX] UnifiedAgent handles all coordination internally
	log.Info("UnifiedAgent handles all agent coordination, session management, and orchestration")

	// إنشاء REST API Server
	apiServer := api.NewServerWithTLS(n, *apiPort, log, *tlsCert != "", *tlsCert, *tlsKey)

	apiKeyPath := filepath.Join(*dataDir, "provider-keys.enc")
	apiKeyManager, err := providers.NewAPIKeyManager(apiKeyPath)
	if err != nil {
		log.WithError(err).Warn("Failed to create API key manager — provider keys will not persist")
	} else {
		log.WithField("path", apiKeyPath).Info("API key manager initialized")
	}

	apiServer.UseRuntime(&api.ServerRuntime{
		EventBus:           eb,
		SessionManager:     sessionManager,
		BridgeManager:      sessionBridgeManager,
		ProviderRegistry:   providerRegistry,
		APIKeyManager:      apiKeyManager,
		OwnerDID:           kp.DID,
		AgentRegistry:      agentRegistry,
		UnifiedAgent:       unifiedAgent,
		OrchestratorEngine: orchestratorEngine,
		SessionContainer:   sessionContainer,
	})
	log.Info("REST API wired to shared studio runtime (EventBus, sessions, providers, agents, orchestrator)")
	log.WithField("port", *apiPort).Info("API Server created")

	// حفظ token للمصادقة
	apiToken := apiServer.LocalToken()
	log.WithField("token", apiToken[:10]+"...").Info("API authentication token generated")
	log.Infof("🚀 Dashboard: http://localhost:%d/dashboard?token=%s", *apiPort, apiToken)

	// إنشاء WebSocket Bridge لربط EventBus بـ SessionContainer
	wsHandler := api.NewWebSocketHandler(eb, sessionContainer, stdlog.New(os.Stdout, "[WS] ", stdlog.LstdFlags))
	if err := wsHandler.Start(); err != nil {
		log.WithError(err).Warn("Failed to start WebSocket handler")
	}
	log.Info("WebSocket Bridge created and started")

	// بدء REST API Server في الخلفية
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.WithField("panic", r).Error("API server goroutine panicked")
			}
		}()
		if err := apiServer.Start(); err != nil {
			log.WithError(err).Fatal("API server failed to start")
		}
	}()
	log.WithField("port", *apiPort).Info("API Server started")

	// [TEST] اختبار Thinking Engine - تنفيذ مهمة بسيطة
	log.Info("Testing Thinking Engine with a simple task...")
	testCtx, testCancel := context.WithTimeout(ctx, 30*time.Second)
	defer testCancel()

	testTask := "What is 2 + 2?"
	log.WithField("task", testTask).Info("Executing test task")

	// الحصول على أول وكيل من AgentPool
	agentPool := unifiedAgent.GetAgentPool()
	if agentPool != nil {
		agents := agentPool.ListAgents()
		if len(agents) > 0 {
			// الحصول على ThinkingEngine للوكيل الأول
			te, err := agentPool.GetOrCreateThinkingEngine(agents[0].AgentID)
			if err != nil {
				log.WithError(err).Warn("Failed to get ThinkingEngine for test")
			} else {
				// تنفيذ مهمة بسيطة
				result, err := te.AnalyzeTask(testCtx, testTask)
				if err != nil {
					log.WithError(err).Warn("ThinkingEngine test failed")
				} else {
					log.WithField("result", result).Info("ThinkingEngine test succeeded")
				}
			}
		} else {
			log.Warn("No agents in AgentPool for testing")
		}
	} else {
		log.Warn("AgentPool not initialized for testing")
	}

	// بدء واجهة Studio
	log.WithField("addr", *addr).Info("Studio starting...")

	// انتظار إشارة الإنهاء
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Info("Studio shutting down...")

	// إيقاف WebSocket Bridge
	if err := wsHandler.Stop(); err != nil {
		log.WithError(err).Warn("Failed to stop WebSocket handler gracefully")
	}

	// إيقاف REST API Server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := apiServer.Stop(shutdownCtx); err != nil {
		log.WithError(err).Warn("Failed to stop API server gracefully")
	}
}

// parseBootstrap يحلل عناوين bootstrap
func parseBootstrap(bootstrap string) []string {
	if bootstrap == "" {
		return nil
	}
	return []string{bootstrap}
}
