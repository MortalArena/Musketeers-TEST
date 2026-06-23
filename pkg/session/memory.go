package session

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
)

// CollectiveMemory الذاكرة الجماعية - العقل الجمعي للجلسة
type CollectiveMemory struct {
	SessionID string `json:"session_id"`

	// 4 أنواع من الذاكرة
	Episodic   []MemoryEvent    `json:"episodic"`   // أحداث (ماذا حدث؟)
	Semantic   []MemoryFact     `json:"semantic"`   // حقائق (ماذا نعرف؟)
	Procedural []MemoryWorkflow `json:"procedural"` // طرق (كيف نفعل؟)
	Meta       []MemoryStrategy `json:"meta"`       // استراتيجيات (كيف نفكر؟)

	// المعرفة الجماعية - ملفات/لينكات/داتا من العميل البشري
	Knowledge []KnowledgeItem `json:"knowledge"` // ملفات المشروع والموارد

	// إحصائيات
	TotalEvents     int `json:"total_events"`
	TotalFacts      int `json:"total_facts"`
	TotalWorkflows  int `json:"total_workflows"`
	TotalStrategies int `json:"total_strategies"`
	TotalKnowledge  int `json:"total_knowledge"`

	DB *badger.DB
	mu sync.RWMutex
}

// [SAFETY] حدود الموارد لمنع استهلاك غير محدود
const (
	// [SAFETY] الحد الأقصى لعدد الأحداث
	MaxEpisodicEvents = 10000
	// [SAFETY] الحد الأقصى لعدد الحقائق
	MaxSemanticFacts = 5000
	// [SAFETY] الحد الأقصى لعدد الورك فلو
	MaxProceduralWorkflows = 1000
	// [SAFETY] الحد الأقصى لعدد الاستراتيجيات
	MaxMetaStrategies = 500
	// [SAFETY] الحد الأقصى لعدد عناصر المعرفة
	MaxKnowledgeItems = 1000
	// [SAFETY] الحد الأقصى لقيمة الثقة
	MaxConfidence = 1.0
	// [SAFETY] الحد الأدنى لقيمة الثقة
	MinConfidence = 0.0
)

// MemoryEvent حدث في الذاكرة العرضية
type MemoryEvent struct {
	ID         string                 `json:"id"`
	Timestamp  time.Time              `json:"timestamp"`
	AgentDID   string                 `json:"agent_did"`
	Action     string                 `json:"action"`
	Context    map[string]interface{} `json:"context"`
	Outcome    string                 `json:"outcome"` // success, failure, partial
	Lessons    []string               `json:"lessons"`
	Confidence float64                `json:"confidence"` // 0.0 - 1.0
	Tags       []string               `json:"tags"`
}

