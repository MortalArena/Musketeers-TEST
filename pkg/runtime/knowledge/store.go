package knowledge

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type KnowledgeStore interface {
	Working() WorkingMemory
	Semantic() SemanticMemory
	Episodic() EpisodicMemory
	Procedural() ProceduralMemory
}

type WorkingMemory interface {
	Set(key string, value any, ttl time.Duration) error
	Get(key string) (any, bool)
	Delete(key string) error
	Clear() error
}

type SemanticMemory interface {
	Store(text string, embedding []float32, metadata map[string]any) error
	Search(query string, limit int) ([]MemoryEntry, error)
	SearchByEmbedding(embedding []float32, limit int) ([]MemoryEntry, error)
}

type EpisodicMemory interface {
	Record(episode Episode) error
	Recall(query string, timeRange *TimeRange) ([]Episode, error)
	Forget(before time.Time) error
}

type ProceduralMemory interface {
	StoreProcedure(name string, steps []ProcedureStep) error
	GetProcedure(name string) ([]ProcedureStep, error)
	ListProcedures() ([]string, error)
}

type MemoryEntry struct {
	ID        string         `json:"id"`
	Text      string         `json:"text"`
	Embedding []float32      `json:"embedding,omitempty"`
	Metadata  map[string]any `json:"metadata"`
	Score     float64        `json:"score"`
	Timestamp time.Time      `json:"timestamp"`
}

type Episode struct {
	ID        string         `json:"id"`
	Timestamp time.Time      `json:"timestamp"`
	Context   map[string]any `json:"context"`
	Outcome   string         `json:"outcome"`
	Embedding []float32      `json:"embedding,omitempty"`
}

type TimeRange struct {
	Start time.Time
	End   time.Time
}

type ProcedureStep struct {
	ID          string         `json:"id"`
	Action      string         `json:"action"`
	Parameters  map[string]any `json:"parameters"`
	Description string         `json:"description"`
}

type DefaultKnowledgeStore struct {
	working    WorkingMemory
	semantic   SemanticMemory
	episodic   EpisodicMemory
	procedural ProceduralMemory
}

func NewDefaultKnowledgeStore() *DefaultKnowledgeStore {
	return &DefaultKnowledgeStore{
		working:    NewInMemoryWorkingMemory(),
		semantic:   NewInMemorySemanticMemory(),
		episodic:   NewInMemoryEpisodicMemory(),
		procedural: NewInMemoryProceduralMemory(),
	}
}

func (s *DefaultKnowledgeStore) Working() WorkingMemory       { return s.working }
func (s *DefaultKnowledgeStore) Semantic() SemanticMemory     { return s.semantic }
func (s *DefaultKnowledgeStore) Episodic() EpisodicMemory     { return s.episodic }
func (s *DefaultKnowledgeStore) Procedural() ProceduralMemory { return s.procedural }

type InMemoryWorkingMemory struct {
	mu     sync.RWMutex
	data   map[string]workingEntry
	stopCh chan struct{}
}

type workingEntry struct {
	value     any
	expiresAt time.Time
}

func NewInMemoryWorkingMemory() *InMemoryWorkingMemory {
	mem := &InMemoryWorkingMemory{data: make(map[string]workingEntry), stopCh: make(chan struct{})}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				_ = r
			}
		}()
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				mem.cleanup()
			case <-mem.stopCh:
				return
			}
		}
	}()
	return mem
}

func (m *InMemoryWorkingMemory) Set(key string, value any, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = workingEntry{value: value, expiresAt: time.Now().Add(ttl)}
	return nil
}

func (m *InMemoryWorkingMemory) Get(key string) (any, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, exists := m.data[key]
	if !exists {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.value, true
}

func (m *InMemoryWorkingMemory) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *InMemoryWorkingMemory) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string]workingEntry)
	return nil
}

func (m *InMemoryWorkingMemory) Close() error {
	close(m.stopCh)
	return nil
}

func (m *InMemoryWorkingMemory) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for key, entry := range m.data {
		if now.After(entry.expiresAt) {
			delete(m.data, key)
		}
	}
}

type InMemorySemanticMemory struct {
	mu      sync.RWMutex
	entries []MemoryEntry
}

func NewInMemorySemanticMemory() *InMemorySemanticMemory {
	return &InMemorySemanticMemory{entries: make([]MemoryEntry, 0)}
}

