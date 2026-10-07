package chrome

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"piper_go/internal/config"
	"piper_go/pkg/db/es"
	"piper_go/pkg/persistence"
	"piper_go/pkg/tpl"
	"piper_go/pkg/util"
	"piper_go/pkg/websocket"
)

// Agent is a long-lived Chrome worker (Java ChromeAgent subset).
type Agent struct {
	ID     string
	Name   string
	cfg    config.Config
	es     *es.Client
	queue  chan map[string]any
	alloc  context.Context
	cancel context.CancelFunc
	mu     sync.Mutex

	buildRunCtx func(*[]map[string]any) *tpl.RunContext
	getTemplate func(string) map[string]any
	routeChild  func(map[string]any)
	taskDone    func(map[string]any)
	persist     *persistence.Persister
}

func NewAgent(name, instID string, cfg config.Config, esClient *es.Client) *Agent {
	id := util.MD5Hex(instID + "::" + name)
	opts := chromedpAllocatorOptions(cfg)
	alloc, cancel := chromedpNewExecAllocator(context.Background(), opts...)
	return &Agent{
		ID:     id,
		Name:   name,
		cfg:    cfg,
		es:     esClient,
		queue:  make(chan map[string]any, 256),
		alloc:  alloc,
		cancel: cancel,
	}
}

func (a *Agent) SetHooks(getTpl func(string) map[string]any, runCtx func(*[]map[string]any) *tpl.RunContext, route func(map[string]any), taskDone func(map[string]any)) {
	a.getTemplate = getTpl
	a.buildRunCtx = runCtx
	a.routeChild = route
	a.taskDone = taskDone
}

func (a *Agent) SetPersister(p *persistence.Persister) {
	a.persist = p
}

func (a *Agent) Submit(token map[string]any) {
	a.queue <- token
}

func (a *Agent) RunLoop() {
	for token := range a.queue {
		a.executeToken(token)
	}
}

func (a *Agent) Close() {
	if a.cancel != nil {
		a.cancel()
	}
}

func (a *Agent) executeToken(token map[string]any) {
	id, _ := token["id"].(string)
	if id != "" {
		tpl.EnsureReqLog(token, tpl.NewReqLogForToken(id))
	}
	var childTokens []map[string]any
	success := false
	defer func() {
		token["success"] = success
		token["update_time"] = time.Now().UnixMilli()
		if a.es != nil && id != "" {
			if err := a.es.Index(context.Background(), "token", id, token); err != nil {
				fmt.Printf("persist completed token %s: %v\n", id, err)
			}
		}
		websocket.BroadcastToken(token)
		if uid, _ := token["uid"].(string); uid != "" {
			websocket.BroadcastMsg(uid, map[string]any{"type": "token_done", "token_id": id, "success": success})
		}
		if a.routeChild != nil {
			for _, ch := range childTokens {
				a.routeChild(ch)
			}
		}
		if a.taskDone != nil {
			a.taskDone(token)
		}
		if success && a.persist != nil {
			a.persist.FromToken(context.Background(), token)
		}
	}()

	tplID, _ := token["tpl_id"].(string)
	tplDoc := a.getTemplate(tplID)
	pageURL, _ := token["url"].(string)
	if pageURL == "" {
		if r, ok := token["r"].(map[string]any); ok {
			pageURL, _ = r["uri"].(string)
		}
	}
	waitReady := flagsContain(token, "WAIT_PAGE_READY")

	tabCtx, tabCancel := chromedpNewContext(a.alloc)
	defer tabCancel()
	session := &Session{ctx: tabCtx, cancel: tabCancel}

	doneCap := make(chan struct{})
	defer close(doneCap)
	var captured []tpl.CapturedResponse
	var capMu sync.Mutex
	_ = session.CaptureNetwork(doneCap, func(url string, body []byte, headers map[string]string) {
		capMu.Lock()
		captured = append(captured, tpl.CapturedResponse{URL: url, Body: body, Headers: headers})
		capMu.Unlock()
	})

	if err := session.NavigateWaitReady(pageURL, waitReady); err != nil {
		tpl.ReqLogDone(token, err)
		return
	}

	runCtx := a.buildRunCtx(&childTokens)
	execAgent := tpl.NewChromeExecAgent(token, session)

	if tplDoc != nil {
		for _, icpt := range tpl.CollectInterceptors(tplDoc) {
			capMu.Lock()
			snap := append([]tpl.CapturedResponse(nil), captured...)
			capMu.Unlock()
			for _, cap := range snap {
				tpl.RunInterceptorMatch(runCtx, execAgent, token, icpt, cap)
			}
		}
		if err := tpl.RunProcedures(runCtx, tplDoc, token, execAgent); err != nil {
			if strings.Contains(err.Error(), "if not satisfy") || strings.Contains(err.Error(), "scheduled retry") {
				tpl.ReqLogDone(token, err)
				return
			}
			tpl.ReqLogDone(token, err)
			return
		}
	}

	if html, err := session.PageSource(); err == nil {
		execAgent.SetResponseText(html)
	}
	tpl.ReqLogDone(token, nil)
	success = true
}

func flagsContain(token map[string]any, flag string) bool {
	flags, ok := token["flags"].([]any)
	if !ok {
		if fs, ok := token["flags"].([]string); ok {
			for _, f := range fs {
				if f == flag {
					return true
				}
			}
		}
		return false
	}
	for _, f := range flags {
		if fmt.Sprint(f) == flag {
			return true
		}
	}
	return false
}
