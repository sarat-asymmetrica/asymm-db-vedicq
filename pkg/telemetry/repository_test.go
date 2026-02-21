package telemetry

import "testing"

func TestValidateSecurityEvent(t *testing.T) {
	rec := SecurityEventRecord{
		ActorType: "service",
		EventType: "authz.decision",
		Severity:  "info",
		Message:   "policy decision evaluated",
	}
	if err := validateSecurityEvent(rec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSecurityEventInvalidSeverity(t *testing.T) {
	rec := SecurityEventRecord{
		ActorType: "service",
		EventType: "authz.decision",
		Severity:  "fatal",
		Message:   "policy decision evaluated",
	}
	if err := validateSecurityEvent(rec); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestValidateEventLinks(t *testing.T) {
	links := []EventLink{
		{LinkKind: "policy_decision", LinkedID: "00000000-0000-0000-0000-000000000000"},
	}
	if err := validateEventLinks(links); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateEventLinksMissingID(t *testing.T) {
	links := []EventLink{
		{LinkKind: "policy_decision", LinkedID: ""},
	}
	if err := validateEventLinks(links); err == nil {
		t.Fatalf("expected validation error")
	}
}