func (m *InMemorySemanticMemory) Store(text string, embedding []float32, metadata map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if metadata == nil {
		metadata = map[string]any{}
	}
	m.entries = append(m.entries, MemoryEntry{
		ID:        generateKnowledgeID(),
		Text:      text,
		Embedding: append([]float32(nil), embedding...),
		Metadata:  metadata,
		Timestamp: time.Now().UTC(),
	})
	return nil
}

func (m *InMemorySemanticMemory) Search(query string, limit int) ([]MemoryEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	query = strings.ToLower(strings.TrimSpace(query))
	results := make([]MemoryEntry, 0)
	for _, entry := range m.entries {
		if query == "" || strings.Contains(strings.ToLower(entry.Text), query) {
			results = append(results, entry)
		}
	}
	return limitEntries(results, limit), nil
}

func (m *InMemorySemanticMemory) SearchByEmbedding(embedding []float32, limit int) ([]MemoryEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	type scoredEntry struct {
		entry MemoryEntry
		score float64
	}
	scored := make([]scoredEntry, 0, len(m.entries))
	for _, entry := range m.entries {
		score := cosineSimilarity(embedding, entry.Embedding)
		scored = append(scored, scoredEntry{entry: entry, score: score})
	}
	sort.Slice(scored, func(i, j int) bool { return scored[i].score > scored[j].score })
	results := make([]MemoryEntry, 0, len(scored))
	for _, item := range scored {
		entry := item.entry
		entry.Score = item.score
		results = append(results, entry)
	}
	return limitEntries(results, limit), nil
}

type InMemoryEpisodicMemory struct {
	mu       sync.RWMutex
	episodes []Episode
}

func NewInMemoryEpisodicMemory() *InMemoryEpisodicMemory {
	return &InMemoryEpisodicMemory{episodes: make([]Episode, 0)}
}

func (m *InMemoryEpisodicMemory) Record(episode Episode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if episode.ID == "" {
		episode.ID = generateKnowledgeID()
	}
	if episode.Timestamp.IsZero() {
		episode.Timestamp = time.Now().UTC()
	}
	if episode.Context == nil {
		episode.Context = map[string]any{}
	}
	m.episodes = append(m.episodes, episode)
	return nil
}

func (m *InMemoryEpisodicMemory) Recall(query string, timeRange *TimeRange) ([]Episode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	query = strings.ToLower(strings.TrimSpace(query))
	results := make([]Episode, 0)
	for _, episode := range m.episodes {
		if timeRange != nil {
			if episode.Timestamp.Before(timeRange.Start) || episode.Timestamp.After(timeRange.End) {
				continue
			}
		}
		if query != "" && !strings.Contains(strings.ToLower(fmt.Sprintf("%v", episode.Context)), query) {
			continue
		}
		results = append(results, episode)
	}
	return results, nil
}

func (m *InMemoryEpisodicMemory) Forget(before time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	remaining := make([]Episode, 0, len(m.episodes))
	for _, episode := range m.episodes {
		if episode.Timestamp.After(before) {
			remaining = append(remaining, episode)
		}
	}
	m.episodes = remaining
	return nil
}

type InMemoryProceduralMemory struct {
	mu         sync.RWMutex
	procedures map[string][]ProcedureStep
}

func NewInMemoryProceduralMemory() *InMemoryProceduralMemory {
	return &InMemoryProceduralMemory{procedures: make(map[string][]ProcedureStep)}
}

func (m *InMemoryProceduralMemory) StoreProcedure(name string, steps []ProcedureStep) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([]ProcedureStep, len(steps))
	for i, step := range steps {
		if step.ID == "" {
			step.ID = fmt.Sprintf("step-%d", i+1)
		}
		if step.Parameters == nil {
			step.Parameters = map[string]any{}
		}
		copied[i] = step
	}
	m.procedures[name] = copied
	return nil
}

func (m *InMemoryProceduralMemory) GetProcedure(name string) ([]ProcedureStep, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	steps, exists := m.procedures[name]
	if !exists {
		return nil, fmt.Errorf("procedure not found: %s", name)
	}
	return append([]ProcedureStep(nil), steps...), nil
}

func (m *InMemoryProceduralMemory) ListProcedures() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.procedures))
	for name := range m.procedures {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func limitEntries(entries []MemoryEntry, limit int) []MemoryEntry {
	if limit <= 0 || limit > len(entries) {
		return entries
	}
	return entries[:limit]
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

var knowledgeSeq uint64

func generateKnowledgeID() string {
	return fmt.Sprintf("mem-%d", atomic.AddUint64(&knowledgeSeq, 1))
}
