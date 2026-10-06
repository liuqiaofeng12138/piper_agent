package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"piper_go/internal/bootstrap"
	"piper_go/internal/config"
	"piper_go/internal/handler"
	"piper_go/internal/middleware"
	"piper_go/internal/svc"
	"piper_go/pkg/distributor"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/pipergo-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	logx.Infof("config %s: noAuth=%v requireDeps=%v prometheus=%s port=%d", *configFile, c.WebAPI.NoAuth, c.WebAPI.RequireDeps, c.WebAPI.PrometheusHost, c.Port)

	// WithCors: OPTIONS preflight for cross-origin dev (8088 → 8888); without this, POST /auth/login gets 405.
	server := rest.MustNewServer(c.RestConf, rest.WithCors())
	defer server.Stop()

	svcCtx := svc.NewServiceContext(c)
	middleware.Register(server, svcCtx)
	handler.RegisterHandlers(server, svcCtx)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := bootstrap.Run(ctx, svcCtx); err != nil {
		logx.Errorf("bootstrap failed: %v", err)
		os.Exit(1)
	}

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	go func() {
		server.Start()
	}()

	<-ctx.Done()
	logx.Info("shutdown signal received, stopping HTTP server...")
	if err := distributor.SaveAgentsRegistry(c.H2.Path); err != nil {
		logx.Errorf("save agents_info.json on shutdown: %v", err)
	}
	server.Stop()
}
