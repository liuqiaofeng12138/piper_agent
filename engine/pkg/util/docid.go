package util

import (
	"fmt"
	"sort"
	"strings"
)

// DocFingerprint builds a stable string from document fields for identity (sorted keys).
func DocFingerprint(fields map[string]any) string {
	if len(fields) == 0 {
		return ""
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		if strings.HasPrefix(k, "_") || k == "id" || k == "create_time" || k == "update_time" {
			continue
		}
		keys = append(keys, k)
	}
	if len(keys) == 0 {
		return ""
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(strings.TrimSpace(fmt.Sprint(fields[k])))
		b.WriteByte('\n')
	}
	return b.String()
}

// AssignDocumentID picks Elasticsearch document id: explicit id, title hash, content fingerprint, or token+index fallback.
func AssignDocumentID(fields map[string]any, explicitID, tokenID, indexID string) string {
	if id := strings.TrimSpace(explicitID); id != "" {
		return id
	}
	if title, ok := fields["title"].(string); ok {
		if t := strings.TrimSpace(title); t != "" {
			return MD5Hex(t)
		}
	}
	if fp := DocFingerprint(fields); fp != "" {
		if tokenID != "" {
			return MD5Hex(tokenID + "::" + fp)
		}
		return MD5Hex(fp)
	}
	return MD5Hex(tokenID + "::" + indexID)
}
