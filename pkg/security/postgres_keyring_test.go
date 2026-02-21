package security

import (
	"encoding/hex"
	"testing"
)

func TestDecodeStoredKeyHex(t *testing.T) {
	raw := []byte("super-secret")
	hexText := hex.EncodeToString(raw)
	got := decodeStoredKey(hexText)
	if string(got) != string(raw) {
		t.Fatalf("decoded mismatch, got %q want %q", string(got), string(raw))
	}
}

func TestDecodeStoredKeyRawFallback(t *testing.T) {
	raw := "not-hex-value"
	got := decodeStoredKey(raw)
	if string(got) != raw {
		t.Fatalf("fallback mismatch, got %q want %q", string(got), raw)
	}
}
