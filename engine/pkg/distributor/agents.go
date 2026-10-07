package distributor

import (
	"errors"
	"sort"
	"sync"
	"time"
)

var (
	agentsMu sync.RWMutex
	agents   = map[string]map[string]any{}
)

// RegisterAgent stores agent snapshot for /agents API (Java Distributor registry subset).
func RegisterAgent(doc map[string]any) {
	id, _ := doc["id"].(string)
	if id == "" {
		return
	}
	agentsMu.Lock()
	enriched := enrichAgent(cloneMapAgent(doc), false)
	agents[id] = enriched
	agentsMu.Unlock()
	if name, _ := enriched["name"].(string); name != "" || id != "" {
		Stats.EnsureAgentMetrics(id, name)
	}
	maybePersistAgents()
}

func UnregisterAgent(id string) {
	agentsMu.Lock()
	delete(agents, id)
	agentsMu.Unlock()
	maybePersistAgents()
}

func GetAgentByID(id string) (map[string]any, error) {
	agentsMu.RLock()
	defer agentsMu.RUnlock()
	a, ok := agents[id]
	if !ok {
		return nil, errors.New("Agent not found")
	}
	return enrichAgent(cloneMapAgent(a), true), nil
}

func ListAgents(status, domain string) []map[string]any {
	agentsMu.RLock()
	defer agentsMu.RUnlock()
	var out []map[string]any
	for _, a := range agents {
		if status != "" {
			if s, _ := a["status"].(string); s != status {
				continue
			}
		}
		if domain != "" {
			accts, _ := a["accounts"].(map[string]any)
			if accts == nil {
				if m, ok := a["accounts"].(map[string]string); ok {
					if _, ok := m[domain]; !ok {
						continue
					}
				} else {
					continue
				}
			} else if accts[domain] == nil {
				continue
			}
		}
		out = append(out, enrichAgent(cloneMapAgent(a), true))
	}
	sort.Slice(out, func(i, j int) bool {
		n1, _ := out[i]["name"].(string)
		n2, _ := out[j]["name"].(string)
		return n1 < n2
	})
	return out
}

func AllAgentsJSON() []map[string]any {
	return ListAgents("", "")
}

func cloneMapAgent(m map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

func agentInt64(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case float32:
		return int64(t)
	default:
		return 0
	}
}

// NewHttpAgentDoc builds a UI-compatible HttpAgent snapshot.
func NewHttpAgentDoc(instID, name string) map[string]any {
	now := time.Now().UnixMilli()
	return map[string]any{
		"id":                 instID + "-" + name,
		"type":               "HttpAgent",
		"name":               name,
		"status":             "Idle",
		"accounts":           map[string]any{},
		"proxy_id":           "",
		"proxy_local":        "",
		"init_time":          now,
		"duration":           int64(0),
		"public_queue_size":  0,
		"local_queue_size":   0,
		"vnc_addr":           "",
	}
}

func enrichAgent(doc map[string]any, refreshDuration bool) map[string]any {
	now := time.Now().UnixMilli()
	if doc["init_time"] == nil {
		doc["init_time"] = now
	}
	initMs := agentInt64(doc["init_time"])
	if refreshDuration || doc["duration"] == nil {
		dur := now - initMs
		if dur < 0 {
			dur = 0
		}
		doc["duration"] = dur
	}
	if doc["public_queue_size"] == nil {
		if qs := agentInt64(doc["queue_size"]); qs > 0 {
			doc["public_queue_size"] = qs
		} else {
			doc["public_queue_size"] = 0
		}
	}
	if doc["local_queue_size"] == nil {
		doc["local_queue_size"] = 0
	}
	if doc["proxy_local"] == nil {
		doc["proxy_local"] = ""
	}
	if doc["accounts"] == nil {
		doc["accounts"] = map[string]any{}
	}
	if doc["vnc_addr"] == nil {
		doc["vnc_addr"] = ""
	}
	return doc
}
