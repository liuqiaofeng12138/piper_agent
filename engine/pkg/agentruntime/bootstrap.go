package agentruntime

import (
	"context"
	"path/filepath"

	"piper_go/internal/bootstrap"
	"piper_go/internal/config"
	"piper_go/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
)

type Options struct {
	RelaxDeps       bool
	SkipStorageWait bool
}

// Bootstrap initializes meta, ES, distributor, and optional Chrome (same as pipergo API).
func Bootstrap(ctx context.Context, configPath string, opt Options) (*Service, error) {
	var c config.Config
	if err := conf.Load(configPath, &c); err != nil {
		return nil, err
	}
	cfgDir := filepath.Dir(configPath)
	if !filepath.IsAbs(c.H2.Path) {
		c.H2.Path = filepath.Join(cfgDir, c.H2.Path)
	}
	if opt.RelaxDeps {
		c.WebAPI.RequireDeps = false
	}
	if opt.SkipStorageWait {
		c.WebAPI.SkipStorageWait = true
	}
	svcCtx := svc.NewServiceContext(c)
	if err := bootstrap.Run(ctx, svcCtx); err != nil {
		return nil, err
	}
	svc := &Service{ctx: svcCtx}
	svc.EnsureHTTPAgent()
	return svc, nil
}

type Service struct {
	ctx *svc.ServiceContext
}

func (s *Service) Svc() *svc.ServiceContext {
	return s.ctx
}
