package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/MortalArena/Musketeers/pkg/providers"
	"github.com/MortalArena/Musketeers/pkg/providers/builtin"
)

func main() {
	// Initialize provider registry
	registry := builtin.NewRegistry()

	// Example: Initialize OpenAI provider
	ctx := context.Background()

	openaiConfig := providers.ProviderConfig{
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		BaseURL: "https://api.openai.com/v1",
		Timeout: 5 * 60, // 5 minutes
	}

	provider, exists := registry.Get(providers.ProviderOpenAI)
	if !exists {
		log.Fatalf("Provider not found in registry")
	}

	if err := provider.Initialize(ctx, openaiConfig); err != nil {
		log.Fatalf("Failed to initialize provider: %v", err)
	}

	// Check provider status
	status := provider.Status()
	fmt.Printf("Provider: %s\n", provider.Name())
	fmt.Printf("Available: %v\n", provider.IsAvailable())
	fmt.Printf("Last Check: %v\n", status.LastCheck)
	fmt.Printf("Models Count: %d\n", status.ModelsCount)

	// List available models
	models, err := provider.ListModels(ctx)
	if err != nil {
		log.Fatalf("Failed to list models: %v", err)
	}

	fmt.Printf("\nAvailable Models:\n")
	for _, model := range models {
		fmt.Printf("- %s (Context: %d)\n", model.ID, model.ContextLength)
	}

	// Example: Get model info
	if len(models) > 0 {
		model, err := provider.GetModel(ctx, models[0].ID)
		if err != nil {
			log.Fatalf("Failed to get model: %v", err)
		}
		fmt.Printf("\nModel Details:\n")
		fmt.Printf("ID: %s\n", model.ID)
		fmt.Printf("Name: %s\n", model.Name)
		fmt.Printf("Context Length: %d\n", model.ContextLength)
		fmt.Printf("Capabilities: %v\n", model.Capabilities)
	}

	// Example: Simple completion request
	if len(models) > 0 {
		req := &providers.CompletionRequest{
			Model: models[0].ID,
			Messages: []providers.Message{
				{
					Role:    providers.RoleUser,
					Content: "Hello, how are you?",
				},
			},
			Temperature: 0.7,
			MaxTokens:   100,
		}

		resp, err := provider.Complete(ctx, req)
		if err != nil {
			log.Printf("Failed to complete: %v", err)
		} else {
			fmt.Printf("\nCompletion Response:\n")
			fmt.Printf("Content: %s\n", resp.Content)
			fmt.Printf("Tokens Used: %d\n", resp.Usage.TotalTokens)
			fmt.Printf("Latency: %v\n", resp.Latency)
		}
	}

	// Close provider
	if err := provider.Close(); err != nil {
		log.Printf("Failed to close provider: %v", err)
	}
}
