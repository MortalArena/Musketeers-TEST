package policy

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
)

type Engine struct {
	mu     sync.RWMutex
	rules  []Rule
	byName map[string]int
}

func NewEngine() *Engine {
	return &Engine{byName: make(map[string]int)}
}

func (e *Engine) AddRule(rule Rule) error {
	if err := rule.validate(); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, exists := e.byName[rule.Name]; exists {
		return fmt.Errorf("rule already exists: %s", rule.Name)
	}
	e.byName[rule.Name] = len(e.rules)
	e.rules = append(e.rules, rule)
	sort.SliceStable(e.rules, func(i, j int) bool {
		return e.rules[i].Priority > e.rules[j].Priority
	})
	rebuild := make(map[string]int, len(e.rules))
	for i, existing := range e.rules {
		rebuild[existing.Name] = i
	}
	e.byName = rebuild
	return nil
}

func (e *Engine) Evaluate(req Request) (Result, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, rule := range e.rules {
		if e.match(rule, req) {
			return Result{Effect: rule.Effect, Rule: rule.Name}, nil
		}
	}
	return Result{Effect: EffectDeny}, nil
}

func (e *Engine) Rule(name string) (Rule, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	idx, exists := e.byName[name]
	if !exists {
		return Rule{}, false
	}
	return e.rules[idx], true
}

func (e *Engine) match(rule Rule, req Request) bool {
	return matchPrincipal(rule.Principals, req.Principal) &&
		matchResource(rule.Resources, req.Resource) &&
		matchConditions(rule.Conditions, req.Context)
}

func matchPrincipal(patterns []Principal, actual Principal) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, pattern := range patterns {
		if pattern.DID == "*" || pattern.DID == actual.DID {
			if len(pattern.Roles) == 0 || hasAny(actual.Roles, pattern.Roles) {
				if attributesMatch(pattern.Attributes, actual.Attributes) {
					return true
				}
			}
		}
	}
	return false
}

func matchResource(patterns []Resource, actual Resource) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, pattern := range patterns {
		if pattern.Type != "" && pattern.Type != actual.Type {
			continue
		}
		if pattern.Action != "" && pattern.Action != actual.Action {
			continue
		}
		if attributesMatch(pattern.Attributes, actual.Attributes) {
			return true
		}
	}
	return false
}

func matchConditions(conditions []Condition, context map[string]any) bool {
	if len(conditions) == 0 {
		return true
	}
	for _, condition := range conditions {
		value, exists := context[condition.Key]
		if !exists {
			return false
		}
		if !compareCondition(value, condition) {
			return false
		}
	}
	return true
}

func compareCondition(value any, condition Condition) bool {
	switch condition.Operator {
	case OpEquals, "":
		return fmt.Sprint(value) == fmt.Sprint(condition.Value)
	case OpNotEquals:
		return fmt.Sprint(value) != fmt.Sprint(condition.Value)
	case OpIn:
		values, ok := condition.Value.([]string)
		if !ok {
			return false
		}
		for _, item := range values {
			if fmt.Sprint(value) == item {
				return true
			}
		}
		return false
	case OpContains:
		return strings.Contains(fmt.Sprint(value), fmt.Sprint(condition.Value))
	default:
		return false
	}
}

func attributesMatch(pattern, actual map[string]string) bool {
	if len(pattern) == 0 {
		return true
	}
	for k, v := range pattern {
		if actual[k] != v {
			return false
		}
	}
	return true
}

func hasAny(actual, allowed []string) bool {
	for _, a := range actual {
		for _, b := range allowed {
			if a == b {
				return true
			}
		}
	}
	return false
}

func RulesEqual(a, b Rule) bool {
	return reflect.DeepEqual(a, b)
}
