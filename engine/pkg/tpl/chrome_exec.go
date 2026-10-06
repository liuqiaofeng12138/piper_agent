package tpl

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ChromeExecAgent drives chromedp via BrowserSession (pkg/chrome).
type ChromeExecAgent struct {
	Token   map[string]any
	Session BrowserSession
	overrideSrc string
}

// BrowserSession is implemented by pkg/chrome.Session.
type BrowserSession interface {
	PageSource() (string, error)
	EvalJS(script string) (string, error)
	ScrollPixel(delta int) error
	ScrollWheel(repeat int) error
	ClickCSS(css string) error
	SetValueCSS(css, value string) error
	Navigate(url string) error
}

func NewChromeExecAgent(token map[string]any, session BrowserSession) *ChromeExecAgent {
	return &ChromeExecAgent{Token: token, Session: session}
}

func (a *ChromeExecAgent) GetSrc() string {
	if a.overrideSrc != "" {
		return a.overrideSrc
	}
	if a.Session == nil {
		return ""
	}
	s, err := a.Session.PageSource()
	if err != nil {
		return ""
	}
	a.syncResponseText(s)
	return s
}

func (a *ChromeExecAgent) SetResponseText(text string) {
	a.overrideSrc = text
	a.syncResponseText(text)
}

func (a *ChromeExecAgent) syncResponseText(s string) {
	r, _ := a.Token["r"].(map[string]any)
	if r == nil {
		r = map[string]any{}
		a.Token["r"] = r
	}
	r["text"] = s
}

func (a *ChromeExecAgent) GetDom() *goquery.Document {
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

func (a *ChromeExecAgent) LoadURL(url string) error {
	return a.ChromeNavigate(url)
}

func (a *ChromeExecAgent) IsChrome() bool { return true }

func (a *ChromeExecAgent) ElementCount(css string) (int, error) {
	script := fmt.Sprintf(`document.querySelectorAll(%q).length`, css)
	out, err := a.Session.EvalJS(script)
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return 0, nil
	}
	return n, nil
}

func (a *ChromeExecAgent) ChromeScroll(proc map[string]any, vars map[string]any) error {
	value := ReplaceVars(strVal(proc, "value"), vars, false)
	scrollType, _ := proc["type"].(string)
	if scrollType == "Pixel" {
		delta, _ := strconv.Atoi(value)
		return a.Session.ScrollPixel(delta)
	}
	repeat, _ := strconv.Atoi(value)
	if repeat > 3000 {
		repeat = 3000
	}
	if repeat <= 0 {
		repeat = 100
	}
	return a.Session.ScrollWheel(repeat)
}

func (a *ChromeExecAgent) ChromeClick(css string) error {
	return a.Session.ClickCSS(css)
}

func (a *ChromeExecAgent) ChromeSetValue(css, valueTpl string, vars map[string]any) error {
	val := ReplaceVars(valueTpl, vars, false)
	return a.Session.SetValueCSS(css, val)
}

func (a *ChromeExecAgent) ChromeNavigate(url string) error {
	return a.Session.Navigate(url)
}

func (a *ChromeExecAgent) ChromeEvalJS(script string) (string, error) {
	return a.Session.EvalJS(script)
}
