package chrome

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

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
	return chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Evaluate(`window.stop()`, nil),
	)
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
	tasks = append(tasks, chromedp.Evaluate(`window.stop()`, nil))
	return chromedp.Run(ctx, tasks...)
}

func (s *Session) PageSource() (string, error) {
	var html string
	err := chromedp.Run(s.ctx, chromedp.OuterHTML("html", &html, chromedp.ByQuery))
	return html, err
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
			go func(requestID network.RequestID, url string, mime string) {
				body, err := network.GetResponseBody(requestID).Do(s.ctx)
				if err != nil {
					return
				}
				h := map[string]string{}
				if mime != "" {
					h["Content-Type"] = mime
				}
				onResponse(url, body, h)
			}(e.RequestID, e.Response.URL, e.Response.MimeType)
		}
	})
	return chromedp.Run(s.ctx, network.Enable())
}
