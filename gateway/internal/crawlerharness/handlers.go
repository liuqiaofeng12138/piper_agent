package crawlerharness

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	commonv1 "piper_agent/runtime/pkg/pb/common/v1"

	"piper_agent/gateway/internal/crawlerrt"
)

type ToolHandlers struct {
	client  *crawlerrt.Client
	session *Session
	cfg     Config
}

func NewToolHandlers(client *crawlerrt.Client, session *Session, cfg Config) *ToolHandlers {
	return &ToolHandlers{client: client, session: session, cfg: cfg}
}

func (h *ToolHandlers) Dispatch(name string, args map[string]any) string {
	switch name {
	case "site_probe":
		return h.siteProbe(args)
	case "search_similar_templates":
		return h.searchSimilarTemplates(args)
	case "load_shared_example_template":
		return h.loadSharedExample(args)
	case "template_author_validate":
		return h.templateValidate(args)
	case "template_author_save":
		return jsonErr("template_author_save not implemented in embedded worker; use validate + run")
	case "param_filler_suggest":
		return h.paramFillerSuggest(args)
	case "list_proxies":
		return h.listProxies(args)
	case "select_proxy":
		return h.selectProxy(args)
	case "runner_execute":
		return h.runnerExecute(args)
	case "runner_wait_for_complete":
		return h.runnerWait(args)
	case "runner_get_data":
		return h.runnerGetData(args)
	default:
		return jsonErr("unknown tool " + name)
	}
}

func jsonErr(msg string) string {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return string(b)
}

func (h *ToolHandlers) siteProbe(args map[string]any) string {
	url := strings.TrimSpace(fmt.Sprint(args["url"]))
	if url == "" {
		url = strings.TrimSpace(fmt.Sprint(args["target"]))
	}
	if url == "" {
		return jsonErr("url required")
	}
	timeout := h.cfg.Harness.SiteProbeTimeoutSeconds
	if v, ok := args["timeout_seconds"].(float64); ok && v > 0 {
		timeout = v
	}
	maxBytes := h.cfg.Harness.SiteProbeMaxBytes
	if v, ok := args["max_body_bytes"].(float64); ok && v > 0 {
		maxBytes = int(v)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout*float64(time.Second)))
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return jsonErr(err.Error())
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return jsonErr(err.Error())
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, int64(maxBytes)))
	out := map[string]any{
		"ok":           true,
		"url":          url,
		"status_code":  resp.StatusCode,
		"content_type": resp.Header.Get("Content-Type"),
		"body_preview": string(body[:min(len(body), 2000)]),
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (h *ToolHandlers) searchSimilarTemplates(args map[string]any) string {
	query := strings.ToLower(fmt.Sprint(args["query"]))
	topK := 5
	if v, ok := intFrom(args["top_k"]); ok && v > 0 {
		topK = v
	}
	type hit struct {
		TemplateID string  `json:"template_id"`
		Name       string  `json:"name"`
		Score      float64 `json:"score"`
		Source     string  `json:"source"`
		Snippet    string  `json:"snippet"`
	}
	var hits []hit
	entries, _ := filepath.Glob(filepath.Join(h.cfg.ExamplesDir, "*.json"))
	for _, path := range entries {
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		name := filepath.Base(path)
		text := strings.ToLower(name + " " + string(b))
		score := 0.0
		for _, word := range strings.Fields(query) {
			if len(word) > 2 && strings.Contains(text, word) {
				score += 1
			}
		}
		if score == 0 && query != "" {
			continue
		}
		var doc map[string]any
		_ = json.Unmarshal(b, &doc)
		tid := fmt.Sprint(doc["id"])
		if tid == "" {
			tid = strings.TrimSuffix(name, ".json")
		}
		hits = append(hits, hit{
			TemplateID: tid,
			Name:       fmt.Sprint(doc["name"]),
			Score:      score,
			Source:     "shared/examples",
			Snippet:    name,
		})
	}
	if len(hits) > topK {
		hits = hits[:topK]
	}
	b, _ := json.Marshal(hits)
	return string(b)
}

func (h *ToolHandlers) loadSharedExample(args map[string]any) string {
	key := fmt.Sprint(args["example_id"])
	if key == "" {
		key = fmt.Sprint(args["filename"])
	}
	if key == "" {
		return jsonErr("example_id or filename required")
	}
	candidates := []string{
		filepath.Join(h.cfg.ExamplesDir, key),
		filepath.Join(h.cfg.ExamplesDir, key+".json"),
	}
	var b []byte
	var filename string
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err == nil {
			b = data
			filename = filepath.Base(p)
			break
		}
	}
	if b == nil {
		return jsonErr("shared example not found: " + key)
	}
	var doc map[string]any
	if json.Unmarshal(b, &doc) != nil {
		return jsonErr("invalid json")
	}
	h.session.PendingTemplate = doc
	tid := fmt.Sprint(doc["id"])
	if tid == "" {
		tid = strings.TrimSuffix(filename, ".json")
	}
	out := map[string]any{
		"template_id":   tid,
		"source":        "shared/examples/templates/" + filename,
		"template_json": string(b),
		"use_verbatim":  true,
	}
	resp, _ := json.Marshal(out)
	return string(resp)
}

