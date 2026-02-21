package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultRateLimitPerMinute = 120
	defaultRateLimitBurst     = 30
)

type serveSecurityConfig struct {
	RequireAuth        bool
	AllowedTokens      map[string]struct{}
	AllowedOrigins     map[string]struct{}
	RateLimitPerMinute int
	RateLimitBurst     int
	TrustProxyHeaders  bool
}

func loadServeSecurityConfigFromEnv() (serveSecurityConfig, error) {
	cfg := serveSecurityConfig{
		RequireAuth:        true,
		AllowedTokens:      map[string]struct{}{},
		AllowedOrigins:     map[string]struct{}{},
		RateLimitPerMinute: defaultRateLimitPerMinute,
		RateLimitBurst:     defaultRateLimitBurst,
		TrustProxyHeaders:  true,
	}

	if v := strings.TrimSpace(os.Getenv("RUNTIME_REQUIRE_AUTH")); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return serveSecurityConfig{}, fmt.Errorf("runtime: invalid RUNTIME_REQUIRE_AUTH")
		}
		cfg.RequireAuth = b
	}
	for _, tok := range splitCSV(os.Getenv("RUNTIME_API_TOKENS")) {
		cfg.AllowedTokens[tok] = struct{}{}
	}
	for _, origin := range splitCSV(os.Getenv("RUNTIME_ALLOWED_ORIGINS")) {
		cfg.AllowedOrigins[origin] = struct{}{}
	}
	if v := strings.TrimSpace(os.Getenv("RUNTIME_RATE_LIMIT_PER_MINUTE")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return serveSecurityConfig{}, fmt.Errorf("runtime: invalid RUNTIME_RATE_LIMIT_PER_MINUTE")
		}
		cfg.RateLimitPerMinute = n
	}
	if v := strings.TrimSpace(os.Getenv("RUNTIME_RATE_LIMIT_BURST")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return serveSecurityConfig{}, fmt.Errorf("runtime: invalid RUNTIME_RATE_LIMIT_BURST")
		}
		cfg.RateLimitBurst = n
	}
	if v := strings.TrimSpace(os.Getenv("RUNTIME_TRUST_PROXY_HEADERS")); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return serveSecurityConfig{}, fmt.Errorf("runtime: invalid RUNTIME_TRUST_PROXY_HEADERS")
		}
		cfg.TrustProxyHeaders = b
	}

	if cfg.RequireAuth && len(cfg.AllowedTokens) == 0 {
		return serveSecurityConfig{}, fmt.Errorf("runtime: RUNTIME_REQUIRE_AUTH=true requires RUNTIME_API_TOKENS")
	}
	if cfg.RateLimitBurst > cfg.RateLimitPerMinute {
		return serveSecurityConfig{}, fmt.Errorf("runtime: RUNTIME_RATE_LIMIT_BURST cannot exceed per-minute limit")
	}
	return cfg, nil
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func withServeMiddlewares(next http.Handler, cfg serveSecurityConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setSecurityHeaders(w)
		if !applyCORSHeaders(w, r, cfg) {
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "no-store")
}

func applyCORSHeaders(w http.ResponseWriter, r *http.Request, cfg serveSecurityConfig) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	if len(cfg.AllowedOrigins) == 0 {
		return false
	}
	if _, ok := cfg.AllowedOrigins[origin]; !ok {
		return false
	}
	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,Idempotency-Key,X-Request-ID,X-API-Key")
	w.Header().Set("Access-Control-Max-Age", "600")
	return true
}

type requestRateLimiter struct {
	mu      sync.Mutex
	limits  map[string]*rateCounter
	limit   int
	window  time.Duration
	clockFn func() time.Time
}

type rateCounter struct {
	windowStart time.Time
	count       int
}

func newRequestRateLimiter(perMinute int) *requestRateLimiter {
	return &requestRateLimiter{
		limits:  make(map[string]*rateCounter),
		limit:   perMinute,
		window:  time.Minute,
		clockFn: time.Now,
	}
}

func (l *requestRateLimiter) allow(key string, burst int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.clockFn()
	c, ok := l.limits[key]
	if !ok || now.Sub(c.windowStart) >= l.window {
		l.limits[key] = &rateCounter{
			windowStart: now,
			count:       1,
		}
		return true
	}
	max := l.limit + burst
	if c.count >= max {
		return false
	}
	c.count++
	return true
}

func extractClientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
			first := strings.TrimSpace(strings.Split(xff, ",")[0])
			if first != "" {
				return first
			}
		}
		if rip := strings.TrimSpace(r.Header.Get("X-Real-IP")); rip != "" {
			return rip
		}
	}
	host := strings.TrimSpace(r.RemoteAddr)
	if host == "" {
		return "unknown"
	}
	ip, _, err := net.SplitHostPort(host)
	if err != nil {
		return host
	}
	return ip
}
