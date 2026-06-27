package thinking

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
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

// Language info for file type detection
type LanguageInfo struct {
	Exts       []string
	Name       string
	SingleLine string // single-line comment prefix
	MultiLine  [2]string
}

var supportedLanguages = []LanguageInfo{
	{Exts: []string{".go"}, Name: "Go", SingleLine: "//", MultiLine: [2]string{"/*", "*/"}},
	{Exts: []string{".js", ".jsx", ".ts", ".tsx", ".mjs"}, Name: "JavaScript/TypeScript", SingleLine: "//", MultiLine: [2]string{"/*", "*/"}},
	{Exts: []string{".py"}, Name: "Python", SingleLine: "#", MultiLine: [2]string{`"""`, `"""`}},
	{Exts: []string{".java"}, Name: "Java", SingleLine: "//", MultiLine: [2]string{"/*", "*/"}},
	{Exts: []string{".rs"}, Name: "Rust", SingleLine: "//", MultiLine: [2]string{"/*", "*/"}},
	{Exts: []string{".cpp", ".cc", ".cxx", ".hpp", ".h"}, Name: "C++", SingleLine: "//", MultiLine: [2]string{"/*", "*/"}},
	{Exts: []string{".c"}, Name: "C", SingleLine: "//", MultiLine: [2]string{"/*", "*/"}},
	{Exts: []string{".cs"}, Name: "C#", SingleLine: "//", MultiLine: [2]string{"/*", "*/"}},
	{Exts: []string{".rb"}, Name: "Ruby", SingleLine: "#", MultiLine: [2]string{"=begin", "=end"}},
	{Exts: []string{".php"}, Name: "PHP", SingleLine: "//", MultiLine: [2]string{"/*", "*/"}},
	{Exts: []string{".swift"}, Name: "Swift", SingleLine: "//", MultiLine: [2]string{"/*", "*/"}},
	{Exts: []string{".kt", ".kts"}, Name: "Kotlin", SingleLine: "//", MultiLine: [2]string{"/*", "*/"}},
	{Exts: []string{".sh", ".bash", ".zsh"}, Name: "Shell", SingleLine: "#", MultiLine: [2]string{": '", "'"}},
	{Exts: []string{".md"}, Name: "Markdown", SingleLine: "", MultiLine: [2]string{}},
	{Exts: []string{".json"}, Name: "JSON", SingleLine: "", MultiLine: [2]string{}},
	{Exts: []string{".yaml", ".yml"}, Name: "YAML", SingleLine: "#", MultiLine: [2]string{}},
	{Exts: []string{".toml"}, Name: "TOML", SingleLine: "#", MultiLine: [2]string{}},
	{Exts: []string{".sql"}, Name: "SQL", SingleLine: "--", MultiLine: [2]string{"/*", "*/"}},
	{Exts: []string{".html", ".htm"}, Name: "HTML", SingleLine: "", MultiLine: [2]string{"<!--", "-->"}},
	{Exts: []string{".css", ".scss", ".less"}, Name: "CSS", SingleLine: "", MultiLine: [2]string{"/*", "*/"}},
}

var extToLang map[string]*LanguageInfo

func init() {
	extToLang = make(map[string]*LanguageInfo)
	for i := range supportedLanguages {
		lang := &supportedLanguages[i]
		for _, ext := range lang.Exts {
			extToLang[ext] = lang
		}
	}
}

