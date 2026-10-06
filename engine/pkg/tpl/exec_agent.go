package tpl

import "github.com/PuerkitoBio/goquery"

// ExecAgent is the subset of Java Agent used during Template.run (HTTP + Chrome).
type ExecAgent interface {
	GetSrc() string
	SetResponseText(text string)
	GetDom() *goquery.Document
	LoadURL(url string) error
	IsChrome() bool
	ElementCount(css string) (int, error)
	ChromeScroll(proc map[string]any, vars map[string]any) error
	ChromeClick(css string) error
	ChromeSetValue(css, valueTpl string, vars map[string]any) error
	ChromeNavigate(url string) error
	ChromeEvalJS(script string) (string, error)
}
