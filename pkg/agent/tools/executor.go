package tools

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// [WHY] ToolExecutor ينفذ الأدوات مع حدود أمان ونظام صلاحيات
// [HOW] يفرض حدود على استدعاءات الأدوات وحجم الملفات والمسارات ويدمج مع ToolRegistry
// [SAFETY] يمنع الحلقات اللانهائية والوصول غير المصرح به
type ToolExecutor struct {
	// حدود الأمان
	MaxToolCallsPerTask int    // [WHY] الحد الأقصى لاستدعاءات الأدوات (50)
	MaxFileSizeBytes    int64  // [WHY] الحد الأقصى لحجم الملف (10MB)
	AllowedBasePath     string // [WHY] المسار المسموح (مجلد الجلسة)

	// حالة التنفيذ
	taskCallCount map[string]int // [WHY] عداد استدعاءات الأدوات لكل مهمة
	taskCallMu    sync.RWMutex   // [SAFETY] لحماية العدادات

	// مدير أقفال الملفات
	fileLockManager *FileLockManager // [WHY] يدير أقفال الملفات لمنع التعارضات

	// [NEW] Registry + Role للتحكم بالصلاحيات
	registry   *ToolRegistry // [WHY] سجل الأدوات للتحقق من الصلاحيات
	agentRole  AgentRole     // [WHY] دور الوكيل الحالي

	// Logger
	logger *zap.Logger
}

// [WHY] NewToolExecutor ينشئ منفذ أدوات جديد بدون registry
func NewToolExecutor(allowedBasePath string, logger *zap.Logger) *ToolExecutor {
	if allowedBasePath == "" {
		allowedBasePath = "."
	}

	return &ToolExecutor{
		MaxToolCallsPerTask: 50,
		MaxFileSizeBytes:    10 * 1024 * 1024,
		AllowedBasePath:     allowedBasePath,
		taskCallCount:       make(map[string]int),
		fileLockManager:     NewFileLockManager("", logger),
		agentRole:           RoleRegular,
		logger:              logger,
	}
}

// [WHY] NewToolExecutorWithRegistry ينشئ منفذ أدوات مع registry ونظام صلاحيات
// [HOW] يهيئ الحدود والعدادات ويسجل registry ودور الوكيل
func NewToolExecutorWithRegistry(allowedBasePath string, registry *ToolRegistry, role AgentRole, logger *zap.Logger) *ToolExecutor {
	if allowedBasePath == "" {
		allowedBasePath = "."
	}
	if registry == nil {
		registry = NewToolRegistry()
	}
	if role == "" {
		role = RoleRegular
	}

	return &ToolExecutor{
		MaxToolCallsPerTask: 50,
		MaxFileSizeBytes:    10 * 1024 * 1024,
		AllowedBasePath:     allowedBasePath,
		taskCallCount:       make(map[string]int),
		fileLockManager:     NewFileLockManager("", logger),
		registry:            registry,
		agentRole:           role,
		logger:              logger,
	}
}

// SetRegistry يضبط سجل الأدوات
func (te *ToolExecutor) SetRegistry(registry *ToolRegistry) {
	te.registry = registry
}

// SetAgentRole يضبط دور الوكيل
func (te *ToolExecutor) SetAgentRole(role AgentRole) {
	te.agentRole = role
}

// GetAgentRole يعيد دور الوكيل الحالي
func (te *ToolExecutor) GetAgentRole() AgentRole {
	return te.agentRole
}

// GetRegistry يعيد سجل الأدوات
func (te *ToolExecutor) GetRegistry() *ToolRegistry {
	return te.registry
}

