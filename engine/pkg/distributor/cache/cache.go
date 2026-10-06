package cache

import (
	"sync"
)

// In-memory caches aligned with one.rewind.nio.distributor.Cache (Phase 1 subset).
var (
	mu sync.RWMutex

	Indices   = map[string]map[string]any{}
	IndexName = map[string]string{} // name -> id

	Templates = map[string]map[string]any{}
	VarsLists = map[string]map[string]any{}
	TaskLists = map[string]map[string]any{}

	IndexTemplates     = map[string][]map[string]string{} // index id -> refs
	TemplateTemplates  = map[string][]map[string]string{} // template id -> refs
	IndexScripts       = map[string][2]string{}           // id -> card, detail
)

func PutIndex(doc map[string]any) {
	id, _ := doc["id"].(string)
	if id == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	Indices[id] = doc
	if name := str(doc, "name"); name != "" {
		IndexName[name] = id
	}
	card := str(doc, "card_view")
	detail := str(doc, "detail_view")
	IndexScripts[id] = [2]string{card, detail}
}

func GetIndexByIDOrName(key string) map[string]any {
	mu.RLock()
	defer mu.RUnlock()
	if doc, ok := Indices[key]; ok {
		return doc
	}
	if id, ok := IndexName[key]; ok {
		return Indices[id]
	}
	return nil
}

func PutTemplate(doc map[string]any) {
	id, _ := doc["id"].(string)
	if id == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	Templates[id] = doc
}

func GetTemplate(id string) map[string]any {
	mu.RLock()
	defer mu.RUnlock()
	return Templates[id]
}

func PutVarsList(doc map[string]any) {
	id, _ := doc["id"].(string)
	if id == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	VarsLists[id] = doc
}

func PutTask(doc map[string]any) {
	id, _ := doc["id"].(string)
	if id == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	TaskLists[id] = doc
}

func DeleteTask(id string) {
	mu.Lock()
	defer mu.Unlock()
	delete(TaskLists, id)
}

func RefList(m map[string][]map[string]string, id string) []map[string]string {
	mu.RLock()
	defer mu.RUnlock()
	if list, ok := m[id]; ok {
		out := make([]map[string]string, len(list))
		copy(out, list)
		return out
	}
	return []map[string]string{}
}

func IndexViewScript(id, viewType string) string {
	mu.RLock()
	defer mu.RUnlock()
	pair, ok := IndexScripts[id]
	if !ok {
		return ""
	}
	if viewType == "detail" {
		return pair[1]
	}
	return pair[0]
}

func LoadAll(indices, templates, varsLists, tasks []map[string]any) {
	mu.Lock()
	defer mu.Unlock()
	clearMap(Indices)
	for k := range IndexName {
		delete(IndexName, k)
	}
	clearMap(Templates)
	clearMap(VarsLists)
	clearMap(TaskLists)
	for _, d := range indices {
		id, _ := d["id"].(string)
		Indices[id] = d
		if name := str(d, "name"); name != "" {
			IndexName[name] = id
		}
		card := str(d, "card_view")
		detail := str(d, "detail_view")
		IndexScripts[id] = [2]string{card, detail}
	}
	for _, d := range templates {
		Templates[str(d, "id")] = d
	}
	for _, d := range varsLists {
		VarsLists[str(d, "id")] = d
	}
	for _, d := range tasks {
		TaskLists[str(d, "id")] = d
	}
}

func clearMap(m map[string]map[string]any) {
	for k := range m {
		delete(m, k)
	}
}

func str(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