func (h *ToolHandlers) templateValidate(args map[string]any) string {
	ctx := context.Background()
	vars := map[string]string{}
	if raw, ok := args["vars"].(map[string]any); ok {
		for k, v := range raw {
			vars[k] = fmt.Sprint(v)
		}
	}
	tid := fmt.Sprint(args["template_id"])
	raw := args["template_json"]
	if raw != nil {
		var doc map[string]any
		switch t := raw.(type) {
		case string:
			_ = json.Unmarshal([]byte(t), &doc)
		default:
			doc, _ = raw.(map[string]any)
		}
		if doc != nil {
			tid = fmt.Sprint(doc["id"])
			if tid == "" {
				tid = fmt.Sprint(doc["name"])
			}
			payload, _ := json.Marshal(doc)
			resp, err := h.client.ValidateTemplate(ctx, tid, fmt.Sprint(doc["name"]), fmt.Sprint(doc["domain"]), payload, vars, false)
			if err != nil {
				return jsonErr(err.Error())
			}
			if resp.GetOk() && tid != "" {
				h.session.ValidatedTemplateIDs[tid] = struct{}{}
				h.session.LastTemplateID = tid
				h.session.PendingTemplate = doc
			}
			return formatValidateResp(resp, tid)
		}
	}
	if tid != "" {
		resp, err := h.client.ValidateTemplate(ctx, tid, "", "", nil, vars, true)
		if err != nil {
			return jsonErr(err.Error())
		}
		if resp.GetOk() {
			h.session.ValidatedTemplateIDs[tid] = struct{}{}
			h.session.LastTemplateID = tid
		}
		return formatValidateResp(resp, tid)
	}
	return jsonErr("template_json or template_id required")
}

func formatValidateResp(resp interface {
	GetOk() bool
	GetDiagnostics() []*commonv1.Diagnostic
}, tid string) string {
	diags := make([]map[string]string, 0)
	for _, d := range resp.GetDiagnostics() {
		diags = append(diags, map[string]string{"code": d.GetCode(), "path": d.GetPath(), "message": d.GetMessage()})
	}
	out := map[string]any{"ok": resp.GetOk(), "template_id": tid, "diagnostics": diags}
	b, _ := json.Marshal(out)
	return string(b)
}

func (h *ToolHandlers) paramFillerSuggest(args map[string]any) string {
	text := fmt.Sprint(args["user_text"])
	vars := map[string]string{}
	if m := regexp.MustCompile(`https?://[^\s"']+`).FindString(text); m != "" {
		vars["url"] = m
	}
	out := map[string]any{"vars": vars}
	b, _ := json.Marshal(out)
	return string(b)
}

func (h *ToolHandlers) listProxies(args map[string]any) string {
	size := 20
	if v, ok := intFrom(args["size"]); ok {
		size = v
	}
	resp, err := h.client.ListProxies(context.Background(), size, fmt.Sprint(args["status"]))
	if err != nil {
		return jsonErr(err.Error())
	}
	rows := make([]map[string]string, 0, len(resp.GetProxies()))
	for _, p := range resp.GetProxies() {
		rows = append(rows, map[string]string{"id": p.GetId(), "name": p.GetName(), "domain": p.GetDomain(), "status": p.GetStatus()})
	}
	out := map[string]any{"proxies": rows, "total": resp.GetTotal()}
	b, _ := json.Marshal(out)
	return string(b)
}

