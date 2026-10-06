package mouse

// Action mirrors Java simulator.mouse.Action JSON.
type Action struct {
	Type string  `json:"type"`
	X    int     `json:"x"`
	Y    int     `json:"y"`
	Time int64   `json:"time"`
	Meta float64 `json:"meta,omitempty"`
}

const (
	TypePress   = "Press"
	TypeRelease = "Release"
	TypeMove    = "Move"
)
