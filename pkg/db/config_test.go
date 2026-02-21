package db

import (
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
		DriverName:     "postgres",
		DatabaseURL:    "postgres://x",
		ConnectTimeout: 1,
		MaxIdleConns:   0,
		MaxOpenConns:   1,
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
		DriverName:      "pgx",
		DatabaseURL:     "postgres://user:pass@:5432/dbname",
		ConnectTimeout:  1,
		MaxIdleConns:    0,
		MaxOpenConns:    1,
		ConnMaxIdleTime: 1,
		ConnMaxLifetime: 1,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error for missing host in DATABASE_URL")
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}
