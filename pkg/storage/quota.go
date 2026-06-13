package storage

import (
	"fmt"
	"sync"
)

const DefaultFreeTierBytes = 1 * 1024 * 1024 * 1024 // 1 GB

// QuotaManager يدير حصص التخزين بشكل آمن ومتزامن
type QuotaManager struct {
	mu     sync.RWMutex
	limits map[string]int64 // did -> الحد الأقصى بالبايت
	usage  map[string]int64 // did -> الاستخدام الحالي بالبايت
}

// NewQuotaManager ينشئ مدير حصص جديد
func NewQuotaManager() *QuotaManager {
	return &QuotaManager{
		limits: make(map[string]int64),
		usage:  make(map[string]int64),
	}
}

// SetLimit يحدد الحد الأقصى (يستخدم عند ترقية الباقة)
func (q *QuotaManager) SetLimit(did string, limitBytes int64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.limits[did] = limitBytes
}

// CheckAndAdd يتحقق من المساحة ويحجزها بشكل ذري (Atomic). هذا هو خط الدفاع الأول.
func (q *QuotaManager) CheckAndAdd(did string, sizeBytes int64) error {
	if sizeBytes <= 0 {
		return fmt.Errorf("invalid size: must be greater than 0")
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	limit, exists := q.limits[did]
	if !exists {
		limit = DefaultFreeTierBytes
		q.limits[did] = limit // تطبيق قاعدة الـ 1GB مجاناً تلقائياً
	}

	currentUsage := q.usage[did]
	if currentUsage+sizeBytes > limit {
		return fmt.Errorf("quota exceeded: limit %d, usage %d, requested %d", limit, currentUsage, sizeBytes)
	}

	q.usage[did] += sizeBytes
	return nil
}

// Release يحرر المساحة عند حذف ملف (يجب استدعاؤه دائماً بعد الحذف)
func (q *QuotaManager) Release(did string, sizeBytes int64) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.usage[did] >= sizeBytes {
		q.usage[did] -= sizeBytes
	} else {
		q.usage[did] = 0 // منع القيم السلبية في حال حدوث خطأ في الحساب
	}
}

// GetUsage يعود بالاستخدام الحالي
func (q *QuotaManager) GetUsage(did string) int64 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.usage[did]
}

// GetLimit يعود بالحد الأقصى
func (q *QuotaManager) GetLimit(did string) int64 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	if limit, exists := q.limits[did]; exists {
		return limit
	}
	return DefaultFreeTierBytes
}

// GetRemaining يعود بالمساحة المتبقية
func (q *QuotaManager) GetRemaining(did string) int64 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	limit := int64(DefaultFreeTierBytes)
	if l, exists := q.limits[did]; exists {
		limit = l
	}
	return limit - q.usage[did]
}

// ResetUsage يعيد تعيين الاستخدام إلى صفر (للاستخدام في الاختبارات أو إعادة التعيين)
func (q *QuotaManager) ResetUsage(did string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.usage[did] = 0
}
