package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	agentv1 "piper_agent/gateway/pkg/pb/agent/v1"
	"piper_agent/gateway/internal/agentclient"
	"piper_agent/gateway/internal/config"
	"piper_agent/gateway/internal/router"
	"piper_agent/gateway/internal/run"
	"piper_agent/gateway/internal/session"
)

type Deps struct {
	Store   session.Store
	Config  *config.Config
	Router  *router.Engine
	Workers *agentclient.Pool
	Runs    *run.Registry
}

type Server struct {
	deps *Deps
}

func NewServer(deps *Deps) *Server {
	return &Server{deps: deps}
}

type agentInfo struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Healthy     bool   `json:"healthy"`
	Address     string `json:"address"`
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/health", s.health)
	mux.HandleFunc("GET /api/v1/agents", s.listAgents)
	mux.HandleFunc("POST /api/v1/conversations", s.createConversation)
	mux.HandleFunc("GET /api/v1/conversations", s.listConversations)
	mux.HandleFunc("GET /api/v1/conversations/{id}/messages", s.listMessages)
	mux.HandleFunc("DELETE /api/v1/conversations/{id}", s.deleteConversation)
	mux.HandleFunc("POST /api/v1/chat/completions", s.chatCompletions)
	mux.HandleFunc("POST /api/v1/runs/{id}/cancel", s.cancelRun)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	checks := map[string]any{}
	overall := "ok"
	for _, a := range s.deps.Config.EnabledAgents() {
		workerErr := ""
		workerOK := false
		if err := s.deps.Workers.Health(r.Context(), a.ID); err != nil {
			workerErr = err.Error()
			overall = "degraded"
		} else {
			workerOK = true
		}
		checks[a.ID+"_worker"] = map[string]any{"ok": workerOK, "address": a.Address, "error": workerErr}
	}
	runtimeOK := false
	if s.deps.Config.RuntimeAddress != "" {
		if err := tcpReachable(s.deps.Config.RuntimeAddress, 2*time.Second); err == nil {
			runtimeOK = true
		}
	}
	checks["runtime_grpc"] = map[string]any{"ok": runtimeOK, "address": s.deps.Config.RuntimeAddress}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  overall,
		"service": "piper-gateway",
		"phase":   "W4",
		"checks":  checks,
	})
}

func tcpReachable(addr string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return err
	}
	return conn.Close()
}

func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	agents := make([]agentInfo, 0, len(s.deps.Config.Agents))
	for _, a := range s.deps.Config.Agents {
		healthy := false
		if a.Enabled {
			healthy = s.deps.Workers.Health(r.Context(), a.ID) == nil
		}
		agents = append(agents, agentInfo{
			ID:          a.ID,
			DisplayName: a.DisplayName,
			Description: a.Description,
			Enabled:     a.Enabled,
			Healthy:     healthy,
			Address:     a.Address,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"agents": agents})
}

func (s *Server) createConversation(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	c := s.deps.Store.CreateConversation(body.Title)
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) listConversations(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"conversations": session.CoalesceConversations(s.deps.Store.ListConversations()),
	})
}

func (s *Server) listMessages(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := s.deps.Store.GetConversation(id); !ok {
		http.Error(w, "conversation not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"conversation_id": id,
		"messages":        session.CoalesceMessages(s.deps.Store.ListMessages(id)),
	})
}

