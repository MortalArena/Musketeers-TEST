package autodiscovery

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/agent/adapters"
	"go.uber.org/zap"
)

// AutoDiscovery نظام الاكتشاف التلقائي للوكلاء على جهاز العميل
type AutoDiscovery struct {
	logger           *zap.Logger
	agentRegistry    *agent.AgentRegistry
	discoveredAgents map[string]*DiscoveredAgent
	mu               sync.RWMutex
}

// DiscoveredAgent وكيل مكتشف تلقائياً
type DiscoveredAgent struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"` // ide, cli, desktop
	AgentType  agent.AgentType        `json:"agent_type"`
	Executable string                 `json:"executable"`
	Version    string                 `json:"version"`
	Status     string                 `json:"status"` // available, unavailable
	LastSeen   time.Time              `json:"last_seen"`
	Metadata   map[string]interface{} `json:"metadata"`
	Approved   bool                   `json:"approved"` // هل وافق العميل على الربط
	ApprovedAt time.Time              `json:"approved_at,omitempty"`
}

// NewAutoDiscovery ينشئ نظام اكتشاف تلقائي جديد
func NewAutoDiscovery(logger *zap.Logger, agentRegistry *agent.AgentRegistry) *AutoDiscovery {
	return &AutoDiscovery{
		logger:           logger,
		agentRegistry:    agentRegistry,
		discoveredAgents: make(map[string]*DiscoveredAgent),
	}
}

// DiscoverAll يكتشف جميع الوكلاء المتاحة على جهاز العميل
func (ad *AutoDiscovery) DiscoverAll(ctx context.Context) ([]*DiscoveredAgent, error) {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	ad.logger.Info("بدء الاكتشاف التلقائي للوكلاء AI")

	var discovered []*DiscoveredAgent

	// 1. اكتشاف IDEs مع AI
	ideAgents, err := ad.discoverIDEs(ctx)
	if err != nil {
		ad.logger.Warn("فشل اكتشاف IDEs", zap.Error(err))
	} else {
		// فلترة IDEs للحصول على IDEs مع AI فقط
		aiIDEs := ad.filterAIAgents(ideAgents)
		discovered = append(discovered, aiIDEs...)
		ad.logger.Info("اكتشف IDEs مع AI", zap.Int("count", len(aiIDEs)))
	}

	// 2. اكتشاف أدوات CLI AI فقط
	cliAgents, err := ad.discoverCLITools(ctx)
	if err != nil {
		ad.logger.Warn("فشل اكتشاف أدوات CLI", zap.Error(err))
	} else {
		// فلترة أدوات CLI للحصول على أدوات AI فقط
		aiCLITools := ad.filterAIAgents(cliAgents)
		discovered = append(discovered, aiCLITools...)
		ad.logger.Info("اكتشف أدوات CLI AI", zap.Int("count", len(aiCLITools)))
	}

	// 3. اكتشاف تطبيقات سطح المكتب AI
	desktopAgents, err := ad.discoverDesktopApps(ctx)
	if err != nil {
		ad.logger.Warn("فشل اكتشاف تطبيقات سطح المكتب", zap.Error(err))
	} else {
		// فلترة تطبيقات سطح المكتب للحصول على تطبيقات AI فقط
		aiDesktopApps := ad.filterAIAgents(desktopAgents)
		discovered = append(discovered, aiDesktopApps...)
		ad.logger.Info("اكتشف تطبيقات سطح المكتب AI", zap.Int("count", len(aiDesktopApps)))
	}

	// حفظ الوكلاء المكتشفة
	for _, agent := range discovered {
		ad.discoveredAgents[agent.ID] = agent
	}

	ad.logger.Info("اكتمل الاكتشاف التلقائي للوكلاء AI", zap.Int("total", len(discovered)))

	return discovered, nil
}

// filterAIAgents يفلتر الوكلاء للحصول على الوكلاء AI فقط
func (ad *AutoDiscovery) filterAIAgents(agents []*DiscoveredAgent) []*DiscoveredAgent {
	var aiAgents []*DiscoveredAgent

	for _, agent := range agents {
		if ad.isAIAgent(agent) {
			aiAgents = append(aiAgents, agent)
		}
	}

	return aiAgents
}

// isAIAgent يتحقق مما إذا كان الوكيل وكيل AI
func (ad *AutoDiscovery) isAIAgent(agent *DiscoveredAgent) bool {
	nameLower := strings.ToLower(agent.Name)
	idLower := strings.ToLower(agent.ID)

	// قائمة الكلمات الدالة للوكلاء AI
	aiKeywords := []string{
		"claude", "gpt", "openai", "codex", "cursor", "hermes", "chatgpt",
		"copilot", "ai", "llm", "assistant", "bot", "agent", "cody",
		"tabnine", "kite", "blackbox", "replit", "sourcegraph",
	}

	// التحقق من الكلمات الدالة
	for _, keyword := range aiKeywords {
		if strings.Contains(nameLower, keyword) || strings.Contains(idLower, keyword) {
			return true
		}
	}

	// التحقق من البيانات الوصفية
	if agent.Metadata != nil {
		if aiType, ok := agent.Metadata["ai_type"].(string); ok {
			if aiType != "" {
				return true
			}
		}
		if appType, ok := agent.Metadata["app_type"].(string); ok {
			if strings.Contains(strings.ToLower(appType), "ai") {
				return true
			}
		}
	}

	return false
}

