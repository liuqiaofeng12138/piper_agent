package txt

import (
	"regexp"
	"strings"
)

var (
	reSpaces   = regexp.MustCompile(`\s+`)
	reHTMLTags = regexp.MustCompile(`(?s)<[^>]+>`)
)

// CleanText strips HTML tags and collapses whitespace (Java ContentCleaner subset).
func CleanText(html string) string {
	s := reHTMLTags.ReplaceAllString(html, " ")
	s = reSpaces.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
