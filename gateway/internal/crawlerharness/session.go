package crawlerharness

type Session struct {
	SessionID            string
	Messages             []map[string]any
	ValidatedTemplateIDs   map[string]struct{}
	LastTemplateID       string
	LastRunID            string
	LastTokenID          string
	PendingTemplate      map[string]any
	SelectedProxyID      string
	RunCount             int
}

func NewSession(convID string) *Session {
	return &Session{
		SessionID:          convID,
		Messages:           nil,
		ValidatedTemplateIDs: make(map[string]struct{}),
	}
}