// discoverIDEs يكتشف IDEs المثبتة على جهاز العميل
func (ad *AutoDiscovery) discoverIDEs(ctx context.Context) ([]*DiscoveredAgent, error) {
	var discovered []*DiscoveredAgent

	// المسارات الشائعة لـ IDEs AI على أنظمة التشغيل المختلفة
	aiIDEPaths := []string{}

	switch runtime.GOOS {
	case "windows":
		aiIDEPaths = []string{
			filepath.Join("C:", "Program Files", "Cursor"),
			filepath.Join("C:", "Users", os.Getenv("USERNAME"), "AppData", "Local", "Programs", "Cursor"),
			filepath.Join("C:", "Program Files", "Windsurf"),
			filepath.Join("C:", "Users", os.Getenv("USERNAME"), "AppData", "Local", "Programs", "Windsurf"),
			filepath.Join("C:", "Program Files", "Antigravity"),
			filepath.Join("C:", "Users", os.Getenv("USERNAME"), "AppData", "Local", "Programs", "Antigravity"),
			filepath.Join("C:", "Program Files", "Zed"),
			filepath.Join("C:", "Users", os.Getenv("USERNAME"), "AppData", "Local", "Programs", "Zed"),
			filepath.Join("C:", "Program Files", "Amp"),
			filepath.Join("C:", "Users", os.Getenv("USERNAME"), "AppData", "Local", "Programs", "Amp"),
			filepath.Join("C:", "Program Files", "Trae"),
			filepath.Join("C:", "Users", os.Getenv("USERNAME"), "AppData", "Local", "Programs", "Trae"),
			filepath.Join("C:", "Program Files", "BLACKBOX"),
			filepath.Join("C:", "Users", os.Getenv("USERNAME"), "AppData", "Local", "Programs", "BLACKBOX"),
			filepath.Join("C:", "Program Files", "Kiro"),
			filepath.Join("C:", "Users", os.Getenv("USERNAME"), "AppData", "Local", "Programs", "Kiro"),
			filepath.Join("C:", "Program Files", "Qoder"),
			filepath.Join("C:", "Users", os.Getenv("USERNAME"), "AppData", "Local", "Programs", "Qoder"),
		}
	case "darwin":
		aiIDEPaths = []string{
			filepath.Join("/Applications", "Cursor.app"),
			filepath.Join("/Applications", "Windsurf.app"),
			filepath.Join("/Applications", "Antigravity.app"),
			filepath.Join("/Applications", "Zed.app"),
			filepath.Join("/Applications", "Amp.app"),
			filepath.Join("/Applications", "Trae.app"),
			filepath.Join("/Applications", "BLACKBOX.app"),
			filepath.Join("/Applications", "Kiro.app"),
			filepath.Join("/Applications", "Qoder.app"),
		}
	case "linux":
		aiIDEPaths = []string{
			filepath.Join("/opt", "cursor"),
			filepath.Join("/opt", "windsurf"),
			filepath.Join("/opt", "antigravity"),
			filepath.Join("/usr", "local", "zed"),
			filepath.Join("/opt", "amp"),
			filepath.Join("/opt", "trae"),
			filepath.Join("/opt", "blackbox"),
			filepath.Join("/opt", "kiro"),
			filepath.Join("/opt", "qoder"),
		}
	}

	// فحص كل مسار
	for _, path := range aiIDEPaths {
		if _, err := os.Stat(path); err == nil {
			// اكتشف IDE AI
			ideName := ad.extractIDEName(path)
			if ideAgent := ad.checkIDEGeneral(ctx, path, ideName); ideAgent != nil {
				// تحديد نوع الوكيل AI
				ideAgent.Metadata["ai_type"] = "ide_agent"
				ideAgent.Metadata["ai_category"] = ad.categorizeAIAgent(ideName)
				discovered = append(discovered, ideAgent)
			}
		}
	}

	// فحص أيضاً من خلال الأوامر الشائعة لـ IDEs AI
	aiIDECommands := []string{
		"cursor",      // Cursor AI IDE
		"windsurf",    // Windsurf AI IDE
		"antigravity", // Antigravity AI IDE
		"zed",         // Zed AI IDE
		"amp",         // Amp AI IDE
		"trae",        // Trae AI IDE
		"blackbox",    // BLACKBOX AI IDE
		"kiro",        // Kiro AI IDE
		"qoder",       // Qoder AI IDE
	}

	for _, cmd := range aiIDECommands {
		if ideAgent := ad.checkIDECommand(ctx, cmd); ideAgent != nil {
			// تحديد نوع الوكيل AI
			ideAgent.Metadata["ai_type"] = "ide_agent"
			ideAgent.Metadata["ai_category"] = ad.categorizeAIAgent(cmd)
			discovered = append(discovered, ideAgent)
		}
	}

	ad.logger.Info("اكتشف IDEs AI", zap.Int("count", len(discovered)))

	return discovered, nil
}

// extractIDEName يستخرج اسم IDE من المسار
func (ad *AutoDiscovery) extractIDEName(path string) string {
	// استخراج اسم المجلد الأخير
	parts := strings.Split(path, string(filepath.Separator))
	if len(parts) > 0 {
		name := parts[len(parts)-1]
		// إزالة .app على Mac
		name = strings.TrimSuffix(name, ".app")
		return name
	}
	return "Unknown IDE"
}

// checkIDEGeneral يتحقق من وجود IDE بشكل عام
func (ad *AutoDiscovery) checkIDEGeneral(ctx context.Context, path, name string) *DiscoveredAgent {
	// محاولة الحصول على version
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// على Windows، نحتاج للبحث عن الملف التنفيذي
		executable := filepath.Join(path, "Code.exe")
		if _, err := os.Stat(executable); err != nil {
			executable = filepath.Join(path, "cursor.exe")
		}
		if _, err := os.Stat(executable); err != nil {
			executable = filepath.Join(path, name+".exe")
		}
		cmd = exec.CommandContext(ctx, executable, "--version")
	} else {
		cmd = exec.CommandContext(ctx, name, "--version")
	}

	output, err := cmd.CombinedOutput()
	version := strings.TrimSpace(string(output))

	if err != nil {
		version = "unknown"
	}

	// تصنيف IDE
	agentType := ad.classifyIDE(name)

	return &DiscoveredAgent{
		ID:         fmt.Sprintf("ide_%s", strings.ToLower(strings.ReplaceAll(name, " ", "_"))),
		Name:       name,
		Type:       "ide",
		AgentType:  agentType,
		Executable: path,
		Version:    version,
		Status:     "available",
		LastSeen:   time.Now(),
		Metadata: map[string]interface{}{
			"ide_type": strings.ToLower(name),
			"path":     path,
		},
	}
}

