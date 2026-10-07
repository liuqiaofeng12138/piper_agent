package session

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Message struct {
	ID        string    `json:"id"`
	Role      Role      `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Conversation struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MemoryStore struct {
	mu            sync.RWMutex
	conversations map[string]*Conversation
	messages      map[string][]Message
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		conversations: make(map[string]*Conversation),
		messages:      make(map[string][]Message),
	}
}

func (s *MemoryStore) CreateConversation(title string) *Conversation {
	now := time.Now().UTC()
	if title == "" {
		title = "新对话"
	}
	c := &Conversation{
		ID:        "c_" + uuid.NewString(),
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.mu.Lock()
	s.conversations[c.ID] = c
	s.messages[c.ID] = nil
	s.mu.Unlock()
	return c
}

func (s *MemoryStore) ListConversations() []*Conversation {
	s.mu.RLock()
	out := make([]*Conversation, 0, len(s.conversations))
	for _, c := range s.conversations {
		out = append(out, c)
	}
	s.mu.RUnlock()
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].UpdatedAt.After(out[i].UpdatedAt) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func (s *MemoryStore) GetConversation(id string) (*Conversation, bool) {
	s.mu.RLock()
	c, ok := s.conversations[id]
	s.mu.RUnlock()
	return c, ok
}

func (s *MemoryStore) AppendMessage(conversationID string, role Role, content string) (Message, bool) {
	s.mu.Lock()
	c, ok := s.conversations[conversationID]
	if !ok {
		s.mu.Unlock()
		return Message{}, false
	}
	msg := Message{
		ID:        "m_" + uuid.NewString(),
		Role:      role,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	}
	s.messages[conversationID] = append(s.messages[conversationID], msg)
	c.UpdatedAt = msg.CreatedAt
	if role == RoleUser && c.Title == "新对话" && len(content) > 0 {
		title := content
		if len(title) > 32 {
			title = title[:32] + "…"
		}
		c.Title = title
	}
	s.mu.Unlock()
	return msg, true
}

func (s *MemoryStore) ListMessages(conversationID string) []Message {
	s.mu.RLock()
	msgs := s.messages[conversationID]
	if msgs == nil {
		s.mu.RUnlock()
		return nil
	}
	out := make([]Message, len(msgs))
	copy(out, msgs)
	s.mu.RUnlock()
	return out
}
