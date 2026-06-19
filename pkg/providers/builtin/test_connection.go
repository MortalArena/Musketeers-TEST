package builtin

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// TestConnection tests if a provider's API endpoint is reachable without requiring an API key
func TestConnection(baseURL string) error {
	if baseURL == "" {
		return fmt.Errorf("base URL is empty")
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create a simple GET request to the base URL
	// Most APIs will return 401/403 without auth, but that's fine - we just want to know if the endpoint is reachable
	req, err := http.NewRequestWithContext(context.Background(), "GET", baseURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", baseURL, err)
	}
	defer resp.Body.Close()

	// Any response (even 401/403) means the endpoint is reachable
	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		return nil
	}

	return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// TestAllConnections tests connections for all providers
func TestAllConnections() map[string]error {
	results := make(map[string]error)

	// Test each provider's base URL
	providers := map[string]string{
		"openai":     "https://api.openai.com/v1",
		"anthropic":  "https://api.anthropic.com/v1",
		"google":     "https://generativelanguage.googleapis.com/v1beta",
		"deepseek":   "https://api.deepseek.com/v1",
		"xai":        "https://api.x.ai/v1",
		"mistral":    "https://api.mistral.ai/v1",
		"cohere":     "https://api.cohere.ai/v1",
		"qwen":       "https://dashscope.aliyuncs.com/compatible-mode/v1",
		"moonshot":   "https://api.moonshot.cn/v1",
		"nvidia":     "https://integrate.api.nvidia.com/v1",
		"xiaomi":     "https://api.xiaomi.com/v1",
		"zai":        "https://open.bigmodel.cn/api/paas/v4",
		"tencent":    "https://hunyuan.tencentcloudapi.com/v1",
		"stepfun":    "https://api.stepfun.com/v1",
		"poolside":   "https://api.poolside.ai/v1",
		"recraft":    "https://api.recraft.ai/v1",
		"sourceful":  "https://api.sourceful.ai/v1",
		"openrouter": "https://openrouter.ai/api/v1",
		"groq":       "https://api.groq.com/openai/v1",
		"perplexity": "https://api.perplexity.ai",
		"togetherai": "https://api.together.xyz/v1",
		"ollama":     "http://localhost:11434",
	}

	for name, url := range providers {
		err := TestConnection(url)
		results[name] = err
		if err != nil {
			fmt.Printf("❌ %s: %v\n", name, err)
		} else {
			fmt.Printf("✅ %s: Connection successful\n", name)
		}
	}

	return results
}
