package tpl

// CollectTokenData aggregates docs and sources from token logs (Java TokenRoute.data).
func CollectTokenData(token map[string]any, fetchToken func(id string) map[string]any) (docs map[string][]map[string]any, sources []map[string]any) {
	docs = map[string][]map[string]any{}
	sources = []map[string]any{}
	collectFromToken(token, docs, &sources, fetchToken)
	return docs, sources
}

func collectFromToken(token map[string]any, docs map[string][]map[string]any, sources *[]map[string]any, fetchToken func(id string) map[string]any) {
	logs, _ := token["logs"].(map[string]any)
	for _, v := range logs {
		list, ok := v.([]any)
		if !ok {
			if lm, ok := v.([]map[string]any); ok {
				for _, log := range lm {
					collectFromLog(log, docs, sources, fetchToken)
				}
			}
			continue
		}
		for _, item := range list {
			log, ok := item.(map[string]any)
			if !ok {
				continue
			}
			collectFromLog(log, docs, sources, fetchToken)
		}
	}
}

func collectFromLog(log map[string]any, docs map[string][]map[string]any, sources *[]map[string]any, fetchToken func(id string) map[string]any) {
	if raw, ok := log["docs"].([]any); ok {
		for _, d := range raw {
			if doc, ok := d.(map[string]any); ok {
				addDoc(docs, doc)
			}
		}
	}
	if raw, ok := log["docs"].([]map[string]any); ok {
		for _, doc := range raw {
			addDoc(docs, doc)
		}
	}
	if raw, ok := log["sources"].([]any); ok {
		for _, s := range raw {
			if m, ok := s.(map[string]any); ok {
				*sources = append(*sources, m)
			}
		}
	}
	if raw, ok := log["sources"].([]map[string]any); ok {
		*sources = append(*sources, raw...)
	}
	if raw, ok := log["tokens"].([]any); ok {
		for _, t := range raw {
			tok, ok := t.(map[string]any)
			if !ok {
				continue
			}
			if id, _ := tok["id"].(string); id != "" && fetchToken != nil {
				if full := fetchToken(id); full != nil {
					collectFromToken(full, docs, sources, fetchToken)
				}
			}
		}
	}
	if raw, ok := log["tokens"].([]map[string]any); ok {
		for _, tok := range raw {
			if id, _ := tok["id"].(string); id != "" && fetchToken != nil {
				if full := fetchToken(id); full != nil {
					collectFromToken(full, docs, sources, fetchToken)
				}
			}
		}
	}
}

func addDoc(docs map[string][]map[string]any, doc map[string]any) {
	indexID, _ := doc["_index_id"].(string)
	if indexID == "" {
		indexID = "_default"
	}
	docs[indexID] = append(docs[indexID], doc)
}
