package security

import "testing"

func TestRuntimeConfigValidate(t *testing.T) {
	cfg := RuntimeConfig{
		NodeName:    "node-a",
		NonceScope:  "default",
		NonceWindow: 1000,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeConfigValidateMissingNode(t *testing.T) {
	cfg := RuntimeConfig{
		NodeName:    "",
		NonceScope:  "default",
		NonceWindow: 1000,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestRuntimeConfigValidateWindow(t *testing.T) {
	cfg := RuntimeConfig{
		NodeName:    "node-a",
		NonceScope:  "default",
		NonceWindow: 0,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error")
	}
}
