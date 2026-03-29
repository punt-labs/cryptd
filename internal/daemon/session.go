package daemon

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// connState tracks per-connection state for a single client session.
type connState struct {
	sessionID string
}

// generateSessionID returns a 32-character hex string from crypto/rand.
func generateSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session ID: %w", err)
	}
	return hex.EncodeToString(b), nil
}
