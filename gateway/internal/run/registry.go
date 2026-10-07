package run

import (
	"context"
	"sync"
)

type CancelFunc func()

type Registry struct {
	mu   sync.Mutex
	runs map[string]CancelFunc
}

func NewRegistry() *Registry {
	return &Registry{runs: make(map[string]CancelFunc)}
}

func (r *Registry) Register(runID string, cancel context.CancelFunc, extra ...CancelFunc) {
	r.mu.Lock()
	combined := func() {
		cancel()
		for _, fn := range extra {
			if fn != nil {
				fn()
			}
		}
	}
	r.runs[runID] = combined
	r.mu.Unlock()
}

func (r *Registry) Cancel(runID string) bool {
	r.mu.Lock()
	fn, ok := r.runs[runID]
	if ok {
		delete(r.runs, runID)
	}
	r.mu.Unlock()
	if ok && fn != nil {
		fn()
	}
	return ok
}

func (r *Registry) Unregister(runID string) {
	r.mu.Lock()
	delete(r.runs, runID)
	r.mu.Unlock()
}
