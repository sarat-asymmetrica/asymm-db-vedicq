package db

import (
	"net/url"
	"os"
	"testing"
)

func TestFromEnvRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	_, err := FromEnv()
	if err == nil {
		t.Fatalf("expected error for missing DATABASE_URL")
	}
}

func TestFromEnvLoadsValidDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb")
	t.Setenv("DB_DRIVER", "")
	t.Setenv("DB_CONNECT_TIMEOUT_SECONDS", "")
	t.Setenv("DB_MAX_IDLE_CONNS", "")
	t.Setenv("DB_MAX_OPEN_CONNS", "")
	t.Setenv("DB_STATEMENT_TIMEOUT_MS", "")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DriverName != "pgx" {
		t.Fatalf("expected default pgx driver, got %q", cfg.DriverName)
	}
	if cfg.DatabaseURL == "" {
		t.Fatalf("database URL should not be empty")
	}
	if cfg.StatementTimeoutMS <= 0 {
		t.Fatalf("expected positive statement timeout, got %d", cfg.StatementTimeoutMS)
	}
	if got := cfg.DatabaseURL; got == "postgres://user:pass@localhost:5432/testdb" {
		t.Fatalf("expected statement timeout param in url, got %q", got)
	}
}

func TestFromEnvNormalizesPostgresAlias(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb")
	t.Setenv("DB_DRIVER", "postgres")
	cfg, err := FromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DriverName != "pgx" {
		t.Fatalf("expected postgres alias normalized to pgx, got %q", cfg.DriverName)
	}
}

func TestConfigValidate(t *testing.T) {
	cfg := Config{
		DriverName:         "postgres",
		DatabaseURL:        "postgres://x",
		ConnectTimeout:     1,
		MaxIdleConns:       0,
		MaxOpenConns:       1,
		StatementTimeoutMS: 1,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	cfg.DatabaseURL = ""
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error for empty database url")
	}
}

func TestConfigValidateRequiresHost(t *testing.T) {
	cfg := Config{
		DriverName:         "pgx",
		DatabaseURL:        "postgres://user:pass@:5432/dbname",
		ConnectTimeout:     1,
		MaxIdleConns:       0,
		MaxOpenConns:       1,
		ConnMaxIdleTime:    1,
		ConnMaxLifetime:    1,
		StatementTimeoutMS: 1,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error for missing host in DATABASE_URL")
	}
}

func TestWithDefaultRuntimeParamsPreservesExistingTimeout(t *testing.T) {
	in := "postgres://user:pass@localhost:5432/testdb?sslmode=disable&statement_timeout=2500"
	out, err := withDefaultRuntimeParams(in, 9000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	u, err := url.Parse(out)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if got := u.Query().Get("statement_timeout"); got != "2500" {
		t.Fatalf("expected existing statement_timeout=2500, got %q", got)
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}
