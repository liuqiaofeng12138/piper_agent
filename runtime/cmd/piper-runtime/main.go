package main

import (
	"flag"
	"log"

	"piper_agent/runtime/internal/config"
	"piper_agent/runtime/internal/grpcserver"
)

func main() {
	configPath := flag.String("f", config.DefaultConfigPath(), "path to unified config (deploy/config/local.yaml)")
	listen := flag.String("listen", "", "override gRPC listen address (e.g. :50051)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if *listen != "" {
		cfg.Listen = *listen
	}
	if err := grpcserver.ListenAndServe(cfg); err != nil {
		log.Fatal(err)
	}
}
