package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// MemoryEntry إدخال ذاكرة
type MemoryEntry struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // "fact", "lesson", "experience", "decision"
	Content     string                 `json:"content"`
	Source      string                 `json:"source"` // agent_id who created this memory
	Importance  float64                `json:"importance"` // 0.0 to 1.0
	AccessCount int                    `json:"access_count"`
	LastAccess  time.Time              `json:"last_access"`
	CreatedAt   time.Time              `json:"created_at"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	Tags        []string               `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// CollectiveMemory الذاكرة الجماعية للجلسة
type CollectiveMemory struct {
	memories   map[string]*MemoryEntry
	logger     *zap.Logger
	mu         sync.RWMutex
	sessionID  string
	maxSize    int
	ttl        time.Duration
}

// NewCollectiveMemory ينشئ ذاكرة جماعية جديدة
func NewCollectiveMemory(sessionID string, maxSize int, ttl time.Duration, logger *zap.Logger) *CollectiveMemory {
	return &CollectiveMemory{
		memories:  make(map[string]*MemoryEntry),
		logger:    logger,
		sessionID: sessionID,
		maxSize:   maxSize,
		ttl:       ttl,
	}
}

// AddMemory يضيف ذاكرة جديدة
func (cm *CollectiveMemory) AddMemory(ctx context.Context, memoryType, content, source string, importance float64, tags []string, metadata map[string]interface{}) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// [WHY] إضافة ذاكرة جديدة للذاكرة الجماعية
	// [HOW] يخزن الذاكرة مع معلومات الأهمية والوصول
	// [SAFETY] يتحقق من الحجم الأقصى وينظف الذكريات القديمة

	// تنظيف الذكريات المنتهية الصلاحية
	cm.cleanupExpiredMemories()

	// التحقق من الحجم الأقصى
	if len(cm.memories) >= cm.maxSize {
		cm.evictLeastImportant()
	}

	memory := &MemoryEntry{
		ID:          fmt.Sprintf("memory_%d", time.Now().UnixNano()),
		Type:        memoryType,
		Content:     content,
		Source:      source,
		Importance:  importance,
		AccessCount: 0,
		LastAccess:  time.Now(),
		CreatedAt:   time.Now(),
		Tags:        tags,
		Metadata:    metadata,
	}

	// تعيين وقت انتهاء الصلاحية
	if cm.ttl > 0 {
		expiresAt := time.Now().Add(cm.ttl)
		memory.ExpiresAt = &expiresAt
	}

	cm.memories[memory.ID] = memory

	cm.logger.Info("تم إضافة ذاكرة جديدة",
		zap.String("session_id", cm.sessionID),
		zap.String("memory_id", memory.ID),
		zap.String("type", memoryType),
		zap.String("source", source),
		zap.Float64("importance", importance),
	)

	return nil
}

// GetMemory يرجع ذاكرة محددة
func (cm *CollectiveMemory) GetMemory(ctx context.Context, memoryID string) (*MemoryEntry, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	memory, ok := cm.memories[memoryID]
	if !ok {
		return nil, fmt.Errorf("ذاكرة غير موجودة: %s", memoryID)
	}

	// تحديث معلومات الوصول
	memory.AccessCount++
	memory.LastAccess = time.Now()

	return memory, nil
}

// GetAllMemories يرجع جميع الذكريات
func (cm *CollectiveMemory) GetAllMemories(ctx context.Context) ([]*MemoryEntry, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	memories := make([]*MemoryEntry, 0, len(cm.memories))
	for _, memory := range cm.memories {
		memories = append(memories, memory)
	}

	return memories, nil
}

// GetMemoriesByType يرجع الذكريات حسب النوع
func (cm *CollectiveMemory) GetMemoriesByType(ctx context.Context, memoryType string) ([]*MemoryEntry, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var result []*MemoryEntry
	for _, memory := range cm.memories {
		if memory.Type == memoryType {
			result = append(result, memory)
		}
	}

	return result, nil
}

// GetMemoriesBySource يرجع الذكريات حسب المصدر
func (cm *CollectiveMemory) GetMemoriesBySource(ctx context.Context, source string) ([]*MemoryEntry, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var result []*MemoryEntry
	for _, memory := range cm.memories {
		if memory.Source == source {
			result = append(result, memory)
		}
	}

	return result, nil
}

// GetMemoriesByTags يرجع الذكريات حسب الوسوم
func (cm *CollectiveMemory) GetMemoriesByTags(ctx context.Context, tags []string) ([]*MemoryEntry, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var result []*MemoryEntry
	for _, memory := range cm.memories {
		// التحقق من أن جميع الوسوم موجودة
		allTagsPresent := true
		for _, tag := range tags {
			tagFound := false
			for _, memoryTag := range memory.Tags {
				if memoryTag == tag {
					tagFound = true
					break
				}
			}
			if !tagFound {
				allTagsPresent = false
				break
			}
		}

		if allTagsPresent {
			result = append(result, memory)
		}
	}

	return result, nil
}

// SearchMemories يبحث في الذكريات
func (cm *CollectiveMemory) SearchMemories(ctx context.Context, query string) ([]*MemoryEntry, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var result []*MemoryEntry
	for _, memory := range cm.memories {
		// بحث بسيط في المحتوى
		if contains(memory.Content, query) {
			result = append(result, memory)
		}
	}

	return result, nil
}

// GetImportantMemories يرجع الذكريات الأهم
func (cm *CollectiveMemory) GetImportantMemories(ctx context.Context, limit int) ([]*MemoryEntry, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	memories := make([]*MemoryEntry, 0, len(cm.memories))
	for _, memory := range cm.memories {
		memories = append(memories, memory)
	}

	// ترتيب الذكريات حسب الأهمية
	for i := 0; i < len(memories); i++ {
		for j := i + 1; j < len(memories); j++ {
			if memories[j].Importance > memories[i].Importance {
				memories[i], memories[j] = memories[j], memories[i]
			}
		}
	}

	if limit > len(memories) {
		limit = len(memories)
	}

	return memories[:limit], nil
}

