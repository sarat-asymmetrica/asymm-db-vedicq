package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sarat-asymmetrica/vedic-platform-experiments/pkg/db"
)

func main() {
	if len(os.Args) < 3 {
		usage()
		os.Exit(2)
	}
	if os.Args[1] != "migrate" {
		usage()
		os.Exit(2)
	}

	switch os.Args[2] {
	case "validate":
		validateCmd(os.Args[3:])
	case "status":
		statusCmd(os.Args[3:])
	case "up":
		upCmd(os.Args[3:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  dbctl migrate validate [--dir path]\n")
	fmt.Fprintf(os.Stderr, "  dbctl migrate status [--dir path]\n")
	fmt.Fprintf(os.Stderr, "  dbctl migrate up [--dir path] [--database-url url] [--dry-run]\n")
}

func defaultMigrationDir() string {
	return filepath.Clean("db/migrations")
}

func validateCmd(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	dir := fs.String("dir", defaultMigrationDir(), "migration directory")
	_ = fs.Parse(args)

	migs, err := db.DiscoverMigrations(*dir)
	if err != nil {
		fatalf("validate: %v", err)
	}
	fmt.Fprintf(os.Stdout, "migrations valid: files=%d last=%04d\n", len(migs), migs[len(migs)-1].Seq)
}

func statusCmd(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	dir := fs.String("dir", defaultMigrationDir(), "migration directory")
	_ = fs.Parse(args)

	migs, err := db.DiscoverMigrations(*dir)
	if err != nil {
		fatalf("status: %v", err)
	}
	fmt.Fprintf(os.Stdout, "migration directory: %s\n", *dir)
	for _, m := range migs {
		fmt.Fprintf(os.Stdout, "%04d  %s  %s\n", m.Seq, m.Name, m.SHA256[:12])
	}
}

func upCmd(args []string) {
	fs := flag.NewFlagSet("up", flag.ExitOnError)
	dir := fs.String("dir", defaultMigrationDir(), "migration directory")
	databaseURL := fs.String("database-url", "", "postgres connection url (defaults to DATABASE_URL)")
	dryRun := fs.Bool("dry-run", false, "validate and print migration count without applying")
	_ = fs.Parse(args)

	migs, err := db.DiscoverMigrations(*dir)
	if err != nil {
		fatalf("up: %v", err)
	}

	if *dryRun {
		fmt.Fprintf(os.Stdout, "dry-run: pending-check only, discovered=%d\n", len(migs))
		return
	}

	cfg, err := db.FromEnv()
	if err != nil {
		fatalf("up: db config: %v", err)
	}
	if *databaseURL != "" {
		cfg.DatabaseURL = *databaseURL
	}
	if err := cfg.Validate(); err != nil {
		fatalf("up: db config: %v", err)
	}

	ctx := context.Background()
	conn, err := db.Open(ctx, cfg)
	if err != nil {
		fatalf("up: connect: %v", err)
	}
	defer conn.Close()

	if err := db.ApplyMigrations(ctx, conn, migs, false); err != nil {
		fatalf("up: %v", err)
	}
	fmt.Fprintf(os.Stdout, "applied migrations successfully: %d\n", len(migs))
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
