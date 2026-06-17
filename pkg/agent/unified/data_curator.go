package unified

import (
	"context"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// DataCurator نظام تنظيم البيانات
type DataCurator struct {
	sessionID string
	logger    *zap.Logger
	mu        sync.RWMutex
}

// NewDataCurator ينشئ نظام تنظيم بيانات جديد
func NewDataCurator(sessionID string, logger *zap.Logger) *DataCurator {
	return &DataCurator{
		sessionID: sessionID,
		logger:    logger,
	}
}

// CurateMemoryEvents ينظم أحداث الذاكرة
func (dc *DataCurator) CurateMemoryEvents(events []*MemoryEvent) []*MemoryEvent {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	curated := []*MemoryEvent{}

	for _, event := range events {
		// تصفية الأحداث غير المفيدة
		if dc.isUseful(event) {
			// تنظيف البيانات
			cleaned := dc.cleanEvent(event)
			curated = append(curated, cleaned)
		}
	}

	dc.logger.Info("تم تنظيم أحداث الذاكرة",
		zap.Int("original_count", len(events)),
		zap.Int("curated_count", len(curated)),
	)

	return curated
}

// isUseful يتحقق من أن الحدث مفيد
func (dc *DataCurator) isUseful(event *MemoryEvent) bool {
	// التحقق من أن الحدث ليس مكرراً
	if event.ID == "" {
		return false
	}

	// التحقق من أن الحدث ليس قديماً جداً
	if time.Since(event.Timestamp) > 24*time.Hour {
		return false
	}

	// التحقق من أن الحدث له أهمية
	if event.Confidence < 0.5 {
		return false
	}

	return true
}

// cleanEvent ينظف البيانات في الحدث
func (dc *DataCurator) cleanEvent(event *MemoryEvent) *MemoryEvent {
	cleaned := &MemoryEvent{
		ID:         event.ID,
		Timestamp:  event.Timestamp,
		AgentDID:   event.AgentDID,
		Action:     strings.TrimSpace(event.Action),
		Context:    dc.cleanContext(event.Context),
		Outcome:    strings.TrimSpace(event.Outcome),
		Lessons:    dc.cleanLessons(event.Lessons),
		Confidence: event.Confidence,
		Tags:       dc.cleanTags(event.Tags),
	}

	return cleaned
}

// cleanContext ينظف البيانات في السياق
func (dc *DataCurator) cleanContext(context map[string]interface{}) map[string]interface{} {
	cleaned := make(map[string]interface{})

	for key, value := range context {
		// إزالة البيانات الحساسة
		if dc.isSensitive(key) {
			continue
		}

		// تنظيف القيم
		cleaned[key] = dc.cleanValue(value)
	}

	return cleaned
}

// isSensitive يتحقق من أن المفتاح حساس
func (dc *DataCurator) isSensitive(key string) bool {
	sensitiveKeys := []string{
		"password", "token", "secret", "key", "credential",
		"api_key", "private_key", "auth",
	}

	for _, sensitive := range sensitiveKeys {
		if strings.Contains(strings.ToLower(key), sensitive) {
			return true
		}
	}

	return false
}

// cleanValue ينظف القيمة
func (dc *DataCurator) cleanValue(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case []string:
		cleaned := []string{}
		for _, s := range v {
			cleaned = append(cleaned, strings.TrimSpace(s))
		}
		return cleaned
	default:
		return v
	}
}

// cleanLessons ينظف الدروس
func (dc *DataCurator) cleanLessons(lessons []string) []string {
	cleaned := []string{}

	for _, lesson := range lessons {
		lesson = strings.TrimSpace(lesson)
		if lesson != "" {
			cleaned = append(cleaned, lesson)
		}
	}

	return cleaned
}

// cleanTags ينظف العلامات
func (dc *DataCurator) cleanTags(tags []string) []string {
	cleaned := []string{}

	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			cleaned = append(cleaned, tag)
		}
	}

	return cleaned
}

// CurateSkillUpdates ينظم تحديثات المهارات
func (dc *DataCurator) CurateSkillUpdates(updates []*SkillUpdate) []*SkillUpdate {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	curated := []*SkillUpdate{}

	for _, update := range updates {
		// تصفية التحديثات غير المفيدة
		if dc.isSkillUpdateUseful(update) {
			curated = append(curated, update)
		}
	}

	dc.logger.Info("تم تنظيم تحديثات المهارات",
		zap.Int("original_count", len(updates)),
		zap.Int("curated_count", len(curated)),
	)

	return curated
}

// isSkillUpdateUseful يتحقق من أن التحديث مفيد
func (dc *DataCurator) isSkillUpdateUseful(update *SkillUpdate) bool {
	// التحقق من أن التحديث ليس قديماً جداً
	if time.Since(update.Timestamp) > 24*time.Hour {
		return false
	}

	// التحقق من أن التحديث له تأثير
	if update.NewLevel <= update.OldLevel {
		return false
	}

	return true
}

// PrepareDataForAgents يجهز البيانات للوكلاء
func (dc *DataCurator) PrepareDataForAgents(ctx context.Context, events []*MemoryEvent, updates []*SkillUpdate) (map[string]interface{}, error) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	// تنظيم البيانات
	curatedEvents := dc.CurateMemoryEvents(events)
	curatedUpdates := dc.CurateSkillUpdates(updates)

	// تجهز البيانات للوكلاء
	preparedData := map[string]interface{}{
		"memory_events":   curatedEvents,
		"skill_updates":   curatedUpdates,
		"prepared_at":     time.Now(),
		"prepared_by":     "DataCurator",
		"session_id":      dc.sessionID,
	}

	dc.logger.Info("تم تجهيز البيانات للوكلاء",
		zap.Int("memory_events", len(curatedEvents)),
		zap.Int("skill_updates", len(curatedUpdates)),
	)

	return preparedData, nil
}
