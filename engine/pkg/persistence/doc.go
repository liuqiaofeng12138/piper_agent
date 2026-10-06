package persistence

import (
	"strings"
	"time"

	"piper_go/pkg/distributor/cache"
	"piper_go/pkg/tpl"
	"piper_go/pkg/util"
)

const IndexSource = "source"

// DocForES converts a mapper doc to Elasticsearch _source (Java DocAdapter).
func DocForES(doc map[string]any, taskID, tplID, tokenID string) map[string]any {
	out := map[string]any{}
	for k, v := range doc {
		if strings.HasPrefix(k, "_") {
			continue
		}
		out[k] = v
	}
	indexID, _ := doc["_index_id"].(string)
	now := time.Now().UnixMilli()
	out["__index"] = indexID
	if taskID != "" {
		out["__task_id"] = taskID
	}
	if tplID != "" {
		out["__tpl_id"] = tplID
	}
	if tokenID != "" {
		out["__token_id"] = tokenID
	}
	explicitID, _ := doc["id"].(string)
	out["id"] = util.AssignDocumentID(out, explicitID, tokenID, indexID)
	out["create_time"] = now
	out["update_time"] = now
	return out
}

// ESIndexNameForDoc returns ES index name for a doc (_index_id / __index → Index.name).
func ESIndexNameForDoc(doc map[string]any) string {
	indexID, _ := doc["_index_id"].(string)
	if indexID == "" {
		indexID, _ = doc["__index"].(string)
	}
	if indexID == "" {
		return "document"
	}
	if idx := cache.GetIndexByIDOrName(indexID); idx != nil {
		if name, _ := idx["name"].(string); name != "" {
			return strings.ToLower(name)
		}
	}
	return strings.ToLower(indexID)
}

// CollectDocsFromToken flattens proc docs with lineage fields (Java Token.getDocs).
func CollectDocsFromToken(token map[string]any) []map[string]any {
	taskID, _ := token["task_id"].(string)
	tplID, _ := token["tpl_id"].(string)
	tokenID, _ := token["id"].(string)
	docsByKey, _ := tpl.CollectTokenData(token, nil)
	var out []map[string]any
	for _, list := range docsByKey {
		for _, d := range list {
			out = append(out, DocForES(d, taskID, tplID, tokenID))
		}
	}
	return out
}
