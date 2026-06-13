package content

import (
	"fmt"
	"sync"
)

// QuotaManager يدير حدود التخزين لكل DID
type QuotaManager struct {
	mu       sync.RWMutex
	limits   map[string]int64 // DID -> Max Bytes
	usage    map[string]int64 // DID -> Current Bytes
}

// NewQuotaManager ينشئ مدير حصص جديد
func NewQuotaManager() *QuotaManager {
	return &QuotaManager{
		limits: make(map[string]int64),
		usage:  make(map[string]int64),
	}
}

// SetLimit يحدد الحد الأقصى للتخزين (مثلاً 10GB = 10 * 1024 * 1024 * 1024)
func (q *QuotaManager) SetLimit(did string, limitBytes int64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.limits[did] = limitBytes
}

// CheckAndAdd يتحقق من المساحة المتاحة ويضيف الاستخدام إذا كان مسموحاً
func (q *QuotaManager) CheckAndAdd(did string, sizeBytes int64) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	limit, exists := q.limits[did]
	if !exists {
		// حد افتراضي 10GB إذا لم يتم تحديده
		limit = 10 * 1024 * 1024 * 1024
		q.limits[did] = limit
	}

	currentUsage := q.usage[did]
	if currentUsage+sizeBytes > limit {
		return fmt.Errorf("quota exceeded for DID %s: limit %d, usage %d, requested %d", did, limit, currentUsage, sizeBytes)
	}

	q.usage[did] += sizeBytes
	return nil
}

// Release يحرر المساحة عند حذف ملف
func (q *QuotaManager) Release(did string, sizeBytes int64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.usage[did] >= sizeBytes {
		q.usage[did] -= sizeBytes
	} else {
		q.usage[did] = 0
	}
}
