package agentruntime

import (
	"context"

	"piper_go/pkg/distributor"
	"piper_go/pkg/persistence"
	"piper_go/pkg/tpl"
)

func (s *Service) RunTemplate(ctx context.Context, templateID string, vars map[string]any, engine, sessionID, proxyID string) (tokenID, agentID string, err error) {
	tplDoc, err := s.GetTemplate(ctx, templateID)
	if err != nil {
		return "", "", err
	}
	btype := tpl.BuilderType(tplDoc)
	if engine == "chrome" || btype == "Chrome" {
		if builder, ok := tplDoc["builder"].(map[string]any); ok {
			builder["type"] = "Chrome"
		}
		engine = "chrome"
	}
	nodeID := ""
	if n := s.ctx.Node(); n != nil {
		nodeID = n.InstID
	}
	opts := tpl.RunOpts{
		UID:      sessionID,
		NodeID:   nodeID,
		Behavior: "DEFAULT",
		ProxyID:  proxyID,
	}
	if engine != "chrome" {
		s.EnsureHTTPAgent()
		if proxyID != "" {
			if err := s.BindProxyToHTTPAgent(proxyID); err != nil {
				return "", "", err
			}
		}
		opts.AgentID = s.HTTAgentID()
	}
	return s.ctx.Dist.RunTemplate(tplDoc, vars, opts)
}

func (s *Service) GetToken(ctx context.Context, id string) (map[string]any, error) {
	return s.ctx.ES.Get(ctx, distributor.ESIndexToken, id)
}

func (s *Service) GetTokenData(ctx context.Context, tokenID string) (map[string]any, error) {
	doc, err := s.ctx.ES.Get(ctx, distributor.ESIndexToken, tokenID)
	if err != nil {
		return nil, err
	}
	fetch := func(tid string) map[string]any {
		d, err := s.ctx.ES.Get(ctx, distributor.ESIndexToken, tid)
		if err != nil {
			return nil
		}
		return d
	}
	return persistence.TokenData(doc, fetch), nil
}
