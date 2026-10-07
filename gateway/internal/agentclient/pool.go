package agentclient

import (
	"context"
	"log"

	"piper_agent/gateway/internal/config"
)

// Pool 维护 agent_id → WorkerClient 的连接池（多子 Agent）。
type Pool struct {
	clients map[string]*WorkerClient
}

// NewPool 按注册表为每个 enabled Agent 建立客户端（连接惰性建立）。
func NewPool(agents []config.AgentSpec) *Pool {
	p := &Pool{clients: make(map[string]*WorkerClient, len(agents))}
	for _, a := range agents {
		if !a.Enabled || a.Address == "" {
			continue
		}
		p.clients[a.ID] = New(a.Address)
	}
	return p
}

// Get 返回指定 Agent 的客户端；不存在返回 nil。
func (p *Pool) Get(agentID string) *WorkerClient {
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
