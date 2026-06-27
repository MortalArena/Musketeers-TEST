package thinking

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/providers"
	"go.uber.org/zap"
)

// RerankSignal represents a source of relevance signal
type RerankSignal int

const (
	SignalBM25     RerankSignal = iota
	SignalEmbedding
	SignalSymbol
	SignalPackage
	SignalRecency
)

// RerankedChunk extends CodeChunk with a relevance score
type RerankedChunk struct {
	*CodeChunk
	Score           float64
	SignalBreakdown map[RerankSignal]float64
}

// ContextReranker is the main entry point for context-aware code search
type ContextReranker struct {
	index   *CodeIndex
	gen     *EmbeddingGenerator
	logger  *zap.Logger
	mu      sync.RWMutex
	ready   bool

	// Provider for real embeddings
	provider providers.Provider

	// Workspace watcher
	watchInterval time.Duration
	watchRunning  bool
	watchStop     chan struct{}

	// Index persistence
	indexPath string

	// Last index refresh time
	lastRefresh time.Time
}

// NewContextReranker creates a new context reranker
func NewContextReranker(projectRoot string, logger *zap.Logger) *ContextReranker {
	indexPath := filepath.Join(projectRoot, ".musketeers", "code_index.json")
	return &ContextReranker{
		index:         NewCodeIndex(projectRoot, logger),
		gen:           NewEmbeddingGenerator(1536),
		logger:        logger,
		watchInterval: 30 * time.Second,
		indexPath:     indexPath,
	}
}

// SetProvider sets the embedding provider for real embeddings
func (cr *ContextReranker) SetProvider(p providers.Provider) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.provider = p
	cr.index.SetProvider(p)
}

// SetIndexPath sets custom persistence path
func (cr *ContextReranker) SetIndexPath(path string) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.indexPath = path
	cr.index.SetIndexPath(path)
}

// EnsureIndexed builds the index if not already done — loads from disk first
func (cr *ContextReranker) EnsureIndexed(ctx context.Context) error {
	cr.mu.RLock()
	if cr.ready {
		cr.mu.RUnlock()
		return nil
	}
	cr.mu.RUnlock()

	cr.mu.Lock()
	defer cr.mu.Unlock()
	if cr.ready { return nil }

	// Try loading from disk first
	if cr.indexPath != "" {
		if err := cr.index.LoadIndex(cr.indexPath); err == nil && len(cr.index.chunks) > 0 {
			cr.logger.Info("تم تحميل الفهرس من الذاكرة", zap.Int("chunks", len(cr.index.chunks)))
		} else {
			// Build from scratch
			if err := cr.index.IndexProject(ctx); err != nil {
				return fmt.Errorf("فشل بناء الفهرس: %w", err)
			}
			if err := cr.index.SaveIndex(); err != nil {
				cr.logger.Warn("فشل حفظ الفهرس", zap.Error(err))
			}
		}
	} else {
		if err := cr.index.IndexProject(ctx); err != nil {
			return fmt.Errorf("فشل بناء الفهرس: %w", err)
		}
	}

	cr.lastRefresh = time.Now()
	cr.ready = true
	return nil
}

// Search performs multi-signal reranking search
func (cr *ContextReranker) Search(ctx context.Context, query string, opts *SearchOptions) ([]*RerankedChunk, error) {
	return cr.search(ctx, query, opts, false)
}

// SearchWithContext searches and enriches with surrounding context (like Cursor's @)
func (cr *ContextReranker) SearchWithContext(ctx context.Context, query string, opts *SearchOptions) ([]*RerankedChunk, error) {
	return cr.search(ctx, query, opts, true)
}

