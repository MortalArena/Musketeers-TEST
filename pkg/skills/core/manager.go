package core

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

// UnifiedSkillsManager مدير المهارات الموحد
type UnifiedSkillsManager struct {
	sessionID string
	logger    *zap.Logger
	mu        sync.RWMutex

	// المهارات القابلة للتنفيذ
	skills   map[string]*Skill
	loader   *SkillLoader
	executor *SkillExecutor

	// مهارات الوكلاء وتطورها
	agentSkills map[string]*AgentSkill

	// إحصائيات
	totalSkills    int
	enabledSkills  int
	disabledSkills int
}

// Skill مهارة قابلة للتنفيذ
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

// AgentSkill مهارات وكيل واحد
type AgentSkill struct {
	AgentDID        string            `json:"agent_did"`
	AgentType       string            `json:"agent_type"`    // coder, designer, tester, etc.
	OverallLevel    int               `json:"overall_level"` // 0-100
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

// SkillLoader يحمل المهارات من الملفات
type SkillLoader struct {
	logger *zap.Logger
}

// SkillExecutor ينفذ المهارات
type SkillExecutor struct {
	logger *zap.Logger
}

// NewUnifiedSkillsManager ينشئ مدير مهارات موحد جديد
func NewUnifiedSkillsManager(sessionID string, logger *zap.Logger) *UnifiedSkillsManager {
	return &UnifiedSkillsManager{
		sessionID:   sessionID,
		logger:      logger,
		skills:      make(map[string]*Skill),
		loader:      NewSkillLoader(logger),
		executor:    NewSkillExecutor(logger),
		agentSkills: make(map[string]*AgentSkill),
	}
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
func (usm *UnifiedSkillsManager) AddSkillDir(dir string) error {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("دليل المهارات غير موجود: %s", dir)
	}

	// تحميل المهارات من الدليل
	if err := usm.loadSkillsFromDir(dir); err != nil {
		return fmt.Errorf("فشل تحميل المهارات من %s: %w", dir, err)
	}

	usm.logger.Info("تم إضافة دليل مهارات",
		zap.String("dir", dir),
		zap.String("session_id", usm.sessionID))
	return nil
}

// loadSkillsFromDir يحمل المهارات من دليل محدد
func (usm *UnifiedSkillsManager) loadSkillsFromDir(dir string) error {
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
			skill, err := usm.loader.LoadSkill(skillPath)
			if err != nil {
				usm.logger.Warn("فشل تحميل مهارة",
					zap.String("skill", entry.Name()),
					zap.Error(err))
				continue
			}

			usm.skills[skill.Name] = skill
			usm.totalSkills++
			if !skill.Disabled {
				usm.enabledSkills++
			} else {
				usm.disabledSkills++
			}

			usm.logger.Info("تم تحميل مهارة",
				zap.String("name", skill.Name),
				zap.String("description", skill.Description))
		}
	}

	return nil
}

// RegisterAgent يسجل وكيلاً ويمنحه مهارات ابتدائية
func (usm *UnifiedSkillsManager) RegisterAgent(agentDID, agentType string) error {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	if _, exists := usm.agentSkills[agentDID]; exists {
		return fmt.Errorf("الوكيل مسجل بالفعل")
	}

	skill := &AgentSkill{
		AgentDID:      agentDID,
		AgentType:     agentType,
		OverallLevel:  50, // يبدأ من 50
		Skills:        make(map[string]*Skill),
		LastEvolution: getCurrentTimestamp(),
	}

	// منح مهارات ابتدائية حسب النوع
	switch agentType {
	case "coder":
		skill.Skills["python"] = &Skill{Name: "Python", Level: 70, Experience: 1000}
		skill.Skills["javascript"] = &Skill{Name: "JavaScript", Level: 70, Experience: 1000}
		skill.Skills["database"] = &Skill{Name: "Database Design", Level: 60, Experience: 500}
		skill.Specializations = []string{"backend", "fullstack"}

	case "designer":
		skill.Skills["ui_design"] = &Skill{Name: "UI Design", Level: 75, Experience: 1200}
		skill.Skills["ux_research"] = &Skill{Name: "UX Research", Level: 70, Experience: 1000}
		skill.Skills["figma"] = &Skill{Name: "Figma", Level: 80, Experience: 1500}
		skill.Specializations = []string{"web", "mobile"}

	case "tester":
		skill.Skills["unit_testing"] = &Skill{Name: "Unit Testing", Level: 80, Experience: 1500}
		skill.Skills["integration_testing"] = &Skill{Name: "Integration Testing", Level: 75, Experience: 1200}
		skill.Skills["security_testing"] = &Skill{Name: "Security Testing", Level: 75, Experience: 1200}
		skill.Specializations = []string{"qa", "automation"}
	}

	usm.agentSkills[agentDID] = skill

	usm.logger.Info("تم تسجيل وكيل",
		zap.String("agent_did", agentDID),
		zap.String("agent_type", agentType))

	return nil
}

// GetSkill يحصل على مهارة بالاسم
func (usm *UnifiedSkillsManager) GetSkill(name string) (*Skill, error) {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	skill, exists := usm.skills[name]
	if !exists {
		return nil, fmt.Errorf("المهارة غير موجودة: %s", name)
	}

	return skill, nil
}

// GetAllSkills يحصل على جميع المهارات
func (usm *UnifiedSkillsManager) GetAllSkills() []*Skill {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	skills := make([]*Skill, 0, len(usm.skills))
	for _, skill := range usm.skills {
		skills = append(skills, skill)
	}

	return skills
}

// SearchSkills يبحث عن مهارات بناءً على الكلمات المفتاحية
func (usm *UnifiedSkillsManager) SearchSkills(query string) []*Skill {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	query = strings.ToLower(query)
	results := make([]*Skill, 0)

	for _, skill := range usm.skills {
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
func (usm *UnifiedSkillsManager) ExecuteSkill(ctx context.Context, skillName string, agentCtx map[string]interface{}) (*SkillResult, error) {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	skill, exists := usm.skills[skillName]
	if !exists {
		return nil, fmt.Errorf("المهارة غير موجودة: %s", skillName)
	}

	if skill.Disabled {
		return nil, fmt.Errorf("المهارة معطلة: %s", skillName)
	}

	return usm.executor.ExecuteSkill(ctx, skill, agentCtx)
}

// GetAgentSkill يحصل على مهارات وكيل
func (usm *UnifiedSkillsManager) GetAgentSkill(agentDID string) (*AgentSkill, error) {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	skill, exists := usm.agentSkills[agentDID]
	if !exists {
		return nil, fmt.Errorf("الوكيل غير مسجل")
	}

	return skill, nil
}

// GetSummary يحصل على ملخص المهارات
func (usm *UnifiedSkillsManager) GetSummary() map[string]interface{} {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	return map[string]interface{}{
		"session_id":        usm.sessionID,
		"total_skills":      usm.totalSkills,
		"enabled_skills":    usm.enabledSkills,
		"disabled_skills":   usm.disabledSkills,
		"registered_agents": len(usm.agentSkills),
	}
}

// LoadSkill يحمل مهارة من مسار محدد
func (sl *SkillLoader) LoadSkill(skillPath string) (*Skill, error) {
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
func (se *SkillExecutor) ExecuteSkill(ctx context.Context, skill *Skill, agentCtx map[string]interface{}) (*SkillResult, error) {
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
func (se *SkillExecutor) executeScript(ctx context.Context, scriptPath string, agentCtx map[string]interface{}) error {
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

// getCurrentTimestamp يحصل على الوقت الحالي
func getCurrentTimestamp() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
