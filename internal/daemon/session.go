package daemon

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"
)

// Session tracks per-connection state for a client session.
// A session persists across reconnects — a client that sends `initialize` with
// its previous session ID gets the same Session back, with its game intact.
type Session struct {
	id          string
	gameID      string // empty until new_game
	passthrough bool   // true = structured JSON responses, false = interpreted + narrated text
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

// generateID returns a 32-character hex string from crypto/rand.
func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate ID: %w", err)
	}
	return hex.EncodeToString(b), nil
}
