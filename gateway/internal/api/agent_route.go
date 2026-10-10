package api

import (
	"context"
	"fmt"
	"strings"

	"piper_agent/gateway/internal/config"
	"piper_agent/gateway/internal/router"
	"piper_agent/gateway/internal/session"
)

// resolveChatAgent 上传文件时直达 doc_rag；本会话已绑定 Agent 时延续（避免追问落到 general_chat 丢失文档上下文）。
func resolveChatAgent(
	ctx context.Context,
	store session.Store,
	cfg *config.Config,
	rtr *router.Engine,
	convID string,
	userMessage string,
	docCount int,
) (string, string) {
	if docCount > 0 {
		id, reason := rtr.ClassifyWithAttachments(ctx, userMessage, docCount)
		if id != "" {
			_ = store.SetPreferredAgent(convID, id)
		}
		return id, reason
	}
	sticky := store.GetPreferredAgent(convID)
	if sticky == "" && cfg.FindAgent(router.AgentDocRAG) != nil && conversationHadDocumentUpload(store, convID) {
		sticky = router.AgentDocRAG
		_ = store.SetPreferredAgent(convID, sticky)
	}
	if sticky != "" {
		if cfg.FindAgent(sticky) != nil {
			return sticky, fmt.Sprintf("session: 延续会话 Agent %q", sticky)
		}
	}
	return rtr.Classify(ctx, userMessage)
}

func conversationHadDocumentUpload(store session.Store, convID string) bool {
	for _, m := range store.ListMessages(convID) {
		if m.Role == session.RoleUser && strings.Contains(m.Content, "[附件:") {
			return true
		}
	}
	return false
}
