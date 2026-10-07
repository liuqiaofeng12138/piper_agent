package distributor

import (
	"context"
	"errors"

	"piper_go/pkg/db/meta"
	"piper_go/pkg/distributor/cache"
	"piper_go/pkg/tpl"
)

// RunTask builds tokens from task + template and submits locally (Phase 2).
func (e *Engine) RunTask(store *meta.Store, taskID, nodeInstID, uid string) error {
	doc, err := store.Get(meta.TableTasks, taskID)
	if err != nil {
		return err
	}
	status, _ := doc["status"].(string)
	if status == "Running" || status == "Waiting_Done" {
		return errors.New("Task is Running/Waiting_Done")
	}
	tplID, _ := doc["tpl_id"].(string)
	tplDoc, err := store.Get(meta.TableTemplates, tplID)
	if err != nil {
		return err
	}

	doc["status"] = "Running"
	_ = store.Upsert(meta.TableTasks, doc)
	cache.PutTask(doc)
	e.beginTaskRun(taskID)

	vars := map[string]any{}
	if vl, ok := doc["vars_list"].([]any); ok && len(vl) > 0 {
		if first, ok := vl[0].(map[string]any); ok {
			vars = first
		}
	}
	opts := tpl.RunOpts{
		UID:      uid,
		TaskID:   taskID,
		NodeID:   nodeInstID,
		Behavior: "DEFAULT",
	}
	_, _, err = e.RunTemplate(tplDoc, vars, opts)
	if err != nil {
		e.finishTaskRun(taskID)
		doc["status"] = "Done"
		_ = store.Upsert(meta.TableTasks, doc)
		cache.PutTask(doc)
	}
	return err
}

func (e *Engine) beginTaskRun(taskID string) {
	e.taskRunsMu.Lock()
	e.taskRuns[taskID] = 1
	e.taskRunsMu.Unlock()
}

func (e *Engine) taskTokenQueued(token map[string]any) {
	taskID, _ := token["task_id"].(string)
	if taskID == "" {
		return
	}
	e.taskRunsMu.Lock()
	if pending, ok := e.taskRuns[taskID]; ok {
		e.taskRuns[taskID] = pending + 1
	}
	e.taskRunsMu.Unlock()
}

// taskTokenFinished is called once per terminal token. Child tokens are
// counted when routed, so a paginated task becomes Done only after its entire
// continuation chain has finished.
func (e *Engine) taskTokenFinished(token map[string]any) {
	taskID, _ := token["task_id"].(string)
	if taskID == "" {
		return
	}
	e.taskRunsMu.Lock()
	pending, ok := e.taskRuns[taskID]
	if !ok {
		e.taskRunsMu.Unlock()
		return
	}
	pending--
	if pending > 0 {
		e.taskRuns[taskID] = pending
		e.taskRunsMu.Unlock()
		return
	}
	delete(e.taskRuns, taskID)
	e.taskRunsMu.Unlock()

	if e.meta == nil {
		return
	}
	doc, err := e.meta.Get(meta.TableTasks, taskID)
	if err != nil {
		return
	}
	if status, _ := doc["status"].(string); status == "Running" {
		doc["status"] = "Done"
		if err := e.meta.Upsert(meta.TableTasks, doc); err == nil {
			cache.PutTask(doc)
		}
	}
}

// TaskTokenFinished is used by Chrome workers to report terminal tokens.
func (e *Engine) TaskTokenFinished(token map[string]any) {
	e.taskTokenFinished(token)
}

func (e *Engine) finishTaskRun(taskID string) {
	e.taskRunsMu.Lock()
	delete(e.taskRuns, taskID)
	e.taskRunsMu.Unlock()
}

func (e *Engine) StopTaskRecord(store *meta.Store, taskID string) error {
	doc, err := store.Get(meta.TableTasks, taskID)
	if err != nil {
		return err
	}
	doc["status"] = "Done"
	_ = store.Upsert(meta.TableTasks, doc)
	cache.PutTask(doc)
	e.StopTask(taskID)
	e.finishTaskRun(taskID)
	return nil
}

func (e *Engine) RunTaskAsync(store *meta.Store, taskID, nodeInstID, uid string) {
	ctx, cancel := context.WithCancel(context.Background())
	e.RegisterRunningTask(taskID, cancel)
	go func() {
		defer cancel()
		_ = e.RunTask(store, taskID, nodeInstID, uid)
		select {
		case <-ctx.Done():
		default:
		}
	}()
}
