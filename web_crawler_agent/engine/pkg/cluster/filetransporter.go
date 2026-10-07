package cluster

import (
	"encoding/json"
	"sync"
	"time"
)

// FileTransporter config + log ring (Java FileTransporter subset; no FTP/MySQL worker in Phase 5 baseline).
type FileTransporter struct {
	mu     sync.RWMutex
	config map[string]any
	logs   []map[string]any
}

var defaultTransporter = &FileTransporter{
	config: map[string]any{
		"enabled":           false,
		"interval":          3600000,
		"reversed_interval": 86400000,
	},
}

func DefaultFileTransporter() *FileTransporter {
	return defaultTransporter
}

func (f *FileTransporter) Config() map[string]any {
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := map[string]any{}
	for k, v := range f.config {
		out[k] = v
	}
	return out
}

func (f *FileTransporter) SetConfig(body []byte) error {
	var patch map[string]any
	if err := json.Unmarshal(body, &patch); err != nil {
		return err
	}
	f.mu.Lock()
	for k, v := range patch {
		f.config[k] = v
	}
	f.mu.Unlock()
	f.appendLog("config updated")
	return nil
}

func (f *FileTransporter) Logs() []map[string]any {
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := make([]map[string]any, len(f.logs))
	copy(out, f.logs)
	return out
}

func (f *FileTransporter) appendLog(msg string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	entry := map[string]any{
		"msg":         msg,
		"update_time": time.Now().UnixMilli(),
	}
	f.logs = append(f.logs, entry)
	if len(f.logs) > 100 {
		f.logs = f.logs[len(f.logs)-100:]
	}
}
