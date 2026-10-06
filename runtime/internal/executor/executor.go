package executor

import (
	"context"

	commonv1 "piper_agent/runtime/pkg/pb/common/v1"
	"piper_go/pkg/agentruntime"
)

type Executor struct {
	svc *agentruntime.Service
}

func New(svc *agentruntime.Service) *Executor {
	return &Executor{svc: svc}
}

func (e *Executor) ListTemplates(ctx context.Context, q string, page, size int) ([]map[string]any, int64, error) {
	return e.svc.ListTemplates(ctx, q, page, size)
}

func (e *Executor) GetTemplate(ctx context.Context, id string) (map[string]any, error) {
	return e.svc.GetTemplate(ctx, id)
}

func (e *Executor) UpsertTemplate(ctx context.Context, doc *commonv1.TemplateDoc) (string, error) {
	return e.svc.UpsertTemplate(ctx, protoToDoc(doc))
}

func (e *Executor) ResolveTemplate(ctx context.Context, doc *commonv1.TemplateDoc, templateID string) (map[string]any, error) {
	return e.svc.ResolveTemplate(ctx, protoToDoc(doc), templateID)
}

func (e *Executor) ValidateHTTPBuild(tplDoc map[string]any, vars map[string]any) []*commonv1.Diagnostic {
	diags := e.svc.ValidateTemplate(tplDoc, vars)
	out := make([]*commonv1.Diagnostic, 0, len(diags))
	for _, d := range diags {
		out = append(out, &commonv1.Diagnostic{Code: d.Code, Message: d.Message, Path: d.Path})
	}
	return out
}

func (e *Executor) RunTemplate(ctx context.Context, templateID string, vars map[string]any, engine, sessionID, proxyID string) (string, string, error) {
	return e.svc.RunTemplate(ctx, templateID, vars, engine, sessionID, proxyID)
}

func (e *Executor) ListProxies(ctx context.Context, status string, page, size int) ([]agentruntime.ProxySummary, int64, error) {
	return e.svc.ListProxies(ctx, status, page, size)
}

func (e *Executor) GetToken(ctx context.Context, id string) (map[string]any, error) {
	return e.svc.GetToken(ctx, id)
}

func (e *Executor) GetTokenData(ctx context.Context, tokenID string) (map[string]any, error) {
	return e.svc.GetTokenData(ctx, tokenID)
}

func TemplateDocFromMap(m map[string]any) *commonv1.TemplateDoc {
	d := agentruntime.TemplateDocFromMap(m)
	if d == nil {
		return nil
	}
	return &commonv1.TemplateDoc{
		Id: d.ID, Name: d.Name, Domain: d.Domain, JsonPayload: d.JSONPayload,
	}
}

func protoToDoc(doc *commonv1.TemplateDoc) *agentruntime.TemplateDoc {
	if doc == nil {
		return nil
	}
	return &agentruntime.TemplateDoc{
		ID: doc.GetId(), Name: doc.GetName(), Domain: doc.GetDomain(), JSONPayload: doc.GetJsonPayload(),
	}
}
