package app

import (
	"piper_agent/gateway/internal/agentclient"
)

// ServeOptions optional in-process integrations (embedded Runtime / web_crawler).
type ServeOptions struct {
	InprocWorkers map[string]agentclient.Worker
}
