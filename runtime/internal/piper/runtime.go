package piper

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"piper_agent/runtime/internal/executor"
	"piper_go/pkg/agentruntime"

	commonv1 "piper_agent/runtime/pkg/pb/common/v1"
	runtimev1 "piper_agent/runtime/pkg/pb/runtime/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Runtime implements gRPC services backed by piper_go (Phase B).
type Runtime struct {
	runtimev1.UnimplementedRuntimeTemplateServer
	runtimev1.UnimplementedRuntimeExecuteServer
	runtimev1.UnimplementedRuntimeMetaServer
	runtimev1.UnimplementedRuntimeDataServer

	exec   *executor.Executor
	maxRun int

	mu          sync.Mutex
	validated   map[string]struct{}
	runs        map[string]*runRecord
	idempotency map[string]string
	activeRuns  int
}

type runRecord struct {
	tokenID      string
	sessionID    string
	startUT      int64
	events       []*runtimev1.RunEvent
	done         chan struct{}
	phase        commonv1.RunPhase
	lastProgress time.Time
}

func NewRuntime(svc *agentruntime.Service, maxConcurrent int) *Runtime {
	if maxConcurrent <= 0 {
		maxConcurrent = 4
	}
	return &Runtime{
		exec:        executor.New(svc),
		maxRun:      maxConcurrent,
		validated:   make(map[string]struct{}),
		runs:        make(map[string]*runRecord),
		idempotency: make(map[string]string),
	}
}

func (r *Runtime) ValidateTemplate(ctx context.Context, req *runtimev1.ValidateTemplateRequest) (*runtimev1.ValidateTemplateResponse, error) {
	if req.GetRequestId() == "" {
		return nil, status.Error(codes.InvalidArgument, "request_id is required")
	}
	vars := varsToMap(req.GetVars())
	tplDoc, err := r.exec.ResolveTemplate(ctx, req.GetTemplate(), req.GetTemplateId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "template: %v", err)
	}
	diags := r.exec.ValidateHTTPBuild(tplDoc, vars)
	if len(diags) > 0 {
		return &runtimev1.ValidateTemplateResponse{Ok: false, Diagnostics: diags}, nil
	}
	id, _ := tplDoc["id"].(string)
	if id == "" {
		id, _ = tplDoc["name"].(string)
	}
	if id != "" {
		r.mu.Lock()
		r.validated[id] = struct{}{}
		r.mu.Unlock()
	}
	return &runtimev1.ValidateTemplateResponse{Ok: true}, nil
}

func (r *Runtime) UpsertTemplate(ctx context.Context, req *runtimev1.UpsertTemplateRequest) (*runtimev1.UpsertTemplateResponse, error) {
	if req.GetRequestId() == "" {
		return nil, status.Error(codes.InvalidArgument, "request_id is required")
	}
	id, err := r.exec.UpsertTemplate(ctx, req.GetTemplate())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "upsert: %v", err)
	}
	return &runtimev1.UpsertTemplateResponse{TemplateId: id}, nil
}

func (r *Runtime) ListTemplates(ctx context.Context, req *runtimev1.ListTemplatesRequest) (*runtimev1.ListTemplatesResponse, error) {
	if req.GetRequestId() == "" {
		return nil, status.Error(codes.InvalidArgument, "request_id is required")
	}
	page, size := int(req.GetPage()), int(req.GetSize())
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	items, total, err := r.exec.ListTemplates(ctx, req.GetQuery(), page, size)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list: %v", err)
	}
	out := make([]*commonv1.TemplateDoc, 0, len(items))
	for _, it := range items {
		out = append(out, executor.TemplateDocFromMap(it))
	}
	return &runtimev1.ListTemplatesResponse{Templates: out, Total: total}, nil
}

func (r *Runtime) GetTemplate(ctx context.Context, req *runtimev1.GetTemplateRequest) (*runtimev1.GetTemplateResponse, error) {
	if req.GetTemplateId() == "" {
		return nil, status.Error(codes.InvalidArgument, "template_id is required")
	}
	doc, err := r.exec.GetTemplate(ctx, req.GetTemplateId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "template: %v", err)
	}
	return &runtimev1.GetTemplateResponse{Template: executor.TemplateDocFromMap(doc)}, nil
}

