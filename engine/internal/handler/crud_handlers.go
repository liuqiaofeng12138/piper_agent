package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/zeromicro/go-zero/rest/pathvar"

	"piper_go/internal/svc"
	"piper_go/pkg/db/meta"
	"piper_go/pkg/distributor/cache"
	piperjson "piper_go/pkg/json"
)

type crudSpec struct {
	table   string
	onWrite func(map[string]any)
}

func metaQuery(svcCtx *svc.ServiceContext, table string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, size := pageSize(r)
		opt := meta.QueryOpts{
			Q:    r.URL.Query().Get("q"),
			Page: page,
			Size: size,
		}
		if table == meta.TableProxies {
			opt.Status = r.URL.Query().Get("status")
		}
		items, total, err := svcCtx.Meta.Query(table, opt)
		if err != nil {
			writeFailure(w, err)
			return
		}
		if table == meta.TableVarsLists && r.URL.Query().Get("type") == "name" {
			for _, it := range items {
				delete(it, "items")
			}
		}
		if table == meta.TableTasks {
			for _, it := range items {
				enrichTaskFromCache(it)
			}
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.SuccessPage(items, page, size, total))
	}
}

func metaGet(svcCtx *svc.ServiceContext, table string, altLookup func(string) map[string]any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		doc, err := svcCtx.Meta.Get(table, id)
		if errors.Is(err, meta.ErrNotFound) && altLookup != nil {
			if alt := altLookup(id); alt != nil {
				piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(alt))
				return
			}
		}
		if err != nil {
			writeFailure(w, err)
			return
		}
		if table == meta.TableTasks {
			enrichTaskFromCache(doc)
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(doc))
	}
}

func metaCreate(svcCtx *svc.ServiceContext, spec crudSpec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := readBody(r)
		if err != nil {
			writeFailure(w, err)
			return
		}
		if len(body) > 0 && body[0] == '[' {
			list, err := meta.ParseJSONArray(body)
			if err != nil {
				writeFailure(w, err)
				return
			}
			for _, doc := range list {
				meta.AssignID(spec.table, doc)
				if spec.onWrite != nil {
					spec.onWrite(doc)
				}
				if err := svcCtx.Meta.Upsert(spec.table, doc); err != nil {
					writeFailure(w, err)
					return
				}
			}
			piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(nil))
			return
		}
		doc, err := meta.ParseJSONObject(body)
		if err != nil {
			writeFailure(w, err)
			return
		}
		meta.AssignID(spec.table, doc)
		if spec.onWrite != nil {
			spec.onWrite(doc)
		}
		if err := svcCtx.Meta.Upsert(spec.table, doc); err != nil {
			piperjson.WriteMsg(w, http.StatusOK, piperjson.Failure())
			return
		}
		id, _ := doc["id"].(string)
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(map[string]string{"id": id}))
	}
}

func metaUpdate(svcCtx *svc.ServiceContext, spec crudSpec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		body, err := readBody(r)
		if err != nil {
			writeFailure(w, err)
			return
		}
		doc, err := meta.ParseJSONObject(body)
		if err != nil {
			writeFailure(w, err)
			return
		}
		doc["id"] = id
		if spec.table == meta.TableTasks {
			doc["status"] = "New"
		}
		if spec.onWrite != nil {
			spec.onWrite(doc)
		}
		if err := svcCtx.Meta.Upsert(spec.table, doc); err != nil {
			piperjson.WriteMsg(w, http.StatusOK, piperjson.Failure())
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(nil))
	}
}

func metaDelete(svcCtx *svc.ServiceContext, table string, before func(map[string]any) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		if before != nil {
			doc, err := svcCtx.Meta.Get(table, id)
			if err != nil {
				writeFailure(w, err)
				return
			}
			if err := before(doc); err != nil {
				writeFailure(w, err)
				return
			}
		}
		if err := svcCtx.Meta.Delete(table, id); err != nil {
			writeFailure(w, err)
			return
		}
		if table == meta.TableTasks {
			cache.DeleteTask(id)
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(nil))
	}
}

func enrichTaskFromCache(task map[string]any) {
	id, _ := task["id"].(string)
	if c, ok := cache.TaskLists[id]; ok {
		if v, ok := c["token_num"]; ok {
			task["token_num"] = v
		}
		if v, ok := c["data_num"]; ok {
			task["data_num"] = v
		}
	}
	vlID, _ := task["vars_list_id"].(string)
	if vlID != "" {
		if vl, ok := cache.VarsLists[vlID]; ok {
			task["vars_list_name"] = vl["name"]
		}
	}
}

func onIndexWrite(doc map[string]any) { cache.PutIndex(doc) }
func onTemplateWrite(doc map[string]any) { cache.PutTemplate(doc) }
func onVarsListWrite(doc map[string]any) { cache.PutVarsList(doc) }
func onTaskWrite(doc map[string]any) { cache.PutTask(doc) }

func taskDeleteGuard(doc map[string]any) error {
	st, _ := doc["status"].(string)
	if st == "Running" || st == "Waiting_Done" {
		return errors.New("task is running")
	}
	return nil
}

func proxyDeleteGuard(doc map[string]any) error {
	if st, _ := doc["status"].(string); st == "Occupied" {
		return errors.New("proxy occupied")
	}
	return nil
}

func cloneMap(in map[string]any) map[string]any {
	b, _ := json.Marshal(in)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out
}
