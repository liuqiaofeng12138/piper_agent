package mock

import (
	"context"
	"fmt"
	"sync"
	"time"

	commonv1 "piper_agent/runtime/pkg/pb/common/v1"
	runtimev1 "piper_agent/runtime/pkg/pb/runtime/v1"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Runtime implements Phase A mock gRPC services (no piper_go).
type Runtime struct {
	runtimev1.UnimplementedRuntimeTemplateServer
	runtimev1.UnimplementedRuntimeExecuteServer

	mu              sync.Mutex
	validated       map[string]struct{}
	templates       map[string]*commonv1.TemplateDoc
	runs            map[string]*runRecord
	idempotency     map[string]string
	maxConcurrent   int
	activeRuns      int
}

type runRecord struct {
	status  *commonv1.RunStatus
	events  []*runtimev1.RunEvent
	done    chan struct{}
	cancel  context.CancelFunc
}

func NewRuntime(maxConcurrent int) *Runtime {
	if maxConcurrent <= 0 {
		maxConcurrent = 4
	}
	return &Runtime{
		validated:     make(map[string]struct{}),
		templates:     make(map[string]*commonv1.TemplateDoc),
		runs:          make(map[string]*runRecord),
		idempotency:   make(map[string]string),
		maxConcurrent: maxConcurrent,
	}
}

func (r *Runtime) ValidateTemplate(_ context.Context, req *runtimev1.ValidateTemplateRequest) (*runtimev1.ValidateTemplateResponse, error) {
	if req.GetRequestId() == "" {
		return nil, status.Error(codes.InvalidArgument, "request_id is required")
	}
	tpl := req.GetTemplate()
	if tpl == nil && req.GetTemplateId() != "" {
		tpl = &commonv1.TemplateDoc{Id: req.GetTemplateId(), Name: req.GetTemplateId(), JsonPayload: []byte(`{"builder":{"type":"Http","url_tpl":"https://example.com"},"procedures":[]}`)}
	}
	if tpl == nil {
		return &runtimev1.ValidateTemplateResponse{
			Ok: false,
			Diagnostics: []*commonv1.Diagnostic{{
				Code:    "TEMPLATE_MISSING",
				Message: "template is required",
				Path:    "template",
			}},
		}, nil
	}
	var diags []*commonv1.Diagnostic
	if tpl.GetName() == "" && tpl.GetId() == "" {
		diags = append(diags, &commonv1.Diagnostic{
			Code: "INVALID_TEMPLATE", Message: "name or id required", Path: "template.name",
		})
	}
	if len(tpl.GetJsonPayload()) == 0 {
		diags = append(diags, &commonv1.Diagnostic{
			Code: "INVALID_TEMPLATE", Message: "json_payload is required (Phase A mock)", Path: "template.json_payload",
		})
	}
	if len(diags) > 0 {
		return &runtimev1.ValidateTemplateResponse{Ok: false, Diagnostics: diags}, nil
	}
	key := tpl.GetId()
	if key == "" {
		key = tpl.GetName()
	}
	r.mu.Lock()
	r.validated[key] = struct{}{}
	if tpl.GetId() != "" {
		r.templates[tpl.GetId()] = tpl
	}
	r.mu.Unlock()
	return &runtimev1.ValidateTemplateResponse{Ok: true}, nil
}

func (r *Runtime) UpsertTemplate(_ context.Context, req *runtimev1.UpsertTemplateRequest) (*runtimev1.UpsertTemplateResponse, error) {
	if req.GetRequestId() == "" {
		return nil, status.Error(codes.InvalidArgument, "request_id is required")
	}
	tpl := req.GetTemplate()
	if tpl == nil {
		return nil, status.Error(codes.InvalidArgument, "template is required")
	}
	id := tpl.GetId()
	if id == "" {
		id = "tpl-" + uuid.NewString()[:8]
		tpl.Id = id
	}
	r.mu.Lock()
	r.templates[id] = tpl
	r.mu.Unlock()
	return &runtimev1.UpsertTemplateResponse{TemplateId: id}, nil
}

func (r *Runtime) RunTemplate(ctx context.Context, req *runtimev1.RunTemplateRequest) (*runtimev1.RunTemplateResponse, error) {
	if req.GetRequestId() == "" {
		return nil, status.Error(codes.InvalidArgument, "request_id is required")
	}
	if req.GetTemplateId() == "" {
		return nil, status.Error(codes.InvalidArgument, "template_id is required")
	}
	if req.GetIdempotencyKey() != "" {
		r.mu.Lock()
		if existing, ok := r.idempotency[req.GetIdempotencyKey()]; ok {
			rec := r.runs[existing]
			r.mu.Unlock()
			if rec != nil {
				return &runtimev1.RunTemplateResponse{RunId: existing, Status: cloneStatus(rec.status)}, nil
			}
		}
		r.mu.Unlock()
	}

	r.mu.Lock()
	if r.activeRuns >= r.maxConcurrent {
		r.mu.Unlock()
		return nil, status.Error(codes.ResourceExhausted, "max concurrent runs reached (mock)")
	}
	if _, ok := r.validated[req.GetTemplateId()]; !ok {
		r.mu.Unlock()
		return nil, status.Error(codes.FailedPrecondition, "template not validated; call ValidateTemplate first (Phase A policy)")
	}
	runID := "run-" + uuid.NewString()[:8]
	tokenID := "token-" + runID
	// Detach from request ctx: RunTemplate RPC returns before the mock run finishes.
	runCtx, cancel := context.WithCancel(context.Background())
	st := &commonv1.RunStatus{
		RunId:     runID,
		SessionId: req.GetSessionId(),
		Phase:     commonv1.RunPhase_RUN_PHASE_RUNNING,
		TokenId:   tokenID,
	}
	rec := &runRecord{
		status: st,
		done:   make(chan struct{}),
		cancel: cancel,
	}
	r.runs[runID] = rec
	if req.GetIdempotencyKey() != "" {
		r.idempotency[req.GetIdempotencyKey()] = runID
	}
	r.activeRuns++
	r.mu.Unlock()

	go r.simulateRun(runCtx, rec)

	return &runtimev1.RunTemplateResponse{
		RunId:  runID,
		Status: cloneStatus(st),
	}, nil
}

func (r *Runtime) simulateRun(ctx context.Context, rec *runRecord) {
	defer func() {
		r.mu.Lock()
		r.activeRuns--
		close(rec.done)
		r.mu.Unlock()
	}()

	appendEvent := func(phase commonv1.RunPhase, msg string) {
		rec.status.Phase = phase
		rec.events = append(rec.events, &runtimev1.RunEvent{
			Status:  cloneStatus(rec.status),
			Message: msg,
		})
	}

	appendEvent(commonv1.RunPhase_RUN_PHASE_RUNNING, "mock: queued on distributor")
	select {
	case <-ctx.Done():
		rec.status.Phase = commonv1.RunPhase_RUN_PHASE_CANCELLED
		rec.events = append(rec.events, &runtimev1.RunEvent{
			Status: cloneStatus(rec.status), Message: "cancelled",
		})
		return
	case <-time.After(300 * time.Millisecond):
	}

	appendEvent(commonv1.RunPhase_RUN_PHASE_RUNNING, fmt.Sprintf("mock: executing engine=%q", "http"))
	select {
	case <-ctx.Done():
		rec.status.Phase = commonv1.RunPhase_RUN_PHASE_CANCELLED
		return
	case <-time.After(400 * time.Millisecond):
	}

	rec.status.Phase = commonv1.RunPhase_RUN_PHASE_SUCCEEDED
	rec.events = append(rec.events, &runtimev1.RunEvent{
		Status: cloneStatus(rec.status), Message: "mock: persisted 3 docs",
	})
}

func (r *Runtime) CancelRun(_ context.Context, req *runtimev1.CancelRunRequest) (*runtimev1.CancelRunResponse, error) {
	if req.GetRunId() == "" {
		return nil, status.Error(codes.InvalidArgument, "run_id is required")
	}
	r.mu.Lock()
	rec, ok := r.runs[req.GetRunId()]
	r.mu.Unlock()
	if !ok {
		return nil, status.Error(codes.NotFound, "run not found")
	}
	if rec.cancel != nil {
		rec.cancel()
	}
	return &runtimev1.CancelRunResponse{}, nil
}

func (r *Runtime) SubscribeRun(req *runtimev1.SubscribeRunRequest, stream runtimev1.RuntimeExecute_SubscribeRunServer) error {
	if req.GetRunId() == "" {
		return status.Error(codes.InvalidArgument, "run_id is required")
	}
	r.mu.Lock()
	rec, ok := r.runs[req.GetRunId()]
	r.mu.Unlock()
	if !ok {
		return status.Error(codes.NotFound, "run not found")
	}

	sent := 0
	for {
		r.mu.Lock()
		events := rec.events
		phase := rec.status.GetPhase()
		r.mu.Unlock()

		for sent < len(events) {
			if err := stream.Send(events[sent]); err != nil {
				return err
			}
			sent++
		}
		if phase == commonv1.RunPhase_RUN_PHASE_SUCCEEDED ||
			phase == commonv1.RunPhase_RUN_PHASE_FAILED ||
			phase == commonv1.RunPhase_RUN_PHASE_CANCELLED {
			return nil
		}
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-rec.done:
			r.mu.Lock()
			for sent < len(rec.events) {
				if err := stream.Send(rec.events[sent]); err != nil {
					r.mu.Unlock()
					return err
				}
				sent++
			}
			r.mu.Unlock()
			return nil
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func cloneStatus(s *commonv1.RunStatus) *commonv1.RunStatus {
	if s == nil {
		return nil
	}
	c := *s
	return &c
}
