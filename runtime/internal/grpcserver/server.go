package grpcserver

import (
	"context"
	"log"
	"net"

	"piper_agent/runtime/internal/config"
	"piper_go/pkg/agentruntime"
	"piper_agent/runtime/internal/mock"
	piperrt "piper_agent/runtime/internal/piper"

	runtimev1 "piper_agent/runtime/pkg/pb/runtime/v1"

	"google.golang.org/grpc"
)

func ListenAndServe(cfg *config.Config) error {
	lis, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		return err
	}
	srv := grpc.NewServer()

	if cfg.Mock {
		mockRT := mock.NewRuntime(cfg.MaxConcurrentRuns)
		runtimev1.RegisterRuntimeTemplateServer(srv, mockRT)
		runtimev1.RegisterRuntimeExecuteServer(srv, mockRT)
		log.Printf("piper-runtime (Phase A mock) listening on %s", cfg.Listen)
		return srv.Serve(lis)
	}

	if cfg.PiperGoConfig == "" {
		log.Fatal("piper_go_config is required when mock=false")
	}
	ctx := context.Background()
	svc, err := agentruntime.Bootstrap(ctx, cfg.PiperGoConfig, agentruntime.Options{
		RelaxDeps:       cfg.RelaxDeps,
		SkipStorageWait: cfg.SkipStorageWait,
	})
	if err != nil {
		return err
	}
	rt := piperrt.NewRuntime(svc, cfg.MaxConcurrentRuns)
	runtimev1.RegisterRuntimeTemplateServer(srv, rt)
	runtimev1.RegisterRuntimeExecuteServer(srv, rt)
	runtimev1.RegisterRuntimeMetaServer(srv, rt)
	runtimev1.RegisterRuntimeDataServer(srv, rt)
	log.Printf("piper-runtime (Phase B piper_go) listening on %s", cfg.Listen)
	return srv.Serve(lis)
}