// checkIDECommand يتحقق من وجود IDE من خلال الأمر
func (ad *AutoDiscovery) checkIDECommand(ctx context.Context, command string) *DiscoveredAgent {
	cmd := exec.CommandContext(ctx, command, "--version")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return nil
	}

	version := strings.TrimSpace(string(output))

	// تصنيف IDE
	agentType := ad.classifyIDE(command)

	return &DiscoveredAgent{
		ID:         fmt.Sprintf("ide_%s", command),
		Name:       command,
		Type:       "ide",
		AgentType:  agentType,
		Executable: command,
		Version:    version,
		Status:     "available",
		LastSeen:   time.Now(),
		Metadata: map[string]interface{}{
			"ide_type": command,
		},
	}
}

// classifyIDE يصنف IDE تلقائياً
func (ad *AutoDiscovery) classifyIDE(name string) agent.AgentType {
	nameLower := strings.ToLower(name)

	// IDEs مع AI
	if strings.Contains(nameLower, "cursor") ||
		strings.Contains(nameLower, "copilot") ||
		strings.Contains(nameLower, "ai") {
		return agent.AgentTypeCustom
	}

	// IDEs عادية
	return agent.AgentTypeIDE
}

// checkVSCode يتحقق من وجود VSCode
func (ad *AutoDiscovery) checkVSCode(ctx context.Context) *DiscoveredAgent {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "code", "--version")
	case "darwin":
		cmd = exec.CommandContext(ctx, "/Applications/Visual Studio Code.app/Contents/MacOS/Electron", "--version")
	case "linux":
		cmd = exec.CommandContext(ctx, "code", "--version")
	default:
		return nil
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	version := strings.TrimSpace(string(output))
	return &DiscoveredAgent{
		ID:         "ide_vscode",
		Name:       "VSCode",
		Type:       "ide",
		AgentType:  agent.AgentTypeIDE,
		Executable: "code",
		Version:    version,
		Status:     "available",
		LastSeen:   time.Now(),
		Metadata: map[string]interface{}{
			"ide_type": "vscode",
		},
	}
}

// checkCursor يتحقق من وجود Cursor
func (ad *AutoDiscovery) checkCursor(ctx context.Context) *DiscoveredAgent {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "cursor", "--version")
	case "darwin":
		cmd = exec.CommandContext(ctx, "/Applications/Cursor.app/Contents/MacOS/Cursor", "--version")
	case "linux":
		cmd = exec.CommandContext(ctx, "cursor", "--version")
	default:
		return nil
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	version := strings.TrimSpace(string(output))
	return &DiscoveredAgent{
		ID:         "ide_cursor",
		Name:       "Cursor",
		Type:       "ide",
		AgentType:  agent.AgentTypeIDE,
		Executable: "cursor",
		Version:    version,
		Status:     "available",
		LastSeen:   time.Now(),
		Metadata: map[string]interface{}{
			"ide_type": "cursor",
		},
	}
}

// checkJetBrains يتحقق من وجود JetBrains
func (ad *AutoDiscovery) checkJetBrains(ctx context.Context) *DiscoveredAgent {
	possiblePaths := []string{
		filepath.Join(os.Getenv("HOME"), "Applications", "JetBrains"),
		filepath.Join("C:", "Program Files", "JetBrains"),
		"/opt/jetbrains",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return &DiscoveredAgent{
				ID:         "ide_jetbrains",
				Name:       "JetBrains IDE",
				Type:       "ide",
				AgentType:  agent.AgentTypeIDE,
				Executable: path,
				Version:    "unknown",
				Status:     "available",
				LastSeen:   time.Now(),
				Metadata: map[string]interface{}{
					"ide_type": "jetbrains",
					"path":     path,
				},
			}
		}
	}

	return nil
}

// discoverCLITools يكتشف أدوات CLI المتاحة على جهاز العميل
func (ad *AutoDiscovery) discoverCLITools(ctx context.Context) ([]*DiscoveredAgent, error) {
	var discovered []*DiscoveredAgent

	// قائمة شاملة من أدوات AI CLI المتاحة في 2026
	aiCLITools := []string{
		// أدوات AI CLI الرئيسية
		"claude",        // Claude Code CLI
		"codex",         // OpenAI Codex CLI
		"aider",         // Aider - terminal AI pair programmer
		"opencode",      // OpenCode - open source AI coding
		"goose",         // Goose - Block's open agent
		"gemini-cli",    // Google Gemini CLI
		"gh",            // GitHub Copilot CLI
		"openclaw",      // OpenClaw AI
		"iflow",         // iFlow CLI
		"kimi-code-cli", // Kimi Code CLI
		"cline",         // Cline CLI
		"roo-code",      // Roo Code CLI
		"continue",      // Continue CLI
		"tabnine",       // Tabnine CLI
		"kilo-code",     // Kilo Code CLI
		"blackbox",      // BLACKBOX CLI
		"kiro",          // Kiro CLI
		"qoder",         // Qoder CLI
		"amp",           // Amp CLI
		"windsurf",      // Windsurf CLI
		"antigravity",   // Antigravity CLI
		"mistral-vibe",  // Mistral Vibe CLI
		"ollama",        // Ollama - local AI runtime
		"llama.cpp",     // llama.cpp - local AI runtime
		"lm-studio",     // LM Studio - local AI runtime
		"vllm",          // vLLM - local AI runtime
		"tabby",         // Tabby - local AI runtime
	}

	// فحص كل أداة AI CLI
	for _, tool := range aiCLITools {
		if agent := ad.checkCLIToolGeneral(ctx, tool, tool); agent != nil {
			// تحديد نوع الوكيل AI
			agent.Metadata["ai_type"] = "cli_agent"
			agent.Metadata["ai_category"] = ad.categorizeAIAgent(tool)
			discovered = append(discovered, agent)
		}
	}

	ad.logger.Info("اكتشف أدوات AI CLI", zap.Int("count", len(discovered)))

	return discovered, nil
}