func (s *Server) deleteConversation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.deps.Store.DeleteConversation(id) {
		http.Error(w, "conversation not found", http.StatusNotFound)
		return
	}
	log.Printf("[chat] conversation deleted id=%s", id)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) chatCompletions(w http.ResponseWriter, r *http.Request) {
	parsed, err := parseChatRequest(r, s.deps.Config.MaxUploadBodyBytes)
	if err != nil {
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}
	req := parsed.Request
	req.Message = strings.TrimSpace(req.Message)
	docCount := len(parsed.Documents)
	if req.Message == "" && docCount == 0 {
		http.Error(w, "message or files required", http.StatusBadRequest)
		return
	}

	convID := req.ConversationID
	if convID == "" {
		c := s.deps.Store.CreateConversation("")
		convID = c.ID
	} else if _, ok := s.deps.Store.GetConversation(convID); !ok {
		http.Error(w, "conversation not found", http.StatusNotFound)
		return
	}

	userDisplay := req.Message
	if docCount > 0 {
		names := make([]string, 0, docCount)
		for _, d := range parsed.Documents {
			if d.Filename != "" {
				names = append(names, d.Filename)
			}
		}
		if len(names) > 0 {
			userDisplay = strings.TrimSpace(req.Message + "\n\n[附件: " + strings.Join(names, ", ") + "]")
		}
	}
	_, _ = s.deps.Store.AppendMessage(convID, session.RoleUser, userDisplay)

	runID := "r_" + uuid.NewString()
	traceID := traceIDFromContext(r.Context())
	agentID, routeReason := resolveChatAgent(
		r.Context(),
		s.deps.Store,
		s.deps.Config,
		s.deps.Router,
		convID,
		req.Message,
		docCount,
	)

	log.Printf("[chat] start trace=%s conv=%s run=%s agent=%s route=%q msg_len=%d",
		traceID, convID, runID, agentID, routeReason, len(req.Message))

	worker := s.deps.Workers.Get(agentID)
	if agentID == "" || worker == nil {
		http.Error(w, "no agent available for this request", http.StatusServiceUnavailable)
		return
	}

	timeout := time.Duration(s.deps.Config.ChatTimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	workerCancel := func() {
		_ = worker.CancelRun(context.Background(), runID)
	}
	s.deps.Runs.Register(runID, cancel, workerCancel)
	defer s.deps.Runs.Unregister(runID)

	grpcReq := &agentv1.ExecuteRequest{
		TraceId:        traceID,
		ConversationId: convID,
		RunId:          runID,
		AgentId:        agentID,
		UserMessage:    req.Message,
		Documents:      parsed.Documents,
	}

	if !req.Stream {
		final, err := s.collectExecute(ctx, worker, grpcReq, nil)
		if err != nil {
			log.Printf("[chat] error trace=%s run=%s err=%v", traceID, runID, err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		log.Printf("[chat] done trace=%s run=%s reply_len=%d", traceID, runID, len(final))
		_, _ = s.deps.Store.AppendMessage(convID, session.RoleAssistant, final)
		writeJSON(w, http.StatusOK, map[string]any{
			"conversation_id": convID,
			"run_id":          runID,
			"agent_id":        agentID,
			"message":         final,
		})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	var assistant strings.Builder
	send := func(ev map[string]any) {
		ev["trace_id"] = traceID
		ev["conversation_id"] = convID
		ev["run_id"] = runID
		ev["agent_id"] = agentID
		b, _ := json.Marshal(ev)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
	}

	err = worker.Execute(ctx, grpcReq, func(ev *agentv1.ExecuteEvent) error {
		if r.Context().Err() != nil {
			return r.Context().Err()
		}
		switch ev.Type {
		case "tool.call":
			log.Printf("[chat] tool trace=%s run=%s tool=%s", traceID, runID, ev.ToolName)
		case "run.progress":
			log.Printf("[chat] progress trace=%s run=%s tool=%s msg=%s", traceID, runID, ev.ToolName, truncateLog(ev.Message, 120))
		case "error":
			log.Printf("[chat] agent_error trace=%s run=%s msg=%s", traceID, runID, ev.Message)
		}
		payload := map[string]any{"type": ev.Type}
		if ev.Delta != "" {
			payload["delta"] = ev.Delta
			assistant.WriteString(ev.Delta)
		}
		if ev.Content != "" {
			payload["content"] = ev.Content
		}
		if ev.ToolName != "" {
			payload["tool_name"] = ev.ToolName
		}
		if ev.ToolDetail != "" {
			payload["tool_detail"] = ev.ToolDetail
		}
		if ev.Message != "" {
			payload["message"] = ev.Message
		}
		send(payload)
		if ev.Type == "message.done" && ev.Content != "" {
			assistant.Reset()
			assistant.WriteString(ev.Content)
		}
		return nil
	})
	if err != nil && !errors.Is(err, io.EOF) && status.Code(err) != codes.Canceled {
		log.Printf("[chat] stream_error trace=%s run=%s err=%v", traceID, runID, err)
		send(map[string]any{"type": "error", "message": err.Error()})
	}

	final := strings.TrimSpace(assistant.String())
	if final != "" {
		_, _ = s.deps.Store.AppendMessage(convID, session.RoleAssistant, final)
	}
	send(map[string]any{"type": "stream.end"})
	log.Printf("[chat] done trace=%s run=%s reply_len=%d", traceID, runID, len(final))
}

func truncateLog(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func (s *Server) collectExecute(
	ctx context.Context,
	worker *agentclient.WorkerClient,
	req *agentv1.ExecuteRequest,
	onEvent func(*agentv1.ExecuteEvent),
) (string, error) {
	var final strings.Builder
	err := worker.Execute(ctx, req, func(ev *agentv1.ExecuteEvent) error {
		if onEvent != nil {
			onEvent(ev)
		}
		if ev.Delta != "" {
			final.WriteString(ev.Delta)
		}
		if ev.Type == "message.done" && ev.Content != "" {
			final.Reset()
			final.WriteString(ev.Content)
		}
		return nil
	})
	if err != nil && !errors.Is(err, io.EOF) && status.Code(err) != codes.Canceled {
		return "", err
	}
	out := strings.TrimSpace(final.String())
	if out == "" {
		out = "（无文本回复）"
	}
	return out, nil
}

func (s *Server) cancelRun(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	ok := s.deps.Runs.Cancel(runID)
	log.Printf("[chat] cancel run=%s ok=%v", runID, ok)
	writeJSON(w, http.StatusOK, map[string]any{
		"run_id":  runID,
		"status":  "cancelled",
		"ok":      ok,
		"message": "cancel signal sent",
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func NewHandler(deps *Deps, authToken string) http.Handler {
	srv := NewServer(deps)
	mux := http.NewServeMux()
	srv.Register(mux)
	var h http.Handler = mux
	h = withMaxBody(deps.Config.MaxRequestBodyBytes, deps.Config.MaxUploadBodyBytes, h)
	h = withRateLimit(deps.Config.RateLimitPerMinute, h)
	h = withTraceID(h)
	h = withCORS(h)
	if authToken != "" {
		h = bearerAuth(authToken, h)
	}
	return h
}
