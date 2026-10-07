package distributor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	agentsH2Path       string
	agentsPersistSkip  int32
)

// SetAgentsH2Path sets the H2 database path used to locate db/agents_info.json (Java InitUtil.agents_info_path).
func SetAgentsH2Path(h2Path string) {
	agentsH2Path = h2Path
}

// AgentsInfoFile returns the agents registry snapshot path next to the H2 meta DB.
func AgentsInfoFile(h2Path string) string {
	return filepath.Join(filepath.Dir(h2Path), "agents_info.json")
}

func suppressAgentsPersist(fn func()) {
	atomic.AddInt32(&agentsPersistSkip, 1)
	defer atomic.AddInt32(&agentsPersistSkip, -1)
	fn()
}

func maybePersistAgents() {
	if agentsH2Path == "" || atomic.LoadInt32(&agentsPersistSkip) > 0 {
		return
	}
	if err := SaveAgentsRegistry(agentsH2Path); err != nil {
		logx.Errorf("persist agents_info.json: %v", err)
	}
}

// SaveAgentsRegistry writes all registered agents to agents_info.json (Java WebAPI shutdown / MiscRoute.restart).
func SaveAgentsRegistry(h2Path string) error {
	list := AllAgentsJSON()
	b, err := json.Marshal(list)
	if err != nil {
		return err
	}
	path := AgentsInfoFile(h2Path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// RestoreAgentsRegistry loads HttpAgent snapshots from disk. ChromeAgent rows are skipped; live Chrome workers re-register at bootstrap.
func RestoreAgentsRegistry(h2Path string) {
	path := AgentsInfoFile(h2Path)
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var list []map[string]any
	if err := json.Unmarshal(b, &list); err != nil {
		logx.Errorf("restore agents_info.json: %v", err)
		return
	}
	suppressAgentsPersist(func() {
		for _, a := range list {
			if st, _ := a["status"].(string); st == "Broken" {
				continue
			}
			typ, _ := a["type"].(string)
			if typ == "ChromeAgent" || typ == "Chrome" {
				continue
			}
			RegisterAgent(a)
		}
	})
}
