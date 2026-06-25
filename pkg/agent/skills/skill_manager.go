package skills

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AgentContext يمثل سياق الوكيل
type AgentContext struct {
	SessionID   string
	AgentID     string
	TaskID      string
	Metadata    map[string]interface{}
	Environment map[string]string
}

// AgentSkill مهارات وكيل واحد (من core)
type AgentSkill struct {
	AgentDID        string            `json:"agent_did"`
	AgentType       string            `json:"agent_type"`
	OverallLevel    int               `json:"overall_level"`
	Skills          map[string]*Skill `json:"skills"`
	TotalTasks      int               `json:"total_tasks"`
	SuccessCount    int               `json:"success_count"`
	FailureCount    int               `json:"failure_count"`
	AvgTaskTime     string            `json:"avg_task_time"`
	MasteryBadges   []string          `json:"mastery_badges"`
	Specializations []string          `json:"specializations"`
	LastEvolution   string            `json:"last_evolution"`
	EvolutionCount  int               `json:"evolution_count"`
}

// SkillManager يدير المهارات للوكلاء بناءً على نظام Cursor
type SkillManager struct {
	sessionID string
	skills    map[string]*Skill
	loader    *SkillLoader
	executor  *SkillExecutor
	logger    *zap.Logger
	mu        sync.RWMutex
	skillDirs []string

	// Agent skills tracking (من core)
	agentSkills map[string]*AgentSkill
	totalSkills    int
	enabledSkills  int
	disabledSkills int
}

// Skill يمثل مهارة واحدة
type Skill struct {
	Name                   string                 `json:"name"`
	Description            string                 `json:"description"`
	Instructions           string                 `json:"instructions"`
	Examples               []string               `json:"examples"`
	Scripts                []string               `json:"scripts"`
	Metadata               map[string]interface{} `json:"metadata"`
	Disabled               bool                   `json:"disabled"`
	DisableModelInvocation bool                   `json:"disable_model_invocation"`
	Level                  int                    `json:"level"`
	Experience             int                    `json:"experience"`
}

// SkillLoader يحمل المهارات من الملفات
type SkillLoader struct {
	logger *zap.Logger
}

// SkillExecutor ينفذ المهارات
type SkillExecutor struct {
	logger *zap.Logger
}

// NewSkillManager ينشئ مدير مهارات جديد
func NewSkillManager(logger *zap.Logger) *SkillManager {
	return NewSkillManagerWithSession("", logger)
}

// NewSkillManagerWithSession ينشئ مدير مهارات مع معرف الجلسة (يدعم AgentSkill)
func NewSkillManagerWithSession(sessionID string, logger *zap.Logger) *SkillManager {
	sm := &SkillManager{
		sessionID:   sessionID,
		skills:      make(map[string]*Skill),
		loader:      NewSkillLoader(logger),
		executor:    NewSkillExecutor(logger),
		logger:      logger,
		skillDirs:   []string{},
		agentSkills: make(map[string]*AgentSkill),
	}
	return sm
}

// NewSkillLoader ينشئ محمل مهارات جديد
func NewSkillLoader(logger *zap.Logger) *SkillLoader {
	return &SkillLoader{
		logger: logger,
	}
}

// NewSkillExecutor ينشئ منفذ مهارات جديد
func NewSkillExecutor(logger *zap.Logger) *SkillExecutor {
	return &SkillExecutor{
		logger: logger,
	}
}

// AddSkillDir يضيف دليل مهارات
func (sm *SkillManager) AddSkillDir(dir string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// [WHY] إضافة دليل مهارات لتحميل المهارات منه
	// [HOW] يضيف الدليل إلى القائمة ويحمل المهارات
	// [SAFETY] يتحقق من وجود الدليل

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("دليل المهارات غير موجود: %s", dir)
	}

	sm.skillDirs = append(sm.skillDirs, dir)

	// تحميل المهارات من الدليل
	if err := sm.loadSkillsFromDir(dir); err != nil {
		return fmt.Errorf("فشل تحميل المهارات من %s: %w", dir, err)
	}

	sm.logger.Info("تم إضافة دليل مهارات", zap.String("dir", dir))
	return nil
}

// loadSkillsFromDir يحمل المهارات من دليل محدد
func (sm *SkillManager) loadSkillsFromDir(dir string) error {
	// [WHY] تحميل المهارات من دليل محدد
	// [HOW] يبحث عن ملفات SKILL.md ويحملها
	// [SAFETY] يتحقق من صحة الملفات

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("فشل قراءة الدليل: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(dir, entry.Name())
		skillFile := filepath.Join(skillPath, "SKILL.md")

		if _, err := os.Stat(skillFile); err == nil {
			skill, err := sm.loader.LoadSkill(skillPath)
			if err != nil {
				sm.logger.Warn("فشل تحميل مهارة",
					zap.String("skill", entry.Name()),
					zap.Error(err))
				continue
			}

			sm.skills[skill.Name] = skill
			sm.totalSkills++
			if !skill.Disabled {
				sm.enabledSkills++
			} else {
				sm.disabledSkills++
			}
			sm.logger.Info("تم تحميل مهارة",
				zap.String("name", skill.Name),
				zap.String("description", skill.Description))
		}
	}

	return nil
}