func (h *ToolHandlers) selectProxy(args map[string]any) string {
	pid := fmt.Sprint(args["proxy_id"])
	if pid == "" {
		return jsonErr("proxy_id required")
	}
	h.session.SelectedProxyID = pid
	b, _ := json.Marshal(map[string]string{"selected_proxy_id": pid})
	return string(b)
}

func (h *ToolHandlers) runnerExecute(args map[string]any) string {
	tid := fmt.Sprint(args["template_id"])
	if tid == "" {
		tid = h.session.LastTemplateID
	}
	if tid == "" {
		return jsonErr("template_id required")
	}
	if h.cfg.Harness.RequireValidateBeforeRun {
		if _, ok := h.session.ValidatedTemplateIDs[tid]; !ok {
			return jsonErr("template " + tid + " not validated; call template_author_validate first")
		}
	}
	if h.cfg.Harness.MaxRunsPerSession > 0 && h.session.RunCount >= h.cfg.Harness.MaxRunsPerSession {
		return jsonErr(fmt.Sprintf("session run limit reached (%d)", h.cfg.Harness.MaxRunsPerSession))
	}
	vars := map[string]string{}
	if raw, ok := args["vars"].(map[string]any); ok {
		for k, v := range raw {
			vars[k] = fmt.Sprint(v)
		}
	}
	engine := fmt.Sprint(args["engine"])
	if engine == "" || engine == "auto" {
		engine = "http"
		if h.session.PendingTemplate != nil {
			if b, ok := h.session.PendingTemplate["builder"].(map[string]any); ok {
				if strings.EqualFold(fmt.Sprint(b["type"]), "chrome") {
					engine = "chrome"
				}
			}
		}
	}
	proxyID := fmt.Sprint(args["proxy_id"])
	if proxyID == "" {
		proxyID = h.session.SelectedProxyID
	}
	resp, err := h.client.RunTemplate(context.Background(), tid, h.session.SessionID, engine, proxyID, "", vars)
	if err != nil {
		return jsonErr(err.Error())
	}
	h.session.RunCount++
	h.session.LastRunID = resp.GetRunId()
	if resp.GetStatus() != nil {
		h.session.LastTokenID = resp.GetStatus().GetTokenId()
	}
	out := map[string]any{
		"run_id":    resp.GetRunId(),
		"token_id":  resp.GetStatus().GetTokenId(),
		"phase":     int(resp.GetStatus().GetPhase()),
		"engine":    engine,
		"proxy_id":  proxyID,
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func (h *ToolHandlers) runnerWait(args map[string]any) string {
	runID := fmt.Sprint(args["run_id"])
	if runID == "" {
		runID = h.session.LastRunID
	}
	if runID == "" {
		return jsonErr("no run_id")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	events, phase, err := h.client.AwaitRun(ctx, runID)
	if err != nil {
		return jsonErr(err.Error())
	}
	phases := make([]string, 0, len(events))
	var messages []string
	for _, ev := range events {
		phases = append(phases, phase.String())
		if ev.GetMessage() != "" {
			messages = append(messages, ev.GetMessage())
		}
	}
	final := phase.String()
	if len(phases) > 0 {
		final = phases[len(phases)-1]
	}
	if len(messages) > 5 {
		messages = messages[len(messages)-5:]
	}
	out := map[string]any{"run_id": runID, "phases": phases, "messages": messages, "final": final}
	b, _ := json.Marshal(out)
	return string(b)
}

func (h *ToolHandlers) runnerGetData(args map[string]any) string {
	tokenID := fmt.Sprint(args["token_id"])
	if tokenID == "" {
		tokenID = h.session.LastTokenID
	}
	if tokenID == "" {
		tokenID = h.session.LastRunID
	}
	if tokenID == "" {
		return jsonErr("no token_id")
	}
	resp, err := h.client.GetTokenData(context.Background(), tokenID)
	if err != nil {
		return jsonErr(err.Error())
	}
	var payload any
	if json.Unmarshal(resp.GetPayloadJson(), &payload) != nil {
		payload = map[string]string{"raw": string(resp.GetPayloadJson())}
	}
	b, _ := json.Marshal(payload)
	if len(b) > 8000 {
		b = b[:8000]
	}
	return string(b)
}
