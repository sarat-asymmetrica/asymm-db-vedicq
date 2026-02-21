package security

import (
	"database/sql"
	"fmt"
	"strings"
)

// RuntimeConfig drives Postgres security runtime wiring.
type RuntimeConfig struct {
	NodeName    string
	NonceScope  string
	NonceWindow uint64
}

func (c RuntimeConfig) Validate() error {
	if strings.TrimSpace(c.NodeName) == "" {
		return fmt.Errorf("security: runtime node name is required")
	}
	if strings.TrimSpace(c.NonceScope) == "" {
		c.NonceScope = "default"
	}
	if c.NonceWindow == 0 {
		return fmt.Errorf("security: runtime nonce window must be > 0")
	}
	return nil
}

// RuntimeDeps bundles Postgres-backed auth/security dependencies.
//
// This stays contract-compatible with Betanet-style auth components:
// 1. KeyResolver for Current/Lookup key resolution.
// 2. RevocationStore for token/session revocation checks.
// 3. NonceStore for crash-safe nonce persistence callbacks.
type RuntimeDeps struct {
	KeyResolver     KeyResolver
	RevocationStore RevocationStore
	NonceStore      *PostgresNonceStore
	NonceStart      uint64
}

func BuildPostgresRuntime(db *sql.DB, cfg RuntimeConfig) (*RuntimeDeps, error) {
	if db == nil {
		return nil, fmt.Errorf("security: nil db handle")
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.NonceScope) == "" {
		cfg.NonceScope = "default"
	}

	keyResolver, err := NewPostgresKeyResolver(db)
	if err != nil {
		return nil, err
	}
	revocation, err := NewPostgresRevocationStore(db)
	if err != nil {
		return nil, err
	}
	nonceStore, err := NewPostgresNonceStore(db, cfg.NodeName, cfg.NonceScope, cfg.NonceWindow)
	if err != nil {
		return nil, err
	}
	start, err := nonceStore.LoadReserved()
	if err != nil {
		return nil, err
	}

	return &RuntimeDeps{
		KeyResolver:     keyResolver,
		RevocationStore: revocation,
		NonceStore:      nonceStore,
		NonceStart:      start,
	}, nil
}
