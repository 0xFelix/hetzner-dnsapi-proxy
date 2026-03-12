package sanitize

import "strings"

// LogValue sanitizes a string for safe logging by removing newlines and carriage returns.
func LogValue(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}
