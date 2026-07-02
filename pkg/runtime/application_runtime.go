package runtime

import (
	"context"
	"fmt"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/agent/unified"
	"github.com/MortalArena/Musketeers/pkg/lifecycle"
	"github.com/MortalArena/Musketeers/pkg/orchestrator"
	"github.com/MortalArena/Musketeers/pkg/providers"
	"github.com/MortalArena/Musketeers/pkg/session"
	"go.uber.org/zap"
)

// ApplicationRuntime Composition Root - يملك كل المكونات الأساسية
type ApplicationRuntime struct {
	// المكونات الأساسية
	providerRegistry   *providers.ProviderRegistry
	agentRegistry      *agent.AgentRegistry
	agentPool          *unified.AgentPool
	sessionContainer   *session.SessionContainer
	orchestratorEngine *orchestrator.OrchestratorEngine

	// Lifecycle
	lifecycle *lifecycle.LifecycleMixin
	logger    *zap.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.RWMutex
}

// NewApplicationRuntime ينشئ ApplicationRuntime جديد
// ProviderRegistry مطلوب — إذا كان nil سيفشل Build()
func NewApplicationRuntime(logger *zap.Logger) *ApplicationRuntime {
	ctx, cancel := context.WithCancel(context.Background())
	return &ApplicationRuntime{
		lifecycle: lifecycle.NewLifecycleMixin(),
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Build يبني كل المكونات
// يجب أن يكون ProviderRegistry قد تم حقنه via SetProviderRegistry قبل استدعاء Build
func (ar *ApplicationRuntime) Build() error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	ar.lifecycle.SetStatus(lifecycle.LifecycleStatusStarting)
	ar.logger.Info("Building ApplicationRuntime")

	if ar.providerRegistry == nil {
		return fmt.Errorf("ProviderRegistry is nil — call SetProviderRegistry() before Build()")
	}
	if ar.agentRegistry == nil {
		ar.agentRegistry = agent.NewAgentRegistry()
		ar.agentRegistry.SetLogger(ar.logger)
	}

	ar.logger.Info("ApplicationRuntime built successfully")
	return nil
}

// SetAgentPool يضبط AgentPool (يُستدعى من main.go بعد إنشاء AgentPool)
func (ar *ApplicationRuntime) SetAgentPool(ap *unified.AgentPool) {
	ar.mu.Lock()
	defer ar.mu.Unlock()
	ar.agentPool = ap
	ar.logger.Info("AgentPool set on ApplicationRuntime")
}

// SetSessionContainer يضبط SessionContainer (يُستدعى من main.go بعد إنشاء SessionContainer)
func (ar *ApplicationRuntime) SetSessionContainer(sc *session.SessionContainer) {
	ar.mu.Lock()
	defer ar.mu.Unlock()
	ar.sessionContainer = sc
	ar.logger.Info("SessionContainer set on ApplicationRuntime")
}

// SetOrchestratorEngine يضبط OrchestratorEngine (يُستدعى من main.go بعد إنشاء OrchestratorEngine)
func (ar *ApplicationRuntime) SetOrchestratorEngine(oe *orchestrator.OrchestratorEngine) {
	ar.mu.Lock()
	defer ar.mu.Unlock()
	ar.orchestratorEngine = oe
	ar.logger.Info("OrchestratorEngine set on ApplicationRuntime")
}

// SetProviderRegistry يضبط ProviderRegistry (يُستدعى من main.go بعد إنشاء ProviderRegistry)
func (ar *ApplicationRuntime) SetProviderRegistry(pr *providers.ProviderRegistry) {
	ar.mu.Lock()
	defer ar.mu.Unlock()
	ar.providerRegistry = pr
	ar.logger.Info("ProviderRegistry set on ApplicationRuntime")
}

// GetAgentPool يرجع AgentPool
func (ar *ApplicationRuntime) GetAgentPool() *unified.AgentPool {
	ar.mu.RLock()
	defer ar.mu.RUnlock()
	return ar.agentPool
}

// GetSessionContainer يرجع SessionContainer
func (ar *ApplicationRuntime) GetSessionContainer() *session.SessionContainer {
	ar.mu.RLock()
	defer ar.mu.RUnlock()
	return ar.sessionContainer
}

// GetOrchestratorEngine يرجع OrchestratorEngine
func (ar *ApplicationRuntime) GetOrchestratorEngine() *orchestrator.OrchestratorEngine {
	ar.mu.RLock()
	defer ar.mu.RUnlock()
	return ar.orchestratorEngine
}

// Inject يربط المكونات ببعضها
func (ar *ApplicationRuntime) Inject() error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	ar.logger.Info("Injecting dependencies")

	// حقن الاعتماديات بين المكونات
	// AgentRegistry يحتاج ProviderRegistry
	// OrchestratorEngine يحتاج AgentRegistry
	// AgentPool يحتاج SessionContainer
	// SessionContainer يحتاج EventBus و DB

	ar.logger.Info("Dependencies injected successfully (partial)")
	return nil
}

