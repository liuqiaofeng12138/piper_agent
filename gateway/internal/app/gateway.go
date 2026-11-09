package app

import (
	"log"
	"net/http"

	"piper_agent/gateway/internal/agentclient"
	"piper_agent/gateway/internal/api"
	"piper_agent/gateway/internal/auth"
	"piper_agent/gateway/internal/artifacts"
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
	artifactsDir, err := artifacts.ResolveDir(cfg.ConfigPath, cfg.ArtifactsDir)
	if err != nil {
		return err
	}
	authSvc, err := auth.New(cfg.Auth)
	if err != nil {
		return err
	}
	deps := &api.Deps{
		Store:        store,
		Config:       cfg,
		Router:       router.New(cfg),
		Workers:      pool,
		Runs:         run.NewRegistry(),
		ArtifactsDir: artifactsDir,
		Auth:         authSvc,
	}
	handler := api.NewHandler(deps)

	log.Printf("piper-gateway listening on %s (agents=%d, classifier=%s, auth=%s)", addr, len(cfg.EnabledAgents()), cfg.Classifier.Mode, cfg.Auth.Mode)
	return http.ListenAndServe(addr, handler)
}
