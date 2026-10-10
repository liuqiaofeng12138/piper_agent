package chrome

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"piper_go/internal/config"
	"piper_go/pkg/db/es"
	"piper_go/pkg/persistence"
	"piper_go/pkg/tpl"
	"piper_go/pkg/util"
	"piper_go/pkg/websocket"

	"github.com/chromedp/chromedp"
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

	browserMu       sync.Mutex
	browserCtx      context.Context
	browserCancel   context.CancelFunc
	browserStarted  bool
	fallbackAlloc   context.Context
	fallbackCancel  context.CancelFunc

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
	a.browserMu.Lock()
	if a.browserCancel != nil {
		a.browserCancel()
		a.browserCancel = nil
		a.browserCtx = nil
		a.browserStarted = false
	}
	a.browserMu.Unlock()
	if a.fallbackCancel != nil {
		a.fallbackCancel()
	}
	if a.cancel != nil {
		a.cancel()
	}
}

func (a *Agent) ensureBrowser() error {
	a.browserMu.Lock()
	defer a.browserMu.Unlock()
	if a.browserStarted && a.browserCtx != nil {
		return nil
	}
	ctx, cancel, err := startBrowserContext(a.alloc)
	dir := strings.TrimSpace(a.cfg.Chrome.UserDataDir)
	if err != nil && dir != "" && shouldUseFallbackProfile(err) {
		fallback := fallbackChromeUserDataDir(dir, a.Name)
		log.Printf("chrome agent %s: profile %q failed (%v); trying clean profile %q", a.Name, dir, err, fallback)
		if a.fallbackAlloc == nil {
			fbCfg := a.cfg
			fbCfg.Chrome.UserDataDir = fallback
			opts := chromedpAllocatorOptions(fbCfg)
			a.fallbackAlloc, a.fallbackCancel = chromedpNewExecAllocator(context.Background(), opts...)
		}
		ctx, cancel, err = startBrowserContext(a.fallbackAlloc)
	}
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return fmt.Errorf("chrome failed to start (try deleting data/chrome_profile or set engine.Chrome.userDataDir to data/chrome_piper): %w", err)
	}
	a.browserCtx = ctx
	a.browserCancel = cancel
	a.browserStarted = true
	log.Printf("chrome agent %s: browser process ready (reused across tokens)", a.Name)
	return nil
}

func startBrowserContext(alloc context.Context) (context.Context, context.CancelFunc, error) {
	ctx, cancel := chromedpNewContext(alloc)
	if err := chromedp.Run(ctx); err != nil {
		cancel()
		return nil, nil, err
	}
	return ctx, cancel, nil
}

func shouldUseFallbackProfile(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "failed to start") ||
		strings.Contains(s, "in use") ||
		strings.Contains(s, "singleton") ||
		strings.Contains(s, "user data directory") ||
		strings.Contains(s, "profile")
}

func fallbackChromeUserDataDir(configuredDir, agentName string) string {
	base := filepath.Dir(configuredDir)
	if base == "" || base == "." {
		base = configuredDir
	}
	return filepath.Join(base, "chrome_piper", strings.ReplaceAll(agentName, "/", "_"))
}

