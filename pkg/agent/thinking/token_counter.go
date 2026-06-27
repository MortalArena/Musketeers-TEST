package thinking

import (
	"context"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/MortalArena/Musketeers/pkg/providers"
	"go.uber.org/zap"
)

// estimateTokens يعطي تقديراً تقريبياً لعدد التوكنز في النص
// [WHY] لا يوجد tokenizer حقيقي، هذا يمنع الفشل الصامت مع الموديلات صغيرة السياق
// heuristic: ~1 token لكل 4 حروف للنصوص المختلطة (عربي + إنجليزي)
func estimateTokens(text string) int {
	if text == "" {
		return 0
	}
	// نستخدم طول النص بالبايت للحروف غير ASCII
	charCount := utf8.RuneCountInString(text)
	tokens := int(math.Ceil(float64(charCount) / 4.0))
	if tokens < 1 {
		tokens = 1
	}
	return tokens
}

// truncateText يقتطع النص ليتناسب مع حد التوكنز المطلوب
func truncateText(text string, maxTokens int) string {
	if maxTokens <= 0 {
		return ""
	}
	if estimateTokens(text) <= maxTokens {
		return text
	}
	// نبدأ باقتطاع تدريجي
	maxChars := maxTokens * 4
	if maxChars >= len(text) {
		return text
	}
	// نقطع مع الحفاظ على نهاية الجملة إن أمكن
	truncated := text[:maxChars]
	lastSentence := strings.LastIndex(truncated, ".")
	if lastSentence > maxChars/2 {
		return text[:lastSentence+1]
	}
	lastNewline := strings.LastIndex(truncated, "\n")
	if lastNewline > maxChars/2 {
		return text[:lastNewline]
	}
	return truncated
}

// getModelContextLength يرجع طول سياق الموديل الحالي
func (te *ThinkingEngine) getModelContextLength(ctx context.Context) int {
	if te.provider == nil || te.modelID == "" {
		return 128000 // افتراضي: 128K توكن
	}
	modelInfo, err := te.provider.GetModel(ctx, te.modelID)
	if err != nil {
		// قيم افتراضية حسب نوع المزود
		return defaultContextLength(te.modelID)
	}
	if modelInfo != nil && modelInfo.ContextLength > 0 {
		return modelInfo.ContextLength
	}
	return defaultContextLength(te.modelID)
}

// defaultContextLength يرجع طول سياق افتراضي حسب اسم الموديل
func defaultContextLength(modelID string) int {
	modelLower := strings.ToLower(modelID)
	switch {
	case strings.Contains(modelLower, "gemini"):
		return 1000000
	case strings.Contains(modelLower, "claude"):
		return 200000
	case strings.Contains(modelLower, "gpt-4"):
		return 128000
	case strings.Contains(modelLower, "gpt-3.5"):
		return 16385
	case strings.Contains(modelLower, "deepseek"):
		return 128000
	case strings.Contains(modelLower, "llama") || strings.Contains(modelLower, "mixtral"):
		return 32768
	case strings.Contains(modelLower, "mistral"):
		return 32768
	case strings.Contains(modelLower, "qwen"):
		return 32768
	case strings.Contains(modelLower, "phi"):
		return 128000
	default:
		return 128000
	}
}

// buildTruncatedRequest يبني طلب LLM مع اقتطاع تلقائي للبرومبت
// [WHY] يمنع الفشل الصامت عندما يتجاوز الطول حد سياق الموديل
func (te *ThinkingEngine) buildTruncatedRequest(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) *providers.CompletionRequest {
	contextLen := te.getModelContextLength(ctx)

	sysTokens := estimateTokens(systemPrompt)
	userTokens := estimateTokens(userPrompt)
	totalPromptTokens := sysTokens + userTokens

	// مساحة متبقية للمخرجات
	availableForOutput := contextLen - totalPromptTokens
	buffer := 100 // حماية من التقدير الخاطئ

	if availableForOutput < maxTokens+buffer {
		// نحتاج للاقتطاع — نبدأ بالمستخدم ثم النظام
		maxUserTokens := contextLen - sysTokens - maxTokens - buffer
		if maxUserTokens > 100 {
			userPrompt = truncateText(userPrompt, maxUserTokens)
		} else {
			// النظام نفسه كبير جداً — نقطع النصين
			maxSysTokens := int(math.Ceil(float64(contextLen) * 0.3))
			systemPrompt = truncateText(systemPrompt, maxSysTokens)
			maxUserTokens = contextLen - estimateTokens(systemPrompt) - maxTokens - buffer
			if maxUserTokens > 0 {
				userPrompt = truncateText(userPrompt, maxUserTokens)
			} else {
				userPrompt = ""
			}
		}
	}

	te.logger.Debug("بناء طلب LLM",
		zap.String("model", te.modelID),
		zap.Int("context_length", contextLen),
		zap.Int("system_tokens_est", sysTokens),
		zap.Int("user_tokens_est", estimateTokens(userPrompt)),
		zap.Int("max_tokens", maxTokens),
	)

	return &providers.CompletionRequest{
		Model: te.modelID,
		Messages: []providers.Message{
			{Role: providers.RoleSystem, Content: systemPrompt},
			{Role: providers.RoleUser, Content: userPrompt},
		},
		MaxTokens:   maxTokens,
		Temperature: 0.3,
	}
}

// buildTruncatedRequestWithTools مثل buildTruncatedRequest لكن مع أدوات
func (te *ThinkingEngine) buildTruncatedRequestWithTools(ctx context.Context, systemPrompt, userPrompt string, maxTokens int, tools []providers.Tool) *providers.CompletionRequest {
	req := te.buildTruncatedRequest(ctx, systemPrompt, userPrompt, maxTokens)
	req.Tools = tools
	req.ResponseFormat = &providers.ResponseFormat{Type: "json"}
	return req
}

// completeWithTruncation يرسل طلب إكمال مع اقتطاع تلقائي للبرومبت
// [WHY] نقطة وحيدة لجميع استدعاءات LLM — تضمن عدم تجاوز حد السياق
func (te *ThinkingEngine) completeWithTruncation(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (*providers.CompletionResponse, error) {
	req := te.buildTruncatedRequest(ctx, systemPrompt, userPrompt, maxTokens)
	return te.provider.Complete(ctx, req)
}

// completeWithTruncationJSON مثل completeWithTruncation لكن مع تنسيق JSON
func (te *ThinkingEngine) completeWithTruncationJSON(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (*providers.CompletionResponse, error) {
	req := te.buildTruncatedRequest(ctx, systemPrompt, userPrompt, maxTokens)
	req.ResponseFormat = &providers.ResponseFormat{Type: "json"}
	return te.provider.Complete(ctx, req)
}

// completeWithTruncationTools مثل completeWithTruncation مع أدوات
func (te *ThinkingEngine) completeWithTruncationTools(ctx context.Context, systemPrompt, userPrompt string, maxTokens int, tools []providers.Tool) (*providers.CompletionResponse, error) {
	req := te.buildTruncatedRequest(ctx, systemPrompt, userPrompt, maxTokens)
	req.Tools = tools
	if len(tools) > 0 {
		req.ResponseFormat = &providers.ResponseFormat{Type: "json"}
	}
	return te.provider.Complete(ctx, req)
}