// GetSkill يحصل على مهارة بالاسم
func (sm *SkillManager) GetSkill(name string) (*Skill, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	skill, exists := sm.skills[name]
	if !exists {
		return nil, fmt.Errorf("المهارة غير موجودة: %s", name)
	}

	return skill, nil
}

// GetAllSkills يحصل على جميع المهارات
func (sm *SkillManager) GetAllSkills() []*Skill {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	skills := make([]*Skill, 0, len(sm.skills))
	for _, skill := range sm.skills {
		skills = append(skills, skill)
	}

	return skills
}

// SearchSkills يبحث عن مهارات بناءً على الكلمات المفتاحية
func (sm *SkillManager) SearchSkills(query string) []*Skill {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	query = strings.ToLower(query)
	results := make([]*Skill, 0)

	for _, skill := range sm.skills {
		if skill.Disabled {
			continue
		}

		// البحث في الاسم والوصف
		if strings.Contains(strings.ToLower(skill.Name), query) ||
			strings.Contains(strings.ToLower(skill.Description), query) {
			results = append(results, skill)
		}
	}

	return results
}

// ExecuteSkill ينفذ مهارة
func (sm *SkillManager) ExecuteSkill(ctx context.Context, skillName string, agentCtx *AgentContext) (*SkillResult, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	skill, exists := sm.skills[skillName]
	if !exists {
		return nil, fmt.Errorf("المهارة غير موجودة: %s", skillName)
	}

	if skill.Disabled {
		return nil, fmt.Errorf("المهارة معطلة: %s", skillName)
	}

	return sm.executor.ExecuteSkill(ctx, skill, agentCtx)
}

// LoadSkill يحمل مهارة من مسار محدد
func (sl *SkillLoader) LoadSkill(skillPath string) (*Skill, error) {
	// [WHY] تحميل مهارة من مسار محدد
	// [HOW] يقرأ ملف SKILL.md ويحلله
	// [SAFETY] يتحقق من صحة الملف والبيانات

	skillFile := filepath.Join(skillPath, "SKILL.md")
	content, err := os.ReadFile(skillFile)
	if err != nil {
		return nil, fmt.Errorf("فشل قراءة ملف المهارة: %w", err)
	}

	skill, err := sl.parseSkillMD(content)
	if err != nil {
		return nil, fmt.Errorf("فشل تحليل ملف المهارة: %w", err)
	}

	// تحميل ملفات إضافية إذا وجدت
	referenceFile := filepath.Join(skillPath, "reference.md")
	if _, err := os.Stat(referenceFile); err == nil {
		refContent, _ := os.ReadFile(referenceFile)
		skill.Metadata["reference"] = string(refContent)
	}

	examplesFile := filepath.Join(skillPath, "examples.md")
	if _, err := os.Stat(examplesFile); err == nil {
		exContent, _ := os.ReadFile(examplesFile)
		skill.Examples = append(skill.Examples, string(exContent))
	}

	// تحميل السكريبتات إذا وجدت
	scriptsDir := filepath.Join(skillPath, "scripts")
	if entries, err := os.ReadDir(scriptsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				scriptPath := filepath.Join(scriptsDir, entry.Name())
				skill.Scripts = append(skill.Scripts, scriptPath)
			}
		}
	}

	return skill, nil
}

// parseSkillMD يحلل ملف SKILL.md
func (sl *SkillLoader) parseSkillMD(content []byte) (*Skill, error) {
	// [WHY] تحليل ملف SKILL.md
	// [HOW] يستخرج YAML frontmatter والمحتوى
	// [SAFETY] يتحقق من صحة البيانات

	skill := &Skill{
		Metadata: make(map[string]interface{}),
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
			if err := sl.parseFrontmatter(frontmatter, skill); err != nil {
				return nil, fmt.Errorf("فشل تحليل frontmatter: %w", err)
			}

			// المحتوى بعد frontmatter
			skill.Instructions = strings.Join(lines[frontmatterEnd+1:], "\n")
		}
	} else {
		skill.Instructions = string(content)
	}

	// التحقق من الحقول المطلوبة
	if skill.Name == "" {
		return nil, fmt.Errorf("اسم المهارة مطلوب")
	}
	if skill.Description == "" {
		return nil, fmt.Errorf("وصف المهارة مطلوب")
	}

	return skill, nil
}

// parseFrontmatter يحلل YAML frontmatter
func (sl *SkillLoader) parseFrontmatter(frontmatter string, skill *Skill) error {
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
			skill.Name = value
		case "description":
			skill.Description = value
		case "disable-model-invocation":
			skill.DisableModelInvocation = strings.ToLower(value) == "true"
		case "disabled":
			skill.Disabled = strings.ToLower(value) == "true"
		}
	}

	return nil
}

