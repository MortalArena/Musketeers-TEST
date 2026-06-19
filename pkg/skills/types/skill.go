package types

import "time"

// Skill مهارة محددة
type Skill struct {
	Name        string               `json:"name"`
	Level       int                  `json:"level"` // 0-100
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
