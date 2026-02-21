package security

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// PostgresNonceStore persists monotonic nonce watermark reservations.
type PostgresNonceStore struct {
	db       *sql.DB
	nodeName string
	scope    string
	window   uint64
}

func NewPostgresNonceStore(db *sql.DB, nodeName, scope string, window uint64) (*PostgresNonceStore, error) {
	if db == nil {
		return nil, fmt.Errorf("security: nil db handle")
	}
	nodeName = strings.TrimSpace(nodeName)
	scope = strings.TrimSpace(scope)
	if nodeName == "" {
		return nil, fmt.Errorf("security: node name is required")
	}
	if scope == "" {
		scope = "default"
	}
	if window == 0 {
		return nil, fmt.Errorf("security: nonce window must be > 0")
	}
	return &PostgresNonceStore{
		db:       db,
		nodeName: nodeName,
		scope:    scope,
		window:   window,
	}, nil
}

// LoadReserved returns the current reserved high watermark for this node/scope.
func (s *PostgresNonceStore) LoadReserved() (uint64, error) {
	var reserved int64
	err := s.db.QueryRow(
		`SELECT reserved_high
		   FROM security.nonce_watermarks
		  WHERE node_name = $1
		    AND nonce_scope = $2`,
		s.nodeName,
		s.scope,
	).Scan(&reserved)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if reserved < 0 {
		return 0, fmt.Errorf("security: negative reserved watermark")
	}
	return uint64(reserved), nil
}

// PersistReserved upserts the reserved high watermark.
func (s *PostgresNonceStore) PersistReserved(reserved uint64) error {
	_, err := s.db.Exec(
		`INSERT INTO security.nonce_watermarks
		 (node_name, nonce_scope, reserved_high, window_size, updated_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (node_name, nonce_scope) DO UPDATE
		 SET reserved_high = EXCLUDED.reserved_high,
		     window_size = EXCLUDED.window_size,
		     updated_at = EXCLUDED.updated_at`,
		s.nodeName,
		s.scope,
		int64(reserved),
		int64(s.window),
		time.Now().UTC(),
	)
	return err
}

func (s *PostgresNonceStore) NodeName() string {
	return s.nodeName
}

func (s *PostgresNonceStore) Scope() string {
	return s.scope
}

func (s *PostgresNonceStore) WindowSize() uint64 {
	return s.window
}
