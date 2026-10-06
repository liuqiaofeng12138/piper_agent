package distributor

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"piper_go/internal/config"
	"piper_go/pkg/chrome"
	"piper_go/pkg/db/es"
	"piper_go/pkg/db/meta"
	"piper_go/pkg/distributor/cache"
	"piper_go/pkg/persistence"
	"piper_go/pkg/piperfunc"
	"piper_go/pkg/tpl"
	"piper_go/pkg/websocket"
)

const (
	ESIndexToken = "token"
	ESIndexLog   = "log"
)

type Engine struct {
	cfg     config.Config
	es      *es.Client
	meta    *meta.Store
	persist *persistence.Persister
	mu      sync.Mutex
	queue   []map[string]any
	stopCh  chan struct{}

	runningTasks map[string]context.CancelFunc
	taskMu       sync.Mutex
	taskRunsMu   sync.Mutex
	taskRuns     map[string]int
}

var (
	defaultEngine *Engine
	engineOnce    sync.Once
)

func Init(cfg config.Config, esClient *es.Client) *Engine {
	engineOnce.Do(func() {
		defaultEngine = &Engine{
			cfg:          cfg,
			es:           esClient,
			queue:        make([]map[string]any, 0),
			stopCh:       make(chan struct{}),
			runningTasks: make(map[string]context.CancelFunc),
			taskRuns:     make(map[string]int),
		}
		go defaultEngine.workerLoop()
	})
	return defaultEngine
}

func (e *Engine) SetMeta(store *meta.Store) {
	if e != nil {
		e.meta = store
	}
}

func (e *Engine) SetPersister(p *persistence.Persister) {
	if e != nil {
		e.persist = p
	}
}

func Default() *Engine {
	return defaultEngine
}

func (e *Engine) Submit(tokens []map[string]any) {
	if e == nil {
		return
	}
	e.mu.Lock()
	e.queue = append(e.queue, tokens...)
	e.mu.Unlock()
	Stats.SetQueue(len(e.queue))
}

func (e *Engine) workerLoop() {
	for {
		select {
		case <-e.stopCh:
			return
		default:
		}
		tok := e.pop()
		if tok == nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		e.execute(tok)
	}
}

func (e *Engine) pop() map[string]any {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.queue) == 0 {
		return nil
	}
	t := e.queue[0]
	e.queue = e.queue[1:]
	Stats.SetQueue(len(e.queue))
	return t
}

func (e *Engine) execute(token map[string]any) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(e.cfg.Requester.TokenTimeout)*time.Millisecond)
	defer cancel()

	id, _ := token["id"].(string)
	if id != "" {
		reqLog := tpl.NewReqLogForToken(id)
		tpl.EnsureReqLog(token, reqLog)
	}

	var childTokens []map[string]any
	success := false
	defer func() {
		token["success"] = success
		token["update_time"] = time.Now().UnixMilli()
		RecordTokenStats(token, success)
		if e.es != nil && id != "" {
			if err := e.es.Index(context.Background(), ESIndexToken, id, token); err != nil {
				log.Printf("persist completed token %s: %v", id, err)
			}
		}
		websocket.BroadcastToken(token)
		if behavior, _ := token["behavior"].(string); behavior == "TEST" {
			websocket.BroadcastAll(map[string]any{
				"type":        "Template",
				"msg":         "Template test finished",
				"create_time": time.Now().UnixMilli(),
				"ref_obj":     token,
			})
		}
		if uid, _ := token["uid"].(string); uid != "" {
			websocket.BroadcastMsg(uid, map[string]any{"type": "token_done", "token_id": id, "success": success})
		}
		if len(childTokens) > 0 {
			for _, child := range childTokens {
				e.RouteToken(child)
			}
		}
		e.taskTokenFinished(token)
		if success && e.persist != nil {
			e.persist.FromToken(context.Background(), token)
		}
	}()

	r, _ := token["r"].(map[string]any)
	if r == nil {
		return
	}
	uri, _ := r["uri"].(string)
	method, _ := r["method"].(string)
	if method == "" {
		method = http.MethodGet
	}
	bodyStr, _ := r["body"].(string)
	var bodyReader io.Reader
	if bodyStr != "" && method != http.MethodGet && method != http.MethodHead {
		bodyReader = strings.NewReader(bodyStr)
	}
	req, err := http.NewRequestWithContext(ctx, method, uri, bodyReader)
	if err != nil {
		tpl.ReqLogDone(token, err)
		return
	}
	if headers, ok := r["headers"].(map[string]any); ok {
		for k, v := range headers {
			req.Header.Set(k, fmt.Sprint(v))
		}
	}
	client := &http.Client{Timeout: time.Duration(e.cfg.Requester.ReadTimeout) * time.Millisecond}
	resp, err := client.Do(req)
	if err != nil {
		r["error"] = err.Error()
		tpl.ReqLogDone(token, err)
		return
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	r["status"] = resp.StatusCode
	r["text"] = string(b)
	httpOK := resp.StatusCode >= 200 && resp.StatusCode < 400
	if !httpOK {
		tpl.ReqLogDone(token, fmt.Errorf("http status %d", resp.StatusCode))
		return
	}

	tplID, _ := token["tpl_id"].(string)
	tplDoc := e.getTemplate(tplID)
	runCtx := e.buildRunContext(&childTokens)
	if tplDoc != nil {
		if err := tpl.RunHTTPProcedures(runCtx, tplDoc, token); err != nil {
			if strings.Contains(err.Error(), "if not satisfy") || strings.Contains(err.Error(), "scheduled retry") {
				tpl.ReqLogDone(token, err)
				return
			}
			tpl.ReqLogDone(token, err)
			return
		}
	}
	tpl.ReqLogDone(token, nil)
	success = true
}

