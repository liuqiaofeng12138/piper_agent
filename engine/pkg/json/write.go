package json

import (
	"encoding/json"
	"net/http"
)

const contentTypeJSON = "application/json; charset=utf-8"

// WriteJSON writes a Msg (or any value) as JSON with Piper default headers.
func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", contentTypeJSON)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// WriteMsg writes the standard Piper envelope.
func WriteMsg(w http.ResponseWriter, status int, body Msg) {
	WriteJSON(w, status, body)
}
