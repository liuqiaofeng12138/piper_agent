package chrome

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// executorCtx 返回挂了 Target 执行器的 ctx：chromedp 原始 ctx 上并没有 cdproto
// executor，直接 network.GetXXX().Do(ctx) 会报 "invalid context"，必须用
// cdp.WithExecutor 包装（chromedp 官方 network 示例的写法）。
func (s *Session) executorCtx(timeout time.Duration) (context.Context, context.CancelFunc, bool) {
	c := chromedp.FromContext(s.ctx)
	if c == nil || c.Target == nil {
		return nil, nil, false
	}
	ectx := cdp.WithExecutor(context.Background(), c.Target)
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	wctx, cancel := context.WithTimeout(ectx, timeout)
	return wctx, cancel, true
}

// Session is a chromedp tab context bound to one token execution.
type Session struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func NewSession(parent context.Context, headless bool, chromePath string) (*Session, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)
	if chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	}
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(parent, opts...)
	ctx, cancel := chromedp.NewContext(allocCtx)
	s := &Session{ctx: ctx, cancel: func() {
		cancel()
		cancelAlloc()
	}}
	if err := chromedp.Run(ctx); err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}

func (s *Session) Close() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Session) Navigate(url string) error {
	ctx, cancel := context.WithTimeout(s.ctx, 120*time.Second)
	defer cancel()
	return chromedp.Run(ctx, chromedp.Navigate(url))
}

func (s *Session) NavigateWaitReady(url string, waitReady bool) error {
	ctx, cancel := context.WithTimeout(s.ctx, 180*time.Second)
	defer cancel()
	tasks := []chromedp.Action{
		network.Enable(),
		chromedp.Navigate(url),
	}
	if waitReady {
		tasks = append(tasks, chromedp.WaitReady("body", chromedp.ByQuery))
	}
	return chromedp.Run(ctx, tasks...)
}

func (s *Session) Reload() error {
	ctx, cancel := context.WithTimeout(s.ctx, 180*time.Second)
	defer cancel()
	return chromedp.Run(ctx, chromedp.Reload())
}

// WaitForSelector waits until css is visible (builder wait_selector).
func (s *Session) WaitForSelector(css string, timeout time.Duration) error {
	css = strings.TrimSpace(css)
	if css == "" {
		return nil
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(s.ctx, timeout)
	defer cancel()
	return chromedp.Run(ctx, chromedp.WaitVisible(css, chromedp.ByQuery))
}

func (s *Session) PageSource() (string, error) {
	var html string
	err := chromedp.Run(s.ctx, chromedp.OuterHTML("html", &html, chromedp.ByQuery))
	return html, err
}

func (s *Session) LocationURL() (string, error) {
	var u string
	err := chromedp.Run(s.ctx, chromedp.Location(&u))
	return u, err
}

func (s *Session) EvalJS(script string) (string, error) {
	var out any
	wrapped := script
	if !strings.HasPrefix(strings.TrimSpace(script), "return ") {
		wrapped = "return (" + script + ")"
	}
	err := chromedp.Run(s.ctx, chromedp.Evaluate(wrapped, &out))
	if err != nil {
		return "", err
	}
	return fmt.Sprint(out), nil
}

// HasWeiboLoginCookie 用 Cookie 判定微博是否真正登录（SUB 是微博登录态的硬标志）。
// SUB 是 HttpOnly，document.cookie 读不到，必须走 CDP network.GetCookies。
func (s *Session) HasWeiboLoginCookie() bool {
	ectx, cancel, ok := s.executorCtx(5 * time.Second)
	if !ok {
		return false
	}
	defer cancel()
	cookies, err := network.GetCookies().Do(ectx)
	if err != nil {
		return false
	}
	for _, c := range cookies {
		if !strings.Contains(c.Domain, "weibo.com") {
			continue
		}
		if c.Name == "SUB" && c.Value != "" {
			return true
		}
	}
	return false
}

func (s *Session) ScrollPixel(delta int) error {
	script := fmt.Sprintf(`document.documentElement.scrollTop = document.documentElement.scrollTop + %d`, delta)
	_, err := s.EvalJS(script)
	return err
}

func (s *Session) ScrollWheel(repeat int) error {
	script := fmt.Sprintf(`for(var i=0;i<%d;i++){ window.scrollBy(0, 120); }`, repeat)
	_, err := s.EvalJS(script)
	return err
}

func (s *Session) ClickCSS(css string) error {
	return chromedp.Run(s.ctx, chromedp.Click(css, chromedp.ByQuery))
}

func (s *Session) SetValueCSS(css, value string) error {
	return chromedp.Run(s.ctx,
		chromedp.Clear(css, chromedp.ByQuery),
		chromedp.SendKeys(css, value, chromedp.ByQuery),
	)
}

// CaptureNetwork enables listener and returns captured bodies until done is closed.
func (s *Session) CaptureNetwork(done <-chan struct{}, onResponse func(url string, body []byte, headers map[string]string)) error {
	// ResponseReceived 时 body 尚未接收完，立刻 GetResponseBody 对 XHR 基本必失败
	// （No resource with given identifier found）。必须先记下 requestID→URL 映射，
	// 等 LoadingFinished 再取 body。
	var mu sync.Mutex
	type respMeta struct{ url, mime string }
	meta := map[network.RequestID]respMeta{}
	var eventCount, bodyErrCount int32

	chromedp.ListenTarget(s.ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventResponseReceived:
			if e.Response == nil || e.Response.URL == "" {
				return
			}
			select {
			case <-done:
				return
			default:
			}
			mu.Lock()
			meta[e.RequestID] = respMeta{url: e.Response.URL, mime: e.Response.MimeType}
			eventCount++
			mu.Unlock()
		case *network.EventLoadingFinished:
			select {
			case <-done:
				return
			default:
			}
			mu.Lock()
			m, ok := meta[e.RequestID]
			if ok {
				delete(meta, e.RequestID)
			}
			mu.Unlock()
			if !ok {
				return
			}
			go func(requestID network.RequestID, url string, mime string) {
				ectx, cancel, ok := s.executorCtx(15 * time.Second)
				if !ok {
					return
				}
				defer cancel()
				body, err := network.GetResponseBody(requestID).Do(ectx)
				if err != nil {
					mu.Lock()
					bodyErrCount++
					n := bodyErrCount
					mu.Unlock()
					if n <= 3 {
						log.Printf("chrome capture: GetResponseBody %s: %v", url, err)
					}
					return
				}
				h := map[string]string{}
				if mime != "" {
					h["Content-Type"] = mime
				}
				onResponse(url, body, h)
			}(e.RequestID, m.url, m.mime)
		}
	})
	if err := chromedp.Run(s.ctx, network.Enable()); err != nil {
		log.Printf("chrome capture: network.Enable: %v", err)
		return err
	}
	log.Printf("chrome capture: network listener attached")
	return nil
}