// [WHY] ExecuteTool ينفذ أداة مع نظام صلاحيات كامل
// [HOW] 1. فحص العداد 2. فحص الصلاحية 3. فحص المسارات 4. أقفال الملفات 5. التنفيذ
// [SAFETY] ثلاث طبقات أمان: عداد، صلاحية، مسار
func (te *ToolExecutor) ExecuteTool(ctx context.Context, taskID, toolName string, params map[string]interface{}) (interface{}, error) {
	// [SAFETY] الطبقة 1: فحص العدادات (atomic check+increment)
	if !te.tryAcquireToolCall(taskID) {
		return nil, fmt.Errorf("تجاوز الحد الأقصى لاستدعاءات الأدوات: %d", te.MaxToolCallsPerTask)
	}

	// [SAFETY] الطبقة 2: فحص الصلاحية عبر registry
	if te.registry != nil {
		def, err := te.registry.Get(toolName)
		if err != nil {
			return nil, fmt.Errorf("أداة غير موجودة: %s", toolName)
		}
		if !def.HasPermission(te.agentRole) {
			return nil, fmt.Errorf("صلاحية مرفوضة: الدور %s لا يمكنه استخدام أداة %s", te.agentRole, toolName)
		}
	}

	// [SAFETY] الطبقة 3: فحص المسارات للملفات
	if toolName == "read_file" || toolName == "write_file" || toolName == "file_list" || toolName == "file_delete" {
		filePath, ok := params["path"].(string)
		if !ok {
			return nil, fmt.Errorf("المعامل path مطلوب")
		}
		if !te.isPathAllowed(filePath) {
			return nil, fmt.Errorf("المسار غير مسموح: %s", filePath)
		}
		if toolName == "read_file" || toolName == "file_list" {
			if err := te.checkFileSize(filePath); err != nil {
				if toolName == "file_list" {
					// file_list يتجاهل خطأ الحجم
				} else {
					return nil, err
				}
			}
		}

		// [SAFETY] أقفال الملفات للكتابة والحذف
		if toolName == "write_file" || toolName == "file_delete" {
			absPath := filepath.Join(te.AllowedBasePath, filePath)
			if err := te.fileLockManager.Lock(ctx, absPath, taskID); err != nil {
				return nil, fmt.Errorf("فشل الحصول على قفل الملف: %w", err)
			}
			defer te.fileLockManager.Unlock(absPath)
		}
	}

	// [HOW] تنفيذ الأداة
	result, err := te.executeToolInternal(ctx, toolName, params)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// [WHY] executeToolInternal ينفذ الأداة فعلياً
// [HOW] يحاول أولاً من الأدوات المدمجة، ثم من registry
// [SAFETY] يستخدم context للإلغاء
func (te *ToolExecutor) executeToolInternal(ctx context.Context, toolName string, params map[string]interface{}) (interface{}, error) {
	// [HOW] الأدوات المدمجة (عمليات ملفات + HTTP + بحث)
	switch toolName {
	case "read_file":
		return te.readFile(ctx, params)
	case "write_file":
		return te.writeFile(ctx, params)
	case "file_list":
		return te.listFiles(ctx, params)
	case "file_delete":
		return te.deleteFile(ctx, params)
	case "http_request":
		return te.httpRequest(ctx, params)
	case "web_search":
		return te.webSearch(ctx, params)
	case "file_search":
		return te.fileSearch(ctx, params)
	case "content_grep":
		return te.contentGrep(ctx, params)
	case "edit_file":
		return te.editFile(ctx, params)
	case "run_tests":
		return te.runTests(ctx, params)
	case "git_operation":
		return te.gitOperation(ctx, params)
	}

	// [HOW] إذا كانت الأداة مسجلة في registry، ننفذها عبر handler
	if te.registry != nil {
		result, err := te.registry.Execute(ctx, te.agentRole, toolName, params)
		if err == nil {
			return result, nil
		}
		// إذا كان الخطأ "tool not found"، نكمل للرسالة الافتراضية
		if !strings.Contains(err.Error(), "tool not found") {
			return nil, err
		}
	}

	return nil, fmt.Errorf("أداة غير مدعومة: %s", toolName)
}

// ============================================================
// أدوات الملفات
// ============================================================

func (te *ToolExecutor) readFile(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("المعامل path مطلوب")
	}
	absPath := filepath.Join(te.AllowedBasePath, path)
	data, err := te.readFileWithContext(ctx, absPath)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"content": string(data),
		"path":    path,
	}, nil
}

