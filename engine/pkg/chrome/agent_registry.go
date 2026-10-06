package chrome

import "time"

// Snapshot returns agent document for /agents API.
func (a *Agent) Snapshot() map[string]any {
	now := time.Now().UnixMilli()
	qs := len(a.queue)
	return map[string]any{
		"id":                a.ID,
		"type":              "ChromeAgent",
		"name":              a.Name,
		"status":            "Idle",
		"proxy_id":          "",
		"proxy_local":       "",
		"accounts":          map[string]any{},
		"queue_size":        qs,
		"public_queue_size": qs,
		"local_queue_size":  0,
		"init_time":         now,
		"duration":          int64(0),
		"vnc_addr":          "",
	}
}
