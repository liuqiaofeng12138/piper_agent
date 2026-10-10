package router

import (
	"context"
	"log"
	"strings"

	"piper_agent/gateway/internal/config"
)

// Engine 意图路由：rule（关键词）或 llm（小模型 JSON 输出），失败回退默认 Agent。
type Engine struct {
	mode       string
	agents     []config.AgentSpec
	defaultID  string
	classifier *LLMClassifier
}

func New(cfg *config.Config) *Engine {
	mode := cfg.Classifier.Mode
	if mode == "" {
		mode = "rule"
	}
	e := &Engine{mode: mode, agents: cfg.EnabledAgents()}
	if d := cfg.DefaultAgent(); d != nil {
		e.defaultID = d.ID
	}
	if mode == "llm" {
		e.classifier = NewLLMClassifier(cfg)
		if e.classifier == nil {
			log.Printf("[router] classifier.mode=llm 但 llm.api_key 未配置，回退 rule 模式")
			e.mode = "rule"
		}
	}
	return e
}

// Classify 返回选中的 agent_id 与路由原因（用于审计日志）。
func (e *Engine) Classify(ctx context.Context, userMessage string) (string, string) {
	return e.ClassifyWithAttachments(ctx, userMessage, 0)
}

// ClassifyWithAttachments 若附带文档则跳过意图识别，直达 doc_rag。
func (e *Engine) ClassifyWithAttachments(ctx context.Context, userMessage string, documentCount int) (string, string) {
	if documentCount > 0 {
		for _, a := range e.agents {
			if a.ID == AgentDocRAG {
				return AgentDocRAG, "upload: 附带文档，直达文档问答 Agent"
			}
		}
	}
	if e.mode == "llm" && e.classifier != nil && len(e.agents) > 1 {
		if id, reason, ok := e.classifier.Classify(ctx, userMessage, e.agents); ok {
			return id, "llm: " + reason
		}
		// LLM 失败/超时/结果非法 → 回退规则
	}
	return e.classifyByRule(userMessage)
}

func (e *Engine) classifyByRule(msg string) (string, string) {
	if id := matchWebCrawlerKeywords(msg, e.agents); id != "" {
		return id, "rule: 命中采集关键词"
	}
	if id := matchPaperSearchKeywords(msg, e.agents); id != "" {
		return id, "rule: 命中论文检索关键词"
	}
	if e.defaultID != "" {
		return e.defaultID, "rule: 默认路由"
	}
	return "", "rule: 无可用 Agent"
}

// matchWebCrawlerKeywords 命中采集类关键词时返回注册表中第一个采集类 Agent。
func matchWebCrawlerKeywords(msg string, agents []config.AgentSpec) string {
	lower := strings.ToLower(msg)
	keywords := []string{
		"抓", "采集", "爬", "抓取", "模板", "模版", "url", "http", "api",
		"index", "es", "chrome", "代理", "列表", "字段",
	}
	matched := false
	for _, k := range keywords {
		if strings.Contains(lower, k) {
			matched = true
			break
		}
	}
	if !matched {
		return ""
	}
	for _, a := range agents {
		if a.ID == "web_crawler" {
			return a.ID
		}
	}
	return ""
}

// matchPaperSearchKeywords 命中学术论文检索类关键词时返回 paper_search Agent。
func matchPaperSearchKeywords(msg string, agents []config.AgentSpec) string {
	lower := strings.ToLower(msg)
	keywords := []string{
		"论文", "文献", "arxiv", "学术", "paper", "papers", "publication",
		"期刊", "会议", "citation", "cite", "综述",
	}
	matched := false
	for _, k := range keywords {
		if strings.Contains(lower, k) {
			matched = true
			break
		}
	}
	if !matched {
		return ""
	}
	for _, a := range agents {
		if a.ID == AgentPaperSearch {
			return a.ID
		}
	}
	return ""
}
