package chrome

import (
	"fmt"
	"sync"

	"piper_go/internal/config"
	"piper_go/pkg/db/es"
	"piper_go/pkg/persistence"
	"piper_go/pkg/tpl"
)

// Distributor routes Chrome tokens to agents (Java ChromeDistributor subset).
type Distributor struct {
	cfg    config.Config
	es     *es.Client
	mu     sync.RWMutex
	agents []*Agent

	getTemplate func(string) map[string]any
	buildRunCtx func(*[]map[string]any) *tpl.RunContext
	routeChild  func(map[string]any)
	taskDone    func(map[string]any)
	persist     *persistence.Persister
}

var (
	defaultDist *Distributor
	distOnce    sync.Once
)

func Init(cfg config.Config, esClient *es.Client) *Distributor {
	distOnce.Do(func() {
		defaultDist = &Distributor{cfg: cfg, es: esClient}
	})
	return defaultDist
}

func Default() *Distributor {
	return defaultDist
}

func (d *Distributor) SetHooks(getTpl func(string) map[string]any, runCtx func(*[]map[string]any) *tpl.RunContext, route func(map[string]any), taskDone func(map[string]any)) {
	d.getTemplate = getTpl
	d.buildRunCtx = runCtx
	d.routeChild = route
	d.taskDone = taskDone
}

func (d *Distributor) SetPersister(p *persistence.Persister) {
	d.persist = p
	d.mu.RLock()
	defer d.mu.RUnlock()
	for _, a := range d.agents {
		a.SetPersister(p)
	}
}

func (d *Distributor) StartAgents(instID string, count int) {
	if count <= 0 {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("CA-%d", i+1)
		a := NewAgent(name, instID, d.cfg, d.es)
		a.SetHooks(d.getTemplate, d.buildRunCtx, d.routeChild, d.taskDone)
		a.SetPersister(d.persist)
		d.agents = append(d.agents, a)
		go a.RunLoop()
	}
}

func (d *Distributor) PickAgentID(preferred string) string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if preferred != "" {
		for _, a := range d.agents {
			if a.ID == preferred || a.Name == preferred {
				return a.ID
			}
		}
	}
	if len(d.agents) == 0 {
		return ""
	}
	return d.agents[0].ID
}

func (d *Distributor) Submit(token map[string]any) error {
	agentID, _ := token["agent_id"].(string)
	d.mu.RLock()
	var agent *Agent
	for _, a := range d.agents {
		if a.ID == agentID {
			agent = a
			break
		}
	}
	if agent == nil && len(d.agents) > 0 {
		agent = d.agents[0]
	}
	d.mu.RUnlock()
	if agent == nil {
		return fmt.Errorf("ChromeDistributor no Agent")
	}
	agent.Submit(token)
	return nil
}

func (d *Distributor) AgentCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.agents)
}

// AddAgent starts one additional Chrome agent (Java AgentRoute.create).
func (d *Distributor) AddAgent(instID string) *Agent {
	d.mu.Lock()
	defer d.mu.Unlock()
	name := fmt.Sprintf("CA-%d", len(d.agents)+1)
	a := NewAgent(name, instID, d.cfg, d.es)
	a.SetHooks(d.getTemplate, d.buildRunCtx, d.routeChild, d.taskDone)
	a.SetPersister(d.persist)
	d.agents = append(d.agents, a)
	go a.RunLoop()
	return a
}

func (d *Distributor) RemoveAgent(id string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i, a := range d.agents {
		if a.ID == id {
			a.Close()
			close(a.queue)
			d.agents = append(d.agents[:i], d.agents[i+1:]...)
			return true
		}
	}
	return false
}

func (d *Distributor) EachAgent(fn func(*Agent)) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	for _, a := range d.agents {
		fn(a)
	}
}
