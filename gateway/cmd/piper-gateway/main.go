package main

import (
	"flag"
	"log"

	"path/filepath"

	"piper_agent/gateway/internal/app"
	"piper_agent/gateway/internal/bootstrap"
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

	abs, _ := filepath.Abs(*configPath)
	repoRoot := bootstrap.RepoRoot(abs)
	if err := app.StartHTTP(cfg, *listen, nil, abs, repoRoot); err != nil {
		log.Fatal(err)
	}
}