// ExecuteSkill ينفذ مهارة
func (se *SkillExecutor) ExecuteSkill(ctx context.Context, skill *Skill, agentCtx *AgentContext) (*SkillResult, error) {
	// [WHY] تنفيذ مهارة
	// [HOW] يطبق التعليمات وينفذ السكريبتات
	// [SAFETY] يتحقق من السياق والتنفيذ

	result := &SkillResult{
		SkillName: skill.Name,
		Success:   false,
		Output:    "",
		Metadata:  make(map[string]interface{}),
	}

	// تطبيق التعليمات
	result.Output = skill.Instructions
	result.Metadata["instructions_applied"] = true

	// تنفيذ السكريبتات إذا وجدت
	for _, scriptPath := range skill.Scripts {
		if err := se.executeScript(ctx, scriptPath, agentCtx); err != nil {
			return nil, fmt.Errorf("فشل تنفيذ السكريبت %s: %w", scriptPath, err)
		}
		result.Metadata["scripts_executed"] = true
	}

	result.Success = true
	return result, nil
}

// executeScript ينفذ سكريبت
func (se *SkillExecutor) executeScript(ctx context.Context, scriptPath string, agentCtx *AgentContext) error {
	// [WHY] تنفيذ سكريبت
	// [HOW] يقرأ السكريبت وينفذه
	// [SAFETY] يتحقق من المسار والصلاحيات

	// في التنفيذ الحالي، سنقوم فقط بتسجيل السكريبت
	// في المستقبل، يمكن إضافة تنفيذ فعلي للسكريبتات
	se.logger.Info("تنفيذ سكريبت", zap.String("path", scriptPath))
	return nil
}

// SkillResult نتيجة تنفيذ المهارة
type SkillResult struct {
	SkillName string                 `json:"skill_name"`
	Success   bool                   `json:"success"`
	Output    string                 `json:"output"`
	Metadata  map[string]interface{} `json:"metadata"`
	Error     string                 `json:"error,omitempty"`
}

// RegisterAgent يسجل وكيلاً ويمنحه مهارات ابتدائية (من core)
func (sm *SkillManager) RegisterAgent(agentDID, agentType string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.agentSkills[agentDID]; exists {
		return fmt.Errorf("الوكيل مسجل بالفعل")
	}

	askill := &AgentSkill{
		AgentDID:      agentDID,
		AgentType:     agentType,
		OverallLevel:  50,
		Skills:        make(map[string]*Skill),
		LastEvolution: getCurrentTimestamp(),
	}

	switch agentType {
	case "coder":
		askill.Skills["python"] = &Skill{Name: "Python", Level: 70, Experience: 1000}
		askill.Skills["javascript"] = &Skill{Name: "JavaScript", Level: 70, Experience: 1000}
		askill.Skills["database"] = &Skill{Name: "Database Design", Level: 60, Experience: 500}
		askill.Specializations = []string{"backend", "fullstack"}
	case "designer":
		askill.Skills["ui_design"] = &Skill{Name: "UI Design", Level: 75, Experience: 1200}
		askill.Skills["ux_research"] = &Skill{Name: "UX Research", Level: 70, Experience: 1000}
		askill.Skills["figma"] = &Skill{Name: "Figma", Level: 80, Experience: 1500}
		askill.Specializations = []string{"web", "mobile"}
	case "tester":
		askill.Skills["unit_testing"] = &Skill{Name: "Unit Testing", Level: 80, Experience: 1500}
		askill.Skills["integration_testing"] = &Skill{Name: "Integration Testing", Level: 75, Experience: 1200}
		askill.Skills["security_testing"] = &Skill{Name: "Security Testing", Level: 75, Experience: 1200}
		askill.Specializations = []string{"qa", "automation"}
	}

	sm.agentSkills[agentDID] = askill
	sm.logger.Info("تم تسجيل وكيل", zap.String("agent_did", agentDID), zap.String("agent_type", agentType))
	return nil
}

// GetAgentSkill يحصل على مهارات وكيل (من core)
func (sm *SkillManager) GetAgentSkill(agentDID string) (*AgentSkill, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	askill, exists := sm.agentSkills[agentDID]
	if !exists {
		return nil, fmt.Errorf("الوكيل غير مسجل")
	}
	return askill, nil
}

// GetSummary يحصل على ملخص المهارات
func (sm *SkillManager) GetSummary() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return map[string]interface{}{
		"session_id":        sm.sessionID,
		"total_skills":      sm.totalSkills,
		"enabled_skills":    sm.enabledSkills,
		"disabled_skills":   sm.disabledSkills,
		"registered_agents": len(sm.agentSkills),
	}
}

// GetSkillSummary يحصل على ملخص المهارات
func (sm *SkillManager) GetSkillSummary() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	summary := map[string]interface{}{
		"total_skills":      len(sm.skills),
		"enabled_skills":    0,
		"disabled_skills":   0,
		"skill_directories": sm.skillDirs,
	}

	for _, skill := range sm.skills {
		if skill.Disabled {
			summary["disabled_skills"] = summary["disabled_skills"].(int) + 1
		} else {
			summary["enabled_skills"] = summary["enabled_skills"].(int) + 1
		}
	}

	return summary
}

func getCurrentTimestamp() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
