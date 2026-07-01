package lifecycle

import (
	"context"
	"sync"
)

// LifecycleStatus حالة دورة الحياة
type LifecycleStatus string

const (
	LifecycleStatusStopped  LifecycleStatus = "stopped"
	LifecycleStatusStarting LifecycleStatus = "starting"
	LifecycleStatusRunning  LifecycleStatus = "running"
	LifecycleStatusStopping LifecycleStatus = "stopping"
	LifecycleStatusError    LifecycleStatus = "error"
)

// Lifecycle واجهة دورة الحياة الموحدة
type Lifecycle interface {
	// Start يبدأ المكون
	Start(ctx context.Context) error

	// Stop يوقف المكون
	Stop(ctx context.Context) error

	// Close يغلق المكون
	Close() error

	// Shutdown يوقف المكون بشكل آمن
	Shutdown(ctx context.Context) error

	// Cancel يلغي العمليات الجارية
	Cancel() error

	// IsRunning يتحقق مما إذا كان المكون يعمل
	IsRunning() bool

	// Status يرجع حالة المكون
	Status() LifecycleStatus
}

// LifecycleMixin mixin لتقليل التكرار في دورة الحياة
type LifecycleMixin struct {
	status LifecycleStatus
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
}

// NewLifecycleMixin ينشئ LifecycleMixin جديد
func NewLifecycleMixin() *LifecycleMixin {
	ctx, cancel := context.WithCancel(context.Background())
	return &LifecycleMixin{
		status: LifecycleStatusStopped,
		ctx:    ctx,
		cancel: cancel,
	}
}

// SetStatus يضبط الحالة
func (lm *LifecycleMixin) SetStatus(status LifecycleStatus) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.status = status
}

// GetStatus يرجع الحالة
func (lm *LifecycleMixin) GetStatus() LifecycleStatus {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.status
}

// Context يرجع context
func (lm *LifecycleMixin) Context() context.Context {
	return lm.ctx
}

// CancelContext يلغي context
func (lm *LifecycleMixin) CancelContext() {
	lm.cancel()
}

// IsRunningMixin يتحقق مما إذا كان يعمل
func (lm *LifecycleMixin) IsRunningMixin() bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.status == LifecycleStatusRunning
}