func (te *ToolExecutor) readFileWithContext(ctx context.Context, path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buf := make([]byte, 4096)
	var result []byte
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			n, err := file.Read(buf)
			if err != nil && err != io.EOF {
				return nil, err
			}
			if n == 0 {
				return result, nil
			}
			result = append(result, buf[:n]...)
		}
	}
}

func (te *ToolExecutor) writeFile(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("المعامل path مطلوب")
	}
	content, ok := params["content"].(string)
	if !ok {
		return nil, fmt.Errorf("المعامل content مطلوب")
	}
	absPath := filepath.Join(te.AllowedBasePath, path)

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return nil, fmt.Errorf("فشل إنشاء المجلد: %w", err)
	}

	err := te.writeFileWithContext(ctx, absPath, []byte(content))
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"success": true,
		"path":    path,
	}, nil
}

func (te *ToolExecutor) writeFileWithContext(ctx context.Context, path string, data []byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	chunkSize := 32768
	for i := 0; i < len(data); i += chunkSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			end := i + chunkSize
			if end > len(data) {
				end = len(data)
			}
			if _, err := file.Write(data[i:end]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (te *ToolExecutor) listFiles(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, _ := params["path"].(string)
	if path == "" {
		path = "."
	}
	absPath := filepath.Join(te.AllowedBasePath, path)

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("فشل قراءة المجلد: %w", err)
	}

	files := make([]map[string]interface{}, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, map[string]interface{}{
			"name":  entry.Name(),
			"dir":   entry.IsDir(),
			"size":  info.Size(),
			"mtime": info.ModTime(),
		})
	}

	return map[string]interface{}{
		"path":  path,
		"files": files,
		"count": len(files),
	}, nil
}

func (te *ToolExecutor) deleteFile(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("المعامل path مطلوب")
	}
	absPath := filepath.Join(te.AllowedBasePath, path)

	if err := os.Remove(absPath); err != nil {
		return nil, fmt.Errorf("فشل حذف الملف: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"path":    path,
	}, nil
}

// ============================================================
// أمان HTTP - SSRF Protection
// ============================================================

func isPrivateURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return true
	}
	if parsed.Scheme != "https" {
		return true
	}
	host := parsed.Hostname()

	blocked := []string{
		"localhost", "127.", "10.", "192.168.", "172.16.",
		"169.254.", "::1", "[::1]", "0.0.0.0",
	}
	for _, b := range blocked {
		if strings.HasPrefix(host, b) {
			return true
		}
	}

	ip := net.ParseIP(host)
	if ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return true
		}
	}

	metadataEndpoints := []string{
		"metadata.google.internal",
		"169.254.169.254",
		"metadata.azure.net",
	}
	for _, endpoint := range metadataEndpoints {
		if host == endpoint {
			return true
		}
	}

	return false
}

func (te *ToolExecutor) httpRequest(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	url, ok := params["url"].(string)
	if !ok {
		return nil, fmt.Errorf("المعامل url مطلوب")
	}
	if isPrivateURL(url) {
		return nil, fmt.Errorf("SSRF: private/internal URLs not allowed: %s", url)
	}

	method, _ := params["method"].(string)
	if method == "" {
		method = "GET"
	}

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			if isPrivateURL(req.URL.String()) {
				return fmt.Errorf("redirect to private URL not allowed: %s", req.URL.String())
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status_code": resp.StatusCode,
		"body":        string(body),
	}, nil
}

// ============================================================
// أدوات البحث
// ============================================================

func (te *ToolExecutor) webSearch(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query, ok := params["query"].(string)
	if !ok {
		return nil, fmt.Errorf("المعامل query مطلوب")
	}
	// البحث عبر HTTP GET لمحرك بحث عام (DuckDuckGo)
	searchURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1", url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Musketeers-Agent/1.0")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("فشل البحث: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"query":   query,
		"results": string(body),
		"source":  "duckduckgo",
	}, nil
}

