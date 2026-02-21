package authz

import "testing"

func TestValidateDecisionRecord(t *testing.T) {
	tier := "T2"
	rec := DecisionRecord{
		Subject:     "user-1",
		Action:      "read:resource",
		ResourceRef: "resource:123",
		ReasonCode:  "policy.allow.read",
		Tier:        &tier,
	}
	if err := validateDecisionRecord(rec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateDecisionRecordInvalidTier(t *testing.T) {
	tier := "T9"
	rec := DecisionRecord{
		Subject:     "user-1",
		Action:      "read:resource",
		ResourceRef: "resource:123",
		ReasonCode:  "policy.allow.read",
		Tier:        &tier,
	}
	if err := validateDecisionRecord(rec); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestValidateTraceSteps(t *testing.T) {
	steps := []TraceStep{
		{StepOrder: 0, RuleID: "allow.read", Matched: true, Outcome: "allow", Reason: "rule.matched"},
	}
	if err := validateTraceSteps(steps); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateTraceStepsInvalidOutcome(t *testing.T) {
	steps := []TraceStep{
		{StepOrder: 0, RuleID: "allow.read", Matched: true, Outcome: "bad", Reason: "rule.matched"},
	}
	if err := validateTraceSteps(steps); err == nil {
		t.Fatalf("expected validation error")
	}
}
