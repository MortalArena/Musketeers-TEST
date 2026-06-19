package evolution

import (
	"fmt"
	"time"

	"github.com/MortalArena/Musketeers/pkg/skills/types"
	"go.uber.org/zap"
)

// XPSystem نظام XP لتطوير المهارات
type XPSystem struct {
	logger *zap.Logger
}

// NewXPSystem ينشئ نظام XP جديد
func NewXPSystem(logger *zap.Logger) *XPSystem {
	return &XPSystem{
		logger: logger,
	}
}

// RecordTaskCompletion يسجل إكمال مهمة ويطور المهارات
func (xp *XPSystem) RecordTaskCompletion(agentDID string, task *types.SkillTask, skills map[string]*types.Skill) error {
	for _, skillName := range task.SkillsUsed {
		if skill, ok := skills[skillName]; ok {
			skill.Experience += task.XPGained
			skill.UsageCount++
			skill.LastUsed = time.Now()

			// حساب المستوى الجديد
			newLevel := calculateLevel(skill.Experience)
			if newLevel > skill.Level {
				skill.Level = newLevel
				xp.logger.Info("تم تطوير المهارة",
					zap.String("agent_id", agentDID),
					zap.String("skill_name", skillName),
					zap.Int("old_level", skill.Level),
					zap.Int("new_level", newLevel))
			}

			// تحديث معدل النجاح
			if skill.UsageCount > 0 {
				if task.Success {
					skill.SuccessRate = (skill.SuccessRate*float64(skill.UsageCount-1) + 1.0) / float64(skill.UsageCount)
				} else {
					skill.SuccessRate = (skill.SuccessRate * float64(skill.UsageCount-1)) / float64(skill.UsageCount)
				}
			}
		}
	}

	return nil
}

// calculateLevel يحسب المستوى من الخبرة
func calculateLevel(experience int) int {
	// كل 100 XP = مستوى واحد
	level := experience / 100
	if level > 100 {
		level = 100
	}
	return level
}

// calculateOverallLevel يحسب المستوى العام
func calculateOverallLevel(skills map[string]*types.Skill) int {
	if len(skills) == 0 {
		return 0
	}

	total := 0
	for _, skill := range skills {
		total += skill.Level
	}

	avg := total / len(skills)
	return avg
}

// checkMasteryBadges يتحقق ويمنح شارات الإتقان
func (xp *XPSystem) checkMasteryBadges(badges []string, skills map[string]*types.Skill) []string {
	// شارة "Expert" - إذا وصلت مهارة إلى 90+
	for name, skill := range skills {
		if skill.Level >= 90 {
			badge := fmt.Sprintf("expert_%s", name)
			if !contains(badges, badge) {
				badges = append(badges, badge)
			}
		}
	}

	// شارة "Master" - إذا وصلت 3 مهارات إلى 95+
	masterCount := 0
	for _, skill := range skills {
		if skill.Level >= 95 {
			masterCount++
		}
	}
	if masterCount >= 3 && !contains(badges, "master") {
		badges = append(badges, "master")
	}

	return badges
}

// contains يتحقق من وجود عنصر في شريحة
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
