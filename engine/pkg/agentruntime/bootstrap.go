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

// BootstrapFromYAML loads engine settings from a YAML document (the `engine:` section body).
func BootstrapFromYAML(ctx context.Context, engineYAML []byte, configBaseDir string, opt Options) (*Service, error) {
	var c config.Config
	if err := conf.LoadFromYamlBytes(engineYAML, &c); err != nil {
		return nil, err
	}
	return bootstrapWithConfig(ctx, c, configBaseDir, opt)
}

func bootstrapWithConfig(ctx context.Context, c config.Config, configBaseDir string, opt Options) (*Service, error) {
	if c.H2.Path != "" && !filepath.IsAbs(c.H2.Path) {
		c.H2.Path = filepath.Join(configBaseDir, c.H2.Path)
	}
	if c.Chrome.UserDataDir != "" && !filepath.IsAbs(c.Chrome.UserDataDir) {
		c.Chrome.UserDataDir = filepath.Join(configBaseDir, c.Chrome.UserDataDir)
	}
	if c.Chrome.Enabled && !c.Chrome.Headless && c.Chrome.ManualLoginWaitSeconds < 0 {
		c.Chrome.ManualLoginWaitSeconds = 120
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
