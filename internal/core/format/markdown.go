package format

import (
	"strings"
	"unicode"
)

// NormalizeMarkdown trims outer space, normalizes newlines to \n, collapses excessive blank lines.
func NormalizeMarkdown(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return s
}

// StripInvisible removes zero-width and BOM-like runes often pasted from the web.
func StripInvisible(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\u200b' || r == '\ufeff' {
			return -1
		}
		if unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, s)
}
