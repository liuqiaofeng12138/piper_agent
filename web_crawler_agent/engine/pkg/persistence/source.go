package persistence

import (
	"strings"
	"time"

	"piper_go/pkg/tpl"
	"piper_go/pkg/util"
)

// SourceForES builds ES document for index `source` (Java Source model).
func SourceForES(src map[string]any, taskID, tplID string) map[string]any {
	now := time.Now().UnixMilli()
	out := map[string]any{
		"update_time": now,
	}
	if id, _ := src["id"].(string); id != "" {
		out["id"] = id
	}
	url, _ := src["url"].(string)
	if url == "" {
		url, _ = src["uri"].(string)
	}
	if url != "" {
		out["uri"] = url
	}
	for _, k := range []string{"name", "mime", "meta", "f_id", "f_type", "width", "height", "size"} {
		if v, ok := src[k]; ok {
			out[k] = v
		}
	}
	if taskID != "" {
		out["__task_id"] = taskID
	}
	if tplID != "" {
		out["__tpl_id"] = tplID
	}
	if out["id"] == nil {
		seed := url
		if seed == "" {
			seed = strings.TrimSpace(metaString(src))
		}
		if seed != "" {
			out["id"] = hashID(seed)
		}
	}
	return out
}

func metaString(src map[string]any) string {
	m, _ := src["meta"].(string)
	return m
}

func hashID(s string) string {
	return util.MD5Hex(s)
}

// CollectSourcesFromToken gathers sources from token logs.
func CollectSourcesFromToken(token map[string]any) []map[string]any {
	_, sources := tpl.CollectTokenData(token, nil)
	taskID, _ := token["task_id"].(string)
	tplID, _ := token["tpl_id"].(string)
	var out []map[string]any
	for _, s := range sources {
		out = append(out, SourceForES(s, taskID, tplID))
	}
	return out
}
