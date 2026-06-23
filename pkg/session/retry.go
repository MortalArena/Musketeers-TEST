package session

import (
	"context"
	"fmt"
	"time"
)

// RetryConfig إعدادات إعادة المحاولة
type RetryConfig struct {
	MaxAttempts  int           // الحد الأقصى لمحاولات إعادة المحاولة
	InitialDelay time.Duration // التأخير الأولي
	MaxDelay     time.Duration // الحد الأقصى للتأخير
	Multiplier   float64       // مضاعف التأخير (للتراجع الأسي)
}

// DefaultRetryConfig إعدادات إعادة المحاولة الافتراضية
var DefaultRetryConfig = RetryConfig{
	MaxAttempts:  3,
	InitialDelay: 1 * time.Second,
	MaxDelay:     30 * time.Second,
	Multiplier:   2.0,
}

// RetryWithBackoff يعيد المحاولة مع تراجع أسي
func RetryWithBackoff(ctx context.Context, config RetryConfig, operation func() error) error {
	var lastErr error
	delay := config.InitialDelay

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		if attempt > 0 {
			// الانتظار قبل إعادة المحاولة
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}

			// زيادة التأخير (تراجع أسي)
			delay = time.Duration(float64(delay) * config.Multiplier)
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
		}

		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// إذا كان الخطأ نهائياً، لا حاجة لإعادة المحاولة
		if IsFatalError(err) {
			return fmt.Errorf("fatal error: %w", err)
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", config.MaxAttempts, lastErr)
}

// IsFatalError يتحقق مما إذا كان الخطأ نهائياً
func IsFatalError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// قائمة بالأخطاء النهائية الشائعة
	fatalErrors := []string{
		"context canceled",
		"context deadline exceeded",
		"maximum limit reached",
		"cannot be empty",
		"too long",
		"invalid",
		"not found",
	}

	for _, fatal := range fatalErrors {
		if containsString(errMsg, fatal) {
			return true
		}
	}

	return false
}

// containsString يتحقق من وجود نص في نص آخر
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

// findSubstring يبحث عن نص فرعي
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// SafeExecute ينفذ عملية بأمان مع معالجة الأخطاء
func SafeExecute(ctx context.Context, operation string, fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in %s: %v", operation, r)
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fn()
	}
}

// SafeExecuteWithRetry ينفذ عملية بأمان مع إعادة المحاولة
func SafeExecuteWithRetry(ctx context.Context, operation string, config RetryConfig, fn func() error) error {
	return SafeExecute(ctx, operation, func() error {
		return RetryWithBackoff(ctx, config, fn)
	})
}
