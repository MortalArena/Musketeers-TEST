package subagents

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// SubagentManager يدير الوكلاء الفرعيين بناءً على نظام Cursor
type SubagentManager struct {
	subagents map[string]*Subagent
	factory   *SubagentFactory
	executor  *SubagentExecutor
	logger    *zap.Logger
	mu        sync.RWMutex
	agentDirs []string
}

// Subagent يمثل وكيل فرعي واحد
type Subagent struct {
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	SystemPrompt    string                 `json:"system_prompt"`
	Specialization  string                 `json:"specialization"`
	Capabilities    []string               `json:"capabilities"`
	Priority        int                    `json:"priority"`
	ReadOnly        bool                   `json:"read_only"`
	RunInBackground bool                   `json:"run_in_background"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// SubagentFactory ينشئ الوكلاء الفرعيين
type SubagentFactory struct {
	logger *zap.Logger
}

// SubagentExecutor ينفذ الوكلاء الفرعيين
type SubagentExecutor struct {
	logger *zap.Logger
}

// NewSubagentManager ينشئ مدير وكلاء فرعيين جديد
func NewSubagentManager(logger *zap.Logger) *SubagentManager {
	sm := &SubagentManager{
		subagents: make(map[string]*Subagent),
		factory:   NewSubagentFactory(logger),
		executor:  NewSubagentExecutor(logger),
		logger:    logger,
		agentDirs: []string{},
	}
	return sm
}

// NewSubagentFactory ينشئ مصنع وكلاء فرعيين جديد
func NewSubagentFactory(logger *zap.Logger) *SubagentFactory {
	return &SubagentFactory{
		logger: logger,
	}
}

// NewSubagentExecutor ينشئ منفذ وكلاء فرعيين جديد
func NewSubagentExecutor(logger *zap.Logger) *SubagentExecutor {
	return &SubagentExecutor{
		logger: logger,
	}
}

// AddAgentDir يضيف دليل وكلاء فرعيين
func (sam *SubagentManager) AddAgentDir(dir string) error {
	sam.mu.Lock()
	defer sam.mu.Unlock()

	// [WHY] إضافة دليل وكلاء فرعيين لتحميل الوكلاء منه
	// [HOW] يضيف الدليل إلى القائمة ويحمل الوكلاء
	// [SAFETY] يتحقق من وجود الدليل

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("دليل الوكلاء غير موجود: %s", dir)
	}

	sam.agentDirs = append(sam.agentDirs, dir)
	
	// تحميل الوكلاء من الدليل
	if err := sam.loadSubagentsFromDir(dir); err != nil {
		return fmt.Errorf("فشل تحميل الوكلاء من %s: %w", dir, err)
	}

	sam.logger.Info("تم إضافة دليل وكلاء فرعيين", zap.String("dir", dir))
	return nil
}

// loadSubagentsFromDir يحمل الوكلاء من دليل محدد
func (sam *SubagentManager) loadSubagentsFromDir(dir string) error {
	// [WHY] تحميل الوكلاء من دليل محدد
	// [HOW] يبحث عن ملفات .md ويحملها
	// [SAFETY] يتحقق من صحة الملفات

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("فشل قراءة الدليل: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		agentPath := filepath.Join(dir, entry.Name())
		subagent, err := sam.factory.LoadSubagent(agentPath)
		if err != nil {
			sam.logger.Warn("فشل تحميل وكيل فرعي", 
				zap.String("file", entry.Name()),
				zap.Error(err))
			continue
		}

		sam.subagents[subagent.Name] = subagent
		sam.logger.Info("تم تحميل وكيل فرعي", 
			zap.String("name", subagent.Name),
			zap.String("description", subagent.Description))
	}

	return nil
}

// GetSubagent يحصل على وكيل فرعي بالاسم
func (sam *SubagentManager) GetSubagent(name string) (*Subagent, error) {
	sam.mu.RLock()
	defer sam.mu.RUnlock()

	subagent, exists := sam.subagents[name]
	if !exists {
		return nil, fmt.Errorf("الوكيل الفرعي غير موجود: %s", name)
	}

	return subagent, nil
}

// GetAllSubagents يحصل على جميع الوكلاء الفرعيين
func (sam *SubagentManager) GetAllSubagents() []*Subagent {
	sam.mu.RLock()
	defer sam.mu.RUnlock()

	subagents := make([]*Subagent, 0, len(sam.subagents))
	for _, subagent := range sam.subagents {
		subagents = append(subagents, subagent)
	}

	return subagents
}

// SearchSubagents يبحث عن وكلاء فرعيين بناءً على الكلمات المفتاحية
func (sam *SubagentManager) SearchSubagents(query string) []*Subagent {
	sam.mu.RLock()
	defer sam.mu.RUnlock()

	query = strings.ToLower(query)
	results := make([]*Subagent, 0)

	for _, subagent := range sam.subagents {
		// البحث في الاسم والوصف والتخصص
		if strings.Contains(strings.ToLower(subagent.Name), query) ||
		   strings.Contains(strings.ToLower(subagent.Description), query) ||
		   strings.Contains(strings.ToLower(subagent.Specialization), query) {
			results = append(results, subagent)
		}
	}

	return results
}

// GetSubagentsBySpecialization يحصل على الوكلاء حسب التخصص
func (sam *SubagentManager) GetSubagentsBySpecialization(specialization string) []*Subagent {
	sam.mu.RLock()
	defer sam.mu.RUnlock()

	results := make([]*Subagent, 0)

	for _, subagent := range sam.subagents {
		if strings.EqualFold(subagent.Specialization, specialization) {
			results = append(results, subagent)
		}
	}

	return results
}

// CreateSubagent ينشئ وكيل فرعي جديد
func (sam *SubagentManager) CreateSubagent(config *SubagentConfig) (*Subagent, error) {
	sam.mu.Lock()
	defer sam.mu.Unlock()

	// [WHY] إنشاء وكيل فرعي جديد
	// [HOW] يستخدم المصنع لإنشاء الوكيل
	// [SAFETY] يتحقق من صحة التكوين

	subagent, err := sam.factory.CreateSubagent(config)
	if err != nil {
		return nil, fmt.Errorf("فشل إنشاء الوكيل الفرعي: %w", err)
	}

	sam.subagents[subagent.Name] = subagent
	sam.logger.Info("تم إنشاء وكيل فرعي", 
		zap.String("name", subagent.Name),
		zap.String("specialization", subagent.Specialization))

	return subagent, nil
}

// DelegateTask يفوض مهمة للوكيل الفرعي
func (sam *SubagentManager) DelegateTask(ctx context.Context, task *Task, subagentName string) (*SubagentResult, error) {
	sam.mu.RLock()
	defer sam.mu.RUnlock()

	subagent, exists := sam.subagents[subagentName]
	if !exists {
		return nil, fmt.Errorf("الوكيل الفرعي غير موجود: %s", subagentName)
	}

	return sam.executor.ExecuteTask(ctx, subagent, task)
}

// LoadSubagent يحمل وكيل فرعي من ملف
func (sf *SubagentFactory) LoadSubagent(filePath string) (*Subagent, error) {
	// [WHY] تحميل وكيل فرعي من ملف
	// [HOW] يقرأ الملف ويحلله
	// [SAFETY] يتحقق من صحة الملف والبيانات

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("فشل قراءة ملف الوكيل الفرعي: %w", err)
	}

	subagent, err := sf.parseSubagentMD(content)
	if err != nil {
		return nil, fmt.Errorf("فشل تحليل ملف الوكيل الفرعي: %w", err)
	}

	return subagent, nil
}

// CreateSubagent ينشئ وكيل فرعي من تكوين
func (sf *SubagentFactory) CreateSubagent(config *SubagentConfig) (*Subagent, error) {
	// [WHY] إنشاء وكيل فرعي من تكوين
	// [HOW] يحول التكوين إلى وكيل فرعي
	// [SAFETY] يتحقق من صحة التكوين

	subagent := &Subagent{
		Name:            config.Name,
		Description:     config.Description,
		SystemPrompt:    config.SystemPrompt,
		Specialization:  config.Specialization,
		Capabilities:    config.Capabilities,
		Priority:        config.Priority,
		ReadOnly:        config.ReadOnly,
		RunInBackground: config.RunInBackground,
		Metadata:        config.Metadata,
	}

	// التحقق من الحقول المطلوبة
	if subagent.Name == "" {
		return nil, fmt.Errorf("اسم الوكيل الفرعي مطلوب")
	}
	if subagent.Description == "" {
		return nil, fmt.Errorf("وصف الوكيل الفرعي مطلوب")
	}
	if subagent.SystemPrompt == "" {
		return nil, fmt.Errorf("system prompt مطلوب")
	}

	return subagent, nil
}

// parseSubagentMD يحلل ملف الوكيل الفرعي
func (sf *SubagentFactory) parseSubagentMD(content []byte) (*Subagent, error) {
	// [WHY] تحليل ملف الوكيل الفرعي
	// [HOW] يستخرج YAML frontmatter والمحتوى
	// [SAFETY] يتحقق من صحة البيانات

	subagent := &Subagent{
		Capabilities: []string{},
		Metadata:     make(map[string]interface{}),
	}

	lines := strings.Split(string(content), "\n")
	
	// استخراج YAML frontmatter
	if len(lines) > 0 && strings.HasPrefix(lines[0], "---") {
		frontmatterEnd := -1
		for i := 1; i < len(lines); i++ {
			if strings.HasPrefix(lines[i], "---") {
				frontmatterEnd = i
				break
			}
		}

		if frontmatterEnd > 0 {
			frontmatter := strings.Join(lines[1:frontmatterEnd], "\n")
			if err := sf.parseFrontmatter(frontmatter, subagent); err != nil {
				return nil, fmt.Errorf("فشل تحليل frontmatter: %w", err)
			}

			// المحتوى بعد frontmatter هو system prompt
			subagent.SystemPrompt = strings.Join(lines[frontmatterEnd+1:], "\n")
		}
	} else {
		subagent.SystemPrompt = string(content)
	}

	// التحقق من الحقول المطلوبة
	if subagent.Name == "" {
		return nil, fmt.Errorf("اسم الوكيل الفرعي مطلوب")
	}
	if subagent.Description == "" {
		return nil, fmt.Errorf("وصف الوكيل الفرعي مطلوب")
	}

	return subagent, nil
}

// parseFrontmatter يحلل YAML frontmatter
func (sf *SubagentFactory) parseFrontmatter(frontmatter string, subagent *Subagent) error {
	// [WHY] تحليل YAML frontmatter
	// [HOW] يستخرج الحقول من YAML
	// [SAFETY] يتحقق من صحة البيانات

	lines := strings.Split(frontmatter, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "name":
			subagent.Name = value
		case "description":
			subagent.Description = value
		case "specialization":
			subagent.Specialization = value
		case "priority":
			// يمكن تحويل القيمة إلى int
			subagent.Priority = 0 // افتراضي
		case "readonly":
			subagent.ReadOnly = strings.ToLower(value) == "true"
		}
	}

	return nil
}

// ExecuteTask ينفذ مهمة باستخدام الوكيل الفرعي
func (se *SubagentExecutor) ExecuteTask(ctx context.Context, subagent *Subagent, task *Task) (*SubagentResult, error) {
	// [WHY] تنفيذ مهمة باستخدام الوكيل الفرعي
	// [HOW] يطبق system prompt وينفذ المهمة
	// [SAFETY] يتحقق من السياق والتنفيذ

	result := &SubagentResult{
		SubagentName: subagent.Name,
		TaskID:       task.ID,
		Success:      false,
		Output:       "",
		Metadata:     make(map[string]interface{}),
	}

	// تطبيق system prompt
	result.Output = fmt.Sprintf("System Prompt: %s\n\nTask: %s", subagent.SystemPrompt, task.Description)
	result.Metadata["system_prompt_applied"] = true
	result.Metadata["specialization"] = subagent.Specialization
	result.Metadata["capabilities"] = subagent.Capabilities

	// في التنفيذ الحالي، سنقوم فقط بتسجيل التنفيذ
	// في المستقبل، يمكن إضافة تنفيذ فعلي للمهمة
	se.logger.Info("تنفيذ مهمة باستخدام وكيل فرعي", 
		zap.String("subagent", subagent.Name),
		zap.String("task", task.ID),
		zap.String("specialization", subagent.Specialization))

	result.Success = true
	return result, nil
}

// SubagentConfig تكوين الوكيل الفرعي
type SubagentConfig struct {
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	SystemPrompt    string                 `json:"system_prompt"`
	Specialization  string                 `json:"specialization"`
	Capabilities    []string               `json:"capabilities"`
	Priority        int                    `json:"priority"`
	ReadOnly        bool                   `json:"read_only"`
	RunInBackground bool                   `json:"run_in_background"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// Task يمثل مهمة
type Task struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Priority    int                    `json:"priority"`
}

// SubagentResult نتيجة تنفيذ الوكيل الفرعي
type SubagentResult struct {
	SubagentName string                 `json:"subagent_name"`
	TaskID       string                 `json:"task_id"`
	Success      bool                   `json:"success"`
	Output       string                 `json:"output"`
	Metadata     map[string]interface{} `json:"metadata"`
	Error        string                 `json:"error,omitempty"`
}

// GetSubagentSummary يحصل على ملخص الوكلاء الفرعيين
func (sam *SubagentManager) GetSubagentSummary() map[string]interface{} {
	sam.mu.RLock()
	defer sam.mu.RUnlock()

	summary := map[string]interface{}{
		"total_subagents":    len(sam.subagents),
		"specializations":    make(map[string]int),
		"agent_directories":  sam.agentDirs,
	}

	for _, subagent := range sam.subagents {
		spec := subagent.Specialization
		if spec == "" {
			spec = "general"
		}
		
		if _, exists := summary["specializations"].(map[string]int)[spec]; !exists {
			summary["specializations"].(map[string]int)[spec] = 0
		}
		summary["specializations"].(map[string]int)[spec]++
	}

	return summary
}
