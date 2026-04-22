// Package mailrender turns HTML email bodies into terminal-friendly output using
// sanitize → HTML-to-Markdown → Glamour, with plain-text fallback.
package mailrender

import (
	"io"
	"mime/quotedprintable"
	"regexp"
	"strings"

	"charm.land/glamour/v2"
	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/microcosm-cc/bluemonday"

	"github.com/termite-mail/termite/internal/tui/linefmt"
)

// MaxHTMLInput is a safety cap on HTML payload size before sanitization.
const MaxHTMLInput = 2 << 20

// RenderBodyLines converts HTML email to styled terminal lines when possible.
// If html is empty or conversion fails, plain text is wrapped and ansi is false.
func RenderBodyLines(htmlInput, plain string, wrapWidth int) (lines []string, ansi bool) {
	plain = strings.ReplaceAll(plain, "\r\n", "\n")
	htmlInput = strings.TrimSpace(htmlInput)
	if htmlInput == "" {
		return splitPlain(plain, wrapWidth), false
	}
	if len(htmlInput) > MaxHTMLInput {
		htmlInput = htmlInput[:MaxHTMLInput]
	}

	// Messages synced before folded-header CTE parsing may still hold raw QP in the DB.
	htmlInput = maybeDecodeQuotedPrintable(htmlInput)

	policy := bluemonday.UGCPolicy()
	safe := policy.Sanitize(htmlInput)

	md, err := htmltomarkdown.ConvertString(safe)
	if err != nil {
		return splitPlain(plain, wrapWidth), false
	}
	md = strings.TrimSpace(md)
	if md == "" {
		return splitPlain(plain, wrapWidth), false
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(wrapWidth),
	)
	if err != nil {
		return splitPlain(plain, wrapWidth), false
	}
	out, err := r.Render(md)
	if err != nil {
		return splitPlain(plain, wrapWidth), false
	}
	out = strings.TrimRight(out, "\n")
	if out == "" {
		return splitPlain(plain, wrapWidth), false
	}
	return strings.Split(out, "\n"), true
}

func splitPlain(plain string, wrapWidth int) []string {
	plain = maybeDecodeQuotedPrintable(plain)
	if plain == "" {
		return []string{""}
	}
	return strings.Split(linefmt.WrapPlainText(plain, wrapWidth), "\n")
}

var qpSoftLineBreak = regexp.MustCompile(`=[ \t]*(?:\r?\n)`)

// maybeDecodeQuotedPrintable decodes obvious quoted-printable leftovers
// (soft line breaks or many =HH hex bytes) without harming normal HTML/text.
func maybeDecodeQuotedPrintable(s string) string {
	if !looksLikeQuotedPrintable(s) {
		return s
	}
	r := quotedprintable.NewReader(strings.NewReader(strings.ReplaceAll(s, "\r\n", "\n")))
	out, err := io.ReadAll(r)
	if err != nil {
		return s
	}
	dec := string(out)
	if len(dec) == 0 {
		return s
	}
	return dec
}

func looksLikeQuotedPrintable(s string) bool {
	if qpSoftLineBreak.MatchString(s) {
		return true
	}
	// UTF-8 text in QP uses many =XX triplets; require several to avoid false positives.
	n := 0
	for i := 0; i+2 < len(s); i++ {
		if s[i] != '=' {
			continue
		}
		if !isHex(s[i+1]) || !isHex(s[i+2]) {
			continue
		}
		n++
		if n >= 8 {
			return true
		}
	}
	return false
}

func isHex(b byte) bool {
	switch {
	case b >= '0' && b <= '9':
		return true
	case b >= 'a' && b <= 'f':
		return true
	case b >= 'A' && b <= 'F':
		return true
	default:
		return false
	}
}
