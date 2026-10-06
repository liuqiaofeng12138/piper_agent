package tpl

import "strings"

// ProcTypeFromJSON returns the simple class name from Gson _type field.
func ProcTypeFromJSON(m map[string]any) string {
	raw, _ := m["_type"].(string)
	if raw == "" {
		return ""
	}
	if i := strings.LastIndex(raw, "."); i >= 0 {
		return raw[i+1:]
	}
	return raw
}
