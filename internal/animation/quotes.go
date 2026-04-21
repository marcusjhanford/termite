package animation

import (
	"crypto/sha256"
	"encoding/binary"
	"time"
)

// ForestQuotes is the curated pool of calming quotes displayed during
// the inbox zero forest scene.
var ForestQuotes = []string{
	"The quieter you become, the more you can hear.",
	"Simplicity is the ultimate sophistication.",
	"You have arrived.",
	"The present moment always will have been.",
	"Rest is not idleness.",
	"Inhale the future. Exhale the past.",
	"Nothing is lost. Everything is transformed.",
	"The inbox is empty. The mind can breathe.",
	"Done is a place you can visit.",
	"Every ending is a clearing.",
}

// SelectQuote deterministically selects a quote based on the account ID
// and the current date. The same account sees the same quote per day
// rather than a random one per visit.
func SelectQuote(accountID string, date time.Time) string {
	// Build a deterministic seed from accountID + date.
	key := accountID + date.Format("2006-01-02")
	h := sha256.Sum256([]byte(key))
	// Use the first 8 bytes of the hash as an index.
	idx := binary.BigEndian.Uint64(h[:8]) % uint64(len(ForestQuotes))
	return ForestQuotes[idx]
}

// RevealQuote returns the quote with only the first charsRevealed characters
// visible, and the rest replaced with spaces. This implements the typewriter
// reveal effect during the forest scene.
func RevealQuote(quote string, charsRevealed int) string {
	runes := []rune(quote)
	if charsRevealed >= len(runes) {
		return quote
	}
	result := make([]rune, len(runes))
	for i, r := range runes {
		if i < charsRevealed {
			result[i] = r
		} else {
			result[i] = ' '
		}
	}
	return string(result)
}
