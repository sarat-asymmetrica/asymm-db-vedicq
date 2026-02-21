package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	dbpkg "github.com/sarat-asymmetrica/vedic-platform-experiments/pkg/db"
	platform "github.com/sarat-asymmetrica/vedic-platform-experiments/pkg/platform"
	securitypkg "github.com/sarat-asymmetrica/vedic-platform-experiments/pkg/security"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  platform_runtime selfcheck [--node-name name] [--nonce-scope scope] [--nonce-window N]\n")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "selfcheck":
		selfcheckCmd(os.Args[2:])
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

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