func (te *ToolExecutor) fileSearch(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	pattern, ok := params["pattern"].(string)
	if !ok {
		return nil, fmt.Errorf("المعامل pattern مطلوب")
	}
	root, _ := params["path"].(string)
	if root == "" {
		root = "."
	}
	absRoot := filepath.Join(te.AllowedBasePath, root)
	if !te.isPathAllowed(root) {
		return nil, fmt.Errorf("المسار غير مسموح: %s", root)
	}
	matches := make([]string, 0)
	err := filepath.WalkDir(absRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			// تجاهل المجلدات المخفية والمجلدات النظامية
			if strings.HasPrefix(d.Name(), ".") || d.Name() == "node_modules" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.Contains(d.Name(), pattern) {
			rel, _ := filepath.Rel(te.AllowedBasePath, path)
			matches = append(matches, rel)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("فشل البحث في الملفات: %w", err)
	}
	return map[string]interface{}{
		"pattern": pattern,
		"matches": matches,
		"count":   len(matches),
	}, nil
}

// ============================================================
// أدوات جديدة: content_grep, edit_file, run_tests, git_operation
// ============================================================

func (te *ToolExecutor) contentGrep(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	pattern, ok := params["pattern"].(string)
	if !ok {
		return nil, fmt.Errorf("المعامل pattern مطلوب (regex)")
	}
	root, _ := params["path"].(string)
	if root == "" {
		root = "."
	}
	includePattern, _ := params["include"].(string)
	absRoot := filepath.Join(te.AllowedBasePath, root)
	if !te.isPathAllowed(root) {
		return nil, fmt.Errorf("المسار غير مسموح: %s", root)
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("نمط regex غير صالح: %w", err)
	}
	type match struct {
		File    string `json:"file"`
		Line    int    `json:"line"`
		Content string `json:"content"`
	}
	results := make([]match, 0)
	err = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") || d.Name() == "node_modules" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if includePattern != "" {
			if matched, _ := filepath.Match(includePattern, d.Name()); !matched {
				return nil
			}
		}
		rel, _ := filepath.Rel(te.AllowedBasePath, path)
		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if re.MatchString(line) {
				results = append(results, match{File: rel, Line: lineNum, Content: strings.TrimSpace(line)})
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("فشل البحث في المحتوى: %w", err)
	}
	return map[string]interface{}{
		"pattern": pattern,
		"results": results,
		"count":   len(results),
	}, nil
}

func (te *ToolExecutor) editFile(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("المعامل path مطلوب")
	}
	operation, _ := params["operation"].(string)
	absPath := filepath.Join(te.AllowedBasePath, path)
	if !te.isPathAllowed(path) {
		return nil, fmt.Errorf("المسار غير مسموح: %s", path)
	}

	switch operation {
	case "read":
		data, err := te.readFileWithContext(ctx, absPath)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"content": string(data),
			"path":    path,
		}, nil

	case "append":
		content, ok := params["content"].(string)
		if !ok {
			return nil, fmt.Errorf("المعامل content مطلوب لعملية append")
		}
		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			return nil, fmt.Errorf("فشل إنشاء المجلد: %w", err)
		}
		file, err := os.OpenFile(absPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("فشل فتح الملف: %w", err)
		}
		defer file.Close()
		if _, err := file.WriteString(content); err != nil {
			return nil, fmt.Errorf("فشل الكتابة: %w", err)
		}
		return map[string]interface{}{
			"success": true,
			"path":    path,
			"action":  "append",
		}, nil

	case "replace":
		oldStr, ok := params["old"].(string)
		if !ok {
			return nil, fmt.Errorf("المعامل old مطلوب لعملية replace")
		}
		newStr, ok2 := params["new"].(string)
		if !ok2 {
			return nil, fmt.Errorf("المعامل new مطلوب لعملية replace")
		}
		// قراءة الملف
		data, err := te.readFileWithContext(ctx, absPath)
		if err != nil {
			return nil, err
		}
		oldContent := string(data)
		if !strings.Contains(oldContent, oldStr) {
			return nil, fmt.Errorf("النص المطلوب استبداله غير موجود في الملف")
		}
		newContent := strings.ReplaceAll(oldContent, oldStr, newStr)
		// إنشاء نسخة احتياطية
		backupPath := absPath + ".bak"
		if err := te.writeFileWithContext(ctx, backupPath, data); err != nil {
			return nil, fmt.Errorf("فشل إنشاء نسخة احتياطية: %w", err)
		}
		// كتابة المحتوى الجديد
		if err := te.writeFileWithContext(ctx, absPath, []byte(newContent)); err != nil {
			// استعادة النسخة الاحتياطية إذا فشلت الكتابة
			restore, _ := os.ReadFile(backupPath)
			os.WriteFile(absPath, restore, 0644)
			return nil, fmt.Errorf("فشل كتابة الملف بعد الاستبدال: %w", err)
		}
		return map[string]interface{}{
			"success":      true,
			"path":         path,
			"backup_path":  path + ".bak",
			"action":       "replace",
			"replacements": strings.Count(oldContent, oldStr),
		}, nil

	default:
		// كتابة كاملة (السلوك الافتراضي)
		content, ok := params["content"].(string)
		if !ok {
			return nil, fmt.Errorf("المعامل content مطلوب")
		}
		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			return nil, fmt.Errorf("فشل إنشاء المجلد: %w", err)
		}
		if err := te.writeFileWithContext(ctx, absPath, []byte(content)); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"success": true,
			"path":    path,
			"action":  "write",
		}, nil
	}
}

