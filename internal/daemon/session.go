package daemon

import (
	"crypto/rand"
	"encoding/hex"
)

// connState tracks per-connection state for a single client session.
type connState struct {
	sessionID string
}

// generateSessionID returns a 32-character hex string from crypto/rand.
func generateSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}
