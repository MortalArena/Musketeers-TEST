package types

import "time"

// MemoryEntry إدخال ذاكرة
type MemoryEntry struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // "fact", "lesson", "experience", "decision"
	Content     string                 `json:"content"`
	Source      string                 `json:"source"` // agent_id who created this memory
	Importance  float64                `json:"importance"` // 0.0 to 1.0
	AccessCount int                    `json:"access_count"`
	LastAccess  time.Time              `json:"last_access"`
	CreatedAt   time.Time              `json:"created_at"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	Tags        []string               `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
}
