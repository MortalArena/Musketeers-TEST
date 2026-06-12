package policy

import (
	"fmt"
	"strings"
	"time"
)

type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
)

type Principal struct {
	DID        string            `json:"did"`
	Roles      []string          `json:"roles,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type Resource struct {
	Type       string            `json:"type"`
	Action     string            `json:"action"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

func (r Resource) String() string {
	return fmt.Sprintf("%s/%s", r.Type, r.Action)
}

func ParseResource(s string) (Resource, error) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return Resource{}, fmt.Errorf("invalid resource format: %s", s)
	}
	return Resource{Type: parts[0], Action: parts[1]}, nil
}

type Operator string

const (
	OpEquals     Operator = "eq"
	OpNotEquals  Operator = "ne"
	OpIn         Operator = "in"
	OpContains   Operator = "contains"
	OpTimeWindow Operator = "time_window"
)

type Condition struct {
	Key      string   `json:"key"`
	Operator Operator `json:"operator"`
	Value    any      `json:"value"`
}

type Rule struct {
	Name       string      `json:"name"`
	Effect     Effect      `json:"effect"`
	Principals []Principal `json:"principals"`
	Resources  []Resource  `json:"resources"`
	Conditions []Condition `json:"conditions,omitempty"`
	Priority   int         `json:"priority,omitempty"`
	ExpiresAt  *time.Time  `json:"expires_at,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
}

type Request struct {
	Principal Principal      `json:"principal"`
	Resource  Resource       `json:"resource"`
	Context   map[string]any `json:"context,omitempty"`
}

type Result struct {
	Effect           Effect `json:"effect"`
	Rule             string `json:"rule,omitempty"`
	RequiresApproval bool   `json:"requires_approval,omitempty"`
}

func (r Rule) validate() error {
	if r.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	if r.Effect != EffectAllow && r.Effect != EffectDeny {
		return fmt.Errorf("invalid effect: %s", r.Effect)
	}
	return nil
}

type ApprovalLevel string

const (
	ApprovalOnce      ApprovalLevel = "once"
	ApprovalSession   ApprovalLevel = "session"
	Approval24h       ApprovalLevel = "24h"
	Approval7d        ApprovalLevel = "7d"
	ApprovalPermanent ApprovalLevel = "permanent"
)
