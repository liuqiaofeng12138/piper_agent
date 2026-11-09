package crawlerrt

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"piper_agent/runtime/pkg/embed"
	commonv1 "piper_agent/runtime/pkg/pb/common/v1"
	runtimev1 "piper_agent/runtime/pkg/pb/runtime/v1"
)

// Client calls embedded Runtime services in-process (no gRPC).
type Client struct {
	svc *embed.Services
}

func New(svc *embed.Services) *Client {
	return &Client{svc: svc}
}

func reqID() string { return uuid.NewString() }

func (c *Client) ValidateTemplate(ctx context.Context, templateID, name, domain string, jsonPayload []byte, vars map[string]string, metaOnly bool) (*runtimev1.ValidateTemplateResponse, error) {
	var doc *commonv1.TemplateDoc
	if !metaOnly {
		doc = &commonv1.TemplateDoc{
			Id:          templateID,
			Name:        name,
			Domain:      domain,
			JsonPayload: jsonPayload,
		}
	}
	req := &runtimev1.ValidateTemplateRequest{
		RequestId:  reqID(),
		Template:   doc,
		TemplateId: templateID,
		Vars:       &commonv1.Vars{Values: vars},
	}
	if metaOnly {
		req.Template = nil
		req.TemplateId = templateID
	}
	return c.svc.Template.ValidateTemplate(ctx, req)
}

func (c *Client) UpsertTemplate(ctx context.Context, templateID, name string, jsonPayload []byte) (*runtimev1.UpsertTemplateResponse, error) {
	req := &runtimev1.UpsertTemplateRequest{
		RequestId: reqID(),
		Template: &commonv1.TemplateDoc{
			Id:          templateID,
			Name:        name,
			JsonPayload: jsonPayload,
		},
	}
	return c.svc.Template.UpsertTemplate(ctx, req)
}

func (c *Client) RunTemplate(ctx context.Context, templateID, sessionID, engine, proxyID, idempotency string, vars map[string]string) (*runtimev1.RunTemplateResponse, error) {
	req := &runtimev1.RunTemplateRequest{
		RequestId:      reqID(),
		SessionId:      sessionID,
		IdempotencyKey: idempotency,
		TemplateId:     templateID,
		Vars:           &commonv1.Vars{Values: vars},
		Engine:         engine,
		ProxyId:        proxyID,
	}
	return c.svc.Execute.RunTemplate(ctx, req)
}

func (c *Client) AwaitRun(ctx context.Context, runID string) ([]*runtimev1.RunEvent, commonv1.RunPhase, error) {
	if p := c.svc.Piper(); p != nil {
		return p.AwaitRun(ctx, runID)
	}
	return nil, commonv1.RunPhase_RUN_PHASE_UNSPECIFIED, fmt.Errorf("AwaitRun requires non-mock embedded runtime")
}

func (c *Client) GetTokenData(ctx context.Context, tokenID string) (*runtimev1.GetTokenDataResponse, error) {
	if c.svc.Data == nil {
		return nil, context.Canceled
	}
	return c.svc.Data.GetTokenData(ctx, &runtimev1.GetTokenDataRequest{RequestId: reqID(), TokenId: tokenID})
}

func (c *Client) GetTemplate(ctx context.Context, templateID string) (*runtimev1.GetTemplateResponse, error) {
	if c.svc.Meta == nil {
		return nil, context.Canceled
	}
	return c.svc.Meta.GetTemplate(ctx, &runtimev1.GetTemplateRequest{RequestId: reqID(), TemplateId: templateID})
}

func (c *Client) ListProxies(ctx context.Context, size int, status string) (*runtimev1.ListProxiesResponse, error) {
	if c.svc.Meta == nil {
		return &runtimev1.ListProxiesResponse{}, nil
	}
	return c.svc.Meta.ListProxies(ctx, &runtimev1.ListProxiesRequest{
		RequestId: reqID(),
		Page:      1,
		Size:      int32(size),
		Status:    status,
	})
}
