package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Open opens a pooled DB connection and validates it with PingContext.
func Open(ctx context.Context, cfg Config) (*sql.DB, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	db, err := sql.Open(cfg.DriverName, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}

	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return db, nil
}

// HealthCheck returns an error if the DB cannot be reached.
func HealthCheck(ctx context.Context, db *sql.DB, timeout time.Duration) error {
	if db == nil {
		return fmt.Errorf("db: nil handle")
	}
	if timeout <= 0 {
		timeout = defaultConnectTimeout
	}
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return db.PingContext(pingCtx)
}
