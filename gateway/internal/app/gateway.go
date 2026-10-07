package app

import (
	"log"
	"net/http"

	"piper_agent/gateway/internal/agentclient"
	"piper_agent/gateway/internal/api"
	"piper_agent/gateway/internal/config"
	"piper_agent/gateway/internal/router"
	"piper_agent/gateway/internal/run"
	"piper_agent/gateway/internal/session"
)

// StartHTTP 启动 API 网关（阻塞）。
func StartHTTP(cfg *config.Config, listenOverride string) error {
	addr := cfg.Listen
	if listenOverride != "" {
		addr = listenOverride
	}

	var store session.Store
	switch cfg.Session.Driver {
	case "memory":
		store = session.NewMemoryStore()
	default:
		dbPath := session.ResolveSQLitePath(cfg.ConfigPath, cfg.Session.SQLitePath)
		sqliteStore, err := session.OpenSQLite(dbPath)
		if err != nil {
			return err
		}
		store = sqliteStore
		log.Printf("session store: sqlite %s", dbPath)
	}

	pool := agentclient.NewPool(cfg.Agents)
	defer pool.Close()
	deps := &api.Deps{
		Store:   store,
		Config:  cfg,
		Router:  router.New(cfg),
		Workers: pool,
		Runs:    run.NewRegistry(),
	}
	handler := api.NewHandler(deps, bearerToken(cfg))

	log.Printf("piper-gateway listening on %s (agents=%d, classifier=%s)", addr, len(cfg.EnabledAgents()), cfg.Classifier.Mode)
	return http.ListenAndServe(addr, handler)
}

func bearerToken(cfg *config.Config) string {
	if cfg.Auth.Mode != "bearer" {
		return ""
	}
	return cfg.Auth.Token
}
