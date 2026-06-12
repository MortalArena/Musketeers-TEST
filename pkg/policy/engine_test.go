package policy

import "testing"

func TestEngineAllowsMatchingPrincipalResourceAndConditions(t *testing.T) {
	engine := NewEngine()
	rule := Rule{
		Name:       "allow-agent-read",
		Effect:     EffectAllow,
		Principals: []Principal{{DID: "did:ia:agent"}},
		Resources:  []Resource{{Type: "content", Action: "read", Attributes: map[string]string{"domain": "Musketeers.ia"}}},
		Conditions: []Condition{{Key: "time", Operator: OpEquals, Value: "day"}},
	}
	if err := engine.AddRule(rule); err != nil {
		t.Fatalf("AddRule returned error: %v", err)
	}
	result, err := engine.Evaluate(Request{
		Principal: Principal{DID: "did:ia:agent"},
		Resource:  Resource{Type: "content", Action: "read", Attributes: map[string]string{"domain": "Musketeers.ia"}},
		Context:   map[string]any{"time": "day"},
	})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if result.Effect != EffectAllow || result.Rule != rule.Name {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestEngineDeniesNonMatchingRule(t *testing.T) {
	engine := NewEngine()
	if err := engine.AddRule(Rule{Name: "deny-all", Effect: EffectDeny, Principals: []Principal{{DID: "*"}}, Resources: []Resource{{Type: "content", Action: "delete"}}}); err != nil {
		t.Fatalf("AddRule returned error: %v", err)
	}
	result, err := engine.Evaluate(Request{Principal: Principal{DID: "did:ia:agent"}, Resource: Resource{Type: "content", Action: "read"}})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if result.Effect != EffectDeny {
		t.Fatalf("expected deny, got %#v", result)
	}
}

func TestEngineRejectsDuplicateRuleName(t *testing.T) {
	engine := NewEngine()
	rule := Rule{Name: "dup", Effect: EffectAllow}
	if err := engine.AddRule(rule); err != nil {
		t.Fatalf("first AddRule returned error: %v", err)
	}
	if err := engine.AddRule(rule); err == nil {
		t.Fatal("expected duplicate rule error")
	}
}
