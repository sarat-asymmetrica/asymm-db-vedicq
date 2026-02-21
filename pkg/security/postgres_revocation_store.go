package security

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// PostgresRevocationStore persists revocation state in security.* tables.
type PostgresRevocationStore struct {
	db *sql.DB
}

func NewPostgresRevocationStore(db *sql.DB) (*PostgresRevocationStore, error) {
	if db == nil {
		return nil, fmt.Errorf("security: nil db handle")
	}
	return &PostgresRevocationStore{db: db}, nil
}

func (s *PostgresRevocationStore) RevokeToken(tokenID string, until time.Time) error {
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return fmt.Errorf("security: token id is required")
	}
	if until.IsZero() {
		return fmt.Errorf("security: token expiry is required")
	}
	_, err := s.db.Exec(
		`INSERT INTO security.revoked_tokens (token_id, expires_at)
		 VALUES ($1, $2)
		 ON CONFLICT (token_id) DO UPDATE
		 SET expires_at = GREATEST(security.revoked_tokens.expires_at, EXCLUDED.expires_at),
		     revoked_at = now()`,
		tokenID,
		until.UTC(),
	)
	return err
}

func (s *PostgresRevocationStore) RevokeSession(sessionID string, until time.Time) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("security: session id is required")
	}
	if until.IsZero() {
		return fmt.Errorf("security: session expiry is required")
	}
	_, err := s.db.Exec(
		`INSERT INTO security.revoked_sessions (session_id, expires_at)
		 VALUES ($1, $2)
		 ON CONFLICT (session_id) DO UPDATE
		 SET expires_at = GREATEST(security.revoked_sessions.expires_at, EXCLUDED.expires_at),
		     revoked_at = now()`,
		sessionID,
		until.UTC(),
	)
	return err
}

func (s *PostgresRevocationStore) IsTokenRevoked(tokenID string, now time.Time) (bool, error) {
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return false, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var exists bool
	if err := s.db.QueryRow(
		`SELECT EXISTS (
		   SELECT 1 FROM security.revoked_tokens
		    WHERE token_id = $1
		      AND expires_at > $2
		)`,
		tokenID,
		now.UTC(),
	).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (s *PostgresRevocationStore) IsSessionRevoked(sessionID string, now time.Time) (bool, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return false, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var exists bool
	if err := s.db.QueryRow(
		`SELECT EXISTS (
		   SELECT 1 FROM security.revoked_sessions
		    WHERE session_id = $1
		      AND expires_at > $2
		)`,
		sessionID,
		now.UTC(),
	).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

// CleanupExpired deletes expired revocation entries for table hygiene.
func (s *PostgresRevocationStore) CleanupExpired(now time.Time) error {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if _, err := s.db.Exec(`DELETE FROM security.revoked_tokens WHERE expires_at <= $1`, now.UTC()); err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM security.revoked_sessions WHERE expires_at <= $1`, now.UTC()); err != nil {
		return err
	}
	return nil
}