func (r *Runtime) ListProxies(ctx context.Context, req *runtimev1.ListProxiesRequest) (*runtimev1.ListProxiesResponse, error) {
	if req.GetRequestId() == "" {
		return nil, status.Error(codes.InvalidArgument, "request_id is required")
	}
	page, size := int(req.GetPage()), int(req.GetSize())
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 50
	}
	items, total, err := r.exec.ListProxies(ctx, req.GetStatus(), page, size)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list proxies: %v", err)
	}
	out := make([]*runtimev1.ProxyDoc, 0, len(items))
	for _, p := range items {
		out = append(out, &runtimev1.ProxyDoc{
			Id: p.ID, Name: p.Name, Domain: p.Domain, Status: p.Status,
		})
	}
	return &runtimev1.ListProxiesResponse{Proxies: out, Total: total}, nil
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
				return &runtimev1.RunTemplateResponse{
					RunId:  existing,
					Status: r.statusFromRecord(rec),
				}, nil
			}
		}
		r.mu.Unlock()
	}

	r.mu.Lock()
	if r.activeRuns >= r.maxRun {
		r.mu.Unlock()
		return nil, status.Error(codes.ResourceExhausted, "max concurrent runs")
	}
	if _, ok := r.validated[req.GetTemplateId()]; !ok {
		r.mu.Unlock()
		return nil, status.Error(codes.FailedPrecondition, "template not validated; call ValidateTemplate first")
	}
	r.mu.Unlock()

	vars := varsToMap(req.GetVars())
	tokenID, _, err := r.exec.RunTemplate(ctx, req.GetTemplateId(), vars, req.GetEngine(), req.GetSessionId(), req.GetProxyId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "run: %v", err)
	}
	runID := tokenID

	startUT := time.Now().UnixMilli()
	if doc, err := r.exec.GetToken(ctx, tokenID); err == nil && doc != nil {
		startUT = int64(jsonNumber(doc["update_time"]))
	}
	rec := &runRecord{
		tokenID:   tokenID,
		sessionID: req.GetSessionId(),
		startUT:   startUT,
		done:      make(chan struct{}),
		phase:     commonv1.RunPhase_RUN_PHASE_RUNNING,
	}
	rec.events = append(rec.events, &runtimev1.RunEvent{
		Status:  r.statusFromRecord(rec),
		Message: "token queued",
	})

	r.mu.Lock()
	r.runs[runID] = rec
	if req.GetIdempotencyKey() != "" {
		r.idempotency[req.GetIdempotencyKey()] = runID
	}
	r.activeRuns++
	r.mu.Unlock()

	go r.watchToken(context.Background(), rec)

	return &runtimev1.RunTemplateResponse{
		RunId:  runID,
		Status: r.statusFromRecord(rec),
	}, nil
}

func (r *Runtime) watchToken(ctx context.Context, rec *runRecord) {
	defer func() {
		r.mu.Lock()
		r.activeRuns--
		close(rec.done)
		r.mu.Unlock()
	}()

	ticker := time.NewTicker(400 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.After(10 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			rec.phase = commonv1.RunPhase_RUN_PHASE_CANCELLED
			r.appendEvent(rec, "cancelled")
			return
		case <-deadline:
			rec.phase = commonv1.RunPhase_RUN_PHASE_FAILED
			r.appendEvent(rec, "watch deadline exceeded")
			return
		case <-ticker.C:
			doc, err := r.exec.GetToken(ctx, rec.tokenID)
			if err != nil || doc == nil {
				continue
			}
			ut := int64(jsonNumber(doc["update_time"]))
			if ut <= rec.startUT {
				if time.Since(rec.lastProgress) >= 8*time.Second {
					rec.lastProgress = time.Now()
					r.appendEvent(rec, "chrome/http run in progress — if a browser is open, finish login there; this can take several minutes")
				}
				continue
			}
			finished, _ := doc["finished"].(bool)
			if !finished {
				if time.Since(rec.lastProgress) >= 8*time.Second {
					rec.lastProgress = time.Now()
					r.appendEvent(rec, "token updated, still running")
				}
				continue
			}
			ok, _ := doc["success"].(bool)
			if ok {
				rec.phase = commonv1.RunPhase_RUN_PHASE_SUCCEEDED
				r.appendEvent(rec, "token succeeded")
			} else {
				rec.phase = commonv1.RunPhase_RUN_PHASE_FAILED
				r.appendEvent(rec, "token finished without success")
			}
			return
		}
	}
}

func (r *Runtime) appendEvent(rec *runRecord, msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec.events = append(rec.events, &runtimev1.RunEvent{
		Status:  r.statusFromRecord(rec),
		Message: msg,
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
	rec.phase = commonv1.RunPhase_RUN_PHASE_CANCELLED
	r.appendEvent(rec, "cancel requested (token may still complete in piper_go queue)")
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
		phase := rec.phase
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
		case <-time.After(150 * time.Millisecond):
		}
	}
}

func (r *Runtime) GetTokenData(ctx context.Context, req *runtimev1.GetTokenDataRequest) (*runtimev1.GetTokenDataResponse, error) {
	if req.GetTokenId() == "" {
		return nil, status.Error(codes.InvalidArgument, "token_id is required")
	}
	payload, err := r.exec.GetTokenData(ctx, req.GetTokenId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "token data: %v", err)
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &runtimev1.GetTokenDataResponse{
		TokenId:     req.GetTokenId(),
		PayloadJson: b,
	}, nil
}

func (r *Runtime) statusFromRecord(rec *runRecord) *commonv1.RunStatus {
	return &commonv1.RunStatus{
		RunId:     rec.tokenID,
		SessionId: rec.sessionID,
		Phase:     rec.phase,
		TokenId:   rec.tokenID,
	}
}

func varsToMap(v *commonv1.Vars) map[string]any {
	out := map[string]any{}
	if v == nil {
		return out
	}
	for k, val := range v.GetValues() {
		out[k] = val
	}
	return out
}

func jsonNumber(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}