// categorizeAIAgent يصنف وكيل AI حسب نوعه
func (ad *AutoDiscovery) categorizeAIAgent(name string) string {
	nameLower := strings.ToLower(name)

	// أدوات AI الرئيسية
	if strings.Contains(nameLower, "claude") {
		return "anthropic"
	}
	if strings.Contains(nameLower, "codex") {
		return "openai"
	}
	if strings.Contains(nameLower, "gemini") {
		return "google"
	}
	if strings.Contains(nameLower, "copilot") || strings.Contains(nameLower, "gh") {
		return "github"
	}

	// أدوات AI مفتوحة المصدر
	if strings.Contains(nameLower, "aider") || strings.Contains(nameLower, "opencode") ||
		strings.Contains(nameLower, "goose") || strings.Contains(nameLower, "cline") ||
		strings.Contains(nameLower, "roo-code") || strings.Contains(nameLower, "continue") ||
		strings.Contains(nameLower, "kilo-code") {
		return "open_source"
	}

	// أدوات AI محلية
	if strings.Contains(nameLower, "ollama") || strings.Contains(nameLower, "llama.cpp") ||
		strings.Contains(nameLower, "lm-studio") || strings.Contains(nameLower, "vllm") ||
		strings.Contains(nameLower, "tabby") {
		return "local"
	}

	// أدوات AI أخرى
	return "other"
}

// filterSystemTools يفلتر أدوات النظام الداخلية
func (ad *AutoDiscovery) filterSystemTools(executables map[string]string) map[string]string {
	filtered := make(map[string]string)

	// أدوات النظام الداخلية التي يجب تخطيها
	systemTools := map[string]bool{
		// Windows system tools
		"cmd": true, "powershell": true, "pwsh": true, "where": true,
		"whoami": true, "hostname": true, "echo": true, "type": true,
		"copy": true, "move": true, "del": true, "dir": true,
		"cls": true, "exit": true, "pause": true,
		"timeout": true, "choice": true, "find": true,
		"sort": true, "fc": true, "comp": true, "expand": true,
		"attrib": true, "icacls": true, "takeown": true,
		// Linux/Mac system tools
		"ls": true, "pwd": true,
		"cat": true, "less": true, "head": true, "tail": true,
		"cp": true, "mv": true, "rm": true, "mkdir": true, "rmdir": true,
		"chmod": true, "chown": true, "ps": true, "kill": true, "top": true,
		"df": true, "du": true, "free": true, "uname": true, "uptime": true,
		"man": true, "help": true, "which": true, "whereis": true,
	}

	for name, path := range executables {
		nameLower := strings.ToLower(name)
		if !systemTools[nameLower] {
			filtered[name] = path
		}
	}

	return filtered
}

// checkCLITool يتحقق من وجود أداة CLI
func (ad *AutoDiscovery) checkCLITool(ctx context.Context, tool string) *DiscoveredAgent {
	cmd := exec.CommandContext(ctx, tool, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// بعض الأدوات لا تدعم --version، جرب بدون
		cmd = exec.CommandContext(ctx, tool)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return nil
		}
	}

	version := strings.TrimSpace(string(output))
	if version == "" {
		version = "unknown"
	}

	return &DiscoveredAgent{
		ID:         fmt.Sprintf("cli_%s", tool),
		Name:       tool,
		Type:       "cli",
		AgentType:  agent.AgentTypeCLI,
		Executable: tool,
		Version:    version,
		Status:     "available",
		LastSeen:   time.Now(),
		Metadata: map[string]interface{}{
			"command": tool,
		},
	}
}

// checkCLIToolGeneral يتحقق من وجود أداة CLI بشكل عام
func (ad *AutoDiscovery) checkCLIToolGeneral(ctx context.Context, name, fullPath string) *DiscoveredAgent {
	// تجربة الحصول على version
	cmd := exec.CommandContext(ctx, name, "--version")
	output, err := cmd.CombinedOutput()
	version := strings.TrimSpace(string(output))

	if err != nil {
		// بعض الأدوات لا تدعم --version، جرب -v
		cmd = exec.CommandContext(ctx, name, "-v")
		output, err = cmd.CombinedOutput()
		version = strings.TrimSpace(string(output))

		if err != nil {
			// بعض الأدوات لا تدعم -v، جرب version
			cmd = exec.CommandContext(ctx, name, "version")
			output, err = cmd.CombinedOutput()
			version = strings.TrimSpace(string(output))

			if err != nil {
				// إذا فشلت جميع المحاولات، نعتبر الأداة موجودة بدون version
				version = "unknown"
			}
		}
	}

	// تصنيف الأداة تلقائياً
	agentType := ad.classifyCLITool(name)

	return &DiscoveredAgent{
		ID:         fmt.Sprintf("cli_%s", name),
		Name:       name,
		Type:       "cli",
		AgentType:  agentType,
		Executable: fullPath,
		Version:    version,
		Status:     "available",
		LastSeen:   time.Now(),
		Metadata: map[string]interface{}{
			"command":   name,
			"full_path": fullPath,
		},
	}
}

