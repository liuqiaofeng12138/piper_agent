package handler

import (
	"errors"
	"net/http"

	"github.com/zeromicro/go-zero/rest/pathvar"

	"piper_go/internal/svc"
	"piper_go/pkg/db/meta"
	"piper_go/pkg/distributor"
	"piper_go/pkg/distributor/cache"
	piperjson "piper_go/pkg/json"
	pkgmeta "piper_go/pkg/meta"
	proxyimpl "piper_go/pkg/proxy/impl"
	"piper_go/pkg/runtimeconfig"
)

func MiscConfigGetHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(runtimeconfig.WebAPIRoot()))
	}
}

func MiscConfigPutHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := readBody(r)
		if err != nil {
			writeFailure(w, err)
			return
		}
		if err := runtimeconfig.MergeWebAPI(body); err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(nil))
	}
}

func MiscExceptionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(pkgmeta.IFExceptions))
	}
}

func MiscGlobalVarsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(distributor.GetGlobalVars()))
	}
}

func IndexRefHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(cache.RefList(cache.IndexTemplates, id)))
	}
}

func IndexViewsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		viewType := r.URL.Query().Get("type")
		if viewType == "" {
			viewType = "card"
		}
		script := cache.IndexViewScript(id, viewType)
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(script))
	}
}

func TemplateMapperTypesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		types := []string{"Index", "Template", "Source", "YouTubeVideo", "BilibiliVideo"}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(types))
	}
}

func TemplateRefHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(cache.RefList(cache.TemplateTemplates, id)))
	}
}

func TemplateVarNamesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success([]string{}))
	}
}

func VarsListCheckHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success([]string{}))
	}
}

func VarsListFirstHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		doc, err := svcCtx.Meta.Get(meta.TableVarsLists, id)
		if err != nil {
			writeFailure(w, err)
			return
		}
		items, _ := doc["items"].([]any)
		if len(items) == 0 {
			writeFailure(w, errors.New("empty vars list"))
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(items[0]))
	}
}

func AccountFilterHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		domain := r.URL.Query().Get("domain")
		username := r.URL.Query().Get("username")
		rows, _, err := svcCtx.Meta.Query(meta.TableAccounts, meta.QueryOpts{Page: 1, Size: 1000, Q: ""})
		if err != nil {
			writeFailure(w, err)
			return
		}
		for _, row := range rows {
			if row["domain"] == domain && row["username"] == username {
				piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(row))
				return
			}
		}
		writeFailure(w, meta.ErrNotFound)
	}
}

func AccountDomainsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		list, err := svcCtx.Meta.Distinct(meta.TableAccounts, "domain", q)
		if err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(list))
	}
}

func AccountUsernamesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		domain := r.URL.Query().Get("domain")
		list, err := svcCtx.Meta.DistinctEq(meta.TableAccounts, "domain", domain)
		if err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(list))
	}
}

func TaskCopyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		doc, err := svcCtx.Meta.Get(meta.TableTasks, id)
		if err != nil {
			if c, ok := cache.TaskLists[id]; ok {
				doc = cloneMap(c)
			} else {
				writeFailure(w, err)
				return
			}
		}
		clone := cloneMap(doc)
		clone["f_id"] = id
		if name, ok := clone["name"].(string); ok {
			clone["name"] = name + "-COPY"
		}
		clone["status"] = "New"
		clone["concat_chunk_size"] = 0
		delete(clone, "id")
		meta.AssignID(meta.TableTasks, clone)
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(clone))
	}
}

func ProxyInitHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		doc, err := svcCtx.Meta.Get(meta.TableProxies, id)
		if err != nil {
			writeFailure(w, err)
			return
		}
		updated, sess, err := proxyimpl.Setup(doc)
		if err != nil {
			writeFailure(w, err)
			return
		}
		proxyimpl.Close(sess)
		if err := svcCtx.Meta.Upsert(meta.TableProxies, updated); err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(nil))
	}
}
