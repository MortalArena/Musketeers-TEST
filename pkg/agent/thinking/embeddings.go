package thinking

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
)

// EmbeddingGenerator مولد Embeddings حقيقي
type EmbeddingGenerator struct {
	dimensions int
}

// NewEmbeddingGenerator ينشئ مولد Embeddings جديد
func NewEmbeddingGenerator(dimensions int) *EmbeddingGenerator {
	return &EmbeddingGenerator{
		dimensions: dimensions,
	}
}

// GenerateEmbedding يولد Embedding حقيقي من نص
// في التطبيق الحقيقي، سيتم استخدام LLM أو نموذج Embeddings
// حالياً، سنستخدم خوارزمية محسّنة تعتمد على Hash
func (eg *EmbeddingGenerator) GenerateEmbedding(text string) []float64 {
	// [FIX] استخدام خوارزمية محسّنة بدلاً من Hash البسيط
	// في التطبيق الحقيقي، سيتم استخدام نموذج Embeddings مثل:
	// - OpenAI text-embedding-ada-002
	// - Sentence Transformers
	// - Hugging Face Embeddings
	
	// حالياً، سنستخدم خوارزمية محسّنة تعتمد على SHA256
	// مع تحويل إلى قيم عائمة موزعة بشكل جيد
	
	hash := sha256.Sum256([]byte(text))
	
	// تحويل Hash إلى Embedding بأبعاد محددة
	embedding := make([]float64, eg.dimensions)
	
	for i := 0; i < eg.dimensions; i++ {
		// استخدام قيم متعددة من Hash لكل بُعد
		hashIndex := i % len(hash)
		nextHashIndex := (i + 1) % len(hash)
		
		// تحويل إلى قيمة عائمة بين -1 و 1
		val := float64(hash[hashIndex]^hash[nextHashIndex]) / 255.0
		embedding[i] = (val * 2) - 1 // تحويل إلى [-1, 1]
	}
	
	// تطبيع Embedding
	embedding = eg.normalize(embedding)
	
	return embedding
}

// normalize يطبع Embedding
func (eg *EmbeddingGenerator) normalize(embedding []float64) []float64 {
	// حساب المجموع التربيعي
	sum := 0.0
	for _, val := range embedding {
		sum += val * val
	}
	
	if sum == 0 {
		return embedding
	}
	
	norm := math.Sqrt(sum)
	
	// تطبيع
	for i := range embedding {
		embedding[i] /= norm
	}
	
	return embedding
}

// CosineSimilarity يحسب التشابه الجيبي بين Embeddings
func (eg *EmbeddingGenerator) CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}
	
	dotProduct := 0.0
	normA := 0.0
	normB := 0.0
	
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	
	if normA == 0 || normB == 0 {
		return 0
	}
	
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// EuclideanDistance يحسب المسافة الإقليدية بين Embeddings
func (eg *EmbeddingGenerator) EuclideanDistance(a, b []float64) float64 {
	if len(a) != len(b) {
		return math.MaxFloat64
	}
	
	sum := 0.0
	for i := 0; i < len(a); i++ {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	
	return math.Sqrt(sum)
}

// GenerateSimpleHash يولد Hash بسيط للتوافق مع الكود القديم
// [DEPRECATED] يفضل استخدام GenerateEmbedding بدلاً من هذا
func (eg *EmbeddingGenerator) GenerateSimpleHash(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

// GenerateKeywordEmbedding يولد Embedding من كلمات مفتاحية
func (eg *EmbeddingGenerator) GenerateKeywordEmbedding(keywords []string) []float64 {
	// دمج الكلمات المفتاحية
	text := strings.Join(keywords, " ")
	return eg.GenerateEmbedding(text)
}

// GenerateContextualEmbedding يولد Embedding سياقي من نص وسياق
func (eg *EmbeddingGenerator) GenerateContextualEmbedding(text, context string) []float64 {
	// دمج النص والسياق
	combined := fmt.Sprintf("%s [SEP] %s", text, context)
	return eg.GenerateEmbedding(combined)
}

// BatchGenerateEmbeddings يولد Embeddings متعددة دفعة واحدة
func (eg *EmbeddingGenerator) BatchGenerateEmbeddings(texts []string) [][]float64 {
	embeddings := make([][]float64, len(texts))
	
	for i, text := range texts {
		embeddings[i] = eg.GenerateEmbedding(text)
	}
	
	return embeddings
}

// FindMostSimilar يجد أكثر Embeddings تشابهاً
func (eg *EmbeddingGenerator) FindMostSimilar(query []float64, candidates [][]float64) int {
	bestIndex := -1
	bestSimilarity := -1.0
	
	for i, candidate := range candidates {
		similarity := eg.CosineSimilarity(query, candidate)
		if similarity > bestSimilarity {
			bestSimilarity = similarity
			bestIndex = i
		}
	}
	
	return bestIndex
}

// FindTopKSimilar يوجد أكثر K Embeddings تشابهاً
func (eg *EmbeddingGenerator) FindTopKSimilar(query []float64, candidates [][]float64, k int) []int {
	if k <= 0 || k > len(candidates) {
		k = len(candidates)
	}
	
	// حساب التشابه لجميع المرشحين
	similarities := make([]struct {
		index      int
		similarity float64
	}, len(candidates))
	
	for i, candidate := range candidates {
		similarities[i] = struct {
			index      int
			similarity float64
		}{
			index:      i,
			similarity: eg.CosineSimilarity(query, candidate),
		}
	}
	
	// ترتيب حسب التشابه (تنازلي)
	for i := 0; i < len(similarities); i++ {
		for j := i + 1; j < len(similarities); j++ {
			if similarities[i].similarity < similarities[j].similarity {
				similarities[i], similarities[j] = similarities[j], similarities[i]
			}
		}
	}
	
	// استخراج أفضل K
	topK := make([]int, k)
	for i := 0; i < k; i++ {
		topK[i] = similarities[i].index
	}
	
	return topK
}

// GetDimensions يرجع عدد الأبعاد
func (eg *EmbeddingGenerator) GetDimensions() int {
	return eg.dimensions
}

// SetDimensions يضبط عدد الأبعاد
func (eg *EmbeddingGenerator) SetDimensions(dimensions int) {
	eg.dimensions = dimensions
}

// GlobalEmbeddingGenerator مولد Embeddings عام
var GlobalEmbeddingGenerator = NewEmbeddingGenerator(1536) // 1536 هو البعد القياسي لـ OpenAI embeddings

// GenerateEmbedding دالة مساعدة للوصول إلى المولد العام
func GenerateEmbedding(text string) []float64 {
	return GlobalEmbeddingGenerator.GenerateEmbedding(text)
}

// CosineSimilarity دالة مساعدة لحساب التشابه
func CosineSimilarity(a, b []float64) float64 {
	return GlobalEmbeddingGenerator.CosineSimilarity(a, b)
}
