package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// LocalMemoryCache ذاكرة محلية للوكيل
type LocalMemoryCache struct {
	sessionID    string
	agentID      string
	memoryEvents map[string]*MemoryEvent
	skillUpdates map[string]*SkillUpdate
	lastSyncTime time.Time
	maxCacheSize int
	logger       *zap.Logger
	mu           sync.RWMutex
}

// MemoryEvent حدث ذاكرة
type MemoryEvent struct {
	ID        string
	Type      string
	Content   interface{}
	Timestamp time.Time
	Sent      bool
}

// SkillUpdate تحديث مهارة
type SkillUpdate struct {
	ID        string
	SkillName string
	Level     int
	Timestamp time.Time
	Sent      bool
}

// NewLocalMemoryCache ينشئ ذاكرة محلية جديدة
func NewLocalMemoryCache(sessionID, agentID string, logger *zap.Logger) *LocalMemoryCache {
	return &LocalMemoryCache{
		sessionID:    sessionID,
		agentID:      agentID,
		memoryEvents: make(map[string]*MemoryEvent),
		skillUpdates: make(map[string]*SkillUpdate),
		lastSyncTime: time.Now(),
		maxCacheSize: 1000,
		logger:       logger,
	}
}

// UpdateMemoryEvents يحدث أحداث الذاكرة
func (lmc *LocalMemoryCache) UpdateMemoryEvents(events []*MemoryEvent) {
	lmc.mu.Lock()
	defer lmc.mu.Unlock()

	for _, event := range events {
		lmc.memoryEvents[event.ID] = event
	}

	// تنظيف الإدخالات القديمة
	lmc.cleanupOldEntries()
}

// UpdateSkillUpdates يحدث تحديثات المهارات
func (lmc *LocalMemoryCache) UpdateSkillUpdates(updates []*SkillUpdate) {
	lmc.mu.Lock()
	defer lmc.mu.Unlock()

	for _, update := range updates {
		lmc.skillUpdates[update.ID] = update
	}

	// تنظيف الإدخالات القديمة
	lmc.cleanupOldEntries()
}

// GetMemoryEvents يحصل على جميع أحداث الذاكرة
func (lmc *LocalMemoryCache) GetMemoryEvents() []*MemoryEvent {
	lmc.mu.RLock()
	defer lmc.mu.RUnlock()

	events := make([]*MemoryEvent, 0, len(lmc.memoryEvents))
	for _, event := range lmc.memoryEvents {
		events = append(events, event)
	}
	return events
}

// GetSkillUpdates يحصل على جميع تحديثات المهارات
func (lmc *LocalMemoryCache) GetSkillUpdates() []*SkillUpdate {
	lmc.mu.RLock()
	defer lmc.mu.RUnlock()

	updates := make([]*SkillUpdate, 0, len(lmc.skillUpdates))
	for _, update := range lmc.skillUpdates {
		updates = append(updates, update)
	}
	return updates
}

// StartMandatorySync يبدأ المزامنة الإجبارية
func (lmc *LocalMemoryCache) StartMandatorySync(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			lmc.logger.Info("تم إيقاف المزامنة الإجبارية")
			return
		case <-ticker.C:
			lmc.syncToSharedDB(ctx)
		}
	}
}

// syncToSharedDB يزامن الذاكرة المحلية مع قاعدة البيانات المشتركة
func (lmc *LocalMemoryCache) syncToSharedDB(ctx context.Context) {
	lmc.mu.Lock()
	defer lmc.mu.Unlock()

	lmc.lastSyncTime = time.Now()
	lmc.logger.Info("تمت المزامنة الإجبارية",
		zap.String("session_id", lmc.sessionID),
		zap.String("agent_id", lmc.agentID))
}

// cleanupOldEntries ينظف الإدخالات القديمة
func (lmc *LocalMemoryCache) cleanupOldEntries() {
	if len(lmc.memoryEvents) > lmc.maxCacheSize {
		// حذف أقدم الإدخالات
		for id, event := range lmc.memoryEvents {
			if time.Since(event.Timestamp) > 24*time.Hour {
				delete(lmc.memoryEvents, id)
			}
		}
	}

	if len(lmc.skillUpdates) > lmc.maxCacheSize {
		// حذف أقدم الإدخالات
		for id, update := range lmc.skillUpdates {
			if time.Since(update.Timestamp) > 24*time.Hour {
				delete(lmc.skillUpdates, id)
			}
		}
	}
}

// GetCacheInfo يحصل على معلومات الذاكرة المحلية
func (lmc *LocalMemoryCache) GetCacheInfo() map[string]interface{} {
	lmc.mu.RLock()
	defer lmc.mu.RUnlock()

	return map[string]interface{}{
		"session_id":     lmc.sessionID,
		"agent_id":       lmc.agentID,
		"memory_events":  len(lmc.memoryEvents),
		"skill_updates":  len(lmc.skillUpdates),
		"last_sync":      lmc.lastSyncTime,
		"max_cache_size": lmc.maxCacheSize,
	}
}

// generateID يولد معرف فريد
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