func (a *Agent) resetBrowser() {
	a.browserMu.Lock()
	if a.browserCancel != nil {
		a.browserCancel()
	}
	a.browserCtx = nil
	a.browserCancel = nil
	a.browserStarted = false
	a.browserMu.Unlock()
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
		token["finished"] = true
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

	log.Printf("chrome agent %s: start token %s url=%s", a.Name, id, pageURL)
	if err := a.ensureBrowser(); err != nil {
		log.Printf("chrome agent %s: ensureBrowser: %v", a.Name, err)
		token["error"] = err.Error()
		tpl.ReqLogDone(token, err)
		return
	}
	// 直接使用常驻浏览器的主标签页，不再每 token 新开标签：
	// 避免空白标签残留，且用户登录操作与自动化在同一标签进行。
	session := &Session{ctx: a.browserCtx, cancel: func() {}}

	doneCap := make(chan struct{})
	defer close(doneCap)
	var captured []tpl.CapturedResponse
	var capMu sync.Mutex
	_ = session.CaptureNetwork(doneCap, func(url string, body []byte, headers map[string]string) {
		capMu.Lock()
		captured = append(captured, tpl.CapturedResponse{URL: url, Body: body, Headers: headers})
		capMu.Unlock()
	})

	log.Printf("chrome agent %s: navigating to %s", a.Name, pageURL)
	if err := session.NavigateWaitReady(pageURL, waitReady); err != nil {
		log.Printf("chrome agent %s: navigate %s: %v", a.Name, pageURL, err)
		a.resetBrowser()
		token["error"] = err.Error()
		tpl.ReqLogDone(token, err)
		return
	}
	loc, _ := session.LocationURL()
	log.Printf("chrome agent %s: navigation finished, location=%s", a.Name, loc)
	if html, _ := session.PageSource(); len(strings.TrimSpace(html)) < 200 {
		log.Printf("chrome agent %s: page is nearly blank (%d bytes), retrying navigation", a.Name, len(html))
		_ = session.Navigate(pageURL)
		loc, _ = session.LocationURL()
		log.Printf("chrome agent %s: retry navigation, location=%s", a.Name, loc)
	}

	runCtx := a.buildRunCtx(&childTokens)
	execAgent := tpl.NewChromeExecAgent(token, session)

	if a.shouldWaitManualLoginBeforeScrape(tplDoc, pageURL) {
		html, _ := session.PageSource()
		loc, _ := session.LocationURL()
		log.Printf("chrome agent %s: login check url=%s html_len=%d", a.Name, loc, len(html))
		log.Printf("chrome agent %s: login detect detail_wbtext=%v mymblog=%v islogin=%v 关注=%v Feed_wrap=%v",
			a.Name,
			strings.Contains(html, "detail_wbtext"),
			strings.Contains(html, "mymblog"),
			strings.Contains(html, `"islogin":1`),
			strings.Contains(html, "关注"),
			strings.Contains(html, "Feed_wrap"))
		need := pageNeedsManualLogin(html, loc)
		if tpl.ChromeBuilderRequireLogin(tplDoc) && !pageLooksLoggedInForScrape(html, loc) {
			need = true
		}
		if need {
			a.waitForManualLogin(session, pageURL, "login required — complete Weibo login in the browser window")
		} else {
			log.Printf("chrome agent %s: logged-in feed detected, skipping manual login wait", a.Name)
		}
	}
	// 登录后重新拉取页面与 Ajax，便于 Interceptor 命中 mymblog 等接口。
	if err := session.NavigateWaitReady(pageURL, waitReady); err != nil {
		log.Printf("chrome agent %s: post-login navigate: %v", a.Name, err)
	}
	if sel, sec := tpl.ChromeBuilderWait(tplDoc); sel != "" {
		timeout := time.Duration(sec) * time.Second
		if timeout <= 0 {
			timeout = 60 * time.Second
		}
		if err := session.WaitForSelector(sel, timeout); err != nil {
			log.Printf("chrome agent %s: wait_selector %q: %v", a.Name, sel, err)
		}
	}
	_ = session.ScrollWheel(5)
	// 登录后 mymblog 等 XHR 由前端异步发起，GetResponseBody 也是异步抓取。
	// 轮询等待：捕获到含 mymblog 的响应即放行，最多等 12s。
	if tplDoc != nil && len(tpl.CollectInterceptors(tplDoc)) > 0 {
		waitDeadline := time.Now().Add(12 * time.Second)
		for time.Now().Before(waitDeadline) {
			capMu.Lock()
			found := false
			for _, c := range captured {
				if strings.Contains(c.URL, "mymblog") {
					found = true
					break
				}
			}
			n := len(captured)
			capMu.Unlock()
			if found {
				log.Printf("chrome agent %s: mymblog response captured (%d total)", a.Name, n)
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
	} else {
		time.Sleep(3 * time.Second)
	}
	if html, err := session.PageSource(); err == nil {
		execAgent.SetResponseText(html)
	}

	if tplDoc != nil {
		a.runInterceptors(runCtx, tplDoc, token, execAgent, &capMu, &captured)
		if err := tpl.RunProcedures(runCtx, tplDoc, token, execAgent); err != nil {
			// LoadUrlAction 在 Java 侧通过 scheduled retry 重新入队；单次 Chrome 执行已完成导航，应继续取页。
			if strings.Contains(err.Error(), "scheduled retry") {
				err = nil
			}
			if err != nil {
				if strings.Contains(err.Error(), "if not satisfy") {
					tpl.ReqLogDone(token, err)
					return
				}
				tpl.ReqLogDone(token, err)
				return
			}
		}
	}

	if html, err := session.PageSource(); err == nil {
		execAgent.SetResponseText(html)
	}
	tpl.ReqLogDone(token, nil)
	success = true
}

func (a *Agent) shouldWaitManualLoginBeforeScrape(tplDoc map[string]any, pageURL string) bool {
	if a.cfg.Chrome.Headless {
		return false
	}
	if tpl.TemplateHasManualLoginProcedure(tplDoc) {
		return false
	}
	if !a.cfg.Chrome.Enabled {
		return false
	}
	// 微博个人页 + require_login：即使 manualLoginWaitSeconds==0 也要等人工登录，否则只有访客壳、抓不到 mymblog。
	if tpl.ChromeBuilderRequireLogin(tplDoc) && strings.Contains(pageURL, "weibo.com/u/") {
		return true
	}
	if a.cfg.Chrome.ManualLoginWaitSeconds == 0 {
		return false
	}
	return true
}

func (a *Agent) runInterceptors(
	runCtx *tpl.RunContext,
	tplDoc map[string]any,
	token map[string]any,
	execAgent *tpl.ChromeExecAgent,
	capMu *sync.Mutex,
	captured *[]tpl.CapturedResponse,
) {
	interceptors := tpl.CollectInterceptors(tplDoc)
	if len(interceptors) > 0 {
		capMu.Lock()
		urls := make([]string, 0, len(*captured))
		for _, c := range *captured {
			urls = append(urls, c.URL)
		}
		capMu.Unlock()
		if len(urls) == 0 {
			log.Printf("chrome agent %s: no network responses captured for interceptors", a.Name)
		} else {
			const maxShow = 8
			if len(urls) > maxShow {
				urls = urls[len(urls)-maxShow:]
			}
			log.Printf("chrome agent %s: captured %d responses, sample: %q", a.Name, len(*captured), urls)
		}
	}
	for _, icpt := range interceptors {
		capMu.Lock()
		snap := append([]tpl.CapturedResponse(nil), *captured...)
		capMu.Unlock()
		for _, cap := range snap {
			tpl.RunInterceptorMatch(runCtx, execAgent, token, icpt, cap)
		}
	}
}

func (a *Agent) waitForManualLogin(session *Session, pageURL, phase string) {
	sec := a.cfg.Chrome.ManualLoginWaitSeconds
	if sec <= 0 {
		// require_login 或默认：首次登录至少给 5 分钟，避免用户没来得及扫码。
		sec = 300
	}
	deadline := time.Now().Add(time.Duration(sec) * time.Second)
	// 最短等待期：避免骨架屏/游客页被 HTML 启发式误判为"已登录"而秒过。
	// 只有 Cookie 判定（SUB）能在最短等待期内提前放行。
	minWait := 8 * time.Second
	start := time.Now()
	log.Printf("chrome agent %s: %s — login in browser (up to %ds, ends early on real login)", a.Name, phase, sec)
	lastLog := time.Now()
	for time.Now().Before(deadline) {
		loc, _ := session.LocationURL()
		if pageURL != "" && (loc == "" || strings.Contains(strings.ToLower(loc), "about:blank")) {
			log.Printf("chrome agent %s: active tab is %q, navigating to %s", a.Name, loc, pageURL)
			_ = session.NavigateWaitReady(pageURL, false)
			loc, _ = session.LocationURL()
		}
		// 权威判定：微博登录成功会种下 SUB cookie。优先用它。
		if session.HasWeiboLoginCookie() {
			log.Printf("chrome agent %s: weibo SUB cookie present (real login), continuing scrape", a.Name)
			return
		}
		// 兜底：HTML 启发式，但需超过最短等待期才生效（骨架屏加载完之前不信任）。
		if time.Since(start) >= minWait {
			html, _ := session.PageSource()
			if pageLooksLoggedInForScrape(html, loc) && pageHasWeiboFeedContent(html) {
				log.Printf("chrome agent %s: logged-in weibo feed content detected, continuing scrape", a.Name)
				return
			}
		}
		if time.Since(lastLog) >= 15*time.Second {
			log.Printf("chrome agent %s: waiting for login, ~%s remaining (no SUB cookie yet)", a.Name, time.Until(deadline).Round(time.Second))
			lastLog = time.Now()
		}
		time.Sleep(3 * time.Second)
	}
	log.Printf("chrome agent %s: manual login wait finished (no SUB cookie detected)", a.Name)
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