// Start يبدأ كل المكونات
func (ar *ApplicationRuntime) Start() error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	ar.logger.Info("Starting ApplicationRuntime")

	// بدء المكونات التي تطبق Lifecycle
	if err := ar.providerRegistry.Start(ar.ctx); err != nil {
		return fmt.Errorf("failed to start ProviderRegistry: %w", err)
	}

	if err := ar.agentRegistry.Start(ar.ctx); err != nil {
		return fmt.Errorf("failed to start AgentRegistry: %w", err)
	}

	// بدء المكونات الأخرى إذا كانت موجودة
	if ar.sessionContainer != nil {
		if err := ar.sessionContainer.Start(ar.ctx); err != nil {
			return fmt.Errorf("failed to start SessionContainer: %w", err)
		}
	}

	if ar.agentPool != nil {
		if err := ar.agentPool.Start(ar.ctx); err != nil {
			return fmt.Errorf("failed to start AgentPool: %w", err)
		}
	}

	if ar.orchestratorEngine != nil {
		if err := ar.orchestratorEngine.Start(ar.ctx); err != nil {
			return fmt.Errorf("failed to start OrchestratorEngine: %w", err)
		}
	}

	ar.lifecycle.SetStatus(lifecycle.LifecycleStatusRunning)
	ar.logger.Info("ApplicationRuntime started successfully")
	return nil
}

// Shutdown يوقف كل المكونات بشكل آمن
func (ar *ApplicationRuntime) Shutdown(ctx context.Context) error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	ar.lifecycle.SetStatus(lifecycle.LifecycleStatusStopping)
	ar.logger.Info("Shutting down ApplicationRuntime")

	// إيقاف المكونات التي تطبق Lifecycle بالترتيب العكسي
	var errors []error

	if ar.orchestratorEngine != nil {
		if err := ar.orchestratorEngine.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to shutdown OrchestratorEngine: %w", err))
		}
	}

	if ar.agentPool != nil {
		if err := ar.agentPool.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to shutdown AgentPool: %w", err))
		}
	}

	if ar.sessionContainer != nil {
		if err := ar.sessionContainer.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to shutdown SessionContainer: %w", err))
		}
	}

	if err := ar.agentRegistry.Shutdown(ctx); err != nil {
		errors = append(errors, fmt.Errorf("failed to shutdown AgentRegistry: %w", err))
	}

	if err := ar.providerRegistry.Shutdown(ctx); err != nil {
		errors = append(errors, fmt.Errorf("failed to shutdown ProviderRegistry: %w", err))
	}

	ar.lifecycle.SetStatus(lifecycle.LifecycleStatusStopped)
	ar.logger.Info("ApplicationRuntime shut down successfully")

	if len(errors) > 0 {
		return fmt.Errorf("shutdown completed with %d errors: %v", len(errors), errors)
	}
	return nil
}

// Cancel يلغي كل العمليات الجارية
func (ar *ApplicationRuntime) Cancel() error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	ar.logger.Info("Cancelling ApplicationRuntime")

	// إلغاء المكونات التي تطبق Lifecycle بالترتيب العكسي
	var errors []error

	if ar.orchestratorEngine != nil {
		if err := ar.orchestratorEngine.Cancel(); err != nil {
			errors = append(errors, fmt.Errorf("failed to cancel OrchestratorEngine: %w", err))
		}
	}

	if ar.agentPool != nil {
		if err := ar.agentPool.Cancel(); err != nil {
			errors = append(errors, fmt.Errorf("failed to cancel AgentPool: %w", err))
		}
	}

	if ar.sessionContainer != nil {
		if err := ar.sessionContainer.Cancel(); err != nil {
			errors = append(errors, fmt.Errorf("failed to cancel SessionContainer: %w", err))
		}
	}

	if err := ar.agentRegistry.Cancel(); err != nil {
		errors = append(errors, fmt.Errorf("failed to cancel AgentRegistry: %w", err))
	}

	if err := ar.providerRegistry.Cancel(); err != nil {
		errors = append(errors, fmt.Errorf("failed to cancel ProviderRegistry: %w", err))
	}

	ar.cancel()
	ar.logger.Info("ApplicationRuntime cancelled successfully")

	if len(errors) > 0 {
		return fmt.Errorf("cancel completed with %d errors: %v", len(errors), errors)
	}
	return nil
}

// IsRunning يتحقق مما إذا كان يعمل
func (ar *ApplicationRuntime) IsRunning() bool {
	return ar.lifecycle.IsRunningMixin()
}

// Status يرجع الحالة
func (ar *ApplicationRuntime) Status() lifecycle.LifecycleStatus {
	return ar.lifecycle.GetStatus()
}

// Close يغلق ApplicationRuntime
func (ar *ApplicationRuntime) Close() error {
	return ar.Shutdown(ar.ctx)
}

// Stop يوقف ApplicationRuntime
func (ar *ApplicationRuntime) Stop(ctx context.Context) error {
	return ar.Shutdown(ctx)
}

// GetAgentRegistry يرجع AgentRegistry
func (ar *ApplicationRuntime) GetAgentRegistry() *agent.AgentRegistry {
	return ar.agentRegistry
}

// GetProviderRegistry يرجع ProviderRegistry
func (ar *ApplicationRuntime) GetProviderRegistry() *providers.ProviderRegistry {
	return ar.providerRegistry
}
