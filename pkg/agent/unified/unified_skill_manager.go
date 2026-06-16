package unified

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

// UnifiedSkillManager نظام المهارات الشامل الذي يدمج جميع وظائف المهارات
type UnifiedSkillManager struct {
	sessionID string
	logger    *zap.Logger
	mu        sync.RWMutex

	// مهارات الوكلاء (من sessionSkills)
	agentSkills map[string]*AgentSkill

	// مهارات المنصة المتقدمة (من skillManager)
	platformSkills map[string]*PlatformSkill

	// أدلة المهارات
	skillDirs []string
}

// AgentSkill مهارات وكيل واحد (من sessionSkills)
type AgentSkill struct {
	AgentDID        string            `json:"agent_did"`
	AgentType       string            `json:"agent_type"`
	OverallLevel    int               `json:"overall_level"`
	Skills          map[string]*Skill `json:"skills"`
	TotalTasks      int               `json:"total_tasks"`
	SuccessCount    int               `json:"success_count"`
	FailureCount    int               `json:"failure_count"`
	AvgTaskTime     time.Duration     `json:"avg_task_time"`
	MasteryBadges   []string          `json:"mastery_badges"`
	Specializations []string          `json:"specializations"`
	LastEvolution   time.Time         `json:"last_evolution"`
	EvolutionCount  int               `json:"evolution_count"`
}

// Skill مهارة محددة (من sessionSkills)
type Skill struct {
	Name        string               `json:"name"`
	Level       int                  `json:"level"`
	Experience  int                  `json:"experience"`
	LastUsed    time.Time            `json:"last_used"`
	UsageCount  int                  `json:"usage_count"`
	SuccessRate float64              `json:"success_rate"`
	SubSkills   map[string]*SubSkill `json:"sub_skills"`
}

// SubSkill مهارة فرعية
type SubSkill struct {
	Name        string  `json:"name"`
	Level       int     `json:"level"`
	Proficiency float64 `json:"proficiency"`
}

// PlatformSkill مهارة المنصة المتقدمة (من skillManager)
type PlatformSkill struct {
	Name                   string                 `json:"name"`
	Description            string                 `json:"description"`
	Instructions           string                 `json:"instructions"`
	Examples               []string               `json:"examples"`
	Scripts                []string               `json:"scripts"`
	Metadata               map[string]interface{} `json:"metadata"`
	Disabled               bool                   `json:"disabled"`
	DisableModelInvocation bool                   `json:"disable_model_invocation"`
}

// SkillTask مهمة مكتملة
type SkillTask struct {
	Name          string        `json:"name"`
	Success       bool          `json:"success"`
	Duration      time.Duration `json:"duration"`
	SkillsUsed    []string      `json:"skills_used"`
	XPGained      int           `json:"xp_gained"`
	LessonLearned string        `json:"lesson_learned"`
}

// AgentContext سياق الوكيل
type AgentContext struct {
	SessionID   string
	AgentID     string
	TaskID      string
	Metadata    map[string]interface{}
	Environment map[string]string
}

// SkillResult نتيجة تنفيذ مهارة
type SkillResult struct {
	Success bool
	Output  interface{}
	Error   error
}

// NewUnifiedSkillManager ينشئ مدير مهارات شامل جديد
func NewUnifiedSkillManager(sessionID string, logger *zap.Logger) *UnifiedSkillManager {
	return &UnifiedSkillManager{
		sessionID:      sessionID,
		logger:         logger,
		agentSkills:    make(map[string]*AgentSkill),
		platformSkills: make(map[string]*PlatformSkill),
		skillDirs:      []string{},
	}
}

