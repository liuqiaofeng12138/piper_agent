package session

// Store 会话与消息持久化（内存或 SQLite）。
type Store interface {
	CreateConversation(title string) *Conversation
	ListConversations() []*Conversation
	GetConversation(id string) (*Conversation, bool)
	AppendMessage(conversationID string, role Role, content string) (Message, bool)
	ListMessages(conversationID string) []Message
}
