package main

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoadServeSecurityConfigRequiresTokenWhenAuthEnabled(t *testing.T) {
	t.Setenv("RUNTIME_REQUIRE_AUTH", "true")
	t.Setenv("RUNTIME_API_TOKENS", "")
	_, err := loadServeSecurityConfigFromEnv()
	if err == nil {
		t.Fatalf("expected error when auth is enabled without tokens")
	}
}

func TestLoadServeSecurityConfigSuccess(t *testing.T) {
	t.Setenv("RUNTIME_REQUIRE_AUTH", "true")
	t.Setenv("RUNTIME_API_TOKENS", "token-a, token-b")
	t.Setenv("RUNTIME_ALLOWED_ORIGINS", "https://app.example.com")
	t.Setenv("RUNTIME_RATE_LIMIT_PER_MINUTE", "200")
	t.Setenv("RUNTIME_RATE_LIMIT_BURST", "20")
	t.Setenv("RUNTIME_TRUST_PROXY_HEADERS", "false")

	cfg, err := loadServeSecurityConfigFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.AllowedTokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(cfg.AllowedTokens))
	}
	if len(cfg.AllowedOrigins) != 1 {
		t.Fatalf("expected 1 allowed origin, got %d", len(cfg.AllowedOrigins))
	}
	if cfg.TrustProxyHeaders {
		t.Fatalf("expected trust proxy headers false")
	}
}

func TestExtractAuthToken(t *testing.T) {
	req := httptest.NewRequest("POST", "/v1/decisions", nil)
	req.Header.Set("Authorization", "Bearer abc123")
	if got := extractAuthToken(req); got != "abc123" {
		t.Fatalf("expected bearer token, got %q", got)
	}
	req.Header.Del("Authorization")
	req.Header.Set("X-API-Key", "api-key-1")
	if got := extractAuthToken(req); got != "api-key-1" {
		t.Fatalf("expected api key token, got %q", got)
	}
}

func TestRateLimiterAllow(t *testing.T) {
	l := newRequestRateLimiter(2)
	now := time.Now()
	l.clockFn = func() time.Time { return now }
	if !l.allow("k", 0) {
		t.Fatalf("expected first request allowed")
	}
	if !l.allow("k", 0) {
		t.Fatalf("expected second request allowed")
	}
	if l.allow("k", 0) {
		t.Fatalf("expected third request blocked")
	}
}
