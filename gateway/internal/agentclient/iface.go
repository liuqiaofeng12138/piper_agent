package agentclient

import (
	"context"

	agentv1 "piper_agent/gateway/pkg/pb/agent/v1"
)

// Worker is a sub-agent execute backend (remote gRPC or in-process).
type Worker interface {
	Health(ctx context.Context) error
	Execute(ctx context.Context, req *agentv1.ExecuteRequest, onEvent func(*agentv1.ExecuteEvent) error) error
	CancelRun(ctx context.Context, runID string) error
	Close()
}
