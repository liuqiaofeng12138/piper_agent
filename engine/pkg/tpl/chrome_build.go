package tpl

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// BuildChromeToken builds a Chrome token (Java Builder.Chrome.build / Token.Chrome).
func BuildChromeToken(tpl map[string]any, vars map[string]any, opts RunOpts) (map[string]any, error) {
	builder, _ := tpl["builder"].(map[string]any)
	if builder == nil {
		return nil, errors.New("template missing builder")
	}
	if btype, _ := builder["type"].(string); btype != "Chrome" {
		return nil, fmt.Errorf("unsupported builder type %q", btype)
	}
	urlTpl, _ := builder["url_tpl"].(string)
	if urlTpl == "" {
		return nil, errors.New("chrome builder missing url_tpl")
	}
	finalURL := ReplaceVars(urlTpl, vars, true)
	domainTpl, _ := builder["domain"].(string)
	domain := ReplaceVars(domainTpl, vars, false)
	if domain == "" {
		if u, err := url.Parse(finalURL); err == nil {
			domain = u.Hostname()
		}
	}
	usernameTpl, _ := builder["username"].(string)
	username := ReplaceVars(usernameTpl, vars, false)

	tplID, _ := tpl["id"].(string)
	now := time.Now().UnixMilli()
	ttl := toInt(builder["ttl"], 0)
	if ttl == 0 {
		ttl = 255
	}
	flags := builder["flags"]
	token := map[string]any{
		"type":        "Chrome",
		"tpl_id":      tplID,
		"url":         finalURL,
		"domain":      domain,
		"username":    username,
		"vars":        vars,
		"behavior":    opts.Behavior,
		"task_id":     opts.TaskID,
		"uid":         opts.UID,
		"node_id":     opts.NodeID,
		"agent_id":    opts.AgentID,
		"success":     false,
		"ttl":         ttl,
		"flags":       flags,
		"create_time": now,
		"update_time": now,
		"logs":        map[string]any{},
		"r": map[string]any{
			"uri":    finalURL,
			"method": http.MethodGet,
		},
	}
	if opts.GenID != "" {
		token["gen_id"] = opts.GenID
	}
	AssignTokenID(token)
	return token, nil
}

// BuildToken picks HTTP or Chrome builder.
func BuildToken(tpl map[string]any, vars map[string]any, opts RunOpts) (map[string]any, error) {
	switch BuilderType(tpl) {
	case "Chrome":
		return BuildChromeToken(tpl, vars, opts)
	default:
		return BuildHTTPToken(tpl, vars, opts)
	}
}
