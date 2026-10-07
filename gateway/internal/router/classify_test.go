package router

import (
	"context"
	"testing"

	"piper_agent/gateway/internal/config"
)

func testConfig() *config.Config {
	return &config.Config{
		Classifier: config.ClassifierConfig{Mode: "rule"},
		Agents: []config.AgentSpec{
			{ID: "web_crawler", DisplayName: "网页采集", Address: "127.0.0.1:15061", Enabled: true},
			{ID: "general_chat", DisplayName: "通用对话", Address: "127.0.0.1:15062", Enabled: true, Default: true},
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
