package agentruntime

import (
	"context"
	"fmt"

	"piper_go/pkg/db/meta"
	"piper_go/pkg/distributor"
)

type ProxySummary struct {
	ID     string
	Name   string
	Domain string
	Status string
}

func (s *Service) ListProxies(_ context.Context, status string, page, size int) ([]ProxySummary, int64, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 50
	}
	items, total, err := s.ctx.Meta.Query(meta.TableProxies, meta.QueryOpts{
		Status: status,
		Page:   int64(page),
		Size:   int64(size),
	})
	if err != nil {
		return nil, 0, err
	}
	out := make([]ProxySummary, 0, len(items))
	for _, it := range items {
		out = append(out, ProxySummary{
			ID:     strVal(it, "id"),
			Name:   strVal(it, "name"),
			Domain: strVal(it, "domain"),
			Status: strVal(it, "status"),
		})
	}
	return out, total, nil
}

func strVal(m map[string]any, k string) string {
	v, _ := m[k].(string)
	return v
}

// BindProxyToHTTPAgent sets proxy on the runtime HttpAgent (Java AgentSetProxy).
func (s *Service) BindProxyToHTTPAgent(proxyID string) error {
	s.EnsureHTTPAgent()
	agentID := s.HTTAgentID()
	if proxyID != "" {
		if _, err := s.ctx.Meta.Get(meta.TableProxies, proxyID); err != nil {
			return fmt.Errorf("proxy %s: %w", proxyID, err)
		}
	}
	a, err := distributor.GetAgentByID(agentID)
	if err != nil {
		return err
	}
	a["proxy_id"] = proxyID
	distributor.RegisterAgent(a)
	return nil
}
