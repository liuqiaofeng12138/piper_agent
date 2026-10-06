package txt

import (
	"strings"
	"unicode/utf8"

	"piper_go/pkg/util"
)

func MD5Hex(s string) string {
	return util.MD5Hex(s)
}

func StrCrop(s string, maxRunes int) string {
	if maxRunes <= 0 || s == "" {
		return s
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	i := 0
	for pos := range s {
		if i == maxRunes {
			return s[:pos]
		}
		i++
	}
	return s
}

func SplitFirst(s, sep string) (before, after string) {
	i := strings.Index(s, sep)
	if i < 0 {
		return s, ""
	}
	return s[:i], s[i+len(sep):]
}

func TrimSpace(s string) string {
	return strings.TrimSpace(s)
}

func Empty(s string) bool {
	return strings.TrimSpace(s) == ""
}