// GetRecentMemories يرجع الذكريات الحديثة
func (cm *CollectiveMemory) GetRecentMemories(ctx context.Context, limit int) ([]*MemoryEntry, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	memories := make([]*MemoryEntry, 0, len(cm.memories))
	for _, memory := range cm.memories {
		memories = append(memories, memory)
	}

	// ترتيب الذكريات حسب وقت الإنشاء
	for i := 0; i < len(memories); i++ {
		for j := i + 1; j < len(memories); j++ {
			if memories[j].CreatedAt.After(memories[i].CreatedAt) {
				memories[i], memories[j] = memories[j], memories[i]
			}
		}
	}

	if limit > len(memories) {
		limit = len(memories)
	}

	return memories[:limit], nil
}

// UpdateMemory يحدث ذاكرة
func (cm *CollectiveMemory) UpdateMemory(ctx context.Context, memoryID string, content string, importance float64, tags []string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	memory, ok := cm.memories[memoryID]
	if !ok {
		return fmt.Errorf("ذاكرة غير موجودة: %s", memoryID)
	}

	memory.Content = content
	memory.Importance = importance
	memory.Tags = tags
	memory.LastAccess = time.Now()

	cm.logger.Info("تم تحديث الذاكرة",
		zap.String("session_id", cm.sessionID),
		zap.String("memory_id", memoryID),
	)

	return nil
}

// DeleteMemory يحذف ذاكرة
func (cm *CollectiveMemory) DeleteMemory(ctx context.Context, memoryID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	_, ok := cm.memories[memoryID]
	if !ok {
		return fmt.Errorf("ذاكرة غير موجودة: %s", memoryID)
	}

	delete(cm.memories, memoryID)

	cm.logger.Info("تم حذف الذاكرة",
		zap.String("session_id", cm.sessionID),
		zap.String("memory_id", memoryID),
	)

	return nil
}

// cleanupExpiredMemories ينظف الذكريات المنتهية الصلاحية
func (cm *CollectiveMemory) cleanupExpiredMemories() {
	now := time.Now()
	for id, memory := range cm.memories {
		if memory.ExpiresAt != nil && memory.ExpiresAt.Before(now) {
			delete(cm.memories, id)
			cm.logger.Info("تم حذف ذاكرة منتهية الصلاحية",
				zap.String("session_id", cm.sessionID),
				zap.String("memory_id", id),
			)
		}
	}
}

// evictLeastImportant يحذف الذكريات الأقل أهمية
func (cm *CollectiveMemory) evictLeastImportant() {
	if len(cm.memories) == 0 {
		return
	}

	// إيجاد الذاكرة الأقل أهمية
	var leastImportantID string
	minImportance := 1.0
	for id, memory := range cm.memories {
		if memory.Importance < minImportance {
			minImportance = memory.Importance
			leastImportantID = id
		}
	}

	if leastImportantID != "" {
		delete(cm.memories, leastImportantID)
		cm.logger.Info("تم حذف ذاكرة أقل أهمية",
			zap.String("session_id", cm.sessionID),
			zap.String("memory_id", leastImportantID),
		)
	}
}

// GetMemorySummary يرجع ملخص الذاكرة
func (cm *CollectiveMemory) GetMemorySummary(ctx context.Context) (map[string]interface{}, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// حساب الإحصائيات
	typeCounts := make(map[string]int)
	sourceCounts := make(map[string]int)
	totalImportance := 0.0

	for _, memory := range cm.memories {
		typeCounts[memory.Type]++
		sourceCounts[memory.Source]++
		totalImportance += memory.Importance
	}

	avgImportance := 0.0
	if len(cm.memories) > 0 {
		avgImportance = totalImportance / float64(len(cm.memories))
	}

	summary := map[string]interface{}{
		"session_id":        cm.sessionID,
		"total_memories":    len(cm.memories),
		"max_size":          cm.maxSize,
		"ttl":               cm.ttl.String(),
		"type_counts":       typeCounts,
		"source_counts":     sourceCounts,
		"avg_importance":    avgImportance,
		"total_importance":  totalImportance,
	}

	return summary, nil
}

// ExportMemories يصدر الذكريات كـ JSON
func (cm *CollectiveMemory) ExportMemories(ctx context.Context) ([]byte, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	memories := make([]*MemoryEntry, 0, len(cm.memories))
	for _, memory := range cm.memories {
		memories = append(memories, memory)
	}

	return json.Marshal(memories)
}

// ImportMemories يستورد الذكريات من JSON
func (cm *CollectiveMemory) ImportMemories(ctx context.Context, data []byte) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var memories []*MemoryEntry
	if err := json.Unmarshal(data, &memories); err != nil {
		return fmt.Errorf("فشل فك تشفير الذكريات: %w", err)
	}

	for _, memory := range memories {
		cm.memories[memory.ID] = memory
	}

	cm.logger.Info("تم استيراد الذكريات",
		zap.String("session_id", cm.sessionID),
		zap.Int("count", len(memories)),
	)

	return nil
}

// contains دالة مساعدة للبحث في النص
func contains(text, query string) bool {
	return len(text) >= len(query) && (text == query || len(query) > 0 && findSubstring(text, query))
}

// findSubstring دالة مساعدة للبحث عن نص فرعي
func findSubstring(text, query string) bool {
	for i := 0; i <= len(text)-len(query); i++ {
		if text[i:i+len(query)] == query {
			return true
		}
	}
	return false
}
