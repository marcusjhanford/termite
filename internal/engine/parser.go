package engine

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"mime/quotedprintable"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"

	"github.com/termite-mail/termite/internal/config"
	"github.com/termite-mail/termite/internal/db"
)

// ParseRawMessage converts a RawMessage (from IMAP fetch) into a db.Message
// suitable for database storage. It parses MIME structure, extracts text and
// HTML bodies, detects attachments, and populates threading headers.
func ParseRawMessage(raw RawMessage, accountID string) (*db.Message, error) {
	env := raw.Envelope

	// Generate a deterministic message ID from the IMAP UID and account.
	messageID := env.MessageID
	if messageID == "" {
		messageID = generateMessageID(accountID, raw.UID)
	}

	// Determine the From address.
	fromAddr := ""
	if len(env.From) > 0 {
		fromAddr = env.From[0]
	}

	// Build JSON arrays for To and Cc.
	toJSON := stringsToJSON(env.To)
	ccJSON := stringsToJSON(env.Cc)

	// Parse the message body.
	bodyText, bodyHTML, hasAttachment := parseBody(raw.Body)

	// Build References value from envelope.
	referencesStr := strings.Join(env.References, " ")

	// Determine read status from flags.
	isRead := false
	for _, f := range raw.Flags {
		if f == imap.FlagSeen {
			isRead = true
			break
		}
	}

	// Determine starred status.
	isStarred := false
	for _, f := range raw.Flags {
		if f == imap.FlagFlagged {
			isStarred = true
			break
		}
	}

	// Date as Unix timestamp.
	dateUnix := env.Date.Unix()
	if env.Date.IsZero() {
		dateUnix = time.Now().Unix()
	}

	// Build raw headers string from envelope data for storage.
	rawHeaders := buildRawHeaders(env)

	msg := &db.Message{
		ID:            messageID,
		AccountID:     accountID,
		UID:           uint32(raw.UID),
		FromAddr:      fromAddr,
		ToAddrs:       toJSON,
		CcAddrs:       ccJSON,
		Subject:       env.Subject,
		Date:          dateUnix,
		BodyText:      bodyText,
		BodyHTML:      bodyHTML,
		RawHeaders:    rawHeaders,
		IsRead:        isRead,
		IsStarred:     isStarred,
		HasAttachment: hasAttachment,
		InReplyTo:     toNullString(firstOrEmpty(env.InReplyTo)),
		References:    toNullString(referencesStr),
	}

	return msg, nil
}

// EvaluateSplitInboxRules evaluates the split inbox rules against a message
// and returns the ID of the first matching split inbox. If no rules match,
// returns "primary" as the default.
func EvaluateSplitInboxRules(msg *db.Message, inboxes []config.SplitInboxConfig) string {
	for _, inbox := range inboxes {
		if matchesInbox(msg, inbox) {
			return inbox.ID
		}
	}
	return "primary"
}

