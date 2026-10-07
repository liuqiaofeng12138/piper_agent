package websocket

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type hub struct {
	msgSessions   map[string]map[*websocket.Conn]struct{}
	tokenSessions map[string]map[*websocket.Conn]struct{}
	mu            sync.RWMutex
}

var defaultHub = &hub{
	msgSessions:   make(map[string]map[*websocket.Conn]struct{}),
	tokenSessions: make(map[string]map[*websocket.Conn]struct{}),
}

func HandleMsg(w http.ResponseWriter, r *http.Request, auth func(token string) (uid string, ok bool), onConnect func(uid string)) {
	token := r.Header.Get("Sec-WebSocket-Protocol")
	uid, ok := auth(token)
	if !ok {
		http.Error(w, "Not Valid", http.StatusUnauthorized)
		return
	}
	conn, err := upgrader.Upgrade(w, r, http.Header{"Sec-WebSocket-Protocol": []string{token}})
	if err != nil {
		return
	}
	register(defaultHub.msgSessions, uid, conn)
	defer unregister(defaultHub.msgSessions, uid, conn)
	if onConnect != nil {
		onConnect(uid)
	}
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

func HandleTokenMsg(w http.ResponseWriter, r *http.Request, auth func(token string) (uid string, ok bool)) {
	token := r.Header.Get("Sec-WebSocket-Protocol")
	uid, ok := auth(token)
	if !ok {
		http.Error(w, "Not Valid", http.StatusUnauthorized)
		return
	}
	conn, err := upgrader.Upgrade(w, r, http.Header{"Sec-WebSocket-Protocol": []string{token}})
	if err != nil {
		return
	}
	register(defaultHub.tokenSessions, uid, conn)
	defer unregister(defaultHub.tokenSessions, uid, conn)
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

func register(m map[string]map[*websocket.Conn]struct{}, uid string, c *websocket.Conn) {
	defaultHub.mu.Lock()
	defer defaultHub.mu.Unlock()
	if m[uid] == nil {
		m[uid] = make(map[*websocket.Conn]struct{})
	}
	m[uid][c] = struct{}{}
}

func unregister(m map[string]map[*websocket.Conn]struct{}, uid string, c *websocket.Conn) {
	defaultHub.mu.Lock()
	defer defaultHub.mu.Unlock()
	if set, ok := m[uid]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(m, uid)
		}
	}
	_ = c.Close()
}

func BroadcastMsg(uid string, payload any) {
	b, _ := json.Marshal(payload)
	defaultHub.mu.RLock()
	defer defaultHub.mu.RUnlock()
	if uid == "" {
		for _, set := range defaultHub.msgSessions {
			for c := range set {
				_ = c.WriteMessage(websocket.TextMessage, b)
			}
		}
		return
	}
	for c := range defaultHub.msgSessions[uid] {
		_ = c.WriteMessage(websocket.TextMessage, b)
	}
}

// BroadcastAll sends to every /msg subscriber (Java MsgPublisher.broadcast(T)).
func BroadcastAll(payload any) {
	BroadcastMsg("", payload)
}

func BroadcastToken(payload any) {
	b, _ := json.Marshal(payload)
	defaultHub.mu.RLock()
	defer defaultHub.mu.RUnlock()
	for _, set := range defaultHub.tokenSessions {
		for c := range set {
			_ = c.WriteMessage(websocket.TextMessage, b)
		}
	}
}
