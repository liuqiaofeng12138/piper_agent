package crawlerworker

import (
	"context"
	"fmt"
	"log"
	"sync"

	agentv1 "piper_agent/gateway/pkg/pb/agent/v1"
	"piper_agent/gateway/internal/crawlerharness"
	"piper_agent/gateway/internal/crawlerrt"
	"piper_agent/runtime/pkg/embed"
)

// Worker is an in-process web_crawler Agent (Harness + embedded Runtime, no Python / no agent gRPC).
type Worker struct {
	cfgPath string
	repoRoot string
	rt      *embed.Services

	mu    sync.Mutex
	loops map[string]*conversation
	cancels map[string]context.CancelFunc
}

type conversation struct {
	loop *crawlerharness.Loop
}

func New(cfgPath, repoRoot string, rt *embed.Services) (*Worker, error) {
	if rt == nil {
		return nil, fmt.Errorf("embedded runtime required for in-process web_crawler")
	}
	return &Worker{
		cfgPath:  cfgPath,
		repoRoot: repoRoot,
		rt:       rt,
		loops:    make(map[string]*conversation),
		cancels:  make(map[string]context.CancelFunc),
	}, nil
}

func (w *Worker) Health(ctx context.Context) error {
	return nil
}

func (w *Worker) Close() {}

func (w *Worker) CancelRun(ctx context.Context, runID string) error {
	w.mu.Lock()
	cancel, ok := w.cancels[runID]
	if ok {
		delete(w.cancels, runID)
	}
	w.mu.Unlock()
	if ok && cancel != nil {
		cancel()
	}
	return nil
}

func (w *Worker) Execute(
	ctx context.Context,
	req *agentv1.ExecuteRequest,
	onEvent func(*agentv1.ExecuteEvent) error,
) error {
	runID := req.GetRunId()
	convID := req.GetConversationId()
	if convID == "" {
		convID = "default"
	}

	runCtx, cancel := context.WithCancel(ctx)
	w.mu.Lock()
	w.cancels[runID] = cancel
	w.mu.Unlock()
	defer func() {
		w.mu.Lock()
		delete(w.cancels, runID)
		w.mu.Unlock()
		cancel()
	}()

	loop, err := w.getLoop(convID)
	if err != nil {
		return err
	}

	cancelled := func() bool {
		select {
		case <-runCtx.Done():
			return true
		default:
			return false
		}
	}

	log.Printf("[crawler-inproc] execute start run=%s conv=%s", runID, convID)
	events := loop.RunTurnIter(runCtx, req.GetUserMessage(), cancelled)
	for _, ev := range events {
		if onEvent != nil {
			if err := onEvent(mapEvent(ev)); err != nil {
				return err
			}
		}
		et, _ := ev["type"].(string)
		if et == "run.cancelled" || et == "error" {
			break
		}
	}
	log.Printf("[crawler-inproc] execute end run=%s conv=%s", runID, convID)
	return nil
}

func (w *Worker) getLoop(convID string) (*crawlerharness.Loop, error) {
	w.mu.Lock()
	if c, ok := w.loops[convID]; ok {
		w.mu.Unlock()
		return c.loop, nil
	}
	w.mu.Unlock()

	cfg := crawlerharness.LoadConfig(w.cfgPath, w.repoRoot)
	sess := crawlerharness.NewSession(convID)
	client := crawlerrt.New(w.rt)
	handlers := crawlerharness.NewToolHandlers(client, sess, cfg)
	reg := crawlerharness.NewRegistry(handlers)
	loop, err := crawlerharness.NewLoop(cfg, reg, sess)
	if err != nil {
		return nil, err
	}

	w.mu.Lock()
	w.loops[convID] = &conversation{loop: loop}
	w.mu.Unlock()
	return loop, nil
}

func mapEvent(ev map[string]any) *agentv1.ExecuteEvent {
	return &agentv1.ExecuteEvent{
		Type:       fmt.Sprint(ev["type"]),
		Delta:      fmt.Sprint(ev["delta"]),
		Content:    fmt.Sprint(ev["content"]),
		ToolName:   fmt.Sprint(ev["tool_name"]),
		ToolDetail: fmt.Sprint(ev["tool_detail"]),
		Message:    fmt.Sprint(ev["message"]),
	}
}