func (e *Engine) getTemplate(id string) map[string]any {
	if doc := cache.GetTemplate(id); doc != nil {
		return doc
	}
	if e.meta != nil && id != "" {
		doc, err := e.meta.Get(meta.TableTemplates, id)
		if err == nil {
			return doc
		}
	}
	return nil
}

func (e *Engine) buildRunContext(childOut *[]map[string]any) *tpl.RunContext {
	return &tpl.RunContext{
		GetTemplate: e.getTemplate,
		QueueToken: func(t map[string]any) {
			*childOut = append(*childOut, t)
		},
		CallFunc: func(funcID string, varNames []string, vars map[string]any) (any, error) {
			if e.meta == nil {
				return nil, fmt.Errorf("meta store not configured")
			}
			return piperfunc.Call(e.meta, funcID, vars)
		},
		ReadTimeout: int(e.cfg.Requester.ReadTimeout),
	}
}

// BuildRunContext exposes template run dependencies for Chrome agents.
func (e *Engine) BuildRunContext(childOut *[]map[string]any) *tpl.RunContext {
	return e.buildRunContext(childOut)
}

// GetTemplate resolves template JSON from cache or meta store.
func (e *Engine) GetTemplate(id string) map[string]any {
	return e.getTemplate(id)
}

// RouteToken submits child tokens to HTTP or Chrome queue (Java distributor routing).
func (e *Engine) RouteToken(t map[string]any) {
	e.taskTokenQueued(t)
	typ, _ := t["type"].(string)
	if typ == "Chrome" {
		if cd := chrome.Default(); cd != nil && cd.AgentCount() > 0 {
			_ = cd.Submit(t)
			return
		}
	}
	e.Submit([]map[string]any{t})
}

func (e *Engine) RunTemplate(tplDoc map[string]any, vars map[string]any, opts tpl.RunOpts) (tokenID, agentID string, err error) {
	btype := tpl.BuilderType(tplDoc)
	switch btype {
	case "Http", "":
		if opts.AgentID == "" {
			opts.AgentID = "http-default"
		}
		token, err := tpl.BuildHTTPToken(tplDoc, vars, opts)
		if err != nil {
			return "", "", err
		}
		id, _ := token["id"].(string)
		// Make the token queryable before returning its ID to the caller. The
		// frontend polls GET /tokens/:id immediately after this response; if we
		// only index at the end of execution, long-running tokens look missing.
		if e.es == nil {
			return "", "", fmt.Errorf("token storage is not configured")
		}
		if err := e.es.Index(context.Background(), ESIndexToken, id, token); err != nil {
			return "", "", fmt.Errorf("persist queued token %s: %w", id, err)
		}
		e.Submit([]map[string]any{token})
		return id, opts.AgentID, nil
	case "Chrome":
		cd := chrome.Default()
		if cd == nil || cd.AgentCount() == 0 {
			return "", "", fmt.Errorf("ChromeDistributor no Agent")
		}
		opts.AgentID = cd.PickAgentID(opts.AgentID)
		token, err := tpl.BuildChromeToken(tplDoc, vars, opts)
		if err != nil {
			return "", "", err
		}
		token["agent_id"] = opts.AgentID
		id, _ := token["id"].(string)
		if e.es == nil {
			return "", "", fmt.Errorf("token storage is not configured")
		}
		if err := e.es.Index(context.Background(), ESIndexToken, id, token); err != nil {
			return "", "", fmt.Errorf("persist queued token %s: %w", id, err)
		}
		if err := cd.Submit(token); err != nil {
			return "", "", err
		}
		return id, opts.AgentID, nil
	default:
		return "", "", fmt.Errorf("Error Builder type")
	}
}

func (e *Engine) RunRawToken(token map[string]any) error {
	if token["id"] == nil || token["id"] == "" {
		now := time.Now().UnixMilli()
		token["create_time"] = now
		token["update_time"] = now
		tpl.AssignTokenID(token)
	}
	if typ, _ := token["type"].(string); typ == "Chrome" {
		if cd := chrome.Default(); cd != nil && cd.AgentCount() > 0 {
			return cd.Submit(token)
		}
		return fmt.Errorf("ChromeDistributor no Agent")
	}
	e.Submit([]map[string]any{token})
	return nil
}

func (e *Engine) RegisterRunningTask(taskID string, cancel context.CancelFunc) {
	e.taskMu.Lock()
	defer e.taskMu.Unlock()
	if old, ok := e.runningTasks[taskID]; ok {
		old()
	}
	e.runningTasks[taskID] = cancel
}

func (e *Engine) StopTask(taskID string) {
	e.taskMu.Lock()
	if cancel, ok := e.runningTasks[taskID]; ok {
		cancel()
		delete(e.runningTasks, taskID)
	}
	e.taskMu.Unlock()
}
