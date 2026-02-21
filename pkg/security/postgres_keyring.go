package security

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
)

// PostgresKeyResolver resolves signing keys from security.signing_key_versions.
//
// Note: key material bootstrap strategy for Sprint B:
// 1. Treat key_hash as either hex-encoded raw key bytes or raw fallback bytes.
// 2. Future hardening can replace this with KMS-backed envelope decryption.
type PostgresKeyResolver struct {
	db *sql.DB
	mu sync.RWMutex
}

func NewPostgresKeyResolver(db *sql.DB) (*PostgresKeyResolver, error) {
	if db == nil {
		return nil, fmt.Errorf("security: nil db handle")
	}
	return &PostgresKeyResolver{db: db}, nil
}

func (r *PostgresKeyResolver) Current() (string, []byte, error) {
	var kid string
	var stored string
	err := r.db.QueryRow(
		`SELECT skv.key_id, skv.key_hash
		   FROM security.signing_key_versions skv
		   JOIN security.signing_keys sk ON sk.id = skv.signing_key_id
		  WHERE skv.is_current = true
		    AND sk.status = 'active'
		    AND (skv.valid_until IS NULL OR skv.valid_until > now())
		  ORDER BY skv.valid_from DESC
		  LIMIT 1`,
	).Scan(&kid, &stored)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil, fmt.Errorf("security: no current signing key")
		}
		return "", nil, err
	}

	key := decodeStoredKey(stored)
	if len(key) == 0 {
		return "", nil, fmt.Errorf("security: current key material is empty")
	}
	return kid, key, nil
}

func (r *PostgresKeyResolver) Lookup(kid string) ([]byte, bool) {
	kid = strings.TrimSpace(kid)
	if kid == "" {
		return nil, false
	}

	var stored string
	err := r.db.QueryRow(
		`SELECT key_hash
		   FROM security.signing_key_versions
		  WHERE key_id = $1
		    AND (valid_until IS NULL OR valid_until > now())
		  LIMIT 1`,
		kid,
	).Scan(&stored)
	if err != nil {
		return nil, false
	}
	key := decodeStoredKey(stored)
	if len(key) == 0 {
		return nil, false
	}
	return key, true
}

func decodeStoredKey(stored string) []byte {
	s := strings.TrimSpace(stored)
	if s == "" {
		return nil
	}
	if decoded, err := hex.DecodeString(s); err == nil && len(decoded) > 0 {
		return decoded
	}
	return []byte(s)
}