// classifyCLITool يصنف أداة CLI تلقائياً
func (ad *AutoDiscovery) classifyCLITool(name string) agent.AgentType {
	nameLower := strings.ToLower(name)

	// أدوات AI
	if strings.Contains(nameLower, "claude") ||
		strings.Contains(nameLower, "gpt") ||
		strings.Contains(nameLower, "openai") ||
		strings.Contains(nameLower, "codex") ||
		strings.Contains(nameLower, "cursor") ||
		strings.Contains(nameLower, "hermes") {
		return agent.AgentTypeCustom
	}

	// أدوات التطوير
	if strings.Contains(nameLower, "git") ||
		strings.Contains(nameLower, "npm") ||
		strings.Contains(nameLower, "yarn") ||
		strings.Contains(nameLower, "cargo") ||
		strings.Contains(nameLower, "pip") ||
		strings.Contains(nameLower, "mvn") ||
		strings.Contains(nameLower, "gradle") ||
		strings.Contains(nameLower, "go") ||
		strings.Contains(nameLower, "rust") ||
		strings.Contains(nameLower, "java") ||
		strings.Contains(nameLower, "python") ||
		strings.Contains(nameLower, "node") {
		return agent.AgentTypeCLI
	}

	// أدوات الحاويات
	if strings.Contains(nameLower, "docker") ||
		strings.Contains(nameLower, "podman") ||
		strings.Contains(nameLower, "k8s") ||
		strings.Contains(nameLower, "kubectl") {
		return agent.AgentTypeCLI
	}

	// أدوات السحابة
	if strings.Contains(nameLower, "aws") ||
		strings.Contains(nameLower, "gcloud") ||
		strings.Contains(nameLower, "az") ||
		strings.Contains(nameLower, "azure") {
		return agent.AgentTypeCLI
	}

	// افتراضياً: CLI
	return agent.AgentTypeCLI
}

// discoverDesktopApps يكتشف تطبيقات سطح المكتب AI المتاحة
func (ad *AutoDiscovery) discoverDesktopApps(ctx context.Context) ([]*DiscoveredAgent, error) {
	var discovered []*DiscoveredAgent

	// المسارات الشائعة لتطبيقات سطح المكتب AI
	aiAppPaths := []string{}

	switch runtime.GOOS {
	case "windows":
		// Program Files
		aiAppPaths = append(aiAppPaths, filepath.Join("C:", "Program Files"))
		aiAppPaths = append(aiAppPaths, filepath.Join("C:", "Program Files (x86)"))
		// AppData
		aiAppPaths = append(aiAppPaths, filepath.Join("C:", "Users", os.Getenv("USERNAME"), "AppData", "Local", "Programs"))
		aiAppPaths = append(aiAppPaths, filepath.Join("C:", "Users", os.Getenv("USERNAME"), "AppData", "Roaming"))
		// Start Menu
		aiAppPaths = append(aiAppPaths, filepath.Join("C:", "ProgramData", "Microsoft", "Windows", "Start Menu", "Programs"))
	case "darwin":
		// Applications folder
		aiAppPaths = append(aiAppPaths, "/Applications")
		// User Applications
		aiAppPaths = append(aiAppPaths, filepath.Join(os.Getenv("HOME"), "Applications"))
	case "linux":
		// /usr/share/applications
		aiAppPaths = append(aiAppPaths, "/usr/share/applications")
		// /usr/local/share/applications
		aiAppPaths = append(aiAppPaths, "/usr/local/share/applications")
		// ~/.local/share/applications
		aiAppPaths = append(aiAppPaths, filepath.Join(os.Getenv("HOME"), ".local", "share", "applications"))
	}

	// فحص كل مسار
	for _, basePath := range aiAppPaths {
		if _, err := os.Stat(basePath); err != nil {
			continue
		}

		// قراءة الملفات في المسار
		entries, err := os.ReadDir(basePath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			// على Mac، التطبيقات تكون .app
			if runtime.GOOS == "darwin" {
				if strings.HasSuffix(entry.Name(), ".app") {
					appPath := filepath.Join(basePath, entry.Name())
					if appAgent := ad.checkDesktopAppGeneral(ctx, appPath, entry.Name()); appAgent != nil {
						// فلترة تطبيقات AI فقط
						if ad.isAIAgent(appAgent) {
							appAgent.Metadata["ai_type"] = "desktop_app"
							appAgent.Metadata["ai_category"] = ad.categorizeAIAgent(entry.Name())
							discovered = append(discovered, appAgent)
						}
					}
				}
			} else if runtime.GOOS == "windows" {
				// على Windows، فحص المجلدات التي تحتوي على ملفات تنفيذية
				if entry.IsDir() {
					appPath := filepath.Join(basePath, entry.Name())
					if appAgent := ad.checkWindowsAppGeneral(ctx, appPath, entry.Name()); appAgent != nil {
						// فلترة تطبيقات AI فقط
						if ad.isAIAgent(appAgent) {
							appAgent.Metadata["ai_type"] = "desktop_app"
							appAgent.Metadata["ai_category"] = ad.categorizeAIAgent(entry.Name())
							discovered = append(discovered, appAgent)
						}
					}
				}
			} else {
				// على Linux، فحص ملفات .desktop
				if strings.HasSuffix(entry.Name(), ".desktop") {
					desktopPath := filepath.Join(basePath, entry.Name())
					if appAgent := ad.checkLinuxDesktopApp(ctx, desktopPath, entry.Name()); appAgent != nil {
						// فلترة تطبيقات AI فقط
						if ad.isAIAgent(appAgent) {
							appAgent.Metadata["ai_type"] = "desktop_app"
							appAgent.Metadata["ai_category"] = ad.categorizeAIAgent(entry.Name())
							discovered = append(discovered, appAgent)
						}
					}
				}
			}
		}
	}

	// فحص تطبيقات سطح المكتب AI الشهيرة بشكل مباشر
	aiDesktopApps := []string{
		"ChatGPT",          // OpenAI ChatGPT Desktop
		"Claude",           // Anthropic Claude Desktop
		"Copilot",          // Microsoft Copilot Desktop
		"Gemini",           // Google Gemini Desktop
		"Perplexity",       // Perplexity Desktop
		"Midjourney",       // Midjourney Desktop
		"Leonardo AI",      // Leonardo AI Desktop
		"Kling AI",         // Kling AI Desktop
		"ElevenLabs",       // ElevenLabs Desktop
		"DeepL",            // DeepL Pro Desktop
		"Google AI Studio", // Google AI Studio Desktop
		"Google Bard",      // Google Bard Desktop
		"Hermes",           // Hermes AI Desktop
		"Codex",            // OpenAI Codex Desktop
	}

	for _, appName := range aiDesktopApps {
		// إنشاء وكيل افتراضي للتطبيق
		agent := &DiscoveredAgent{
			ID:         fmt.Sprintf("desktop_%s", strings.ToLower(strings.ReplaceAll(appName, " ", "_"))),
			Name:       appName,
			Type:       "desktop",
			AgentType:  agent.AgentTypeCustom,
			Executable: appName,
			Version:    "unknown",
			Status:     "available",
			LastSeen:   time.Now(),
			Metadata: map[string]interface{}{
				"ai_type":     "desktop_app",
				"ai_category": ad.categorizeAIAgent(appName),
				"app_type":    "ai_assistant",
			},
		}

		// التحقق من وجود التطبيق
		if ad.checkDesktopAppExists(appName) {
			discovered = append(discovered, agent)
		}
	}

	ad.logger.Info("اكتشف تطبيقات سطح المكتب AI", zap.Int("count", len(discovered)))

	return discovered, nil
}