// RegisterAgent يسجل وكيلاً ويمنحه مهارات ابتدائية
func (usm *UnifiedSkillManager) RegisterAgent(agentDID, agentType string) error {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	if _, exists := usm.agentSkills[agentDID]; exists {
		return fmt.Errorf("الوكيل مسجل بالفعل")
	}

	skill := &AgentSkill{
		AgentDID:      agentDID,
		AgentType:     agentType,
		OverallLevel:  50,
		Skills:        make(map[string]*Skill),
		LastEvolution: time.Now(),
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
	usm.logger.Info("تم تسجيل وكيل في نظام المهارات الشامل",
		zap.String("agent_did", agentDID),
		zap.String("agent_type", agentType))

	return nil
}

// RecordTaskCompletion يسجل إكمال مهمة ويطور المهارات
func (usm *UnifiedSkillManager) RecordTaskCompletion(agentDID string, task SkillTask) error {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	skill, exists := usm.agentSkills[agentDID]
	if !exists {
		return fmt.Errorf("الوكيل غير مسجل")
	}

	skill.TotalTasks++

	if task.Success {
		skill.SuccessCount++

		// تطوير المهارات المستخدمة
		for _, skillName := range task.SkillsUsed {
			if s, ok := skill.Skills[skillName]; ok {
				s.Experience += task.XPGained
				s.UsageCount++
				s.LastUsed = time.Now()

				// حساب المستوى الجديد
				newLevel := usm.calculateLevel(s.Experience)
				if newLevel > s.Level {
					s.Level = newLevel
					skill.EvolutionCount++
					skill.LastEvolution = time.Now()
				}

				// تحديث معدل النجاح
				s.SuccessRate = float64(s.UsageCount) / float64(skill.TotalTasks)
			}
		}

		// منح شارات الإتقان
		usm.checkMasteryBadges(skill)

	} else {
		skill.FailureCount++
	}

	// تحديث متوسط وقت المهمة
	totalTime := skill.AvgTaskTime * time.Duration(skill.TotalTasks-1)
	totalTime += task.Duration
	skill.AvgTaskTime = totalTime / time.Duration(skill.TotalTasks)

	// تحديث المستوى العام
	skill.OverallLevel = usm.calculateOverallLevel(skill)

	return nil
}

// AddSkillDir يضيف دليل مهارات (من Cursor)
func (usm *UnifiedSkillManager) AddSkillDir(dir string) error {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("دليل المهارات غير موجود: %s", dir)
	}

	usm.skillDirs = append(usm.skillDirs, dir)

	// تحميل المهارات من الدليل
	if err := usm.loadSkillsFromDir(dir); err != nil {
		return fmt.Errorf("فشل تحميل المهارات من %s: %w", dir, err)
	}

	usm.logger.Info("تم إضافة دليل مهارات", zap.String("dir", dir))
	return nil
}

// loadSkillsFromDir يحمل المهارات من دليل محدد
func (usm *UnifiedSkillManager) loadSkillsFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			skillDir := filepath.Join(dir, entry.Name())
			skillFile := filepath.Join(skillDir, "SKILL.md")

			if _, err := os.Stat(skillFile); err == nil {
				skill, err := usm.loadSkillFromFile(skillFile)
				if err != nil {
					usm.logger.Warn("فشل تحميل مهارة", zap.String("file", skillFile), zap.Error(err))
					continue
				}

				usm.platformSkills[skill.Name] = skill
				usm.logger.Info("تم تحميل مهارة", zap.String("name", skill.Name))
			}
		}
	}

	return nil
}

// loadSkillFromFile يحمل مهارة من ملف
func (usm *UnifiedSkillManager) loadSkillFromFile(filePath string) (*PlatformSkill, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	skill := &PlatformSkill{
		Name:         filepath.Base(filepath.Dir(filePath)),
		Instructions: string(content),
		Metadata:     make(map[string]interface{}),
	}

	return skill, nil
}

// GetSkill يحصل على مهارة المنصة المتقدمة
func (usm *UnifiedSkillManager) GetSkill(name string) (*PlatformSkill, error) {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	skill, exists := usm.platformSkills[name]
	if !exists {
		return nil, fmt.Errorf("المهارة غير موجودة: %s", name)
	}

	return skill, nil
}

