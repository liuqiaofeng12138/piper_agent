package session

// Store 会话与消息持久化（内存或 SQLite）。
type Store interface {
	CreateConversation(title string) *Conversation
	ListConversations() []*Conversation
	GetConversation(id string) (*Conversation, bool)
	AppendMessage(conversationID string, role Role, content string) (Message, bool)
	ListMessages(conversationID string) []Message
	DeleteConversation(id string) bool
	// PreferredAgent 本会话后续消息优先路由的 Agent（如上传文档后固定 doc_rag）。
	GetPreferredAgent(conversationID string) string
	SetPreferredAgent(conversationID string, agentID string) bool
}
