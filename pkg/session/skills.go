package session

import (
	"fmt"
	"sync"
	"time"
)

// SkillsManager يدير مهارات الوكلاء وتطورها
type SkillsManager struct {
	SessionID        string                 `json:"session_id"`
	AgentSkills      map[string]*AgentSkill `json:"agent_skills"` // DID -> Skill
	EvolutionEnabled bool                   `json:"evolution_enabled"`
	mu               sync.RWMutex
}

// [SAFETY] حدود الموارد لمنع استهلاك غير محدود
const (
	// [SAFETY] الحد الأقصى للمستوى
	MaxSkillLevel = 100
	// [SAFETY] الحد الأدنى للمستوى
	MinSkillLevel = 0
	// [SAFETY] الحد الأقصى للخبرة
	MaxExperience = 100000
	// [SAFETY] الحد الأقصى لعدد المهارات لكل وكيل
	MaxSkillsPerAgent = 50
	// [SAFETY] الحد الأقصى لعدد المهارات الفرعية لكل مهارة
	MaxSubSkillsPerSkill = 10
	// [SAFETY] الحد الأقصى لمعدل النجاح
	MaxSuccessRate = 1.0
	// [SAFETY] الحد الأدنى لمعدل النجاح
	MinSuccessRate = 0.0
)

// AgentSkill مهارات وكيل واحد
type AgentSkill struct {
	AgentDID        string            `json:"agent_did"`
	AgentType       string            `json:"agent_type"`    // coder, designer, tester, etc.
	OverallLevel    int               `json:"overall_level"` // 0-100
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

// Skill مهارة محددة
type Skill struct {
	Name        string               `json:"name"`
	Level       int                  `json:"level"`      // 0-100
	Experience  int                  `json:"experience"` // XP points
	LastUsed    time.Time            `json:"last_used"`
	UsageCount  int                  `json:"usage_count"`
	SuccessRate float64              `json:"success_rate"` // 0.0 - 1.0
	SubSkills   map[string]*SubSkill `json:"sub_skills"`
}

// SubSkill مهارة فرعية
type SubSkill struct {
	Name        string  `json:"name"`
	Level       int     `json:"level"`
	Proficiency float64 `json:"proficiency"` // 0.0 - 1.0
}

// SkillTask مهمة مكتملة (لتسجيلها في المهارات)
type SkillTask struct {
	Name          string        `json:"name"`
	Success       bool          `json:"success"`
	Duration      time.Duration `json:"duration"`
	SkillsUsed    []string      `json:"skills_used"`
	XPGained      int           `json:"xp_gained"`
	LessonLearned string        `json:"lesson_learned"`
}

// NewSkillsManager ينشئ مدير مهارات جديد
func NewSkillsManager(sessionID string) *SkillsManager {
	return &SkillsManager{
		SessionID:        sessionID,
		AgentSkills:      make(map[string]*AgentSkill),
		EvolutionEnabled: true,
	}
}

// RegisterAgent يسجل وكيلاً ويمنحه مهارات ابتدائية
func (sm *SkillsManager) RegisterAgent(agentDID, agentType string) error {
	// [SAFETY] التحقق من صحة المدخلات
	if agentDID == "" {
		return fmt.Errorf("agent DID cannot be empty")
	}
	if agentType == "" {
		return fmt.Errorf("agent type cannot be empty")
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// [SAFETY] التحقق من الحد الأقصى للوكلاء
	if len(sm.AgentSkills) >= MaxAgents {
		return fmt.Errorf("maximum agents limit reached (%d)", MaxAgents)
	}

	if _, exists := sm.AgentSkills[agentDID]; exists {
		return fmt.Errorf("الوكيل مسجل بالفعل")
	}

	skill := &AgentSkill{
		AgentDID:      agentDID,
		AgentType:     agentType,
		OverallLevel:  50, // يبدأ من 50
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

	case "security":
		skill.Skills["vulnerability_assessment"] = &Skill{Name: "Vulnerability Assessment", Level: 85, Experience: 2000}
		skill.Skills["penetration_testing"] = &Skill{Name: "Penetration Testing", Level: 80, Experience: 1500}
		skill.Skills["code_audit"] = &Skill{Name: "Code Audit", Level: 85, Experience: 2000}
		skill.Specializations = []string{"appsec", "infrastructure"}
	}

	// [SAFETY] التحقق من الحد الأقصى للمهارات
	if len(skill.Skills) > MaxSkillsPerAgent {
		return fmt.Errorf("maximum skills per agent limit reached (%d)", MaxSkillsPerAgent)
	}

	sm.AgentSkills[agentDID] = skill
	return nil
}

// RecordTaskCompletion يسجل إكمال مهمة ويطور المهارات
func (sm *SkillsManager) RecordTaskCompletion(agentDID string, task SkillTask) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	skill, exists := sm.AgentSkills[agentDID]
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
				newLevel := calculateLevel(s.Experience)
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
		sm.checkMasteryBadges(skill)

	} else {
		skill.FailureCount++
	}

	// تحديث متوسط وقت المهمة
	totalTime := skill.AvgTaskTime * time.Duration(skill.TotalTasks-1)
	totalTime += task.Duration
	skill.AvgTaskTime = totalTime / time.Duration(skill.TotalTasks)

	// تحديث المستوى العام
	skill.OverallLevel = calculateOverallLevel(skill)

	return nil
}

// checkMasteryBadges يتحقق ويمنح شارات الإتقان
func (sm *SkillsManager) checkMasteryBadges(skill *AgentSkill) {
	// شارة "Expert" - إذا وصلت مهارة إلى 90+
	for name, s := range skill.Skills {
		if s.Level >= 90 {
			badge := fmt.Sprintf("expert_%s", name)
			if !containsSlice(skill.MasteryBadges, badge) {
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
	if masterCount >= 3 && !containsSlice(skill.MasteryBadges, "master") {
		skill.MasteryBadges = append(skill.MasteryBadges, "master")
	}

	// شارة "World-Class" - إذا وصل المستوى العام إلى 95+
	if skill.OverallLevel >= 95 && !containsSlice(skill.MasteryBadges, "world_class") {
		skill.MasteryBadges = append(skill.MasteryBadges, "world_class")
	}
}

// calculateLevel يحسب المستوى من الخبرة
func calculateLevel(experience int) int {
	// [SAFETY] التحقق من الحد الأقصى للخبرة
	if experience > MaxExperience {
		experience = MaxExperience
	}

	// كل 100 XP = مستوى واحد
	level := experience / 100
	if level > MaxSkillLevel {
		level = MaxSkillLevel
	}
	return level
}

// calculateOverallLevel يحسب المستوى العام
func calculateOverallLevel(skill *AgentSkill) int {
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

// containsSlice يتحقق من وجود عنصر في شريحة
func containsSlice(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