// checkDesktopAppExists يتحقق من وجود تطبيق سطح المكتب
func (ad *AutoDiscovery) checkDesktopAppExists(appName string) bool {
	// فحص المسارات الشائعة للتطبيقات على أنظمة التشغيل المختلفة
	switch runtime.GOOS {
	case "windows":
		// فحص Program Files
		appPaths := []string{
			filepath.Join("C:", "Program Files", appName),
			filepath.Join("C:", "Program Files (x86)", appName),
			filepath.Join("C:", "Users", os.Getenv("USERNAME"), "AppData", "Local", "Programs", appName),
			filepath.Join("C:", "Users", os.Getenv("USERNAME"), "AppData", "Roaming", appName),
		}
		for _, path := range appPaths {
			if _, err := os.Stat(path); err == nil {
				return true
			}
		}
	case "darwin":
		// فحص Applications folder
		appPath := filepath.Join("/Applications", appName+".app")
		if _, err := os.Stat(appPath); err == nil {
			return true
		}
		// فحص User Applications
		userAppPath := filepath.Join(os.Getenv("HOME"), "Applications", appName+".app")
		if _, err := os.Stat(userAppPath); err == nil {
			return true
		}
	case "linux":
		// فحص مسارات التطبيقات على Linux
		appPaths := []string{
			filepath.Join("/usr/share/applications", appName+".desktop"),
			filepath.Join("/usr/local/share/applications", appName+".desktop"),
			filepath.Join(os.Getenv("HOME"), ".local", "share", "applications", appName+".desktop"),
		}
		for _, path := range appPaths {
			if _, err := os.Stat(path); err == nil {
				return true
			}
		}
	}

	return false
}

// checkDesktopAppGeneral يتحقق من وجود تطبيق سطح المكتب بشكل عام (Mac)
func (ad *AutoDiscovery) checkDesktopAppGeneral(ctx context.Context, path, name string) *DiscoveredAgent {
	// إزالة .app من الاسم
	appName := strings.TrimSuffix(name, ".app")

	// تصنيف التطبيق
	agentType := ad.classifyDesktopApp(appName)

	return &DiscoveredAgent{
		ID:         fmt.Sprintf("desktop_%s", strings.ToLower(strings.ReplaceAll(appName, " ", "_"))),
		Name:       appName,
		Type:       "desktop",
		AgentType:  agentType,
		Executable: path,
		Version:    "unknown",
		Status:     "available",
		LastSeen:   time.Now(),
		Metadata: map[string]interface{}{
			"app_type": strings.ToLower(appName),
			"path":     path,
			"os":       "darwin",
		},
	}
}

// checkWindowsAppGeneral يتحقق من وجود تطبيق Windows بشكل عام
func (ad *AutoDiscovery) checkWindowsAppGeneral(ctx context.Context, path, name string) *DiscoveredAgent {
	// البحث عن ملف تنفيذي في المجلد
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}

	var executablePath string
	for _, entry := range entries {
		if !entry.IsDir() {
			nameLower := strings.ToLower(entry.Name())
			if strings.HasSuffix(nameLower, ".exe") {
				// إذا كان الملف التنفيذي يحمل نفس اسم المجلد أو اسم شائع
				if strings.HasPrefix(nameLower, strings.ToLower(name)) ||
					strings.Contains(nameLower, "launcher") ||
					strings.Contains(nameLower, "app") {
					executablePath = filepath.Join(path, entry.Name())
					break
				}
			}
		}
	}

	if executablePath == "" {
		return nil
	}

	// تصنيف التطبيق
	agentType := ad.classifyDesktopApp(name)

	return &DiscoveredAgent{
		ID:         fmt.Sprintf("desktop_%s", strings.ToLower(strings.ReplaceAll(name, " ", "_"))),
		Name:       name,
		Type:       "desktop",
		AgentType:  agentType,
		Executable: executablePath,
		Version:    "unknown",
		Status:     "available",
		LastSeen:   time.Now(),
		Metadata: map[string]interface{}{
			"app_type": strings.ToLower(name),
			"path":     path,
			"os":       "windows",
		},
	}
}

// checkLinuxDesktopApp يتحقق من وجود تطبيق Linux بشكل عام
func (ad *AutoDiscovery) checkLinuxDesktopApp(ctx context.Context, path, name string) *DiscoveredAgent {
	// قراءة ملف .desktop
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	// استخراج الاسم من ملف .desktop
	var appName string
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Name=") {
			appName = strings.TrimPrefix(line, "Name=")
			break
		}
	}

	if appName == "" {
		appName = strings.TrimSuffix(name, ".desktop")
	}

	// تصنيف التطبيق
	agentType := ad.classifyDesktopApp(appName)

	return &DiscoveredAgent{
		ID:         fmt.Sprintf("desktop_%s", strings.ToLower(strings.ReplaceAll(appName, " ", "_"))),
		Name:       appName,
		Type:       "desktop",
		AgentType:  agentType,
		Executable: path,
		Version:    "unknown",
		Status:     "available",
		LastSeen:   time.Now(),
		Metadata: map[string]interface{}{
			"app_type": strings.ToLower(appName),
			"path":     path,
			"os":       "linux",
		},
	}
}

