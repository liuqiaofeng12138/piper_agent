package session

// Store 会话与消息持久化（内存或 SQLite）。
type Store interface {
	CreateConversation(userID, title string) *Conversation
	ListConversations(userID string) []*Conversation
	GetConversation(id, userID string) (*Conversation, bool)
	AppendMessage(conversationID, userID string, role Role, content string) (Message, bool)
	ListMessages(conversationID, userID string) []Message
	DeleteConversation(id, userID string) bool
	// PreferredAgent 本会话后续消息优先路由的 Agent（如上传文档后固定 doc_rag）。
	GetPreferredAgent(conversationID string) string
	SetPreferredAgent(conversationID string, agentID string) bool
}
