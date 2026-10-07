package tpl

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"piper_go/pkg/util"
)

// BuildHTTPToken builds a token document from template JSON (Phase 2 HTTP path).
func BuildHTTPToken(tpl map[string]any, vars map[string]any, opts RunOpts) (map[string]any, error) {
	builder, _ := tpl["builder"].(map[string]any)
	if builder == nil {
		return nil, errors.New("template missing builder")
	}
	btype, _ := builder["type"].(string)
	if btype != "Http" && btype != "" {
		return nil, fmt.Errorf("unsupported builder type %q", btype)
	}
	urlTpl, _ := builder["url_tpl"].(string)
	if urlTpl == "" {
		return nil, errors.New("http builder missing url_tpl")
	}
	method, _ := builder["http_method"].(string)
	if method == "" {
		method = http.MethodGet
	}
	bodyTpl, _ := builder["body_tpl"].(string)
	headersTpl, _ := builder["headers_tpl"].(map[string]any)

	finalURL := ReplaceVars(urlTpl, vars, true)
	finalURL = strings.ReplaceAll(finalURL, "|", "%7c")
	finalURL = strings.ReplaceAll(finalURL, ";", "%3b")
	u, err := url.Parse(finalURL)
	if err != nil {
		return nil, err
	}
	domain := u.Hostname()

	headers := map[string]any{}
	for k, v := range headersTpl {
		headers[ReplaceVars(k, vars, false)] = ReplaceVars(fmt.Sprint(v), vars, false)
	}
	body := ReplaceVars(bodyTpl, vars, false)

	tplID, _ := tpl["id"].(string)
	now := time.Now().UnixMilli()
	token := map[string]any{
		"type":        "Http",
		"tpl_id":      tplID,
		"domain":      domain,
		"vars":        vars,
		"behavior":    opts.Behavior,
		"task_id":     opts.TaskID,
		"uid":         opts.UID,
		"node_id":     opts.NodeID,
		"agent_id":    opts.AgentID,
		"proxy_id":    opts.ProxyID,
		"success":     false,
		"create_time": now,
		"update_time": now,
		"logs":        map[string]any{},
		"r": map[string]any{
			"uri":     finalURL,
			"method":  method,
			"headers": headers,
			"body":    body,
		},
	}
	if opts.GenID != "" {
		token["gen_id"] = opts.GenID
	}
	AssignTokenID(token)
	return token, nil
}

// AssignTokenID sets fingerprint and id (Java Token.setId).
func AssignTokenID(token map[string]any) {
	tplID, _ := token["tpl_id"].(string)
	domain, _ := token["domain"].(string)
	username, _ := token["username"].(string)
	vars, _ := token["vars"].(map[string]any)
	varsJSON, _ := json.Marshal(vars)
	fp := util.MD5Hex(fmt.Sprintf("%s::%s::%s::%s", tplID, domain, username, string(varsJSON)))
	var ct int64
	switch t := token["create_time"].(type) {
	case int64:
		ct = t
	case float64:
		ct = int64(t)
	default:
		ct = time.Now().UnixMilli()
	}
	token["fingerprint"] = fp
	token["id"] = fp + fmt.Sprintf("%016x", ct)
}

type RunOpts struct {
	UID      string
	TaskID   string
	NodeID   string
	AgentID  string
	ProxyID  string
	GenID    string
	Behavior string // TEST or DEFAULT
}

func ReplaceVars(tpl string, vars map[string]any, isURL bool) string {
	if tpl == "" {
		return tpl
	}
	out := tpl
	for k, v := range vars {
		repl := fmt.Sprint(v)
		if isURL {
			repl = strings.ReplaceAll(repl, " ", "+")
		}
		out = strings.ReplaceAll(out, "{{"+k+"}}", repl)
	}
	return out
}

func BuilderType(tpl map[string]any) string {
	builder, _ := tpl["builder"].(map[string]any)
	if builder == nil {
		return ""
	}
	t, _ := builder["type"].(string)
	return t
}