// Symbol extraction patterns per language (regex-based for non-Go files)
var symbolPatterns = map[string][]struct {
	Name string
	Re   *regexp.Regexp
	Type string
}{
	"JavaScript/TypeScript": {
		{"function", regexp.MustCompile(`(?:export\s+)?(?:async\s+)?function\s+(\w+)\s*\(`), "function"},
		{"class", regexp.MustCompile(`(?:export\s+)?(?:abstract\s+)?class\s+(\w+)`), "class"},
		{"method", regexp.MustCompile(`(\w+)\s*[=(]\s*(?:async\s+)?\([^)]*\)\s*[{=]>`), "method"},
		{"arrow", regexp.MustCompile(`(?:export\s+)?(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?\(`), "function"},
		{"interface", regexp.MustCompile(`(?:export\s+)?interface\s+(\w+)`), "interface"},
		{"type", regexp.MustCompile(`(?:export\s+)?type\s+(\w+)\s*=`), "type"},
	},
	"Python": {
		{"function", regexp.MustCompile(`(?:async\s+)?def\s+(\w+)\s*\(`), "function"},
		{"class", regexp.MustCompile(`class\s+(\w+)`), "class"},
		{"method", regexp.MustCompile(`(?:async\s+)?def\s+(\w+)\s*\(self`), "method"},
	},
	"Java": {
		{"class", regexp.MustCompile(`(?:public|private|protected)?\s*(?:abstract|final)?\s*class\s+(\w+)`), "class"},
		{"interface", regexp.MustCompile(`(?:public|private|protected)?\s*interface\s+(\w+)`), "interface"},
		{"method", regexp.MustCompile(`(?:public|private|protected)?\s*(?:static)?\s*(?:\w+)\s+(\w+)\s*\(`), "method"},
		{"enum", regexp.MustCompile(`(?:public|private|protected)?\s*enum\s+(\w+)`), "enum"},
	},
	"Rust": {
		{"function", regexp.MustCompile(`(?:pub\s+)?(?:async\s+)?fn\s+(\w+)`), "function"},
		{"struct", regexp.MustCompile(`(?:pub\s+)?struct\s+(\w+)`), "struct"},
		{"impl", regexp.MustCompile(`(?:pub\s+)?impl\s+(\w+)`), "impl"},
		{"enum", regexp.MustCompile(`(?:pub\s+)?enum\s+(\w+)`), "enum"},
		{"trait", regexp.MustCompile(`(?:pub\s+)?trait\s+(\w+)`), "trait"},
	},
	"C++": {
		{"class", regexp.MustCompile(`class\s+(\w+)`), "class"},
		{"struct", regexp.MustCompile(`struct\s+(\w+)`), "struct"},
		{"function", regexp.MustCompile(`\w+\s+(\w+)\s*\([^)]*\)\s*\{`), "function"},
	},
	"C#": {
		{"class", regexp.MustCompile(`class\s+(\w+)`), "class"},
		{"method", regexp.MustCompile(`(?:public|private|protected|internal)?\s*(?:static|virtual|override|async)?\s*(?:\w+\??)\s+(\w+)\s*\(`), "method"},
		{"interface", regexp.MustCompile(`interface\s+(\w+)`), "interface"},
		{"enum", regexp.MustCompile(`enum\s+(\w+)`), "enum"},
	},
	"Swift": {
		{"class", regexp.MustCompile(`class\s+(\w+)`), "class"},
		{"struct", regexp.MustCompile(`struct\s+(\w+)`), "struct"},
		{"function", regexp.MustCompile(`func\s+(\w+)`), "function"},
		{"method", regexp.MustCompile(`func\s+(\w+)\s*\(`), "method"},
		{"enum", regexp.MustCompile(`enum\s+(\w+)`), "enum"},
		{"protocol", regexp.MustCompile(`protocol\s+(\w+)`), "protocol"},
	},
	"Kotlin": {
		{"class", regexp.MustCompile(`class\s+(\w+)`), "class"},
		{"object", regexp.MustCompile(`object\s+(\w+)`), "object"},
		{"function", regexp.MustCompile(`fun\s+(\w+)`), "function"},
		{"method", regexp.MustCompile(`fun\s+(\w+)\s*\(`), "method"},
		{"interface", regexp.MustCompile(`interface\s+(\w+)`), "interface"},
		{"enum", regexp.MustCompile(`enum\s+class\s+(\w+)`), "enum"},
	},
	"Ruby": {
		{"class", regexp.MustCompile(`class\s+(\w+)`), "class"},
		{"module", regexp.MustCompile(`module\s+(\w+)`), "module"},
		{"method", regexp.MustCompile(`def\s+(\w+)`), "method"},
	},
	"PHP": {
		{"class", regexp.MustCompile(`class\s+(\w+)`), "class"},
		{"interface", regexp.MustCompile(`interface\s+(\w+)`), "interface"},
		{"function", regexp.MustCompile(`function\s+(\w+)`), "function"},
		{"method", regexp.MustCompile(`(?:public|private|protected)?\s*function\s+(\w+)`), "method"},
	},
	"Shell": {
		{"function", regexp.MustCompile(`(\w+)\s*\(\)`), "function"},
	},
	"SQL": {
		{"table", regexp.MustCompile(`CREATE\s+TABLE\s+(\w+)`), "table"},
		{"view", regexp.MustCompile(`CREATE\s+VIEW\s+(\w+)`), "view"},
		{"function", regexp.MustCompile(`CREATE\s+FUNCTION\s+(\w+)`), "function"},
		{"procedure", regexp.MustCompile(`CREATE\s+PROCEDURE\s+(\w+)`), "procedure"},
	},
	"HTML": {
		{"tag", regexp.MustCompile(`<(\w+)`), "tag"},
		{"id", regexp.MustCompile(`id=["']([^"']+)["']`), "id"},
		{"class", regexp.MustCompile(`class=["']([^"']+)["']`), "class"},
	},
	"CSS": {
		{"class", regexp.MustCompile(`\.([\w-]+)`), "class"},
		{"id", regexp.MustCompile(`#([\w-]+)`), "id"},
		{"element", regexp.MustCompile(`([\w-]+)\s*\{`), "element"},
	},
}