// MemoryFact حقيقة في الذاكرة الدلالية
type MemoryFact struct {
	ID         string    `json:"id"`
	Statement  string    `json:"statement"`
	Category   string    `json:"category"` // technical, business, user, etc.
	Confidence float64   `json:"confidence"`
	Source     string    `json:"source"`
	VerifiedBy []string  `json:"verified_by"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Tags       []string  `json:"tags"`
}

// MemoryWorkflow workflow في الذاكرة الإجرائية
type MemoryWorkflow struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Steps       []WorkflowStep `json:"steps"`
	SuccessRate float64        `json:"success_rate"` // 0.0 - 1.0
	AvgDuration time.Duration  `json:"avg_duration"`
	UsedCount   int            `json:"used_count"`
	CreatedAt   time.Time      `json:"created_at"`
	Tags        []string       `json:"tags"`
}

// WorkflowStep خطوة في workflow
type WorkflowStep struct {
	Order          int           `json:"order"`
	Action         string        `json:"action"`
	AgentType      string        `json:"agent_type"`
	ExpectedOutput string        `json:"expected_output"`
	Timeout        time.Duration `json:"timeout"`
}

// MemoryStrategy استراتيجية في الذاكرة الوصفية
type MemoryStrategy struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	WhenToUse     string    `json:"when_to_use"`
	HowToUse      string    `json:"how_to_use"`
	Effectiveness float64   `json:"effectiveness"` // 0.0 - 1.0
	Examples      []string  `json:"examples"`
	CreatedAt     time.Time `json:"created_at"`
}

// KnowledgeItem عنصر معرفة من العميل البشري (ملف/لينك/داتا)
type KnowledgeItem struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // file, link, data, image, document
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Content     string    `json:"content"`                // محتوى Markdown المحول
	OriginalURL string    `json:"original_url,omitempty"` // الرابط الأصلي (إن وجد)
	FilePath    string    `json:"file_path,omitempty"`    // مسار الملف المحلي
	ProcessedAt time.Time `json:"processed_at"`
	ProcessedBy string    `json:"processed_by"` // الوكيل الذي قام بالتحويل
	Category    string    `json:"category"`     // requirements, design, reference, etc.
	Tags        []string  `json:"tags"`
	Priority    int       `json:"priority"` // 1-10
}

// NewCollectiveMemory ينشئ ذاكرة جماعية جديدة
func NewCollectiveMemory(sessionID string, db *badger.DB) *CollectiveMemory {
	return &CollectiveMemory{
		SessionID:  sessionID,
		Episodic:   make([]MemoryEvent, 0),
		Semantic:   make([]MemoryFact, 0),
		Procedural: make([]MemoryWorkflow, 0),
		Meta:       make([]MemoryStrategy, 0),
		Knowledge:  make([]KnowledgeItem, 0),
		DB:         db,
	}
}

// RecordEvent يسجل حدثاً في الذاكرة العرضية
func (cm *CollectiveMemory) RecordEvent(event MemoryEvent) error {
	// [SAFETY] التحقق من صحة المدخلات
	if event.AgentDID == "" {
		return fmt.Errorf("agent DID cannot be empty")
	}
	if event.Action == "" {
		return fmt.Errorf("action cannot be empty")
	}
	if event.Outcome == "" {
		return fmt.Errorf("outcome cannot be empty")
	}
	if event.Confidence < MinConfidence || event.Confidence > MaxConfidence {
		return fmt.Errorf("confidence must be between %.1f and %.1f", MinConfidence, MaxConfidence)
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// [SAFETY] التحقق من الحد الأقصى للأحداث
	if len(cm.Episodic) >= MaxEpisodicEvents {
		return fmt.Errorf("maximum episodic events limit reached (%d)", MaxEpisodicEvents)
	}

	event.ID = fmt.Sprintf("evt_%d", len(cm.Episodic)+1)
	event.Timestamp = time.Now()

	cm.Episodic = append(cm.Episodic, event)
	cm.TotalEvents++

	data, _ := json.Marshal(event)
	return cm.DB.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(event.ID), data)
	})
}

// LearnFact يتعلم حقيقة جديدة في الذاكرة الدلالية
func (cm *CollectiveMemory) LearnFact(fact MemoryFact) error {
	// [SAFETY] التحقق من صحة المدخلات
	if fact.Statement == "" {
		return fmt.Errorf("statement cannot be empty")
	}
	if fact.Category == "" {
		return fmt.Errorf("category cannot be empty")
	}
	if fact.Confidence < MinConfidence || fact.Confidence > MaxConfidence {
		return fmt.Errorf("confidence must be between %.1f and %.1f", MinConfidence, MaxConfidence)
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// [SAFETY] التحقق من الحد الأقصى للحقائق
	if len(cm.Semantic) >= MaxSemanticFacts {
		return fmt.Errorf("maximum semantic facts limit reached (%d)", MaxSemanticFacts)
	}

	// التحقق من عدم التكرار
	for i, existing := range cm.Semantic {
		if existing.Statement == fact.Statement {
			// تحديث الثقة إذا كانت أعلى
			if fact.Confidence > existing.Confidence {
				cm.Semantic[i].Confidence = fact.Confidence
				cm.Semantic[i].UpdatedAt = time.Now()
			}
			return nil
		}
	}

	fact.ID = fmt.Sprintf("fact_%d", len(cm.Semantic)+1)
	fact.CreatedAt = time.Now()
	fact.UpdatedAt = time.Now()

	cm.Semantic = append(cm.Semantic, fact)
	cm.TotalFacts++

	data, _ := json.Marshal(fact)
	return cm.DB.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(fact.ID), data)
	})
}

// DiscoverWorkflow يكتشف workflow جديد في الذاكرة الإجرائية
func (cm *CollectiveMemory) DiscoverWorkflow(workflow MemoryWorkflow) error {
	// [SAFETY] التحقق من صحة المدخلات
	if workflow.Name == "" {
		return fmt.Errorf("workflow name cannot be empty")
	}
	if workflow.SuccessRate < MinConfidence || workflow.SuccessRate > MaxConfidence {
		return fmt.Errorf("success rate must be between %.1f and %.1f", MinConfidence, MaxConfidence)
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// [SAFETY] التحقق من الحد الأقصى للورك فلو
	if len(cm.Procedural) >= MaxProceduralWorkflows {
		return fmt.Errorf("maximum procedural workflows limit reached (%d)", MaxProceduralWorkflows)
	}

	workflow.ID = fmt.Sprintf("wf_%d", len(cm.Procedural)+1)
	workflow.CreatedAt = time.Now()

	cm.Procedural = append(cm.Procedural, workflow)
	cm.TotalWorkflows++

	data, _ := json.Marshal(workflow)
	return cm.DB.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(workflow.ID), data)
	})
}

// DevelopStrategy يطور استراتيجية جديدة في الذاكرة الوصفية
func (cm *CollectiveMemory) DevelopStrategy(strategy MemoryStrategy) error {
	// [SAFETY] التحقق من صحة المدخلات
	if strategy.Name == "" {
		return fmt.Errorf("strategy name cannot be empty")
	}
	if strategy.WhenToUse == "" {
		return fmt.Errorf("when to use cannot be empty")
	}
	if strategy.HowToUse == "" {
		return fmt.Errorf("how to use cannot be empty")
	}
	if strategy.Effectiveness < MinConfidence || strategy.Effectiveness > MaxConfidence {
		return fmt.Errorf("effectiveness must be between %.1f and %.1f", MinConfidence, MaxConfidence)
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// [SAFETY] التحقق من الحد الأقصى للاستراتيجيات
	if len(cm.Meta) >= MaxMetaStrategies {
		return fmt.Errorf("maximum meta strategies limit reached (%d)", MaxMetaStrategies)
	}

	strategy.ID = fmt.Sprintf("strat_%d", len(cm.Meta)+1)
	strategy.CreatedAt = time.Now()

	cm.Meta = append(cm.Meta, strategy)
	cm.TotalStrategies++

	data, _ := json.Marshal(strategy)
	return cm.DB.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(strategy.ID), data)
	})
}

// GetBestWorkflow يعيد أفضل workflow لمهمة معينة
func (cm *CollectiveMemory) GetBestWorkflow(taskType string) *MemoryWorkflow {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var best *MemoryWorkflow
	var bestScore float64

	for i := range cm.Procedural {
		wf := &cm.Procedural[i]
		// حساب الدرجة: success_rate * (1 / avg_duration)
		score := wf.SuccessRate * (1.0 / float64(wf.AvgDuration.Seconds()+1))

		if score > bestScore {
			bestScore = score
			best = wf
		}
	}

	return best
}

// QueryEvents يبحث في الأحداث
func (cm *CollectiveMemory) QueryEvents(filters map[string]interface{}) []MemoryEvent {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var results []MemoryEvent
	for _, event := range cm.Episodic {
		if matchesFilters(event, filters) {
			results = append(results, event)
		}
	}
	return results
}

// matchesFilters يتحقق من تطابق الحدث مع الفلاتر
func matchesFilters(event MemoryEvent, filters map[string]interface{}) bool {
	for key, value := range filters {
		switch key {
		case "agent_did":
			if event.AgentDID != value.(string) {
				return false
			}
		case "outcome":
			if event.Outcome != value.(string) {
				return false
			}
		case "tags":
			tags := value.([]string)
			found := false
			for _, tag := range tags {
				for _, eventTag := range event.Tags {
					if tag == eventTag {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}

// Clone ينسخ الذاكرة (للتصدير/الاستيراد)
func (cm *CollectiveMemory) Clone() *CollectiveMemory {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	clone := &CollectiveMemory{
		SessionID:       cm.SessionID,
		TotalEvents:     cm.TotalEvents,
		TotalFacts:      cm.TotalFacts,
		TotalWorkflows:  cm.TotalWorkflows,
		TotalStrategies: cm.TotalStrategies,
		TotalKnowledge:  cm.TotalKnowledge,
		DB:              cm.DB,
	}

	clone.Episodic = make([]MemoryEvent, len(cm.Episodic))
	copy(clone.Episodic, cm.Episodic)

	clone.Semantic = make([]MemoryFact, len(cm.Semantic))
	copy(clone.Semantic, cm.Semantic)

	clone.Procedural = make([]MemoryWorkflow, len(cm.Procedural))
	copy(clone.Procedural, cm.Procedural)

	clone.Meta = make([]MemoryStrategy, len(cm.Meta))
	copy(clone.Meta, cm.Meta)

	clone.Knowledge = make([]KnowledgeItem, len(cm.Knowledge))
	copy(clone.Knowledge, cm.Knowledge)

	return clone
}

// AddKnowledge يضيف عنصر معرفة جديد (ملف/لينك/داتا)
func (cm *CollectiveMemory) AddKnowledge(item KnowledgeItem) error {
	// [SAFETY] التحقق من صحة المدخلات
	if item.Type == "" {
		return fmt.Errorf("knowledge type cannot be empty")
	}
	if item.Name == "" {
		return fmt.Errorf("knowledge name cannot be empty")
	}
	if item.Content == "" && item.OriginalURL == "" && item.FilePath == "" {
		return fmt.Errorf("knowledge must have content, URL, or file path")
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// [SAFETY] التحقق من الحد الأقصى
	if len(cm.Knowledge) >= MaxKnowledgeItems {
		return fmt.Errorf("maximum knowledge items limit reached (%d)", MaxKnowledgeItems)
	}

	item.ID = fmt.Sprintf("kno_%d", len(cm.Knowledge)+1)
	item.ProcessedAt = time.Now()

	cm.Knowledge = append(cm.Knowledge, item)
	cm.TotalKnowledge++

	// حفظ في DB
	data, _ := json.Marshal(item)
	return cm.DB.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(fmt.Sprintf("knowledge_%s_%s", cm.SessionID, item.ID)), data)
	})
}

// GetKnowledgeByCategory يحصل على المعرفة حسب الفئة
func (cm *CollectiveMemory) GetKnowledgeByCategory(category string) []KnowledgeItem {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var result []KnowledgeItem
	for _, item := range cm.Knowledge {
		if item.Category == category {
			result = append(result, item)
		}
	}
	return result
}

// GetKnowledgeByPriority يحصل على المعرفة حسب الأولوية
func (cm *CollectiveMemory) GetKnowledgeByPriority(minPriority int) []KnowledgeItem {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var result []KnowledgeItem
	for _, item := range cm.Knowledge {
		if item.Priority >= minPriority {
			result = append(result, item)
		}
	}
	return result
}

// SearchKnowledge يبحث في المعرفة حسب الكلمات المفتاحية
func (cm *CollectiveMemory) SearchKnowledge(query string) []KnowledgeItem {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var result []KnowledgeItem
	for _, item := range cm.Knowledge {
		// البحث في الاسم والوصف والمحتوى
		if contains(item.Name, query) || contains(item.Description, query) || contains(item.Content, query) {
			result = append(result, item)
		}
		// البحث في الوسوم
		for _, tag := range item.Tags {
			if contains(tag, query) {
				result = append(result, item)
				break
			}
		}
	}
	return result
}

// contains دالة مساعدة للبحث
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr))
}
