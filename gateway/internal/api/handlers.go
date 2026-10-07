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
	Store  session.Store
	Config *config.Config
	Router *router.Engine
	Worker *agentclient.WorkerClient
	Runs   *run.Registry
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
	workerOK := false
	workerErr := ""
	if s.deps.Config.Agents.WebCrawler.Enabled {
		if err := s.deps.Worker.Health(r.Context()); err != nil {
			workerErr = err.Error()
		} else {
			workerOK = true
		}
	}
	runtimeOK := false
	if s.deps.Config.RuntimeAddress != "" {
		if err := tcpReachable(s.deps.Config.RuntimeAddress, 2*time.Second); err == nil {
			runtimeOK = true
		}
	}
	overall := "ok"
	if s.deps.Config.Agents.WebCrawler.Enabled && !workerOK {
		overall = "degraded"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  overall,
		"service": "piper-gateway",
		"phase":   "W3",
		"checks": map[string]any{
			"web_crawler_worker": map[string]any{"ok": workerOK, "address": s.deps.Config.Agents.WebCrawler.Address, "error": workerErr},
			"runtime_grpc":       map[string]any{"ok": runtimeOK, "address": s.deps.Config.RuntimeAddress},
		},
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
	healthy := false
	if s.deps.Config.Agents.WebCrawler.Enabled {
		healthy = s.deps.Worker.Health(r.Context()) == nil
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"agents": []agentInfo{
			{
				ID:          router.AgentWebCrawler,
				DisplayName: "网页采集",
				Description: "自然语言描述采集任务，调用 Piper Runtime 执行抓取",
				Enabled:     s.deps.Config.Agents.WebCrawler.Enabled,
				Healthy:     healthy,
				Address:     s.deps.Config.Agents.WebCrawler.Address,
			},
		},
	})
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

type chatRequest struct {
	ConversationID string `json:"conversation_id"`
	Message        string `json:"message"`
	Stream         bool   `json:"stream"`
}

func (s *Server) chatCompletions(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
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

	_, _ = s.deps.Store.AppendMessage(convID, session.RoleUser, req.Message)

	runID := "r_" + uuid.NewString()
	traceID := traceIDFromContext(r.Context())
	agentID := s.deps.Router.Classify(req.Message)

	log.Printf("[chat] start trace=%s conv=%s run=%s agent=%s msg_len=%d",
		traceID, convID, runID, agentID, len(req.Message))

	if agentID != router.AgentWebCrawler || !s.deps.Config.Agents.WebCrawler.Enabled {
		http.Error(w, "no agent available for this request", http.StatusServiceUnavailable)
		return
	}

	timeout := time.Duration(s.deps.Config.ChatTimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	workerCancel := func() {
		_ = s.deps.Worker.CancelRun(context.Background(), runID)
	}
	s.deps.Runs.Register(runID, cancel, workerCancel)
	defer s.deps.Runs.Unregister(runID)

	grpcReq := &agentv1.ExecuteRequest{
		TraceId:        traceID,
		ConversationId: convID,
		RunId:          runID,
		AgentId:        agentID,
		UserMessage:    req.Message,
	}

	if !req.Stream {
		final, err := s.collectExecute(ctx, grpcReq, nil)
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

	err := s.deps.Worker.Execute(ctx, grpcReq, func(ev *agentv1.ExecuteEvent) error {
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
	req *agentv1.ExecuteRequest,
	onEvent func(*agentv1.ExecuteEvent),
) (string, error) {
	var final strings.Builder
	err := s.deps.Worker.Execute(ctx, req, func(ev *agentv1.ExecuteEvent) error {
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
	h = withMaxBody(deps.Config.MaxRequestBodyBytes, h)
	h = withRateLimit(deps.Config.RateLimitPerMinute, h)
	h = withTraceID(h)
	h = withCORS(h)
	if authToken != "" {
		h = bearerAuth(authToken, h)
	}
	return h
}
