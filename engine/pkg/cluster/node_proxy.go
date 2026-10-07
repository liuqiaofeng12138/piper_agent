package cluster

import (
	"context"
	"time"

	"piper_go/pkg/cluster/model"
	"piper_go/pkg/distributor"
)

// StartNodeProxyRefresh updates NodeInfo.proxies from agent registry (Java initNodeProxyInfo, 5s).
func StartNodeProxyRefresh(ctx context.Context, getNode func() *model.NodeInfo, setNode func(*model.NodeInfo)) {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n := getNode()
				if n == nil {
					continue
				}
				copy := *n
				copy.Proxies = collectProxyInfos()
				setNode(&copy)
			}
		}
	}()
}

func collectProxyInfos() []any {
	var out []any
	for _, a := range distributor.ListAgents("", "") {
		pid, _ := a["proxy_id"].(string)
		if pid == "" {
			continue
		}
		out = append(out, map[string]any{
			"proxy_id": pid,
			"agent_id": a["id"],
		})
	}
	return out
}
