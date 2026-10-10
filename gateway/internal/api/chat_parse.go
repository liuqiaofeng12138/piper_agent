package api

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	agentv1 "piper_agent/gateway/pkg/pb/agent/v1"
)

type chatRequest struct {
	ConversationID string `json:"conversation_id"`
	Message        string `json:"message"`
	Stream         bool   `json:"stream"`
}

type parsedChat struct {
	Request   chatRequest
	Documents []*agentv1.UploadedDocument
}

func parseChatRequest(r *http.Request, maxUploadBytes int64) (parsedChat, error) {
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		return parseMultipartChat(r, maxUploadBytes)
	}
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return parsedChat{}, err
	}
	if !req.Stream {
		req.Stream = true
	}
	return parsedChat{Request: req}, nil
}

func parseMultipartChat(r *http.Request, maxUploadBytes int64) (parsedChat, error) {
	if maxUploadBytes <= 0 {
		maxUploadBytes = 32 << 20
	}
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		return parsedChat{}, err
	}
	req := chatRequest{
		ConversationID: strings.TrimSpace(r.FormValue("conversation_id")),
		Message:        strings.TrimSpace(r.FormValue("message")),
		Stream:         true,
	}
	if r.FormValue("stream") == "false" {
		req.Stream = false
	}

	var docs []*agentv1.UploadedDocument
	if r.MultipartForm != nil {
		for _, headers := range r.MultipartForm.File["files"] {
			doc, err := readUploadedFile(headers)
			if err != nil {
				return parsedChat{}, err
			}
			docs = append(docs, doc)
		}
	}
	return parsedChat{Request: req, Documents: docs}, nil
}

func readUploadedFile(fh *multipart.FileHeader) (*agentv1.UploadedDocument, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 32<<20))
	if err != nil {
		return nil, err
	}
	mime := fh.Header.Get("Content-Type")
	return &agentv1.UploadedDocument{
		Filename: fh.Filename,
		MimeType: mime,
		Data:     data,
	}, nil
}