// classifyDesktopApp يصنف تطبيق سطح المكتب تلقائياً
func (ad *AutoDiscovery) classifyDesktopApp(name string) agent.AgentType {
	nameLower := strings.ToLower(name)

	// تطبيقات AI
	if strings.Contains(nameLower, "claude") ||
		strings.Contains(nameLower, "gpt") ||
		strings.Contains(nameLower, "openai") ||
		strings.Contains(nameLower, "codex") ||
		strings.Contains(nameLower, "cursor") ||
		strings.Contains(nameLower, "hermes") ||
		strings.Contains(nameLower, "chatgpt") ||
		strings.Contains(nameLower, "copilot") ||
		strings.Contains(nameLower, "ai") {
		return agent.AgentTypeCustom
	}

	// تطبيقات سطح المكتب العادية
	return agent.AgentTypeCustom
}

// checkClaudeDesktop يتحقق من وجود Claude Desktop
func (ad *AutoDiscovery) checkClaudeDesktop(ctx context.Context) *DiscoveredAgent {
	possiblePaths := []string{
		filepath.Join(os.Getenv("HOME"), "Applications", "Claude.app"),
		filepath.Join("C:", "Users", os.Getenv("USERNAME"), "AppData", "Local", "Programs", "Claude"),
		filepath.Join("C:", "Users", os.Getenv("USERNAME"), "AppData", "Local", "Programs", "Anthropic", "Claude"),
		"/usr/local/bin/claude",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return &DiscoveredAgent{
				ID:         "desktop_claude",
				Name:       "Claude Desktop",
				Type:       "ai_agent",
				AgentType:  agent.AgentTypeCustom,
				Executable: path,
				Version:    "unknown",
				Status:     "available",
				LastSeen:   time.Now(),
				Metadata: map[string]interface{}{
					"app_type": "claude-desktop",
					"path":     path,
					"ai_type":  "claude",
				},
			}
		}
	}

	return nil
}

// checkCodexApp يتحقق من وجود Codex App
func (ad *AutoDiscovery) checkCodexApp(ctx context.Context) *DiscoveredAgent {
	possiblePaths := []string{
		filepath.Join(os.Getenv("HOME"), "Applications", "Codex.app"),
		filepath.Join("C:", "Users", os.Getenv("USERNAME"), "AppData", "Local", "Programs", "Codex"),
		filepath.Join("C:", "Users", os.Getenv("USERNAME"), "AppData", "Local", "Programs", "OpenAI", "Codex"),
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return &DiscoveredAgent{
				ID:         "desktop_codex",
				Name:       "Codex App",
				Type:       "ai_agent",
				AgentType:  agent.AgentTypeCustom,
				Executable: path,
				Version:    "unknown",
				Status:     "available",
				LastSeen:   time.Now(),
				Metadata: map[string]interface{}{
					"app_type": "codex-app",
					"path":     path,
					"ai_type":  "codex",
				},
			}
		}
	}

	return nil
}

// checkHermes يتحقق من وجود Hermes
func (ad *AutoDiscovery) checkHermes(ctx context.Context) *DiscoveredAgent {
	possiblePaths := []string{
		filepath.Join(os.Getenv("HOME"), "Applications", "Hermes.app"),
		filepath.Join("C:", "Users", os.Getenv("USERNAME"), "AppData", "Local", "Programs", "Hermes"),
		filepath.Join("C:", "Program Files", "Hermes"),
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return &DiscoveredAgent{
				ID:         "desktop_hermes",
				Name:       "Hermes",
				Type:       "ai_agent",
				AgentType:  agent.AgentTypeCustom,
				Executable: path,
				Version:    "unknown",
				Status:     "available",
				LastSeen:   time.Now(),
				Metadata: map[string]interface{}{
					"app_type": "hermes",
					"path":     path,
					"ai_type":  "hermes",
				},
			}
		}
	}

	return nil
}

// checkCursorAI يتحقق من وجود Cursor AI IDE
func (ad *AutoDiscovery) checkCursorAI(ctx context.Context) *DiscoveredAgent {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "cursor", "--version")
	case "darwin":
		cmd = exec.CommandContext(ctx, "/Applications/Cursor.app/Contents/MacOS/Cursor", "--version")
	case "linux":
		cmd = exec.CommandContext(ctx, "cursor", "--version")
	default:
		return nil
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	version := strings.TrimSpace(string(output))
	return &DiscoveredAgent{
		ID:         "ide_cursor_ai",
		Name:       "Cursor AI",
		Type:       "ai_agent",
		AgentType:  agent.AgentTypeIDE,
		Executable: "cursor",
		Version:    version,
		Status:     "available",
		LastSeen:   time.Now(),
		Metadata: map[string]interface{}{
			"ide_type": "cursor",
			"ai_type":  "cursor-ai",
		},
	}
}

