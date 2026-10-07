package router

import (
	"strings"
)

const AgentWebCrawler = "web_crawler"

// Engine 轻量意图路由（Phase W2：规则 + 默认 web_crawler）。
type Engine struct {
	mode string
}

func New(mode string) *Engine {
	if mode == "" {
		mode = "rule"
	}
	return &Engine{mode: mode}
}

func (e *Engine) Classify(userMessage string) string {
	if e.mode != "rule" {
		return AgentWebCrawler
	}
	_ = matchWebCrawlerKeywords(userMessage)
	return AgentWebCrawler
}

func matchWebCrawlerKeywords(msg string) bool {
	lower := strings.ToLower(msg)
	keywords := []string{
		"抓", "采集", "爬", "抓取", "模板", "模版", "url", "http", "api",
		"index", "es", "chrome", "代理", "列表", "字段",
	}
	for _, k := range keywords {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
}
