package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
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

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		hctx, cancel := context.WithTimeout(context.Background(), *healthTimeout)
		defer cancel()
		if err := rt.HealthCheck(hctx, *healthTimeout); err != nil {
			http.Error(w, "unhealthy", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		hctx, cancel := context.WithTimeout(context.Background(), *healthTimeout)
		defer cancel()
		if err := rt.HealthCheck(hctx, *healthTimeout); err != nil {
			http.Error(w, "not-ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("asymm-db-vedicq-runtime"))
	})

	addr := net.JoinHostPort(*host, strconv.Itoa(*port))
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
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