// checkVSCodeAI يتحقق من وجود VSCode مع إضافات AI
func (ad *AutoDiscovery) checkVSCodeAI(ctx context.Context) *DiscoveredAgent {
	// التحقق من وجود VSCode
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "code", "--version")
	case "darwin":
		cmd = exec.CommandContext(ctx, "/Applications/Visual Studio Code.app/Contents/MacOS/Electron", "--version")
	case "linux":
		cmd = exec.CommandContext(ctx, "code", "--version")
	default:
		return nil
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	version := strings.TrimSpace(string(output))

	// التحقق من وجود إضافات AI
	extensionsPath := filepath.Join(os.Getenv("USERPROFILE"), ".vscode", "extensions")
	if runtime.GOOS == "darwin" {
		extensionsPath = filepath.Join(os.Getenv("HOME"), ".vscode", "extensions")
	}

	// البحث عن إضافات AI الشائعة
	aiExtensions := []string{
		"GitHub.copilot",
		"GitHub.copilot-chat",
		"AmazonWebServices.aws-toolkit-vscode",
		"ms-python.python",
		"ms-vscode.vscode-typescript-next",
	}

	hasAIExtension := false
	for _, ext := range aiExtensions {
		if _, err := os.Stat(filepath.Join(extensionsPath, ext)); err == nil {
			hasAIExtension = true
			break
		}
	}

	if !hasAIExtension {
		return nil
	}

	return &DiscoveredAgent{
		ID:         "ide_vscode_ai",
		Name:       "VSCode with AI Extensions",
		Type:       "ai_agent",
		AgentType:  agent.AgentTypeIDE,
		Executable: "code",
		Version:    version,
		Status:     "available",
		LastSeen:   time.Now(),
		Metadata: map[string]interface{}{
			"ide_type":          "vscode",
			"ai_type":           "vscode-ai-extensions",
			"has_ai_extensions": true,
		},
	}
}

// GetDiscoveredAgents يعيد جميع الوكلاء المكتشفة
func (ad *AutoDiscovery) GetDiscoveredAgents() []*DiscoveredAgent {
	ad.mu.RLock()
	defer ad.mu.RUnlock()

	agents := make([]*DiscoveredAgent, 0, len(ad.discoveredAgents))
	for _, agent := range ad.discoveredAgents {
		agents = append(agents, agent)
	}

	return agents
}

// GetDiscoveredAgent يعيد وكيل مكتشف محدد
func (ad *AutoDiscovery) GetDiscoveredAgent(id string) (*DiscoveredAgent, error) {
	ad.mu.RLock()
	defer ad.mu.RUnlock()

	agent, exists := ad.discoveredAgents[id]
	if !exists {
		return nil, fmt.Errorf("الوكيل غير موجود: %s", id)
	}

	return agent, nil
}

// ApproveAgent يوافق على ربط وكيل مكتشف
func (ad *AutoDiscovery) ApproveAgent(id string) error {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	agent, exists := ad.discoveredAgents[id]
	if !exists {
		return fmt.Errorf("الوكيل غير موجود: %s", id)
	}

	agent.Approved = true
	agent.ApprovedAt = time.Now()

	ad.logger.Info("تمت الموافقة على الوكيل", zap.String("agent_id", id))

	return nil
}

// RejectAgent يرفض ربط وكيل مكتشف
func (ad *AutoDiscovery) RejectAgent(id string) error {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	agent, exists := ad.discoveredAgents[id]
	if !exists {
		return fmt.Errorf("الوكيل غير موجود: %s", id)
	}

	agent.Approved = false
	agent.Status = "rejected"

	ad.logger.Info("تم رفض الوكيل", zap.String("agent_id", id))

	return nil
}

// RegisterApprovedAgents يسجل الوكلاء الموافق عليها في AgentRegistry
func (ad *AutoDiscovery) RegisterApprovedAgents(ctx context.Context) error {
	ad.mu.RLock()
	defer ad.mu.RUnlock()

	for _, discovered := range ad.discoveredAgents {
		if !discovered.Approved {
			continue
		}

		// إنشاء Adapter حسب النوع
		var adapter agent.UnifiedAgent
		var err error

		switch discovered.Type {
		case "ide":
			adapter, err = ad.createIDEAdapter(discovered)
		case "cli":
			adapter, err = ad.createCLIAdapter(discovered)
		case "desktop":
			adapter, err = ad.createDesktopAdapter(discovered)
		default:
			ad.logger.Warn("نوع وكيل غير مدعوم", zap.String("type", discovered.Type))
			continue
		}

		if err != nil {
			ad.logger.Error("فشل إنشاء Adapter", zap.String("agent_id", discovered.ID), zap.Error(err))
			continue
		}

		// تسجيل في AgentRegistry
		if err := ad.agentRegistry.Register(adapter, nil); err != nil {
			ad.logger.Error("فشل تسجيل الوكيل", zap.String("agent_id", discovered.ID), zap.Error(err))
		} else {
			ad.logger.Info("تم تسجيل الوكيل بنجاح", zap.String("agent_id", discovered.ID))
		}
	}

	return nil
}

// createIDEAdapter ينشئ IDE Adapter
func (ad *AutoDiscovery) createIDEAdapter(discovered *DiscoveredAgent) (agent.UnifiedAgent, error) {
	ideType := "vscode"
	if metadata, ok := discovered.Metadata["ide_type"].(string); ok {
		ideType = metadata
	}

	config := &adapters.IDEConfig{
		IDEType:     ideType,
		Name:        discovered.Name,
		ProjectPath: "./",
	}

	return adapters.NewIDEAdapter(config), nil
}

// createCLIAdapter ينشئ CLI Adapter
func (ad *AutoDiscovery) createCLIAdapter(discovered *DiscoveredAgent) (agent.UnifiedAgent, error) {
	config := &adapters.CLIConfig{
		Command: discovered.Executable,
		Args:    []string{},
		Name:    discovered.Name,
	}

	return adapters.NewCLIAdapter(config), nil
}

// createDesktopAdapter ينشئ Desktop Adapter
func (ad *AutoDiscovery) createDesktopAdapter(discovered *DiscoveredAgent) (agent.UnifiedAgent, error) {
	appPath := discovered.Executable
	if metadata, ok := discovered.Metadata["path"].(string); ok {
		appPath = metadata
	}

	config := &adapters.DesktopAppConfig{
		Name:              discovered.Name,
		Executable:        appPath,
		CommunicationMode: "websocket",
		AutoStart:         false,
	}

	return adapters.NewDesktopAppAdapter(config, ad.logger)
}
