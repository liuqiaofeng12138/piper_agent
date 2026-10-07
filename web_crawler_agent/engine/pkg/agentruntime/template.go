package agentruntime

import (
	"context"
	"encoding/json"
	"fmt"

	"piper_go/pkg/chrome"
	"piper_go/pkg/db/meta"
	"piper_go/pkg/distributor/cache"
	"piper_go/pkg/tpl"
)

type Diagnostic struct {
	Code    string
	Message string
	Path    string
}

type TemplateDoc struct {
	ID          string
	Name        string
	Domain      string
	JSONPayload []byte
}

func (s *Service) ListTemplates(_ context.Context, q string, page, size int) ([]map[string]any, int64, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	return s.ctx.Meta.Query(meta.TableTemplates, meta.QueryOpts{Q: q, Page: int64(page), Size: int64(size)})
}

func (s *Service) GetTemplate(_ context.Context, id string) (map[string]any, error) {
	if doc := cache.GetTemplate(id); doc != nil {
		return doc, nil
	}
	return s.ctx.Meta.Get(meta.TableTemplates, id)
}

func (s *Service) UpsertTemplate(_ context.Context, doc *TemplateDoc) (string, error) {
	m, err := templateMapFromDoc(doc)
	if err != nil {
		return "", err
	}
	if err := s.ctx.Meta.Upsert(meta.TableTemplates, m); err != nil {
		return "", err
	}
	cache.PutTemplate(m)
	id, _ := m["id"].(string)
	return id, nil
}

func (s *Service) ResolveTemplate(_ context.Context, doc *TemplateDoc, templateID string) (map[string]any, error) {
	if doc != nil && (doc.ID != "" || doc.Name != "" || len(doc.JSONPayload) > 0) {
		return templateMapFromDoc(doc)
	}
	if templateID == "" {
		return nil, fmt.Errorf("template_id or template body required")
	}
	return s.GetTemplate(context.Background(), templateID)
}

func (s *Service) ValidateBuild(tplDoc map[string]any, vars map[string]any) []Diagnostic {
	btype := tpl.BuilderType(tplDoc)
	if btype == "Chrome" {
		cd := chrome.Default()
		if cd == nil || cd.AgentCount() == 0 {
			return []Diagnostic{{
				Code: "CHROME_UNAVAILABLE", Message: "Chrome.enabled is false or no chrome agents", Path: "builder.type",
			}}
		}
		_, err := tpl.BuildChromeToken(tplDoc, vars, tpl.RunOpts{Behavior: "TEST", AgentID: cd.PickAgentID("")})
		if err != nil {
			return []Diagnostic{{Code: "BUILD_TOKEN_FAILED", Message: err.Error(), Path: "builder"}}
		}
		return nil
	}
	s.EnsureHTTPAgent()
	_, err := tpl.BuildHTTPToken(tplDoc, vars, tpl.RunOpts{Behavior: "TEST", AgentID: s.HTTAgentID()})
	if err != nil {
		return []Diagnostic{{Code: "BUILD_TOKEN_FAILED", Message: err.Error(), Path: "builder"}}
	}
	return nil
}

func TemplateDocFromMap(m map[string]any) *TemplateDoc {
	if m == nil {
		return nil
	}
	b, _ := json.Marshal(m)
	id, _ := m["id"].(string)
	name, _ := m["name"].(string)
	domain, _ := m["domain"].(string)
	return &TemplateDoc{ID: id, Name: name, Domain: domain, JSONPayload: b}
}

func templateMapFromDoc(doc *TemplateDoc) (map[string]any, error) {
	if doc == nil {
		return nil, fmt.Errorf("template is nil")
	}
	out := map[string]any{}
	if len(doc.JSONPayload) > 0 {
		if err := json.Unmarshal(doc.JSONPayload, &out); err != nil {
			return nil, err
		}
	}
	if doc.ID != "" {
		out["id"] = doc.ID
	}
	if doc.Name != "" {
		out["name"] = doc.Name
	}
	if doc.Domain != "" {
		out["domain"] = doc.Domain
	}
	if out["id"] == nil && out["name"] == nil {
		return nil, fmt.Errorf("template id or name required")
	}
	return out, nil
}
