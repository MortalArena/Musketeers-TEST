package main

import (
	"context"
	"fmt"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/agent/adapters"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	ctx := context.Background()

	// ==================== 1. CLI Agents ====================
	fmt.Println("=== 1. Setting up CLI Agents ===")

	cliAdapter := adapters.NewMultiCLIAdapter(logger)

	// إضافة 4 CLI agents
	cliAdapter.AddCLIInstance("claude-code-1", "claude-code", &adapters.CLIConfig{
		Name:    "claude-code",
		Command: "claude",
		Args:    []string{"--api-key", "CLAUDE_KEY"},
	})

	cliAdapter.AddCLIInstance("opencode-1", "opencode", &adapters.CLIConfig{
		Name:    "opencode",
		Command: "opencode",
		Args:    []string{"--api-key", "OPENCODE_KEY"},
	})

	cliAdapter.AddCLIInstance("codex-1", "codex", &adapters.CLIConfig{
		Name:    "codex",
		Command: "codex",
		Args:    []string{"--api-key", "OPENAI_KEY"},
	})

	cliAdapter.AddCLIInstance("gemini-1", "gemini", &adapters.CLIConfig{
		Name:    "gemini",
		Command: "gemini",
		Args:    []string{"--api-key", "GOOGLE_KEY"},
	})

	// عرض جميع نسخ CLI
	cliInstances := cliAdapter.GetAllCLIInstances()
	fmt.Printf("✓ Registered %d CLI agents:\n", len(cliInstances))
	for _, inst := range cliInstances {
		fmt.Printf("  - %s (%s)\n", inst.AgentName, inst.InstanceID)
	}

	// ==================== 2. Desktop Apps ====================
	fmt.Println("\n=== 2. Setting up Desktop Apps ===")

	desktopAdapter := adapters.NewMultiDesktopAdapter(logger)

	// إضافة 3 Desktop apps
	desktopAdapter.AddDesktopInstance("claude-desktop-1", "claude-desktop", &adapters.DesktopAppConfig{
		Name:              "claude-desktop",
		Executable:        "/Applications/Claude.app/Contents/MacOS/Claude",
		CommunicationMode: "websocket",
		WebSocketURL:      "ws://localhost:8080",
	})

	desktopAdapter.AddDesktopInstance("codex-app-1", "codex-app", &adapters.DesktopAppConfig{
		Name:              "codex-app",
		Executable:        "/Applications/Codex.app/Contents/MacOS/Codex",
		CommunicationMode: "http",
		HTTPBaseURL:       "http://localhost:3000",
	})

	desktopAdapter.AddDesktopInstance("hermes-1", "hermes", &adapters.DesktopAppConfig{
		Name:              "hermes",
		Executable:        "/Applications/Hermes.app/Contents/MacOS/Hermes",
		CommunicationMode: "websocket",
		WebSocketURL:      "ws://localhost:8081",
	})

	// عرض جميع نسخ Desktop
	desktopInstances := desktopAdapter.GetAllDesktopInstances()
	fmt.Printf("✓ Registered %d desktop apps:\n", len(desktopInstances))
	for _, inst := range desktopInstances {
		fmt.Printf("  - %s (%s)\n", inst.AgentName, inst.InstanceID)
	}

	// ==================== 3. IDEs ====================
	fmt.Println("\n=== 3. Setting up IDEs ===")

	ideAdapter := adapters.NewMultiIDEAdapter(logger)

	// إضافة 3 IDEs
	ideAdapter.AddIDEInstance("cursor-1", "cursor", &adapters.IDEConfig{
		IDEType:     "cursor",
		Name:        "Cursor",
		ProjectPath: "/project1",
	})

	ideAdapter.AddIDEInstance("vscode-1", "vscode", &adapters.IDEConfig{
		IDEType:     "vscode",
		Name:        "VS Code",
		ProjectPath: "/project2",
	})

	ideAdapter.AddIDEInstance("windsurf-1", "windsurf", &adapters.IDEConfig{
		IDEType:     "windsurf",
		Name:        "Windsurf",
		ProjectPath: "/project3",
	})

	// عرض جميع نسخ IDE
	ideInstances := ideAdapter.GetAllIDEInstances()
	fmt.Printf("✓ Registered %d IDEs:\n", len(ideInstances))
	for _, inst := range ideInstances {
		fmt.Printf("  - %s (%s)\n", inst.AgentName, inst.InstanceID)
	}

	// ==================== 4. IDE Extensions ====================
	fmt.Println("\n=== 4. Setting up IDE Extensions ===")

	// إضافة extensions داخل VS Code
	ideAdapter.AddIDEExtensionInstance("vscode-cline-1", "vscode", "cline", &adapters.IDEExtensionConfig{
		IDEType:           "vscode",
		ExtensionName:     "cline",
		CommunicationMode: "websocket",
		WebSocketURL:      "ws://localhost:8082",
	})

	ideAdapter.AddIDEExtensionInstance("vscode-copilot-1", "vscode", "copilot", &adapters.IDEExtensionConfig{
		IDEType:           "vscode",
		ExtensionName:     "copilot",
		CommunicationMode: "http",
		HTTPBaseURL:       "http://localhost:3003",
	})

	ideAdapter.AddIDEExtensionInstance("vscode-continue-1", "vscode", "continue", &adapters.IDEExtensionConfig{
		IDEType:           "vscode",
		ExtensionName:     "continue",
		CommunicationMode: "http",
		HTTPBaseURL:       "http://localhost:3002",
	})

	// إضافة extensions داخل Cursor
	ideAdapter.AddIDEExtensionInstance("cursor-cline-1", "cursor", "cline", &adapters.IDEExtensionConfig{
		IDEType:           "cursor",
		ExtensionName:     "cline",
		CommunicationMode: "websocket",
		WebSocketURL:      "ws://localhost:8084",
	})

	// عرض جميع نسخ Extensions
	extInstances := ideAdapter.GetAllExtensionInstances()
	fmt.Printf("✓ Registered %d IDE extensions:\n", len(extInstances))
	for _, inst := range extInstances {
		fmt.Printf("  - %s (%s)\n", inst.AgentName, inst.InstanceID)
	}

	// ==================== 5. تنفيذ المهام ====================
	fmt.Println("\n=== 5. Executing Tasks ===")

	task := &agent.AgentTask{
		ID:          "task-1",
		Title:       "Create a REST API",
		Description: "Create a simple REST API with Go",
	}

	// تنفيذ على Claude Code فقط
	fmt.Println("\n5.1 Execute on Claude Code only:")
	result, err := cliAdapter.ExecuteOnCLI(ctx, "claude-code-1", task)
	if err != nil {
		fmt.Printf("  ✗ Error: %v\n", err)
	} else {
		fmt.Printf("  ✓ Success: %v\n", result.Success)
	}

	// تنفيذ على جميع CLI agents
	fmt.Println("\n5.2 Execute on all CLI agents:")
	results, err := cliAdapter.ExecuteOnAllCLI(ctx, task)
	if err != nil {
		fmt.Printf("  ✗ Error: %v\n", err)
	} else {
		fmt.Printf("  ✓ Executed on %d agents\n", len(results))
		for id := range results {
			fmt.Printf("    - %s\n", id)
		}
	}

	// تنفيذ على جميع IDEs
	fmt.Println("\n5.3 Execute on all IDEs:")
	results, err = ideAdapter.ExecuteOnAllIDEs(ctx, task)
	if err != nil {
		fmt.Printf("  ✗ Error: %v\n", err)
	} else {
		fmt.Printf("  ✓ Executed on %d IDEs\n", len(results))
	}

	// تنفيذ على جميع extensions
	fmt.Println("\n5.4 Execute on all IDE extensions:")
	results, err = ideAdapter.ExecuteOnAllExtensions(ctx, task)
	if err != nil {
		fmt.Printf("  ✗ Error: %v\n", err)
	} else {
		fmt.Printf("  ✓ Executed on %d extensions\n", len(results))
	}

	// ==================== 6. الإحصائيات ====================
	fmt.Println("\n=== 6. Statistics ===")
	fmt.Printf("CLI Agents: %d\n", len(cliAdapter.GetAllCLIInstances()))
	fmt.Printf("Desktop Apps: %d\n", len(desktopAdapter.GetAllDesktopInstances()))
	fmt.Printf("IDEs: %d\n", len(ideAdapter.GetAllIDEInstances()))
	fmt.Printf("IDE Extensions: %d\n", len(ideAdapter.GetAllExtensionInstances()))
	fmt.Printf("Total: %d agents\n",
		len(cliAdapter.GetAllCLIInstances())+
			len(desktopAdapter.GetAllDesktopInstances())+
			len(ideAdapter.GetAllIDEInstances())+
			len(ideAdapter.GetAllExtensionInstances()))

	// ==================== 7. Extensions لـ IDE معين ====================
	fmt.Println("\n=== 7. Extensions for VS Code ===")
	vscodeExtensions := ideAdapter.GetExtensionsByIDE("vscode")
	fmt.Printf("VS Code has %d extensions:\n", len(vscodeExtensions))
	for _, ext := range vscodeExtensions {
		fmt.Printf("  - %s\n", ext.AgentName)
	}

	fmt.Println("\n=== Multi-Agent Setup Complete ===")
}
