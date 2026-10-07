package cluster

import (
	"encoding/json"

	"piper_go/pkg/db/meta"
	"piper_go/pkg/distributor/cache"
)

// Pack mirrors Java task.Pack JSON for misc export/import.
type Pack struct {
	Tasks     []map[string]any `json:"tasks"`
	VarsLists []map[string]any `json:"vars_lists"`
	Templates []map[string]any `json:"templates"`
	Indices   []map[string]any `json:"indices"`
	Docs      []map[string]any `json:"docs"`
}

func ExportPack(store *meta.Store, taskIDs []string) (*Pack, error) {
	p := &Pack{
		Tasks:     []map[string]any{},
		VarsLists: []map[string]any{},
		Templates: []map[string]any{},
		Indices:   []map[string]any{},
		Docs:      []map[string]any{},
	}
	tplSeen := map[string]struct{}{}
	idxSeen := map[string]struct{}{}
	vlSeen := map[string]struct{}{}

	for _, id := range taskIDs {
		task, err := store.Get(meta.TableTasks, id)
		if err != nil {
			if t, ok := cache.TaskLists[id]; ok {
				task = clonePackMap(t)
			} else {
				continue
			}
		} else {
			task = clonePackMap(task)
		}
		task["status"] = "New"
		task["node_id"] = nil
		task["concat_chunk_size"] = 0
		p.Tasks = append(p.Tasks, task)

		if vlID, _ := task["vars_list_id"].(string); vlID != "" {
			if _, ok := vlSeen[vlID]; !ok {
				if vl, err := store.Get(meta.TableVarsLists, vlID); err == nil {
					p.VarsLists = append(p.VarsLists, vl)
					vlSeen[vlID] = struct{}{}
				}
			}
		}
		tplID, _ := task["tpl_id"].(string)
		if tplID == "" {
			continue
		}
		collectTemplatePack(store, p, tplID, tplSeen, idxSeen)
	}
	return p, nil
}

func collectTemplatePack(store *meta.Store, p *Pack, tplID string, tplSeen, idxSeen map[string]struct{}) {
	if _, ok := tplSeen[tplID]; ok {
		return
	}
	tpl, err := store.Get(meta.TableTemplates, tplID)
	if err != nil {
		tpl = cache.Templates[tplID]
	}
	if tpl == nil {
		return
	}
	tplSeen[tplID] = struct{}{}
	p.Templates = append(p.Templates, tpl)
	for _, ref := range cache.RefList(cache.TemplateTemplates, tplID) {
		tid := ref["id"]
		if tid == "" {
			tid = ref["ref_id"]
		}
		if tid != "" {
			collectTemplatePack(store, p, tid, tplSeen, idxSeen)
		}
	}
	for _, ref := range cache.RefList(cache.IndexTemplates, tplID) {
		iid := ref["id"]
		if iid == "" {
			iid = ref["ref_id"]
		}
		if iid == "" {
			continue
		}
		if _, seen := idxSeen[iid]; seen {
			continue
		}
		idx, err := store.Get(meta.TableIndices, iid)
		if err == nil {
			idxSeen[iid] = struct{}{}
			p.Indices = append(p.Indices, idx)
		}
	}
}

func ImportPack(store *meta.Store, source []byte) error {
	var p Pack
	if err := json.Unmarshal(source, &p); err != nil {
		return err
	}
	for _, idx := range p.Indices {
		_ = store.Upsert(meta.TableIndices, idx)
		cache.PutIndex(idx)
	}
	for _, tpl := range p.Templates {
		_ = store.Upsert(meta.TableTemplates, tpl)
		cache.PutTemplate(tpl)
	}
	for _, vl := range p.VarsLists {
		_ = store.Upsert(meta.TableVarsLists, vl)
		cache.PutVarsList(vl)
	}
	for _, task := range p.Tasks {
		delete(task, "id")
		meta.AssignID(meta.TableTasks, task)
		_ = store.Upsert(meta.TableTasks, task)
		cache.PutTask(task)
	}
	return nil
}

func clonePackMap(m map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		out[k] = v
	}
	return out
}
