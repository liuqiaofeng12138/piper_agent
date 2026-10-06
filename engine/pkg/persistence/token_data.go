package persistence

import (
	"strings"

	"piper_go/pkg/distributor/cache"
	"piper_go/pkg/tpl"
)

// TokenData aggregates token logs with docs grouped by index name (Java TokenRoute.data).
func TokenData(token map[string]any, fetchToken func(id string) map[string]any) map[string]any {
	tokenID, _ := token["id"].(string)
	docsByID, sources := tpl.CollectTokenData(token, fetchToken)
	docsByName := map[string][]map[string]any{}
	for indexID, list := range docsByID {
		name := indexID
		if idx := cache.GetIndexByIDOrName(indexID); idx != nil {
			if n, _ := idx["name"].(string); n != "" {
				name = n
			}
		}
		name = strings.ToLower(name)
		docsByName[name] = append(docsByName[name], list...)
	}
	return map[string]any{
		"token_id": tokenID,
		"docs":     docsByName,
		"sources":  sources,
	}
}