// CodeChunk represents a semantic chunk of code
type CodeChunk struct {
	ID          string            `json:"id"`
	FilePath    string            `json:"file_path"`
	ChunkType   string            `json:"chunk_type"`
	Name        string            `json:"name"`
	Content     string            `json:"content"`
	StartLine   int               `json:"start_line"`
	EndLine     int               `json:"end_line"`
	Language    string            `json:"language"`
	Package     string            `json:"package"`
	Embedding   []float64         `json:"-"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	LastIndexed time.Time         `json:"last_indexed"`
	FileModTime time.Time         `json:"file_mod_time"`
	Tokens      []string          `json:"-"` // for BM25
	DocFreq     map[string]int    `json:"-"` // term frequency in this chunk
}

// CodeIndex manages the searchable code index with BM25 + embedding search
type CodeIndex struct {
	mu          sync.RWMutex
	chunks      []*CodeChunk
	embedding   *EmbeddingGenerator
	logger      *zap.Logger
	projectRoot string
	chunkByPath map[string][]*CodeChunk
	chunkByName map[string][]*CodeChunk

	// BM25 state
	avgDocLen float64
	totalDocs int
	docFreq   map[string]int // term -> #docs containing it

	// Provider embedding support
	provider providers.Provider

	// Index persistence
	indexPath string
	dirty     bool
}

// NewCodeIndex creates a new code index
func NewCodeIndex(projectRoot string, logger *zap.Logger) *CodeIndex {
	return &CodeIndex{
		chunks:      make([]*CodeChunk, 0),
		embedding:   NewEmbeddingGenerator(1536),
		logger:      logger,
		projectRoot: projectRoot,
		chunkByPath: make(map[string][]*CodeChunk),
		chunkByName: make(map[string][]*CodeChunk),
		docFreq:     make(map[string]int),
	}
}

// SetProvider sets an embedding provider for real embeddings
func (ci *CodeIndex) SetProvider(p providers.Provider) {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	ci.provider = p
}

// SetIndexPath sets persistence path for the index
func (ci *CodeIndex) SetIndexPath(path string) {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	ci.indexPath = path
}

// IndexProject scans and indexes all source files in the project
func (ci *CodeIndex) IndexProject(ctx context.Context) error {
	ci.logger.Info("بدء فهرسة المشروع", zap.String("root", ci.projectRoot))
	start := time.Now()

	var sourceFiles []string
	err := filepath.WalkDir(ci.projectRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == ".git" || name == "vendor" || name == ".next" ||
				name == "__pycache__" || name == ".venv" || name == "venv" || name == "target" ||
				name == "bin" || name == "obj" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		// Only index supported languages
		ext := strings.ToLower(filepath.Ext(path))
		if _, ok := extToLang[ext]; ok {
			sourceFiles = append(sourceFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("فشل مسح المشروع: %w", err)
	}

	var allChunks []*CodeChunk
	for _, file := range sourceFiles {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		chunks := ci.indexFile(file)
		allChunks = append(allChunks, chunks...)
	}

	ci.mu.Lock()
	ci.chunks = allChunks
	ci.chunkByPath = make(map[string][]*CodeChunk)
	ci.chunkByName = make(map[string][]*CodeChunk)
	for _, chunk := range allChunks {
		ci.chunkByPath[chunk.FilePath] = append(ci.chunkByPath[chunk.FilePath], chunk)
		if chunk.Name != "" {
			ci.chunkByName[strings.ToLower(chunk.Name)] = append(ci.chunkByName[strings.ToLower(chunk.Name)], chunk)
		}
	}
	ci.buildBM25Index(allChunks)
	ci.dirty = true
	ci.mu.Unlock()

	go ci.generateEmbeddingsForAll(ctx, allChunks)

	ci.logger.Info("اكتملت فهرسة المشروع",
		zap.Int("files", len(sourceFiles)),
		zap.Int("chunks", len(allChunks)),
		zap.Int("languages", ci.countLanguages(allChunks)),
		zap.Duration("duration", time.Since(start)),
	)
	return nil
}

// buildBM25Index computes BM25 statistics
func (ci *CodeIndex) buildBM25Index(chunks []*CodeChunk) {
	ci.totalDocs = len(chunks)
	totalTokens := 0
	ci.docFreq = make(map[string]int)

	for _, chunk := range chunks {
		chunk.Tokens = tokenize(chunk.Content)
		chunk.DocFreq = make(map[string]int)
		seen := make(map[string]bool)
		for _, t := range chunk.Tokens {
			chunk.DocFreq[t]++
			if !seen[t] {
				seen[t] = true
				ci.docFreq[t]++
			}
		}
		totalTokens += len(chunk.Tokens)
	}

	if ci.totalDocs > 0 {
		ci.avgDocLen = float64(totalTokens) / float64(ci.totalDocs)
	}
}

// indexFile parses a file and extracts semantic chunks
func (ci *CodeIndex) indexFile(filePath string) []*CodeChunk {
	relPath, _ := filepath.Rel(ci.projectRoot, filePath)
	ext := strings.ToLower(filepath.Ext(filePath))

	src, err := os.ReadFile(filePath)
	if err != nil {
		ci.logger.Warn("فشل قراءة الملف", zap.String("file", relPath), zap.Error(err))
		return nil
	}

	content := string(src)
	info, _ := os.Stat(filePath)
	modTime := info.ModTime()

	lang := extToLang[ext]
	langName := "Unknown"
	langSingle := ""
	if lang != nil {
		langName = lang.Name
		langSingle = lang.SingleLine
	}

	// Go gets full AST parsing
	if ext == ".go" {
		chunks := ci.indexGoFile(relPath, content, modTime)
		if len(chunks) > 0 {
			return chunks
		}
	}

	// Other languages: regex-based symbol extraction
	return ci.indexGenericFile(relPath, content, ext, langName, langSingle, modTime)
}

// indexGoFile parses a Go file with full AST
func (ci *CodeIndex) indexGoFile(relPath, content string, modTime time.Time) []*CodeChunk {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		return nil
	}

	packageName := ""
	if f.Name != nil {
		packageName = f.Name.Name
	}

	type rawChunk struct {
		name      string
		chunkType string
		content   string
		startLine int
		endLine   int
	}

	var chunks []rawChunk

	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			startPos := fset.Position(d.Pos())
			endPos := fset.Position(d.End())
			funcName := d.Name.Name
			if d.Recv != nil && len(d.Recv.List) > 0 {
				recvType := goExprToString(d.Recv.List[0].Type)
				funcName = fmt.Sprintf("(%s).%s", recvType, funcName)
			}
			chunks = append(chunks, rawChunk{funcName, "function", content[startPos.Offset:endPos.Offset], startPos.Line, endPos.Line})

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					startPos := fset.Position(s.Pos())
					endPos := fset.Position(s.End())
					chunks = append(chunks, rawChunk{s.Name.Name, "type", content[startPos.Offset:endPos.Offset], startPos.Line, endPos.Line})
				}
			}
		}
	}

	if len(chunks) == 0 {
		return nil
	}

	sort.Slice(chunks, func(i, j int) bool { return chunks[i].startLine < chunks[j].startLine })

	result := make([]*CodeChunk, len(chunks))
	for i, c := range chunks {
		result[i] = &CodeChunk{
			ID:          fmt.Sprintf("%s:%d-%d", relPath, c.startLine, c.endLine),
			FilePath:    relPath,
			ChunkType:   c.chunkType,
			Name:        c.name,
			Content:     c.content,
			StartLine:   c.startLine,
			EndLine:     c.endLine,
			Language:    "Go",
			Package:     packageName,
			LastIndexed: time.Now(),
			FileModTime: modTime,
		}
	}
	return result
}

// indexGenericFile uses regex to extract symbols from non-Go files
func (ci *CodeIndex) indexGenericFile(relPath, content, ext, langName, commentPrefix string, modTime time.Time) []*CodeChunk {
	lines := strings.Split(content, "\n")

	// Try regex patterns
	patterns, hasPatterns := symbolPatterns[langName]

	// If no patterns or file is small, whole-file chunk
	if !hasPatterns || len(lines) < 3 {
		return []*CodeChunk{{
			ID:          relPath,
			FilePath:    relPath,
			ChunkType:   "file",
			Name:        filepath.Base(relPath),
			Content:     content,
			StartLine:   1,
			EndLine:     len(lines),
			Language:    langName,
			LastIndexed: time.Now(),
			FileModTime: modTime,
		}}
	}

	// Detect symbols with regex
	type match struct {
		name      string
		typ       string
		startLine int
	}

	var rawMatches []struct {
		lineNum int
		text    string
		name    string
		typ     string
	}

	for i, line := range lines {
		for _, pat := range patterns {
			parts := pat.Re.FindStringSubmatch(line)
			if len(parts) >= 2 {
				rawMatches = append(rawMatches, struct {
					lineNum int
					text    string
					name    string
					typ     string
				}{i + 1, line, parts[1], pat.Type})
			}
		}
	}

	if len(rawMatches) == 0 {
		return []*CodeChunk{{
			ID: relPath, FilePath: relPath, ChunkType: "file",
			Name: filepath.Base(relPath), Content: content,
			StartLine: 1, EndLine: len(lines),
			Language: langName, LastIndexed: time.Now(), FileModTime: modTime,
		}}
	}

	// Deduplicate: keep first match per symbol name
	seen := make(map[string]bool)
	var unique []struct {
		lineNum int
		name    string
		typ     string
	}
	for _, m := range rawMatches {
		key := strings.ToLower(m.name + ":" + m.typ)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, struct {
				lineNum int
				name    string
				typ     string
			}{m.lineNum, m.name, m.typ})
		}
	}

	result := make([]*CodeChunk, len(unique))
	for i, m := range unique {
		var endLine int
		if i+1 < len(unique) {
			endLine = unique[i+1].lineNum - 1
		} else {
			endLine = len(lines)
		}
		if endLine-m.lineNum > 100 {
			endLine = m.lineNum + 100
		}
		chunkContent := strings.Join(lines[m.lineNum-1:endLine], "\n")

		result[i] = &CodeChunk{
			ID:       fmt.Sprintf("%s:%d-%d", relPath, m.lineNum, endLine),
			FilePath: relPath, ChunkType: m.typ, Name: m.name,
			Content: chunkContent, StartLine: m.lineNum, EndLine: endLine,
			Language: langName, LastIndexed: time.Now(), FileModTime: modTime,
		}
	}
	return result
}

// generateEmbeddingsForAll generates embeddings for all chunks (async)
func (ci *CodeIndex) generateEmbeddingsForAll(ctx context.Context, chunks []*CodeChunk) {
	for _, chunk := range chunks {
		if ctx.Err() != nil {
			return
		}
		chunk.Embedding = ci.embedding.GenerateEmbedding(chunk.Content)
	}
	ci.logger.Info("اكتملت توليد Embeddings", zap.Int("chunks", len(chunks)))
}

// BM25Score computes BM25 relevance for a query against a chunk
const (
	bm25K1 = 1.2
	bm25B  = 0.75
)

func (ci *CodeIndex) bm25Score(query string, chunk *CodeChunk) float64 {
	queryTokens := tokenize(query)
	if len(queryTokens) == 0 || ci.totalDocs == 0 {
		return 0
	}

	score := 0.0
	seen := make(map[string]bool)

	for _, qt := range queryTokens {
		if seen[qt] {
			continue
		}
		seen[qt] = true

		// IDF
		df := ci.docFreq[qt]
		if df == 0 {
			continue
		}
		idf := math.Log(1 + (float64(ci.totalDocs)-float64(df)+0.5)/(float64(df)+0.5))

		// TF in this chunk
		tf := float64(chunk.DocFreq[qt])
		docLen := float64(len(chunk.Tokens))

		// BM25 formula
		numer := tf * (bm25K1 + 1)
		denom := tf + bm25K1*(1-bm25B+bm25B*docLen/ci.avgDocLen)
		score += idf * numer / denom
	}

	return score
}

// Search searches with BM25 + embedding hybrid scoring
func (ci *CodeIndex) Search(ctx context.Context, query string, maxResults int) []*CodeChunk {
	if maxResults <= 0 {
		maxResults = 10
	}

	queryEmbedding := ci.embedding.GenerateEmbedding(query)

	ci.mu.RLock()
	chunks := make([]*CodeChunk, len(ci.chunks))
	copy(chunks, ci.chunks)
	totalDocs := ci.totalDocs
	ci.mu.RUnlock()

	if totalDocs == 0 {
		return nil
	}

	type scored struct {
		chunk    *CodeChunk
		bm25     float64
		cosine   float64
		combined float64
	}

	var results []scored
	for _, chunk := range chunks {
		bm25 := ci.bm25Score(query, chunk)

		var cosine float64
		if chunk.Embedding != nil && len(chunk.Embedding) == len(queryEmbedding) {
			cosine = ci.embedding.CosineSimilarity(queryEmbedding, chunk.Embedding)
		}

		// Normalize BM25 to [0,1] using sigmoid-ish normalization
		bm25Norm := bm25 / (bm25 + 1)

		// Combined: 0.5 BM25 + 0.5 cosine (if cosine available)
		var combined float64
		if cosine > 0 {
			combined = bm25Norm*0.5 + cosine*0.5
		} else {
			combined = bm25Norm
		}

		if combined > 0 {
			results = append(results, scored{chunk, bm25, cosine, combined})
		}
	}

	sort.Slice(results, func(i, j int) bool { return results[i].combined > results[j].combined })

	if len(results) > maxResults {
		results = results[:maxResults]
	}

	out := make([]*CodeChunk, len(results))
	for i, r := range results {
		out[i] = r.chunk
	}
	return out
}

// QueryContext answers a natural language question about the codebase
func (ci *CodeIndex) QueryContext(query string) *CodeContextResult {
	queryLower := strings.ToLower(query)

	type scored struct {
		chunk *CodeChunk
		score float64
	}

	var results []scored

	ci.mu.RLock()
	for _, chunk := range ci.chunks {
		score := 0.0
		contentLower := strings.ToLower(chunk.Content)

		terms := strings.Fields(queryLower)
		for _, term := range terms {
			if len(term) < 3 {
				continue
			}
			if strings.Contains(contentLower, term) {
				score += 1.0
			}
		}

		if chunk.Name != "" && strings.Contains(queryLower, strings.ToLower(chunk.Name)) {
			score += 5.0
		}
		if chunk.Package != "" && strings.Contains(queryLower, chunk.Package) {
			score += 3.0
		}

		if score > 0 {
			results = append(results, scored{chunk, score})
		}
	}
	ci.mu.RUnlock()

	sort.Slice(results, func(i, j int) bool { return results[i].score > results[j].score })
	if len(results) > 10 {
		results = results[:10]
	}

	chunks := make([]*CodeChunk, len(results))
	for i, r := range results {
		chunks[i] = r.chunk
	}

	return &CodeContextResult{
		Query:      query,
		Chunks:     chunks,
		Summary:    ci.generateSummary(query, chunks),
		TotalFound: len(results),
	}
}

// SaveIndex persists index to disk
func (ci *CodeIndex) SaveIndex() error {
	ci.mu.RLock()
	if !ci.dirty {
		ci.mu.RUnlock()
		return nil
	}
	path := ci.indexPath
	chunks := ci.chunks
	ci.mu.RUnlock()

	if path == "" {
		return nil
	}

	data, err := json.Marshal(chunks)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}

	ci.mu.Lock()
	ci.dirty = false
	ci.mu.Unlock()

	return nil
}

// LoadIndex loads index from disk
func (ci *CodeIndex) LoadIndex(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var chunks []*CodeChunk
	if err := json.Unmarshal(data, &chunks); err != nil {
		return err
	}

	ci.mu.Lock()
	ci.chunks = chunks
	ci.chunkByPath = make(map[string][]*CodeChunk)
	ci.chunkByName = make(map[string][]*CodeChunk)
	for _, chunk := range chunks {
		ci.chunkByPath[chunk.FilePath] = append(ci.chunkByPath[chunk.FilePath], chunk)
		if chunk.Name != "" {
			ci.chunkByName[strings.ToLower(chunk.Name)] = append(ci.chunkByName[strings.ToLower(chunk.Name)], chunk)
		}
	}
	ci.buildBM25Index(chunks)
	ci.dirty = false
	ci.mu.Unlock()

	ci.logger.Info("تم تحميل الفهرس من القرص", zap.String("path", path), zap.Int("chunks", len(chunks)))
	return nil
}

// SearchBySymbol finds chunks by exact symbol name
func (ci *CodeIndex) SearchBySymbol(name string) []*CodeChunk {
	ci.mu.RLock()
	defer ci.mu.RUnlock()
	return ci.chunkByName[strings.ToLower(name)]
}

// generateSummary creates a human-readable answer
func (ci *CodeIndex) generateSummary(query string, chunks []*CodeChunk) string {
	if len(chunks) == 0 {
		return "لم أجد معلومات متعلقة بسؤالك في قاعدة الشيفرة."
	}

	files := make(map[string][]string)
	for _, c := range chunks {
		files[c.FilePath] = append(files[c.FilePath], c.Name)
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("وجدت %d نتيجة متعلقة بـ \"%s\":", len(chunks), query))
	for path, names := range files {
		parts = append(parts, fmt.Sprintf("• %s (%s)", path, strings.Join(uniqueStrings(names), ", ")))
	}
	return strings.Join(parts, "\n")
}

func (ci *CodeIndex) countLanguages(chunks []*CodeChunk) int {
	seen := make(map[string]bool)
	for _, c := range chunks {
		seen[c.Language] = true
	}
	return len(seen)
}

// CodeContextResult holds the result of a context query
type CodeContextResult struct {
	Query      string       `json:"query"`
	Chunks     []*CodeChunk `json:"chunks"`
	Summary    string       `json:"summary"`
	TotalFound int          `json:"total_found"`
}

func (r *CodeContextResult) ToJSON() string {
	b, _ := json.MarshalIndent(r, "", "  ")
	return string(b)
}

// helpers
func goExprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + goExprToString(t.X)
	case *ast.SelectorExpr:
		return goExprToString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + goExprToString(t.Elt)
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func uniqueStrings(s []string) []string {
	seen := make(map[string]bool)
	var r []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			r = append(r, v)
		}
	}
	return r
}

// tokenize splits text into lowercase tokens
func tokenize(text string) []string {
	re := regexp.MustCompile(`[a-zA-Z_]\w*`)
	matches := re.FindAllString(text, -1)
	result := make([]string, len(matches))
	for i, m := range matches {
		result[i] = strings.ToLower(m)
	}
	return result
}
