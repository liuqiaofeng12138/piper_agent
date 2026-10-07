package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempConfig(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "local.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadAgentsListFormat(t *testing.T) {
	path := writeTempConfig(t, `
agents:
  - id: web_crawler
    display_name: "网页采集"
    description: "采集"
    listen: "127.0.0.1:15061"
    enabled: true
  - id: general_chat
    display_name: "通用对话"
    listen: "127.0.0.1:15062"
    enabled: true
    default: true
  - id: doc_rag
    enabled: false
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Agents) != 3 {
		t.Fatalf("agents len = %d, want 3", len(cfg.Agents))
	}
	if cfg.FindAgent("general_chat") == nil {
		t.Fatal("general_chat should be found")
	}
	if cfg.FindAgent("doc_rag") != nil {
		t.Fatal("doc_rag disabled, should not be found")
	}
	if d := cfg.DefaultAgent(); d == nil || d.ID != "general_chat" {
		t.Fatalf("default agent = %+v, want general_chat", d)
	}
	if got := cfg.Agents[0].Address; got != "127.0.0.1:15061" {
		t.Fatalf("web_crawler address = %s", got)
	}
	if got := cfg.Agents[0].WorkerModule(); got != "piper_agent.workers.web_crawler" {
		t.Fatalf("worker module = %s", got)
	}
	// 内置默认注册表的 web_crawler 指向 web_crawler_agent 项目
	def, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if got := def.Agents[0].WorkerModule(); got != "web_crawler_agent.workers.web_crawler" {
		t.Fatalf("default worker module = %s", got)
	}
}

func TestLoadAgentsLegacyMapFormat(t *testing.T) {
	path := writeTempConfig(t, `
agents:
  web_crawler:
    listen: "127.0.0.1:15061"
    enabled: true
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Agents) != 1 || cfg.Agents[0].ID != "web_crawler" {
		t.Fatalf("agents = %+v", cfg.Agents)
	}
	if d := cfg.DefaultAgent(); d == nil || d.ID != "web_crawler" {
		t.Fatalf("default agent = %+v, want web_crawler", d)
	}
}

func TestLoadClassifierAndLLM(t *testing.T) {
	path := writeTempConfig(t, `
gateway:
  classifier:
    mode: llm
    model: deepseek-chat
    timeout_ms: 2500
llm:
  model: deepseek-flash
  api_key: "sk-test"
  base_url: "https://api.deepseek.com"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Classifier.Mode != "llm" || cfg.Classifier.Model != "deepseek-chat" || cfg.Classifier.TimeoutMs != 2500 {
		t.Fatalf("classifier = %+v", cfg.Classifier)
	}
	if cfg.LLM.APIKey != "sk-test" || cfg.LLM.BaseURL != "https://api.deepseek.com" {
		t.Fatalf("llm = %+v", cfg.LLM)
	}
}

func TestDefaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if d := cfg.DefaultAgent(); d == nil || d.ID != "web_crawler" {
		t.Fatalf("default agent = %+v, want web_crawler", d)
	}
	if cfg.Classifier.Mode != "rule" {
		t.Fatalf("classifier mode = %s", cfg.Classifier.Mode)
	}
}
