package eventbus

import (
	"context"
	"sync"
	"time"
)

// DeadLetterQueue قائمة انتظار للأحداث المرفوضة (In-memory version)
type DeadLetterQueue struct {
	entries  []DLQEntry
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	eventBus *EventBus // Reference to parent EventBus for retry
	maxSize  int       // Maximum number of entries in DLQ
}

// DLQEntry إدخال في DLQ
type DLQEntry struct {
	Event      Event
	Attempts   int
	LastRetry  time.Time
	NextRetry  time.Time
	ErrorCount int
	Errors     []string
}

// NewDeadLetterQueue ينشئ DLQ جديدة
func NewDeadLetterQueue(eventBus *EventBus, maxSize int) *DeadLetterQueue {
	ctx, cancel := context.WithCancel(context.Background())

	dlq := &DeadLetterQueue{
		entries:  make([]DLQEntry, 0),
		ctx:      ctx,
		cancel:   cancel,
		eventBus: eventBus,
		maxSize:  maxSize,
	}

	// بدء background retry worker
	go dlq.retryWorker()

	return dlq
}

// Add يضيف حدثاً إلى DLQ
func (dlq *DeadLetterQueue) Add(event Event) error {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	// التحقق من الحد الأقصى
	if len(dlq.entries) >= dlq.maxSize {
		// إذا امتلأت، نحذف أقدم إدخال
		dlq.entries = dlq.entries[1:]
	}

	entry := DLQEntry{
		Event:      event,
		Attempts:   0,
		LastRetry:  time.Time{},
		NextRetry:  time.Now().Add(5 * time.Second), // أول إعادة محاولة بعد 5 ثواني
		ErrorCount: 0,
		Errors:     []string{},
	}

	dlq.entries = append(dlq.entries, entry)

	return nil
}

// retryWorker يعالج إعادة محاولة الأحداث في الخلفية
func (dlq *DeadLetterQueue) retryWorker() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-dlq.ctx.Done():
			return
		case <-ticker.C:
			dlq.processRetry()
		}
	}
}

// processRetry يعالج الأحداث الجاهزة لإعادة المحاولة
func (dlq *DeadLetterQueue) processRetry() {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	now := time.Now()
	remainingEntries := make([]DLQEntry, 0)

	for _, entry := range dlq.entries {
		// التحقق من أن الحدث جاهز لإعادة المحاولة
		if entry.NextRetry.Before(now) && entry.Attempts < 10 {
			// محاولة إعادة النشر
			entry.Attempts++
			entry.LastRetry = now
			entry.NextRetry = now.Add(calculateBackoff(entry.Attempts))

			// محاولة إعادة النشر مباشرة
			select {
			case dlq.eventBus.eventQueue <- entry.Event:
				// إذا نجح، نحذف الإدخال
				continue
			default:
				// إذا فشل، نحتفظ به للإعادة المحاولة التالية
				entry.ErrorCount++
				entry.Errors = append(entry.Errors, "queue full during retry")
			}
		}

		// إذا وصلنا إلى الحد الأقصى للمحاولات، نحذف الحدث
		if entry.Attempts >= 10 {
			continue
		}

		remainingEntries = append(remainingEntries, entry)
	}

	dlq.entries = remainingEntries
}

// calculateBackoff يحسب وقت الانتظار للإعادة المحاولة (Exponential Backoff)
func calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: 5s, 10s, 20s, 40s, 80s, 160s, 320s, 640s, 1280s, 2560s
	base := 5 * time.Second
	max := 5 * time.Minute
	backoff := base * time.Duration(1<<uint(attempt-1))
	if backoff > max {
		backoff = max
	}
	return backoff
}

// GetSize يحصل على حجم DLQ
func (dlq *DeadLetterQueue) GetSize() int {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()
	return len(dlq.entries)
}

// Clear يمسح جميع الأحداث من DLQ
func (dlq *DeadLetterQueue) Clear() {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()
	dlq.entries = make([]DLQEntry, 0)
}

// Close يغلق DLQ
func (dlq *DeadLetterQueue) Close() {
	dlq.cancel()
}
