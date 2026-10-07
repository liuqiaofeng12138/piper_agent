package main

import (
	"flag"
	"log"

	"piper_agent/gateway/internal/bootstrap"
	"piper_agent/gateway/internal/config"
)

func main() {
	configPath := flag.String("f", config.DefaultConfigPath(), "path to unified config (deploy/config/local.yaml)")
	listen := flag.String("listen", "", "override gateway HTTP listen (e.g. :8080)")
	flag.Parse()

	if err := bootstrap.RunAll(*configPath, *listen); err != nil {
		log.Fatal(err)
	}
}
