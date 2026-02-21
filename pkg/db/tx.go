package db

import (
	"context"
	"database/sql"
	"fmt"
)

// WithTx wraps a function in a transaction with automatic rollback on error.
func WithTx(ctx context.Context, db *sql.DB, opts *sql.TxOptions, fn func(*sql.Tx) error) error {
	if db == nil {
		return fmt.Errorf("db: nil handle")
	}
	if fn == nil {
		return fmt.Errorf("db: tx callback is required")
	}

	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