func (te *ToolExecutor) runTests(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	command, ok := params["command"].(string)
	if !ok {
		command = "go test ./..."
	}
	dir, _ := params["dir"].(string)
	timeoutSec := 300
	if t, ok := params["timeout"].(float64); ok && t > 0 {
		timeoutSec = int(t)
	}
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()
	runDir := te.AllowedBasePath
	if dir != "" {
		absDir := filepath.Join(te.AllowedBasePath, dir)
		runDir = absDir
	}
	cmd := exec.CommandContext(execCtx, "cmd", "/C", command)
	cmd.Dir = runDir
	output, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return map[string]interface{}{
		"command":   command,
		"output":    string(output),
		"exit_code": exitCode,
		"success":   exitCode == 0,
	}, nil
}

func (te *ToolExecutor) gitOperation(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	operation, ok := params["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("المعامل operation مطلوب (clone, add, commit, push, pull, branch, checkout, status, log)")
	}
	dir, _ := params["dir"].(string)
	runDir := te.AllowedBasePath
	if dir != "" {
		runDir = filepath.Join(te.AllowedBasePath, dir)
	}
	var cmd *exec.Cmd
	switch operation {
	case "status":
		cmd = exec.CommandContext(ctx, "git", "-C", runDir, "status")
	case "log":
		limit := 10
		if l, ok := params["limit"].(float64); ok {
			limit = int(l)
		}
		cmd = exec.CommandContext(ctx, "git", "-C", runDir, "log", fmt.Sprintf("--max-count=%d", limit), "--oneline")
	case "add":
		files, _ := params["files"].(string)
		if files == "" {
			files = "."
		}
		cmd = exec.CommandContext(ctx, "git", "-C", runDir, "add", files)
	case "commit":
		message, _ := params["message"].(string)
		if message == "" {
			return nil, fmt.Errorf("المعامل message مطلوب لعملية commit")
		}
		cmd = exec.CommandContext(ctx, "git", "-C", runDir, "commit", "-m", message)
	case "push":
		remote, _ := params["remote"].(string)
		branch, _ := params["branch"].(string)
		args := []string{"-C", runDir, "push"}
		if remote != "" {
			args = append(args, remote)
		}
		if branch != "" {
			args = append(args, branch)
		}
		cmd = exec.CommandContext(ctx, "git", args...)
	case "pull":
		remote, _ := params["remote"].(string)
		branch, _ := params["branch"].(string)
		args := []string{"-C", runDir, "pull"}
		if remote != "" {
			args = append(args, remote)
		}
		if branch != "" {
			args = append(args, branch)
		}
		cmd = exec.CommandContext(ctx, "git", args...)
	case "branch":
		branchName, _ := params["branch"].(string)
		action, _ := params["git_action"].(string)
		args := []string{"-C", runDir, "branch"}
		if action == "create" && branchName != "" {
			args = append(args, branchName)
		} else if action == "delete" && branchName != "" {
			args = append(args, "-d", branchName)
		} else if action == "list" {
			// فقط git branch
		}
		cmd = exec.CommandContext(ctx, "git", args...)
	case "checkout":
		branch, _ := params["branch"].(string)
		if branch == "" {
			return nil, fmt.Errorf("المعامل branch مطلوب لعملية checkout")
		}
		create, _ := params["create"].(bool)
		args := []string{"-C", runDir, "checkout"}
		if create {
			args = append(args, "-b")
		}
		args = append(args, branch)
		cmd = exec.CommandContext(ctx, "git", args...)
	case "diff":
		argsParam, _ := params["args"].(string)
		args := []string{"-C", runDir, "diff"}
		if argsParam != "" {
			args = append(args, strings.Fields(argsParam)...)
		}
		cmd = exec.CommandContext(ctx, "git", args...)
	default:
		argsParam, _ := params["args"].(string)
		args := []string{"-C", runDir, operation}
		if argsParam != "" {
			args = append(args, strings.Fields(argsParam)...)
		}
		cmd = exec.CommandContext(ctx, "git", args...)
	}
	output, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return map[string]interface{}{
		"operation": operation,
		"output":    string(output),
		"exit_code": exitCode,
		"success":   exitCode == 0,
	}, nil
}

