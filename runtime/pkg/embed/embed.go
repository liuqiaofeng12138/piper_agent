// Package embed starts Piper Runtime in-process (no piper-runtime subprocess).
package embed

import (
	"context"
	"fmt"
	"path/filepath"

	"piper_go/pkg/agentruntime"
	rtcfg "piper_agent/runtime/internal/config"
	"piper_agent/runtime/internal/mock"
	piperrt "piper_agent/runtime/internal/piper"
	runtimev1 "piper_agent/runtime/pkg/pb/runtime/v1"
)

// Services holds Runtime gRPC servicers for direct in-process calls (no network).
type Services struct {
	Template runtimev1.RuntimeTemplateServer
	Execute  runtimev1.RuntimeExecuteServer
	Meta     runtimev1.RuntimeMetaServer
	Data     runtimev1.RuntimeDataServer
}

// Piper returns the concrete Piper runtime when not in mock mode (for AwaitRun helpers).
func (s *Services) Piper() *piperrt.Runtime {
	if rt, ok := s.Execute.(*piperrt.Runtime); ok {
		return rt
	}
	return nil
}

// Start loads config and bootstraps Runtime inside the current process.
func Start(configPath string) (*Services, error) {
	cfg, err := rtcfg.Load(configPath)
	if err != nil {
		return nil, err
	}
	if cfg.Mock {
		rt := mock.NewRuntime(cfg.MaxConcurrentRuns)
		return &Services{Template: rt, Execute: rt}, nil
	}
	if len(cfg.EngineYAML()) == 0 {
		return nil, fmt.Errorf("engine section required when mock=false")
	}
	configDir := filepath.Dir(cfg.ConfigPath)
	ctx := context.Background()
	svc, err := agentruntime.BootstrapFromYAML(ctx, cfg.EngineYAML(), configDir, agentruntime.Options{
		RelaxDeps:       cfg.RelaxDeps,
		SkipStorageWait: cfg.SkipStorageWait,
	})
	if err != nil {
		return nil, err
	}
	rt := piperrt.NewRuntime(svc, cfg.MaxConcurrentRuns)
	return &Services{
		Template: rt,
		Execute:  rt,
		Meta:     rt,
		Data:     rt,
	}, nil
}
