package android

import (
	"fmt"
	"sync"
)

// Agent is a placeholder Android collector (Java AndroidAgent baseline).
type Agent struct {
	ID     string
	Name   string
	Status string
	mu     sync.Mutex
}

func NewAgent(name string) *Agent {
	return &Agent{ID: name, Name: name, Status: "Idle"}
}

func (a *Agent) Snapshot() map[string]any {
	a.mu.Lock()
	defer a.mu.Unlock()
	return map[string]any{
		"id":     a.ID,
		"type":   "AndroidAgent",
		"name":   a.Name,
		"status": a.Status,
	}
}

// RunTask is a stub; real implementation would drive ADB (Java AdbUtil).
func (a *Agent) RunTask(task map[string]any) error {
	_ = task
	return fmt.Errorf("AndroidAgent not implemented in piper_go baseline")
}
