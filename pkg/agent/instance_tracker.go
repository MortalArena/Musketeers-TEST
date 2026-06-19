package agent

import (
	"fmt"
	"sync"
	"time"
)

// InstanceTracker متتبع نسخ الوكلاء
type InstanceTracker struct {
	mu              sync.RWMutex
	instanceCounters map[string]int // model -> counter
	sessionCounters map[string]int // sessionID -> counter
}

// NewInstanceTracker ينشئ متتبع نسخ جديد
func NewInstanceTracker() *InstanceTracker {
	return &InstanceTracker{
		instanceCounters: make(map[string]int),
		sessionCounters:  make(map[string]int),
	}
}

// GenerateInstanceID يولد معرف فريد لنسخة الوكيل
func (it *InstanceTracker) GenerateInstanceID(provider, model string) string {
	it.mu.Lock()
	defer it.mu.Unlock()

	key := fmt.Sprintf("%s-%s", provider, model)
	it.instanceCounters[key]++
	counter := it.instanceCounters[key]

	return fmt.Sprintf("%s-%d", model, counter)
}

// GenerateSessionInstanceID يولد معرف فريد لنسخة الوكيل في جلسة محددة
func (it *InstanceTracker) GenerateSessionInstanceID(sessionID, provider, model string) string {
	it.mu.Lock()
	defer it.mu.Unlock()

	sessionKey := fmt.Sprintf("%s-%s-%s", sessionID, provider, model)
	it.sessionCounters[sessionKey]++
	counter := it.sessionCounters[sessionKey]

	return fmt.Sprintf("%s-%d", model, counter)
}

// GenerateAPIKeyID يولد معرف فريد لمفتاح API
func (it *InstanceTracker) GenerateAPIKeyID(humanClientID, provider string) string {
	it.mu.Lock()
	defer it.mu.Unlock()

	key := fmt.Sprintf("%s-%s", humanClientID, provider)
	it.instanceCounters[key]++
	counter := it.instanceCounters[key]

	return fmt.Sprintf("api_key_%s_%d", provider, counter)
}

// GenerateUniqueAgentID يولد معرف فريد للوكيل
func (it *InstanceTracker) GenerateUniqueAgentID(provider, model, instanceID, humanClientID string) string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("agent_%s_%s_%s_%s_%d", provider, model, instanceID, humanClientID, timestamp)
}

// GetInstanceCount يحصل على عدد النسخ لنموذج معين
func (it *InstanceTracker) GetInstanceCount(provider, model string) int {
	it.mu.RLock()
	defer it.mu.RUnlock()

	key := fmt.Sprintf("%s-%s", provider, model)
	return it.instanceCounters[key]
}

// GetSessionInstanceCount يحصل على عدد النسخ لنموذج معين في جلسة محددة
func (it *InstanceTracker) GetSessionInstanceCount(sessionID, provider, model string) int {
	it.mu.RLock()
	defer it.mu.RUnlock()

	sessionKey := fmt.Sprintf("%s-%s-%s", sessionID, provider, model)
	return it.sessionCounters[sessionKey]
}

// ResetCounters يعيد تعيين العدادات
func (it *InstanceTracker) ResetCounters() {
	it.mu.Lock()
	defer it.mu.Unlock()

	it.instanceCounters = make(map[string]int)
	it.sessionCounters = make(map[string]int)
}

// ResetSessionCounters يعيد تعيين عدادات جلسة محددة
func (it *InstanceTracker) ResetSessionCounters(sessionID string) {
	it.mu.Lock()
	defer it.mu.Unlock()

	for key := range it.sessionCounters {
		if len(key) > len(sessionID) && key[:len(sessionID)] == sessionID {
			delete(it.sessionCounters, key)
		}
	}
}

// GetDisplayDisplayName يولد اسم عرض للوكيل
func (it *InstanceTracker) GetDisplayDisplayName(provider, model, instanceID, humanClientName string) string {
	if humanClientName != "" {
		return fmt.Sprintf("%s (%s) - %s", model, humanClientName, instanceID)
	}
	return fmt.Sprintf("%s - %s", model, instanceID)
}
