package db

import (
	"crypto/sha256"
	"encoding/hex"
)

// SHA256Hex returns the lower-case sha256 hex digest of the input bytes.
func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