// matchesInbox checks whether a message matches all rules for a given split inbox.
// An inbox matches if:
//   - The message's account is in the inbox's accounts list (or accounts is empty = all).
//   - All rules match the message.
func matchesInbox(msg *db.Message, inbox config.SplitInboxConfig) bool {
	// Check account filter.
	if len(inbox.Accounts) > 0 {
		found := false
		for _, a := range inbox.Accounts {
			if a == msg.AccountID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// All rules must match.
	for _, rule := range inbox.Rules {
		if !matchesRule(msg, rule) {
			return false
		}
	}

	return true
}

// matchesRule evaluates a single InboxRule against a message.
func matchesRule(msg *db.Message, rule config.InboxRule) bool {
	fieldValue := getFieldValue(msg, rule.Field)

	// Check "exists" condition (typically for headers like list_unsubscribe).
	if rule.Exists != nil {
		exists := fieldValue != ""
		if exists != *rule.Exists {
			return false
		}
	}

	// Check "contains" conditions - at least one must match.
	if len(rule.Contains) > 0 {
		matched := false
		lowerField := strings.ToLower(fieldValue)
		for _, c := range rule.Contains {
			if strings.Contains(lowerField, strings.ToLower(c)) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check "not_contains" conditions - none must match.
	if len(rule.NotContains) > 0 {
		lowerField := strings.ToLower(fieldValue)
		for _, c := range rule.NotContains {
			if strings.Contains(lowerField, strings.ToLower(c)) {
				return false
			}
		}
	}

	return true
}

// getFieldValue extracts the relevant field value from a message for rule evaluation.
func getFieldValue(msg *db.Message, field string) string {
	switch field {
	case "from":
		return msg.FromAddr
	case "to":
		return msg.ToAddrs
	case "subject":
		return msg.Subject
	case "list_unsubscribe":
		// The List-Unsubscribe header is stored in raw headers.
		return extractHeader(msg.RawHeaders, "List-Unsubscribe")
	case "header":
		// Generic header lookup - requires HeaderName in the rule,
		// which is handled at the rule level.
		return msg.RawHeaders
	default:
		return ""
	}
}

// extractHeader extracts a specific header value from the raw headers string.
func extractHeader(rawHeaders, headerName string) string {
	lowerName := strings.ToLower(headerName) + ":"
	for _, line := range strings.Split(rawHeaders, "\n") {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), lowerName) {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

// parseBody extracts text and HTML content from a raw message body.
// It performs basic MIME boundary detection and part extraction.
// Returns (plainText, html, hasAttachment).
func parseBody(body []byte) (string, string, bool) {
	if len(body) == 0 {
		return "", "", false
	}

	bodyStr := string(body)

	// Check if this is a multipart message.
	if idx := strings.Index(bodyStr, "Content-Type:"); idx >= 0 {
		// Look for multipart boundary in the headers.
		headerEnd := strings.Index(bodyStr, "\r\n\r\n")
		if headerEnd == -1 {
			headerEnd = strings.Index(bodyStr, "\n\n")
		}
		if headerEnd == -1 {
			// Single part message, try to extract body after headers.
			return extractSinglePartBody(bodyStr)
		}

		headers := bodyStr[:headerEnd]
		bodyContent := bodyStr[headerEnd:]

		// Extract boundary for multipart.
		boundary := extractBoundary(headers)
		if boundary != "" {
			return parseMultipart(bodyContent, boundary)
		}

		// Single part with headers.
		return extractSinglePartBody(bodyStr)
	}

	// No Content-Type header, treat as plain text.
	return bodyStr, "", false
}

// extractSinglePartBody extracts the body from a single-part message.
func extractSinglePartBody(msg string) (string, string, bool) {
	// Find the header/body separator.
	separators := []string{"\r\n\r\n", "\n\n"}
	for _, sep := range separators {
		if idx := strings.Index(msg, sep); idx >= 0 {
			headers := strings.ToLower(msg[:idx])
			body := decodeMIMEBody(headers, msg[idx+len(sep):])

			if strings.Contains(headers, "content-type: text/html") ||
				strings.Contains(headers, "content-type:text/html") {
				return "", body, false
			}
			return body, "", false
		}
	}
	return msg, "", false
}

// extractBoundary extracts the MIME boundary from content-type headers.
func extractBoundary(headers string) string {
	lower := strings.ToLower(headers)
	idx := strings.Index(lower, "boundary=")
	if idx == -1 {
		return ""
	}

	rest := headers[idx+len("boundary="):]
	rest = strings.TrimSpace(rest)

	if len(rest) > 0 && rest[0] == '"' {
		// Quoted boundary.
		end := strings.Index(rest[1:], `"`)
		if end >= 0 {
			return rest[1 : end+1]
		}
		return rest[1:]
	}

	// Unquoted boundary - ends at whitespace, semicolon, or newline.
	end := strings.IndexAny(rest, " \t\r\n;")
	if end >= 0 {
		return rest[:end]
	}
	return rest
}

// parseMultipart parses a multipart MIME body and extracts text/html parts.
func parseMultipart(body, boundary string) (string, string, bool) {
	delimiter := "--" + boundary
	parts := strings.Split(body, delimiter)

	var textBody, htmlBody string
	hasAttachment := false

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "--" {
			continue
		}

		// Separate part headers from part body.
		var partHeaders, partBody string
		for _, sep := range []string{"\r\n\r\n", "\n\n"} {
			if idx := strings.Index(part, sep); idx >= 0 {
				partHeaders = strings.ToLower(part[:idx])
				partBody = part[idx+len(sep):]
				break
			}
		}
		if partHeaders == "" {
			continue
		}

		// Check for attachments.
		if strings.Contains(partHeaders, "content-disposition: attachment") ||
			strings.Contains(partHeaders, "content-disposition:attachment") {
			hasAttachment = true
			continue
		}

		// Extract text/plain.
		if strings.Contains(partHeaders, "content-type: text/plain") ||
			strings.Contains(partHeaders, "content-type:text/plain") {
			textBody = decodeMIMEBody(partHeaders, partBody)
		}

		// Extract text/html.
		if strings.Contains(partHeaders, "content-type: text/html") ||
			strings.Contains(partHeaders, "content-type:text/html") {
			htmlBody = decodeMIMEBody(partHeaders, partBody)
		}
	}

	return textBody, htmlBody, hasAttachment
}

// decodeMIMEBody applies Content-Transfer-Encoding (quoted-printable, base64) when present.
func decodeMIMEBody(headersBlock string, body string) string {
	h := strings.ToLower(headersBlock)
	if strings.Contains(h, "content-transfer-encoding: quoted-printable") ||
		strings.Contains(h, "content-transfer-encoding:quoted-printable") {
		r := quotedprintable.NewReader(strings.NewReader(body))
		out, err := io.ReadAll(r)
		if err == nil {
			return string(out)
		}
	}
	if strings.Contains(h, "content-transfer-encoding: base64") ||
		strings.Contains(h, "content-transfer-encoding:base64") {
		s := strings.NewReplacer("\r\n", "", "\n", "", "\r", "").Replace(body)
		s = strings.ReplaceAll(s, " ", "")
		if raw, err := base64.StdEncoding.DecodeString(s); err == nil {
			return string(raw)
		}
	}
	return body
}

// generateMessageID creates a deterministic message ID when the IMAP envelope
// does not provide one.
func generateMessageID(accountID string, uid imap.UID) string {
	data := fmt.Sprintf("%s:%d", accountID, uid)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16])
}

// stringsToJSON converts a string slice to a JSON array string.
func stringsToJSON(strs []string) string {
	if len(strs) == 0 {
		return "[]"
	}
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i, s := range strs {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteByte('"')
		// Escape double quotes and backslashes in the string.
		escaped := strings.ReplaceAll(s, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		buf.WriteString(escaped)
		buf.WriteByte('"')
	}
	buf.WriteByte(']')
	return buf.String()
}

// toNullString converts a string to a sql.NullString.
// Empty strings result in a NULL value.
func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// firstOrEmpty returns the first element of a string slice, or empty string.
func firstOrEmpty(strs []string) string {
	if len(strs) > 0 {
		return strs[0]
	}
	return ""
}

// buildRawHeaders reconstructs a raw headers string from envelope data.
// This is a simplified representation used for rule evaluation and storage.
func buildRawHeaders(env Envelope) string {
	var buf bytes.Buffer
	if env.Subject != "" {
		fmt.Fprintf(&buf, "Subject: %s\n", env.Subject)
	}
	if len(env.From) > 0 {
		fmt.Fprintf(&buf, "From: %s\n", strings.Join(env.From, ", "))
	}
	if len(env.To) > 0 {
		fmt.Fprintf(&buf, "To: %s\n", strings.Join(env.To, ", "))
	}
	if len(env.Cc) > 0 {
		fmt.Fprintf(&buf, "Cc: %s\n", strings.Join(env.Cc, ", "))
	}
	if env.MessageID != "" {
		fmt.Fprintf(&buf, "Message-ID: <%s>\n", env.MessageID)
	}
	if len(env.InReplyTo) > 0 {
		fmt.Fprintf(&buf, "In-Reply-To: <%s>\n", strings.Join(env.InReplyTo, "> <"))
	}
	if len(env.References) > 0 {
		refs := make([]string, len(env.References))
		for i, r := range env.References {
			refs[i] = "<" + r + ">"
		}
		fmt.Fprintf(&buf, "References: %s\n", strings.Join(refs, " "))
	}
	if !env.Date.IsZero() {
		fmt.Fprintf(&buf, "Date: %s\n", env.Date.Format(time.RFC1123Z))
	}
	return buf.String()
}
