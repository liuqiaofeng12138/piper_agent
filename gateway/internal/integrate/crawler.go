package integrate

import (
	"fmt"

	"piper_agent/gateway/internal/agentclient"
	"piper_agent/gateway/internal/config"
	"piper_agent/gateway/internal/crawlerworker"
	"piper_agent/runtime/pkg/embed"
)

// InprocWorkers starts embedded Piper Runtime + Go web_crawler when enabled in agents registry.
func InprocWorkers(cfg *config.Config, configPath, repoRoot string) (map[string]agentclient.Worker, error) {
	if !cfg.WebCrawlerEnabled() {
		return nil, nil
	}
	rt, err := embed.Start(configPath)
	if err != nil {
		return nil, fmt.Errorf("runtime embed: %w", err)
	}
	w, err := crawlerworker.New(configPath, repoRoot, rt)
	if err != nil {
		return nil, fmt.Errorf("web_crawler worker: %w", err)
	}
	return map[string]agentclient.Worker{
		"web_crawler": w,
	}, nil
}
