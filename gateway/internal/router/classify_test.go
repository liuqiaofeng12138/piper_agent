package router

import (
	"context"
	"strings"
	"testing"

	"piper_agent/gateway/internal/config"
)

func testConfig() *config.Config {
	return &config.Config{
		Classifier: config.ClassifierConfig{Mode: "rule"},
		Agents: []config.AgentSpec{
			{ID: "web_crawler", DisplayName: "网页采集", Address: "127.0.0.1:15061", Enabled: true},
			{ID: "general_chat", DisplayName: "通用对话", Address: "127.0.0.1:15062", Enabled: true, Default: true},
			{ID: "doc_rag", DisplayName: "文档问答", Address: "127.0.0.1:15063", Enabled: true},
			{ID: "paper_search", DisplayName: "论文检索", Address: "127.0.0.1:15064", Enabled: true},
		},
	}
}

func TestRuleClassifyCrawlMessage(t *testing.T) {
	e := New(testConfig())
	id, _ := e.Classify(context.Background(), "帮我抓取这个 API 的 title 字段")
	if id != "web_crawler" {
		t.Fatalf("agent = %s, want web_crawler", id)
	}
}

func TestRuleClassifyPaperMessage(t *testing.T) {
	e := New(testConfig())
	id, reason := e.Classify(context.Background(), "查询最新的5篇关于ai infra的论文，汇总展示给我")
	if id != "paper_search" {
		t.Fatalf("agent = %s, want paper_search", id)
	}
	if !strings.Contains(reason, "论文") {
		t.Fatalf("reason = %q", reason)
	}
}

func TestRuleClassifyChatMessage(t *testing.T) {
	e := New(testConfig())
	id, reason := e.Classify(context.Background(), "今天天气怎么样，适合出门吗")
	if id != "general_chat" {
		t.Fatalf("agent = %s, want general_chat", id)
	}
	if reason == "" {
		t.Fatal("reason should not be empty")
	}
}

func TestUploadSkipsIntentToDocRAG(t *testing.T) {
	e := New(testConfig())
	id, reason := e.ClassifyWithAttachments(context.Background(), "帮我抓取网页", 2)
	if id != "doc_rag" {
		t.Fatalf("agent = %s, want doc_rag", id)
	}
	if !strings.Contains(reason, "upload") {
		t.Fatalf("reason = %q", reason)
	}
}

func TestLLMModeWithoutKeyFallsBackToRule(t *testing.T) {
	cfg := testConfig()
	cfg.Classifier.Mode = "llm"
	e := New(cfg)
	if e.mode != "rule" {
		t.Fatalf("mode = %s, want rule (no api key)", e.mode)
	}
	id, _ := e.Classify(context.Background(), "讲个笑话")
	if id != "general_chat" {
		t.Fatalf("agent = %s, want general_chat", id)
	}
}