// search internal implementation
func (cr *ContextReranker) search(ctx context.Context, query string, opts *SearchOptions, withSurrounding bool) ([]*RerankedChunk, error) {
	if opts == nil { opts = DefaultSearchOptions() }
	if err := cr.EnsureIndexed(ctx); err != nil { return nil, err }

	query = cleanQuery(query)
	queryLower := strings.ToLower(query)

	chunks := cr.index.Search(ctx, query, opts.MaxCandidates)
	qEmbed := cr.gen.GenerateEmbedding(query)

	var results []*RerankedChunk
	for _, chunk := range chunks {
		breakdown := make(map[RerankSignal]float64)

		// BM25 score (normalized)
		bm25 := cr.index.bm25Score(query, chunk)
		breakdown[SignalBM25] = (bm25 / (bm25 + 1)) * opts.Weights.Keywords
		var score float64
		score += breakdown[SignalBM25]

		// Embedding similarity
		if chunk.Embedding != nil && len(chunk.Embedding) == len(qEmbed) {
			breakdown[SignalEmbedding] = cr.gen.CosineSimilarity(qEmbed, chunk.Embedding) * opts.Weights.Embedding
			score += breakdown[SignalEmbedding]
		}

		// Symbol name match
		symScore := 0.0
		if chunk.Name != "" {
			nl := strings.ToLower(chunk.Name)
			switch {
			case nl == queryLower:
				symScore = 1.0
			case strings.Contains(nl, queryLower) || strings.Contains(queryLower, nl):
				symScore = 0.7
			case partialMatch(queryLower, nl):
				symScore = 0.4
			}
		}
		breakdown[SignalSymbol] = symScore * opts.Weights.Symbol
		score += breakdown[SignalSymbol]

		// Package match
		pkgScore := 0.0
		if chunk.Package != "" && strings.Contains(queryLower, chunk.Package) {
			pkgScore = 0.8
		}
		breakdown[SignalPackage] = pkgScore * opts.Weights.Package
		score += breakdown[SignalPackage]

		// Recency
		recentScore := 0.5
		breakdown[SignalRecency] = recentScore * opts.Weights.Recency
		score += breakdown[SignalRecency]

		results = append(results, &RerankedChunk{
			CodeChunk:       chunk,
			Score:           score,
			SignalBreakdown: breakdown,
		})
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if len(results) > opts.MaxResults { results = results[:opts.MaxResults] }

	// Expand with surrounding context if requested
	if withSurrounding && len(results) > 0 {
		results = cr.expandWithContext(results)
	}

	return results, nil
}

// expandWithContext adds surrounding chunks for richer context (like Cursor's @ expands)
func (cr *ContextReranker) expandWithContext(chunks []*RerankedChunk) []*RerankedChunk {
	cr.index.mu.RLock()
	byPath := cr.index.chunkByPath
	cr.index.mu.RUnlock()

	seen := make(map[string]bool)
	var expanded []*RerankedChunk

	for _, ch := range chunks {
		if !seen[ch.ID] {
			seen[ch.ID] = true
			expanded = append(expanded, ch)
		}

		// Add neighboring chunks from same file (2 before/after)
		fileChunks := byPath[ch.FilePath]
		if len(fileChunks) < 2 { continue }

		chunkIdx := -1
		for i, fc := range fileChunks {
			if fc.ID == ch.ID { chunkIdx = i; break }
		}
		if chunkIdx < 0 { continue }

		for offset := -2; offset <= 2; offset++ {
			if offset == 0 { continue }
			idx := chunkIdx + offset
			if idx < 0 || idx >= len(fileChunks) { continue }
			if !seen[fileChunks[idx].ID] {
				seen[fileChunks[idx].ID] = true
				expanded = append(expanded, &RerankedChunk{
					CodeChunk: fileChunks[idx],
					Score:     ch.Score * 0.5, // context chunks get half score
				})
			}
		}
	}

	return expanded
}

// Query parses @-style queries and returns formatted results
func (cr *ContextReranker) Query(ctx context.Context, rawQuery string) (*QueryResult, error) {
	start := time.Now()
	if err := cr.EnsureIndexed(ctx); err != nil { return nil, err }

	qType := detectQueryType(rawQuery)
	query := cleanQuery(rawQuery)

	var chunks []*RerankedChunk
	var err error

	switch qType {
	case queryTypeSearch:
		chunks, err = cr.Search(ctx, query, nil)
	case queryTypeContext:
		chunks, err = cr.Search(ctx, query, &SearchOptions{
			MaxCandidates: 5, MaxResults: 5,
			Weights: &Weights{Embedding: 0.4, Keywords: 0.3, Symbol: 0.2, Package: 0.05, Recency: 0.05},
		})
	case queryTypeSymbol:
		chunks, err = cr.searchByName(ctx, query)
	}

	if err != nil { return nil, err }

	result := &QueryResult{
		Query: rawQuery, QueryType: qType.String(),
		Chunks: chunks, TotalFound: len(chunks),
		Duration: time.Since(start),
	}
	result.Summary = formatResultSummary(chunks, query)
	return result, nil
}

// searchByName searches by exact symbol name match
func (cr *ContextReranker) searchByName(ctx context.Context, name string) ([]*RerankedChunk, error) {
	queryLower := strings.ToLower(name)
	var results []*RerankedChunk

	chunks := cr.index.Search(ctx, name, 20)
	for _, chunk := range chunks {
		symScore := 0.0
		if chunk.Name != "" {
			nl := strings.ToLower(chunk.Name)
			switch {
			case nl == queryLower:
				symScore = 1.0
			case strings.Contains(nl, queryLower):
				symScore = 0.8
			}
		}
		if symScore > 0 {
			results = append(results, &RerankedChunk{
				CodeChunk: chunk, Score: symScore,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if len(results) > 10 { results = results[:10] }
	return results, nil
}

// ExtractQuery detects @-style queries in user input
func (cr *ContextReranker) ExtractQuery(userInput string) (string, bool) {
	re := regexp.MustCompile(`@([\w/\-.]+(?:\s*[\w/\-.]+)*)|\bcontext:\s*(.+?)(?:\n|$)`)

	matches := re.FindStringSubmatch(userInput)
	if matches == nil { return "", false }

	if matches[1] != "" { return strings.TrimSpace(matches[1]), true }
	if matches[2] != "" { return strings.TrimSpace(matches[2]), true }
	return "", false
}

// LazyResolveSymbol tries to find a symbol definition by name (like Cursor's @ goto)
func (cr *ContextReranker) LazyResolveSymbol(name string) *CodeChunk {
	chunks := cr.index.SearchBySymbol(name)
	if len(chunks) == 0 { return nil }
	return chunks[0]
}

// StartWorkspaceWatcher starts a goroutine that periodically refreshes the index
func (cr *ContextReranker) StartWorkspaceWatcher(ctx context.Context) {
	cr.mu.Lock()
	if cr.watchRunning {
		cr.mu.Unlock()
		return
	}
	cr.watchRunning = true
	cr.watchStop = make(chan struct{})
	cr.mu.Unlock()

	go func() {
		ticker := time.NewTicker(cr.watchInterval)
		defer ticker.Stop()
		defer func() {
			cr.mu.Lock()
			cr.watchRunning = false
			cr.mu.Unlock()
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case <-cr.watchStop:
				return
			case <-ticker.C:
				if err := cr.refreshIndex(ctx); err != nil {
					cr.logger.Warn("فشل تحديث الفهرس", zap.Error(err))
				}
			}
		}
	}()

	cr.logger.Info("بدء مراقبة مساحة العمل", zap.Duration("interval", cr.watchInterval))
}

// StopWorkspaceWatcher stops the workspace watcher
func (cr *ContextReranker) StopWorkspaceWatcher() {
	cr.mu.RLock()
	if !cr.watchRunning {
		cr.mu.RUnlock()
		return
	}
	stop := cr.watchStop
	cr.mu.RUnlock()

	close(stop)
}

// refreshIndex checks for file changes and re-indexes
func (cr *ContextReranker) refreshIndex(ctx context.Context) error {
	if !cr.ready { return nil }

	// Simple check: re-index if project files changed
	// In production, use fsnotify for real-time file watching
	cr.mu.RLock()
	chunks := cr.index.chunks
	cr.mu.RUnlock()

	needsRefresh := false
	checked := 0

	for _, chunk := range chunks {
		checked++
		fullPath := filepath.Join(cr.index.projectRoot, chunk.FilePath)
		info, err := os.Stat(fullPath)
		if err != nil {
			needsRefresh = true
			break
		}
		if info.ModTime().After(chunk.FileModTime) {
			needsRefresh = true
			break
		}
		if checked > 100 { break } // sample check for performance
	}

	if needsRefresh {
		cr.logger.Info("تم الكشف عن تغييرات — إعادة فهرسة")
		if err := cr.index.IndexProject(ctx); err != nil {
			return err
		}
		if err := cr.index.SaveIndex(); err != nil {
			cr.logger.Warn("فشل حفظ الفهرس", zap.Error(err))
		}
		cr.lastRefresh = time.Now()
	}

	return nil
}

// GetIndex returns the underlying code index
func (cr *ContextReranker) GetIndex() *CodeIndex {
	return cr.index
}

// GetLastRefresh returns last refresh time
func (cr *ContextReranker) GetLastRefresh() time.Time {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.lastRefresh
}

// --- Query types ---

type QueryType int

const (
	queryTypeSearch  QueryType = iota
	queryTypeContext
	queryTypeSymbol
)

func (q QueryType) String() string {
	switch q {
	case queryTypeSearch: return "search"
	case queryTypeContext: return "context"
	case queryTypeSymbol: return "symbol"
	}
	return "unknown"
}

func detectQueryType(q string) QueryType {
	if strings.HasPrefix(q, "context:") || strings.HasPrefix(q, "ctx:") {
		return queryTypeContext
	}
	if !strings.Contains(q, " ") && len(q) > 1 {
		return queryTypeSymbol
	}
	return queryTypeSearch
}

// --- Options ---

type SearchOptions struct {
	MaxCandidates int
	MaxResults    int
	Weights       *Weights
}

type Weights struct {
	Embedding float64
	Keywords  float64
	Symbol    float64
	Package   float64
	Recency   float64
}

var DefaultWeights = &Weights{
	Embedding: 0.35, Keywords: 0.35, Symbol: 0.2, Package: 0.05, Recency: 0.05,
}

func DefaultSearchOptions() *SearchOptions {
	w := *DefaultWeights
	return &SearchOptions{
		MaxCandidates: 50, MaxResults: 10,
		Weights: &w,
	}
}

// --- Results ---

type QueryResult struct {
	Query      string           `json:"query"`
	QueryType  string           `json:"query_type"`
	Chunks     []*RerankedChunk `json:"chunks"`
	Summary    string           `json:"summary"`
	TotalFound int              `json:"total_found"`
	Duration   time.Duration    `json:"duration"`
}

// helpers
func cleanQuery(q string) string {
	q = strings.TrimSpace(q)
	q = strings.TrimPrefix(q, "@")
	q = strings.TrimPrefix(q, "context:")
	q = strings.TrimPrefix(q, "ctx:")
	return strings.TrimSpace(q)
}

func partialMatch(a, b string) bool {
	if len(a) < 2 || len(b) < 2 { return false }
	return strings.Contains(a, b) || strings.Contains(b, a) || editDistance(a, b) <= 2
}

func editDistance(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 { return lb }
	if lb == 0 { return la }
	dp := make([][]int, la+1)
	for i := range dp {
		dp[i] = make([]int, lb+1)
		dp[i][0] = i
	}
	for j := range dp[0] { dp[0][j] = j }
	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 0
			if a[i-1] != b[j-1] { cost = 1 }
			dp[i][j] = min(dp[i-1][j]+1, dp[i][j-1]+1, dp[i-1][j-1]+cost)
		}
	}
	return dp[la][lb]
}

func min(a ...int) int {
	m := a[0]
	for _, v := range a[1:] {
		if v < m { m = v }
	}
	return m
}

func formatResultSummary(chunks []*RerankedChunk, query string) string {
	if len(chunks) == 0 {
		return fmt.Sprintf("لم أجد نتائج لـ \"%s\"", query)
	}

	fileGroups := make(map[string][]string)
	for _, c := range chunks {
		sym := c.Name
		if sym == "" { sym = fmt.Sprintf("line %d", c.StartLine) }
		fileGroups[c.FilePath] = append(fileGroups[c.FilePath],
			fmt.Sprintf("%s (score: %.2f)", sym, c.Score))
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("نتائج البحث عن \"%s\":\n", query))
	for path, syms := range fileGroups {
		b.WriteString(fmt.Sprintf("  %s:\n", path))
		for _, s := range syms {
			b.WriteString(fmt.Sprintf("    - %s\n", s))
		}
	}
	return b.String()
}
