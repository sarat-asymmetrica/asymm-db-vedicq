package db

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultDriverName     = "pgx"
	defaultConnectTimeout = 5 * time.Second
	defaultMaxIdleConns   = 5
	defaultMaxOpenConns   = 20
	defaultConnMaxIdle    = 5 * time.Minute
	defaultConnMaxLife    = 30 * time.Minute
)

// Config defines database connection settings.
type Config struct {
	DriverName      string
	DatabaseURL     string
	ConnectTimeout  time.Duration
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
}

// FromEnv loads DB config from environment variables.
func FromEnv() (Config, error) {
	cfg := Config{
		DriverName:      firstNonEmpty(os.Getenv("DB_DRIVER"), defaultDriverName),
		DatabaseURL:     strings.TrimSpace(os.Getenv("DATABASE_URL")),
		ConnectTimeout:  defaultConnectTimeout,
		MaxIdleConns:    defaultMaxIdleConns,
		MaxOpenConns:    defaultMaxOpenConns,
		ConnMaxIdleTime: defaultConnMaxIdle,
		ConnMaxLifetime: defaultConnMaxLife,
	}
	cfg.DriverName = normalizeDriverName(cfg.DriverName)

	if v := strings.TrimSpace(os.Getenv("DB_CONNECT_TIMEOUT_SECONDS")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return Config{}, fmt.Errorf("db: invalid DB_CONNECT_TIMEOUT_SECONDS")
		}
		cfg.ConnectTimeout = time.Duration(n) * time.Second
	}
	if v := strings.TrimSpace(os.Getenv("DB_MAX_IDLE_CONNS")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return Config{}, fmt.Errorf("db: invalid DB_MAX_IDLE_CONNS")
		}
		cfg.MaxIdleConns = n
	}
	if v := strings.TrimSpace(os.Getenv("DB_MAX_OPEN_CONNS")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return Config{}, fmt.Errorf("db: invalid DB_MAX_OPEN_CONNS")
		}
		cfg.MaxOpenConns = n
	}

	return cfg, cfg.Validate()
}

// Validate enforces basic config correctness.
func (c Config) Validate() error {
	if strings.TrimSpace(c.DriverName) == "" {
		return fmt.Errorf("db: driver name is required")
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("db: database url is required")
	}
	if c.ConnectTimeout <= 0 {
		return fmt.Errorf("db: connect timeout must be > 0")
	}
	if c.MaxOpenConns <= 0 {
		return fmt.Errorf("db: max open conns must be > 0")
	}
	if c.MaxIdleConns < 0 {
		return fmt.Errorf("db: max idle conns must be >= 0")
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func normalizeDriverName(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	switch n {
	case "", "pgx":
		return defaultDriverName
	case "postgres", "postgresql":
		// Compatibility alias: DATABASE_URL often uses "postgres" nomenclature,
		// while the registered database/sql driver in this project is pgx.
		return "pgx"
	default:
		return n
	}
}
