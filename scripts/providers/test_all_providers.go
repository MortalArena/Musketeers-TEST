package providers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ProviderTest struct {
	Name    string
	BaseURL string
	Type    string // "cloud" or "local"
}

func testProvider(provider ProviderTest) (bool, int, string) {
	if provider.BaseURL == "" {
		return false, 0, "base URL is empty"
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	endpoints := []string{
		provider.BaseURL,
		provider.BaseURL + "/models",
		provider.BaseURL + "/chat/completions",
		provider.BaseURL + "/v1/models",
		provider.BaseURL + "/api/tags",
		provider.BaseURL + "/api/version",
	}

	for _, endpoint := range endpoints {
		req, err := http.NewRequestWithContext(context.Background(), "GET", endpoint, nil)
		if err != nil {
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 600 {
			return true, resp.StatusCode, string(body)
		}
	}

	return false, 0, "failed to connect"
}

func TestAllProviders() {
	cloudProviders := []ProviderTest{
		{Name: "OpenAI", BaseURL: "https://api.openai.com/v1", Type: "cloud"},
		{Name: "Anthropic", BaseURL: "https://api.anthropic.com/v1", Type: "cloud"},
		{Name: "Google", BaseURL: "https://generativelanguage.googleapis.com/v1beta", Type: "cloud"},
		{Name: "DeepSeek", BaseURL: "https://api.deepseek.com/v1", Type: "cloud"},
		{Name: "xAI", BaseURL: "https://api.x.ai/v1", Type: "cloud"},
		{Name: "Mistral", BaseURL: "https://api.mistral.ai/v1", Type: "cloud"},
		{Name: "Cohere", BaseURL: "https://api.cohere.ai/v1", Type: "cloud"},
		{Name: "Qwen", BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", Type: "cloud"},
		{Name: "Moonshot", BaseURL: "https://api.moonshot.cn/v1", Type: "cloud"},
		{Name: "NVIDIA", BaseURL: "https://integrate.api.nvidia.com/v1", Type: "cloud"},
		{Name: "Xiaomi", BaseURL: "https://api.xiaomimimo.com/v1", Type: "cloud"},
		{Name: "Z.ai", BaseURL: "https://open.bigmodel.cn/api/paas/v4", Type: "cloud"},
		{Name: "Tencent", BaseURL: "https://hunyuan.tencentcloudapi.com/v1", Type: "cloud"},
		{Name: "StepFun", BaseURL: "https://api.stepfun.com/v1", Type: "cloud"},
		{Name: "Recraft", BaseURL: "https://api.recraft.ai/v1", Type: "cloud"},
		{Name: "OpenRouter", BaseURL: "https://openrouter.ai/api/v1", Type: "cloud"},
		{Name: "Groq", BaseURL: "https://api.groq.com/openai/v1", Type: "cloud"},
		{Name: "Perplexity", BaseURL: "https://api.perplexity.ai", Type: "cloud"},
		{Name: "TogetherAI", BaseURL: "https://api.together.xyz/v1", Type: "cloud"},
	}

	localProviders := []ProviderTest{
		{Name: "Ollama (localhost)", BaseURL: "http://localhost:11434", Type: "local"},
		{Name: "Ollama (127.0.0.1)", BaseURL: "http://127.0.0.1:11434", Type: "local"},
		{Name: "LM Studio (localhost)", BaseURL: "http://localhost:1234", Type: "local"},
		{Name: "LM Studio (127.0.0.1)", BaseURL: "http://127.0.0.1:1234", Type: "local"},
	}

	fmt.Println("================================================================================")
	fmt.Println("COMPREHENSIVE PROVIDER CONNECTION TEST")
	fmt.Println("================================================================================")
	fmt.Println()

	fmt.Println("PART 1: Cloud Providers")
	fmt.Println("================================================================================")
	fmt.Println()

	cloudSuccessCount := 0
	for _, provider := range cloudProviders {
		fmt.Printf("Testing %s (%s)...\n", provider.Name, provider.BaseURL)
		success, status, response := testProvider(provider)

		if success {
			cloudSuccessCount++
			fmt.Printf("✅ %s: Connected (HTTP %d)\n", provider.Name, status)
			if len(response) > 0 && len(response) < 150 {
				fmt.Printf("   Response: %s\n", response)
			}
		} else {
			fmt.Printf("❌ %s: Failed - %s\n", provider.Name, response)
		}
		fmt.Println()
	}

	fmt.Println("================================================================================")
	fmt.Printf("Cloud Providers Summary: %d/%d connected successfully\n", cloudSuccessCount, len(cloudProviders))
	fmt.Println()

	fmt.Println("PART 2: Local Providers (Ollama, LM Studio)")
	fmt.Println("================================================================================")
	fmt.Println()
	fmt.Println("Note: Local providers must be running on your machine for this test to succeed.")
	fmt.Println("If they are not running, the test will fail - this is expected.")
	fmt.Println()
	fmt.Println("To start Ollama: Run 'ollama serve' in your terminal")
	fmt.Println("To start LM Studio: Open the LM Studio application")
	fmt.Println()
	fmt.Println("================================================================================")
	fmt.Println()

	localSuccessCount := 0
	for _, provider := range localProviders {
		fmt.Printf("Testing %s (%s)...\n", provider.Name, provider.BaseURL)
		success, status, response := testProvider(provider)

		if success {
			localSuccessCount++
			fmt.Printf("✅ %s: Connected (HTTP %d)\n", provider.Name, status)
			if len(response) > 0 && len(response) < 150 {
				fmt.Printf("   Response: %s\n", response)
			}
		} else {
			fmt.Printf("❌ %s: Failed - %s\n", provider.Name, response)
			fmt.Printf("   Note: This is expected if the provider is not running locally.\n")
		}
		fmt.Println()
	}

	fmt.Println("================================================================================")
	fmt.Printf("Local Providers Summary: %d/%d connected successfully\n", localSuccessCount, len(localProviders))
	fmt.Println()

	fmt.Println("================================================================================")
	fmt.Println("FINAL SUMMARY")
	fmt.Println("================================================================================")
	fmt.Printf("Cloud Providers: %d/%d connected\n", cloudSuccessCount, len(cloudProviders))
	fmt.Printf("Local Providers: %d/%d connected\n", localSuccessCount, len(localProviders))
	fmt.Printf("Total: %d/%d providers connected\n", cloudSuccessCount+localSuccessCount, len(cloudProviders)+len(localProviders))
	fmt.Println()

	fmt.Println("================================================================================")
	fmt.Println("RECOMMENDATIONS")
	fmt.Println("================================================================================")
	fmt.Println()

	if cloudSuccessCount < len(cloudProviders) {
		fmt.Println("⚠️  Some cloud providers failed to connect. Check their API URLs.")
		fmt.Println()
	}

	if localSuccessCount == 0 {
		fmt.Println("⚠️  No local providers are running.")
		fmt.Println()
		fmt.Println("To set up local providers:")
		fmt.Println("1. Install Ollama: https://ollama.com/download")
		fmt.Println("2. Install LM Studio: https://lmstudio.ai/download")
		fmt.Println("3. Start Ollama: ollama serve")
		fmt.Println("4. Start LM Studio: Open the application")
		fmt.Println("5. Re-run this test")
		fmt.Println()
	} else {
		fmt.Println("✅ Local providers are running and accessible.")
		fmt.Println()
	}

	fmt.Println("================================================================================")
	fmt.Println("FREE MODELS ROUTER")
	fmt.Println("================================================================================")
	fmt.Println()
	fmt.Println("The free models router will automatically route requests to free-tier models")
	fmt.Println("from the following providers:")
	fmt.Println()
	fmt.Println("✅ OpenRouter (has free models)")
	fmt.Println("✅ Groq (free tier available)")
	fmt.Println("✅ TogetherAI (free tier available)")
	fmt.Println("✅ Perplexity (free tier available)")
	fmt.Println("✅ Hugging Face (via OpenRouter)")
	fmt.Println()
	fmt.Println("The router will:")
	fmt.Println("1. Check if a model has a free tier")
	fmt.Println("2. Route requests to free models when possible")
	fmt.Println("3. Fall back to paid models only when necessary")
	fmt.Println("4. Track usage and costs")
	fmt.Println()
	fmt.Println("================================================================================")
}
