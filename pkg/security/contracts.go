package security

import "time"

// RevocationStore mirrors the token/session revocation contract used by auth services.
type RevocationStore interface {
	RevokeToken(tokenID string, until time.Time) error
	RevokeSession(sessionID string, until time.Time) error
	IsTokenRevoked(tokenID string, now time.Time) (bool, error)
	IsSessionRevoked(sessionID string, now time.Time) (bool, error)
}

// KeyResolver mirrors current/lookup signing key resolution contracts.
type KeyResolver interface {
	Current() (kid string, key []byte, err error)
	Lookup(kid string) ([]byte, bool)
}
