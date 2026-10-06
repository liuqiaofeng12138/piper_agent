package tpl

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// HTTPAgent implements template Agent for HTTP tokens (Java HttpAgent subset).
type HTTPAgent struct {
	Client *http.Client
	Ctx    context.Context
	Token  map[string]any
}

func NewHTTPAgent(token map[string]any, readTimeoutMs int) *HTTPAgent {
	return &HTTPAgent{
		Client: &http.Client{Timeout: time.Duration(readTimeoutMs) * time.Millisecond},
		Ctx:    context.Background(),
		Token:  token,
	}
}

func (a *HTTPAgent) GetSrc() string {
	r, _ := a.Token["r"].(map[string]any)
	if r == nil {
		return ""
	}
	text, _ := r["text"].(string)
	return text
}

func (a *HTTPAgent) SetResponseText(text string) {
	r, _ := a.Token["r"].(map[string]any)
	if r == nil {
		r = map[string]any{}
		a.Token["r"] = r
	}
	r["text"] = text
}

func (a *HTTPAgent) IsChrome() bool { return false }

func (a *HTTPAgent) ElementCount(_ string) (int, error) { return 0, nil }

func (a *HTTPAgent) ChromeScroll(_ map[string]any, _ map[string]any) error { return nil }

func (a *HTTPAgent) ChromeClick(_ string) error { return nil }

func (a *HTTPAgent) ChromeSetValue(_, _ string, _ map[string]any) error { return nil }

func (a *HTTPAgent) ChromeNavigate(url string) error { return a.LoadURL(url) }

func (a *HTTPAgent) ChromeEvalJS(_ string) (string, error) { return "", nil }

func (a *HTTPAgent) GetDom() *goquery.Document {
	src := a.GetSrc()
	if src == "" {
		return nil
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(src))
	if err != nil {
		return nil
	}
	return doc
}

func (a *HTTPAgent) LoadURL(url string) error {
	req, err := http.NewRequestWithContext(a.Ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := a.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	r, _ := a.Token["r"].(map[string]any)
	if r == nil {
		r = map[string]any{}
		a.Token["r"] = r
	}
	r["text"] = string(b)
	r["status"] = resp.StatusCode
	return nil
}
