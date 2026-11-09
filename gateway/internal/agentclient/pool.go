package agentclient

import (
	"context"
	"log"

	"piper_agent/gateway/internal/config"
)

// Pool 维护 agent_id → Worker 的连接池（多子 Agent）。
type Pool struct {
	clients map[string]Worker
}

// NewPool 按注册表为每个 enabled Agent 建立客户端（连接惰性建立）。
func NewPool(agents []config.AgentSpec, overrides map[string]Worker) *Pool {
	p := &Pool{clients: make(map[string]Worker, len(agents))}
	for _, a := range agents {
		if !a.Enabled {
			continue
		}
		if w, ok := overrides[a.ID]; ok && w != nil {
			p.clients[a.ID] = w
			continue
		}
		if a.Address == "" {
			continue
		}
		p.clients[a.ID] = New(a.Address)
	}
	return p
}

// Get 返回指定 Agent 的 Worker；不存在返回 nil。
func (p *Pool) Get(agentID string) Worker {
	if p == nil {
		return nil
	}
	return p.clients[agentID]
}

// Health 探测指定 Agent；未注册返回错误。
func (p *Pool) Health(ctx context.Context, agentID string) error {
	c := p.Get(agentID)
	if c == nil {
		return errNotRegistered(agentID)
	}
	return c.Health(ctx)
}

// Set replaces or registers an in-process worker (e.g. embedded web_crawler).
func (p *Pool) Set(agentID string, w Worker) {
	if p == nil || w == nil {
		return
	}
	if p.clients == nil {
		p.clients = make(map[string]Worker)
	}
	p.clients[agentID] = w
}

// Close 关闭全部连接。
func (p *Pool) Close() {
	if p == nil {
		return
	}
	for id, c := range p.clients {
		log.Printf("[worker] closing client agent=%s", id)
		c.Close()
	}
}

type errNotRegistered string

func (e errNotRegistered) Error() string {
	return "agent not registered: " + string(e)
}