// ============================================================
// أدوات المساعدة
// ============================================================

func (te *ToolExecutor) tryAcquireToolCall(taskID string) bool {
	te.taskCallMu.Lock()
	defer te.taskCallMu.Unlock()
	count := te.taskCallCount[taskID]
	if count >= te.MaxToolCallsPerTask {
		return false
	}
	te.taskCallCount[taskID] = count + 1
	return true
}

func (te *ToolExecutor) isPathAllowed(path string) bool {
	cleanPath := filepath.Clean(path)
	if filepath.IsAbs(cleanPath) {
		return false
	}
	if strings.Contains(cleanPath, "..") {
		return false
	}
	absPath, err := filepath.Abs(filepath.Join(te.AllowedBasePath, cleanPath))
	if err != nil {
		return false
	}
	allowedAbsPath, err := filepath.Abs(te.AllowedBasePath)
	if err != nil {
		return false
	}
	// المسموح: المسار يساوي تماماً المسار الأساسي أو يبدأ به + separator
	if absPath == allowedAbsPath {
		return true
	}
	return strings.HasPrefix(absPath, allowedAbsPath+string(filepath.Separator))
}

func (te *ToolExecutor) checkFileSize(path string) error {
	absPath := filepath.Join(te.AllowedBasePath, path)
	info, err := os.Stat(absPath)
	if err != nil {
		return nil // الملف غير موجود
	}
	if info.Size() > te.MaxFileSizeBytes {
		return fmt.Errorf("حجم الملف يتجاوز الحد الأقصى: %d bytes", info.Size())
	}
	return nil
}

// ResetTaskCallCount يصفر عداد مهمة
func (te *ToolExecutor) ResetTaskCallCount(taskID string) {
	te.taskCallMu.Lock()
	defer te.taskCallMu.Unlock()
	delete(te.taskCallCount, taskID)
}

// GetTaskCallCount يحصل على عداد مهمة
func (te *ToolExecutor) GetTaskCallCount(taskID string) int {
	te.taskCallMu.RLock()
	defer te.taskCallMu.RUnlock()
	return te.taskCallCount[taskID]
}

// GetAvailableTools يعيد قائمة الأدوات المسموحة لدور الوكيل الحالي
func (te *ToolExecutor) GetAvailableTools() []ToolInfo {
	if te.registry == nil {
		return nil
	}
	return te.registry.GetToolsByRole(te.agentRole)
}
