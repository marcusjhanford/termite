package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/termite-mail/termite/internal/db"
)

// subjectPrefixRE matches common reply/forward prefixes like Re:, Fwd:, etc.
// It handles nested prefixes (e.g., "Re: Re: Fwd:") and is case-insensitive.
var subjectPrefixRE = regexp.MustCompile(`(?i)^(re|fwd|fw)\s*:\s*`)

// BuildThreads groups messages into conversation threads using JWZ-style threading.
//
// The algorithm:
//  1. Index all messages by Message-ID.
//  2. Link messages via In-Reply-To and References headers, building a parent map.
//  3. Walk parent chains to find the root message-id for each thread.
//  4. Messages with no threading headers fall back to subject-based grouping
//     (after stripping Re:/Fwd: prefixes).
//  5. Each thread is identified by a deterministic hash of the root message-id
//     or the normalized subject.
//
// Returns a map of thread ID -> messages in that thread (ordered by date ascending).
func BuildThreads(msgs []db.Message) map[string][]db.Message {
	if len(msgs) == 0 {
		return nil
	}

	// Step 1: Index messages by Message-ID.
	byMessageID := make(map[string]*db.Message, len(msgs))
	for i := range msgs {
		mid := msgs[i].ID
		if mid != "" {
			byMessageID[mid] = &msgs[i]
		}
	}

	// Step 2: Build parent map from In-Reply-To and References.
	// parentOf[child-message-id] = parent-message-id
	parentOf := make(map[string]string)
	for i := range msgs {
		m := &msgs[i]

		// Use In-Reply-To as the direct parent.
		if m.InReplyTo.Valid && m.InReplyTo.String != "" {
			parentOf[m.ID] = m.InReplyTo.String
		}

		// Process References header: each ref[i] is a parent of ref[i+1].
		refs := parseReferencesHeader(m.References.String)
		for j := 1; j < len(refs); j++ {
			parentOf[refs[j]] = refs[j-1]
		}
		// The last reference is the parent of this message if we don't
		// already have one from In-Reply-To.
		if len(refs) > 0 {
			if _, ok := parentOf[m.ID]; !ok {
				parentOf[m.ID] = refs[len(refs)-1]
			}
		}
	}

	// Step 3: Walk parent chains to find root message-id for each message.
	rootOf := make(map[string]string)
	for i := range msgs {
		mid := msgs[i].ID
		root := findRoot(mid, parentOf)
		rootOf[mid] = root
	}

	// Step 4: Group messages by root. Fall back to subject-based grouping
	// for messages with no threading headers.
	threadGroups := make(map[string][]db.Message)
	subjectGroups := make(map[string]string) // normalized-subject -> thread-id

	for i := range msgs {
		m := msgs[i]
		root := rootOf[m.ID]

		// If the root is the message itself AND it has no threading headers,
		// try subject-based fallback.
		if root == m.ID && !hasThreadingHeaders(m) {
			normalizedSubject := normalizeSubject(m.Subject)
			if normalizedSubject == "" {
				// No subject at all; give it its own thread.
				threadID := makeThreadID(m.ID)
				threadGroups[threadID] = append(threadGroups[threadID], m)
				continue
			}

			if existingThreadID, ok := subjectGroups[normalizedSubject]; ok {
				threadGroups[existingThreadID] = append(threadGroups[existingThreadID], m)
				continue
			}

			threadID := makeThreadID(normalizedSubject)
			subjectGroups[normalizedSubject] = threadID
			threadGroups[threadID] = append(threadGroups[threadID], m)
			continue
		}

		threadID := makeThreadID(root)
		// Register the normalized subject for this thread so subject-based
		// fallback can find it.
		normalizedSubject := normalizeSubject(m.Subject)
		if normalizedSubject != "" {
			if _, ok := subjectGroups[normalizedSubject]; !ok {
				subjectGroups[normalizedSubject] = threadID
			}
		}
		threadGroups[threadID] = append(threadGroups[threadID], m)
	}

	// Sort messages within each thread by date ascending.
	for threadID := range threadGroups {
		sortMessagesByDate(threadGroups[threadID])
	}

	return threadGroups
}

// findRoot walks the parent chain to find the root message-id.
// Guards against cycles with a visited set.
func findRoot(messageID string, parentOf map[string]string) string {
	visited := make(map[string]bool)
	current := messageID
	for {
		visited[current] = true
		parent, ok := parentOf[current]
		if !ok || parent == "" || visited[parent] {
			return current
		}
		current = parent
	}
}

// hasThreadingHeaders returns true if the message has In-Reply-To or References.
func hasThreadingHeaders(m db.Message) bool {
	if m.InReplyTo.Valid && m.InReplyTo.String != "" {
		return true
	}
	if m.References.Valid && m.References.String != "" {
		return true
	}
	return false
}

// normalizeSubject strips Re:/Fwd:/Fw: prefixes and trims whitespace,
// producing a canonical form for subject-based threading comparison.
func normalizeSubject(subject string) string {
	s := strings.TrimSpace(subject)
	for {
		stripped := subjectPrefixRE.ReplaceAllString(s, "")
		stripped = strings.TrimSpace(stripped)
		if stripped == s {
			break
		}
		s = stripped
	}
	return strings.ToLower(s)
}

// makeThreadID generates a deterministic thread ID from a key string
// (either a root message-id or a normalized subject).
func makeThreadID(key string) string {
	hash := sha256.Sum256([]byte(key))
	return "thread_" + hex.EncodeToString(hash[:12])
}

// parseReferencesHeader parses a References header value into individual message-ids.
// References are space-separated, optionally wrapped in angle brackets.
func parseReferencesHeader(refs string) []string {
	if refs == "" {
		return nil
	}

	// Split on whitespace; each token is a message-id possibly wrapped in < >.
	parts := strings.Fields(refs)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.TrimPrefix(p, "<")
		p = strings.TrimSuffix(p, ">")
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// sortMessagesByDate sorts messages by date ascending (oldest first).
func sortMessagesByDate(msgs []db.Message) {
	for i := 1; i < len(msgs); i++ {
		for j := i; j > 0 && msgs[j].Date < msgs[j-1].Date; j-- {
			msgs[j], msgs[j-1] = msgs[j-1], msgs[j]
		}
	}
}
