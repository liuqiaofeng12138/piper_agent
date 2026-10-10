package api

import (
	"context"
	"testing"

	"piper_agent/gateway/internal/config"
	"piper_agent/gateway/internal/router"
	"piper_agent/gateway/internal/session"
)

func testRouterCfg() *config.Config {
	return &config.Config{
		Classifier: config.ClassifierConfig{Mode: "rule"},
		Agents: []config.AgentSpec{
			{ID: "web_crawler", Enabled: true, Default: false},
			{ID: "general_chat", Enabled: true, Default: true},
			{ID: "doc_rag", Enabled: true},
		},
	}
}

func TestResolveChatAgentStickyDocRAG(t *testing.T) {
	store := session.NewMemoryStore()
	c := store.CreateConversation("")
	_ = store.SetPreferredAgent(c.ID, "doc_rag")
	rtr := router.New(testRouterCfg())

	id, reason := resolveChatAgent(context.Background(), store, testRouterCfg(), rtr, c.ID, "继续问简历方向", 0)
	if id != "doc_rag" {
		t.Fatalf("agent=%s want doc_rag", id)
	}
	if reason == "" {
		t.Fatal("reason should not be empty")
	}
}

func TestResolveChatAgentFromAttachmentHistory(t *testing.T) {
	store := session.NewMemoryStore()
	c := store.CreateConversation("")
	_, _ = store.AppendMessage(c.ID, session.RoleUser, "总结\n\n[附件: resume.pdf]")
	rtr := router.New(testRouterCfg())

	id, _ := resolveChatAgent(context.Background(), store, testRouterCfg(), rtr, c.ID, "适合什么方向", 0)
	if id != "doc_rag" {
		t.Fatalf("agent=%s want doc_rag", id)
	}
	if store.GetPreferredAgent(c.ID) != "doc_rag" {
		t.Fatal("preferred agent should be persisted")
	}
}
