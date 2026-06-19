package builtin

import (
	"github.com/MortalArena/Musketeers/pkg/providers"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/anthropic"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/cohere"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/custom"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/deepseek"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/google"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/groq"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/mistral"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/moonshot"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/nvidia"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/ollama"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/openai"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/openrouter"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/perplexity"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/poolside"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/qwen"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/recraft"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/sourceful"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/stepfun"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/tencent"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/togetherai"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/xai"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/xiaomi"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin/zai"
)

// NewRegistry creates a new provider registry with all builtin providers
func NewRegistry() *providers.ProviderRegistry {
	registry := providers.NewProviderRegistry()

	// Official Providers (22 providers with official APIs)
	registry.Register(providers.ProviderOpenAI, openai.New())
	registry.Register(providers.ProviderAnthropic, anthropic.New())
	registry.Register(providers.ProviderGoogle, google.New())
	registry.Register(providers.ProviderDeepSeek, deepseek.New())
	registry.Register(providers.ProviderXAI, xai.New())
	registry.Register(providers.ProviderMistral, mistral.New())
	registry.Register(providers.ProviderQwen, qwen.New())
	registry.Register(providers.ProviderMoonshot, moonshot.New())
	registry.Register(providers.ProviderNVIDIA, nvidia.New())
	registry.Register(providers.ProviderXiaomi, xiaomi.New())
	registry.Register(providers.ProviderZAI, zai.New())
	registry.Register(providers.ProviderTencent, tencent.New())
	registry.Register(providers.ProviderStepFun, stepfun.New())
	registry.Register(providers.ProviderPoolside, poolside.New())
	registry.Register(providers.ProviderRecraft, recraft.New())
	registry.Register(providers.ProviderSourceful, sourceful.New())
	registry.Register(providers.ProviderOpenRouter, openrouter.New())
	registry.Register(providers.ProviderCohere, cohere.New())
	registry.Register(providers.ProviderGroq, groq.New())
	registry.Register(providers.ProviderTogetherAI, togetherai.New())
	registry.Register(providers.ProviderPerplexity, perplexity.New())

	// Local Providers
	registry.Register(providers.ProviderOllama, ollama.New())

	// Custom Provider
	registry.Register(providers.ProviderCustom, custom.New())

	return registry
}
