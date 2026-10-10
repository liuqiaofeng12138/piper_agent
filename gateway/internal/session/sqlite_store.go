package session

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func OpenSQLite(path string) (*SQLiteStore, error) {
	if path == "" {
		return nil, fmt.Errorf("sqlite path required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS conversations (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS messages (
  id TEXT PRIMARY KEY,
  conversation_id TEXT NOT NULL,
  role TEXT NOT NULL,
  content TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY(conversation_id) REFERENCES conversations(id)
);
CREATE INDEX IF NOT EXISTS idx_messages_conv ON messages(conversation_id, created_at);
`)
	if err != nil {
		return err
	}
	_, _ = s.db.Exec(`ALTER TABLE conversations ADD COLUMN preferred_agent TEXT NOT NULL DEFAULT ''`)
	return nil
}

func (s *SQLiteStore) CreateConversation(title string) *Conversation {
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
	_, _ = s.db.Exec(
		`INSERT INTO conversations(id, title, created_at, updated_at) VALUES (?,?,?,?)`,
		c.ID, c.Title, c.CreatedAt.Format(time.RFC3339Nano), c.UpdatedAt.Format(time.RFC3339Nano),
	)
	return c
}

func (s *SQLiteStore) ListConversations() []*Conversation {
	rows, err := s.db.Query(`SELECT id, title, created_at, updated_at FROM conversations ORDER BY updated_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*Conversation
	for rows.Next() {
		var c Conversation
		var created, updated string
		if err := rows.Scan(&c.ID, &c.Title, &created, &updated); err != nil {
			continue
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		c.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		out = append(out, &c)
	}
	return out
}

func (s *SQLiteStore) GetConversation(id string) (*Conversation, bool) {
	row := s.db.QueryRow(`SELECT id, title, created_at, updated_at FROM conversations WHERE id=?`, id)
	var c Conversation
	var created, updated string
	if err := row.Scan(&c.ID, &c.Title, &created, &updated); err != nil {
		return nil, false
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	c.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return &c, true
}

func (s *SQLiteStore) AppendMessage(conversationID string, role Role, content string) (Message, bool) {
	if _, ok := s.GetConversation(conversationID); !ok {
		return Message{}, false
	}
	msg := Message{
		ID:        "m_" + uuid.NewString(),
		Role:      role,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	}
	_, err := s.db.Exec(
		`INSERT INTO messages(id, conversation_id, role, content, created_at) VALUES (?,?,?,?,?)`,
		msg.ID, conversationID, string(role), content, msg.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return Message{}, false
	}
	title := ""
	if role == RoleUser {
		_ = s.db.QueryRow(`SELECT title FROM conversations WHERE id=?`, conversationID).Scan(&title)
		if title == "新对话" && content != "" {
			t := content
			if len(t) > 32 {
				t = t[:32] + "…"
			}
			_, _ = s.db.Exec(`UPDATE conversations SET title=?, updated_at=? WHERE id=?`,
				t, msg.CreatedAt.Format(time.RFC3339Nano), conversationID)
		} else {
			_, _ = s.db.Exec(`UPDATE conversations SET updated_at=? WHERE id=?`,
				msg.CreatedAt.Format(time.RFC3339Nano), conversationID)
		}
	} else {
		_, _ = s.db.Exec(`UPDATE conversations SET updated_at=? WHERE id=?`,
			msg.CreatedAt.Format(time.RFC3339Nano), conversationID)
	}
	return msg, true
}

func (s *SQLiteStore) GetPreferredAgent(conversationID string) string {
	var agent string
	err := s.db.QueryRow(
		`SELECT preferred_agent FROM conversations WHERE id=?`,
		conversationID,
	).Scan(&agent)
	if err != nil || agent == "" {
		return ""
	}
	return agent
}

func (s *SQLiteStore) SetPreferredAgent(conversationID string, agentID string) bool {
	res, err := s.db.Exec(
		`UPDATE conversations SET preferred_agent=? WHERE id=?`,
		agentID, conversationID,
	)
	if err != nil {
		return false
	}
	n, _ := res.RowsAffected()
	return n > 0
}

func (s *SQLiteStore) DeleteConversation(id string) bool {
	res, err := s.db.Exec(`DELETE FROM conversations WHERE id=?`, id)
	if err != nil {
		return false
	}
	_, _ = s.db.Exec(`DELETE FROM messages WHERE conversation_id=?`, id)
	n, _ := res.RowsAffected()
	return n > 0
}

func (s *SQLiteStore) ListMessages(conversationID string) []Message {
	rows, err := s.db.Query(
		`SELECT id, role, content, created_at FROM messages WHERE conversation_id=? ORDER BY created_at ASC`,
		conversationID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		var m Message
		var role string
		var created string
		if err := rows.Scan(&m.ID, &role, &m.Content, &created); err != nil {
			continue
		}
		m.Role = Role(role)
		m.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, m)
	}
	return out
}

func ResolveSQLitePath(configPath, rel string) string {
	if rel == "" {
		rel = "../../data/gateway.db"
	}
	if filepath.IsAbs(rel) {
		return rel
	}
	if configPath != "" {
		return filepath.Clean(filepath.Join(filepath.Dir(configPath), rel))
	}
	return filepath.Clean(rel)
}
