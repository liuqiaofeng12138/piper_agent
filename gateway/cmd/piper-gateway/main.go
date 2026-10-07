package main

import (
	"flag"
	"log"

	"piper_agent/gateway/internal/app"
	"piper_agent/gateway/internal/config"
)

func main() {
	configPath := flag.String("f", config.DefaultConfigPath(), "path to unified config (deploy/config/local.yaml)")
	listen := flag.String("listen", "", "override gateway listen address (e.g. :8080)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if err := app.StartHTTP(cfg, *listen); err != nil {
		log.Fatal(err)
	}
}
