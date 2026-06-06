package neith

import (
	"strings"
)

func sanitizeHTML(html string) string {
	sanitized := strings.ReplaceAll(html, "\n", "")
	sanitized = strings.ReplaceAll(sanitized, "\t", "")
	return sanitized
}
