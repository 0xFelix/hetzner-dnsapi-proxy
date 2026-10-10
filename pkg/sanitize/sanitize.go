package sanitize

import (
	"strings"
	"unicode"
)

// LogValue sanitizes a string for safe logging by removing all characters
// that are not printable, e.g. newlines, carriage returns and terminal
// escape sequences.
func LogValue(s string) string {
	return strings.Map(func(r rune) rune {
		if !unicode.IsPrint(r) {
			return -1
		}
		return r
	}, s)
}