// SearchSkills يبحث عن مهارات
func (usm *UnifiedSkillManager) SearchSkills(query string) []*PlatformSkill {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	results := []*PlatformSkill{}
	queryLower := strings.ToLower(query)

	for _, skill := range usm.platformSkills {
		if strings.Contains(strings.ToLower(skill.Name), queryLower) ||
			strings.Contains(strings.ToLower(skill.Description), queryLower) {
			results = append(results, skill)
		}
	}

	return results
}

// ExecuteSkill ينفذ مهارة
func (usm *UnifiedSkillManager) ExecuteSkill(ctx context.Context, skillName string, agentCtx *AgentContext) (*SkillResult, error) {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	skill, exists := usm.platformSkills[skillName]
	if !exists {
		return &SkillResult{Success: false, Error: fmt.Errorf("المهارة غير موجودة: %s", skillName)}, nil
	}

	if skill.Disabled {
		return &SkillResult{Success: false, Error: fmt.Errorf("المهارة معطلة: %s", skillName)}, nil
	}

	// تنفيذ المهارة (محاكاة)
	result := &SkillResult{
		Success: true,
		Output:  fmt.Sprintf("تم تنفيذ مهارة: %s", skillName),
	}

	return result, nil
}

// GetAllSkills يحصل على جميع مهارات المنصة المتقدمة
func (usm *UnifiedSkillManager) GetAllSkills() []*PlatformSkill {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	skills := make([]*PlatformSkill, 0, len(usm.platformSkills))
	for _, skill := range usm.platformSkills {
		skills = append(skills, skill)
	}

	return skills
}

// GetSkillSummary يحصل على ملخص المهارات
func (usm *UnifiedSkillManager) GetSkillSummary() map[string]interface{} {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	return map[string]interface{}{
		"total_agent_skills":    len(usm.agentSkills),
		"total_platform_skills": len(usm.platformSkills),
		"skill_dirs":            usm.skillDirs,
	}
}

// checkMasteryBadges يتحقق ويمنح شارات الإتقان
func (usm *UnifiedSkillManager) checkMasteryBadges(skill *AgentSkill) {
	// شارة "Expert" - إذا وصلت مهارة إلى 90+
	for name, s := range skill.Skills {
		if s.Level >= 90 {
			badge := fmt.Sprintf("expert_%s", name)
			if !usm.contains(skill.MasteryBadges, badge) {
				skill.MasteryBadges = append(skill.MasteryBadges, badge)
			}
		}
	}

	// شارة "Master" - إذا وصلت 3 مهارات إلى 95+
	masterCount := 0
	for _, s := range skill.Skills {
		if s.Level >= 95 {
			masterCount++
		}
	}
	if masterCount >= 3 && !usm.contains(skill.MasteryBadges, "master") {
		skill.MasteryBadges = append(skill.MasteryBadges, "master")
	}

	// شارة "World-Class" - إذا وصل المستوى العام إلى 95+
	if skill.OverallLevel >= 95 && !usm.contains(skill.MasteryBadges, "world_class") {
		skill.MasteryBadges = append(skill.MasteryBadges, "world_class")
	}
}

// calculateLevel يحسب المستوى من الخبرة
func (usm *UnifiedSkillManager) calculateLevel(experience int) int {
	level := experience / 100
	if level > 100 {
		level = 100
	}
	return level
}

// calculateOverallLevel يحسب المستوى العام
func (usm *UnifiedSkillManager) calculateOverallLevel(skill *AgentSkill) int {
	if len(skill.Skills) == 0 {
		return 0
	}

	total := 0
	for _, s := range skill.Skills {
		total += s.Level
	}

	avg := total / len(skill.Skills)

	// مكافأة التنوع
	bonus := 0
	highSkills := 0
	for _, s := range skill.Skills {
		if s.Level >= 80 {
			highSkills++
		}
	}
	if highSkills >= 5 {
		bonus = 5
	}

	// مكافأة الشارات
	bonus += len(skill.MasteryBadges) * 2

	overall := avg + bonus
	if overall > 100 {
		overall = 100
	}

	return overall
}

// contains يتحقق من وجود عنصر في شريحة
func (usm *UnifiedSkillManager) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
