package daemon

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"
)

// connState tracks per-connection state for a single client session.
type connState struct {
	sessionID string
}

// maxSessionIDLen is the maximum length for a client-provided session ID.
const maxSessionIDLen = 128

// sanitizeSessionID validates and sanitizes a client-provided session ID.
// Returns the sanitized ID, or empty string if the input is invalid
// (empty, too long, or contains only control characters).
func sanitizeSessionID(id string) string {
	if id == "" || len(id) > maxSessionIDLen {
		return ""
	}
	// Strip control characters (newlines, tabs, etc.) to prevent log injection.
	clean := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, id)
	if clean == "" {
		return ""
	}
	return clean
}

// generateSessionID returns a 32-character hex string from crypto/rand.
func generateSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session ID: %w", err)
	}
	return hex.EncodeToString(b), nil
}
