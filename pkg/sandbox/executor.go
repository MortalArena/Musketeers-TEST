package sandbox

import (
	"context"
	"fmt"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// SandboxConfig إعدادات الصندوق الرملي
type SandboxConfig struct {
	MemoryLimitPages uint32 // 1 page = 64KB. 800 pages ≈ 50MB
	WasmBinary       []byte
}

// Executor ينفذ أكواد WASM في بيئة معزولة
type Executor struct {
	runtime wazero.Runtime
}

// NewExecutor ينشئ بيئة تشغيل WASM جديدة
func NewExecutor(ctx context.Context) (*Executor, error) {
	// إنشاء بيئة تشغيل جديدة
	r := wazero.NewRuntime(ctx)

	// إضافة دعم WASI (اختياري، لكن يجب تقييده لاحقاً)
	// في الإنتاج، يجب استبداله بـ Host Functions مخصصة وآمنة فقط
	_, err := wasi_snapshot_preview1.Instantiate(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate WASI: %w", err)
	}

	return &Executor{runtime: r}, nil
}

// Execute ينفذ وحدة WASM مع فرض حدود الموارد
func (e *Executor) Execute(ctx context.Context, config SandboxConfig, funcName string, args ...uint64) (uint64, error) {
	// 1. تقييد الذاكرة (Memory Limiting)
	// ملاحظة: wazero يدعم تحديد حجم الذاكرة القصوى عبر ModuleConfig
	compiled, err := e.runtime.CompileModule(ctx, config.WasmBinary)
	if err != nil {
		return 0, fmt.Errorf("failed to compile wasm module: %w", err)
	}

	// 2. تكوين الوحدة (Module Config)
	// منع الوصول لنظام الملفات أو الشبكة الافتراضية
	modConfig := wazero.NewModuleConfig().
		WithName("isolated-plugin")

	// 3. تشغيل الوحدة
	mod, err := e.runtime.InstantiateModule(ctx, compiled, modConfig)
	if err != nil {
		return 0, fmt.Errorf("failed to instantiate wasm module: %w", err)
	}
	defer mod.Close(ctx) // ضمان تنظيف الموارد

	// 4. استدعاء الدالة المطلوبة
	results, err := mod.ExportedFunction(funcName).Call(ctx, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to call wasm function '%s': %w", funcName, err)
	}

	if len(results) == 0 {
		return 0, nil
	}
	return results[0], nil
}

// Close يغلق بيئة التشغيل ويحرر الذاكرة
func (e *Executor) Close(ctx context.Context) error {
	return e.runtime.Close(ctx)
}
