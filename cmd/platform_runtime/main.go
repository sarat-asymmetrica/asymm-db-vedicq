package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	dbpkg "github.com/sarat-asymmetrica/vedic-platform-experiments/pkg/db"
	platform "github.com/sarat-asymmetrica/vedic-platform-experiments/pkg/platform"
	securitypkg "github.com/sarat-asymmetrica/vedic-platform-experiments/pkg/security"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  platform_runtime selfcheck [--node-name name] [--nonce-scope scope] [--nonce-window N]\n")
	fmt.Fprintf(os.Stderr, "  platform_runtime serve [--host addr] [--port N] [--node-name name] [--nonce-scope scope] [--nonce-window N]\n")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "selfcheck":
		selfcheckCmd(os.Args[2:])
	case "serve":
		serveCmd(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func selfcheckCmd(args []string) {
	fs := flag.NewFlagSet("selfcheck", flag.ExitOnError)
	nodeName := fs.String("node-name", "platform-node", "node name for nonce watermark scope")
	nonceScope := fs.String("nonce-scope", "default", "nonce scope")
	nonceWindow := fs.Uint64("nonce-window", 1000, "nonce reservation window")
	_ = fs.Parse(args)

	dbCfg, err := dbpkg.FromEnv()
	if err != nil {
		fatalf("load db config: %v", err)
	}
	secCfg := securitypkg.RuntimeConfig{
		NodeName:    *nodeName,
		NonceScope:  *nonceScope,
		NonceWindow: *nonceWindow,
	}

	ctx := context.Background()
	rt, err := platform.BuildPhase1Runtime(ctx, dbCfg, secCfg)
	if err != nil {
		fatalf("build runtime: %v", err)
	}
	defer rt.Close()

	if err := rt.HealthCheck(ctx, 5*time.Second); err != nil {
		fatalf("health check: %v", err)
	}
	fmt.Fprintln(os.Stdout, "platform_runtime: selfcheck ok")
}

func serveCmd(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	host := fs.String("host", getEnvOrDefault("HOST", "0.0.0.0"), "http bind host")
	port := fs.Int("port", getEnvOrDefaultInt("PORT", 8080), "http bind port")
	nodeName := fs.String("node-name", "platform-node", "node name for nonce watermark scope")
	nonceScope := fs.String("nonce-scope", "default", "nonce scope")
	nonceWindow := fs.Uint64("nonce-window", 1000, "nonce reservation window")
	healthTimeout := fs.Duration("health-timeout", 5*time.Second, "database health check timeout")
	writeTimeout := fs.Duration("write-timeout", 8*time.Second, "api write timeout")
	idempotencyTTL := fs.Duration("idempotency-ttl", 24*time.Hour, "idempotency key retention window")
	shutdownTimeout := fs.Duration("shutdown-timeout", 10*time.Second, "graceful shutdown timeout")
	_ = fs.Parse(args)

	dbCfg, err := dbpkg.FromEnv()
	if err != nil {
		fatalf("load db config: %v", err)
	}
	secCfg := securitypkg.RuntimeConfig{
		NodeName:    *nodeName,
		NonceScope:  *nonceScope,
		NonceWindow: *nonceWindow,
	}
	serveSecCfg, err := loadServeSecurityConfigFromEnv()
	if err != nil {
		fatalf("load runtime security config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rt, err := platform.BuildPhase1Runtime(ctx, dbCfg, secCfg)
	if err != nil {
		fatalf("build runtime: %v", err)
	}
	defer rt.Close()

	if err := rt.HealthCheck(ctx, *healthTimeout); err != nil {
		fatalf("startup health check: %v", err)
	}

	logServeStartup(dbCfg, serveSecCfg, *host, *port, *healthTimeout, *writeTimeout, *idempotencyTTL)

	mux := http.NewServeMux()
	api := newHTTPAPI(rt, *healthTimeout, *writeTimeout, *idempotencyTTL, serveSecCfg)
	mux.HandleFunc("/livez", api.handleLiveness)
	mux.HandleFunc("/healthz", api.handleHealthz)
	mux.HandleFunc("/readyz", api.handleReadyz)
	mux.HandleFunc("/v1/decisions", api.handleDecisionWrite)
	mux.HandleFunc("/v1/telemetry/events", api.handleTelemetryWrite)
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("asymm-db-vedicq-runtime"))
	})

	addr := net.JoinHostPort(*host, strconv.Itoa(*port))
	server := &http.Server{
		Addr:              addr,
		Handler:           withServeMiddlewares(mux, serveSecCfg),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()
	fmt.Fprintf(os.Stdout, "platform_runtime: serving on %s\n", addr)

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), *shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			fatalf("graceful shutdown: %v", err)
		}
		fmt.Fprintln(os.Stdout, "platform_runtime: shutdown complete")
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			fatalf("http server: %v", err)
		}
	}
}

func getEnvOrDefault(name, fallback string) string {
	v := os.Getenv(name)
	if v == "" {
		return fallback
	}
	return v
}

func getEnvOrDefaultInt(name string, fallback int) int {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logServeStartup(
	dbCfg dbpkg.Config,
	serveSecCfg serveSecurityConfig,
	host string,
	port int,
	healthTimeout time.Duration,
	writeTimeout time.Duration,
	idempotencyTTL time.Duration,
) {
	dbHost := "(unknown)"
	u, err := url.Parse(dbCfg.DatabaseURL)
	if err == nil && u != nil && u.Hostname() != "" {
		dbHost = u.Hostname()
	}
	fmt.Fprintf(
		os.Stdout,
		"platform_runtime: startup mode=serve host=%s port=%d db_driver=%s db_host=%s max_open=%d max_idle=%d stmt_timeout_ms=%d health_timeout=%s write_timeout=%s idempotency_ttl=%s\n",
		host,
		port,
		dbCfg.DriverName,
		dbHost,
		dbCfg.MaxOpenConns,
		dbCfg.MaxIdleConns,
		dbCfg.StatementTimeoutMS,
		// Security posture is safe to log as booleans/counts only.
		healthTimeout.String(),
		writeTimeout.String(),
		idempotencyTTL.String(),
	)
	fmt.Fprintf(
		os.Stdout,
		"platform_runtime: security require_auth=%t token_count=%d allowed_origins=%d rate_limit_per_min=%d rate_limit_burst=%d trust_proxy_headers=%t\n",
		serveSecCfg.RequireAuth,
		len(serveSecCfg.AllowedTokens),
		len(serveSecCfg.AllowedOrigins),
		serveSecCfg.RateLimitPerMinute,
		serveSecCfg.RateLimitBurst,
		serveSecCfg.TrustProxyHeaders,
	)
}
