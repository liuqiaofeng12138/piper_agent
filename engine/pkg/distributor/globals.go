package distributor

import "sync"

// Vars mirrors Distributor.vars (global template vars).
var (
	varsMu sync.RWMutex
	Vars   = map[string]any{}
)

func GetGlobalVars() map[string]any {
	varsMu.RLock()
	defer varsMu.RUnlock()
	out := make(map[string]any, len(Vars))
	for k, v := range Vars {
		out[k] = v
	}
	return out
}
