package main

import (
	"flag"
	"log"
	"net/http"

	"piper_agent/gateway/internal/agentclient"
	"piper_agent/gateway/internal/api"
	"piper_agent/gateway/internal/config"
	"piper_agent/gateway/internal/router"
	"piper_agent/gateway/internal/run"
	"piper_agent/gateway/internal/session"
)

func main() {
	configPath := flag.String("f", config.DefaultConfigPath(), "path to unified config (deploy/config/local.yaml)")
	listen := flag.String("listen", "", "override gateway listen address (e.g. :8080)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	addr := cfg.Listen
	if *listen != "" {
		addr = *listen
	}

	var store session.Store
	var sqliteCloser func()
	switch cfg.Session.Driver {
	case "memory":
		store = session.NewMemoryStore()
	default:
		dbPath := session.ResolveSQLitePath(cfg.ConfigPath, cfg.Session.SQLitePath)
		sqliteStore, err := session.OpenSQLite(dbPath)
		if err != nil {
			log.Fatalf("sqlite session: %v", err)
		}
		store = sqliteStore
		sqliteCloser = func() { _ = sqliteStore.Close() }
		log.Printf("session store: sqlite %s", dbPath)
	}

	worker := agentclient.New(cfg.Agents.WebCrawler.Address)
	deps := &api.Deps{
		Store:  store,
		Config: cfg,
		Router: router.New(cfg.Classifier.Mode),
		Worker: worker,
		Runs:   run.NewRegistry(),
	}
	handler := api.NewHandler(deps, bearerToken(cfg))

	log.Printf("piper-gateway listening on %s (config=%s, worker=%s)", addr, cfg.ConfigPath, cfg.Agents.WebCrawler.Address)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
	if sqliteCloser != nil {
		sqliteCloser()
	}
}

func bearerToken(cfg *config.Config) string {
	if cfg.Auth.Mode != "bearer" {
		return ""
	}
	return cfg.Auth.Token
}
