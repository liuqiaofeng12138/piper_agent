package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/rest/pathvar"

	"piper_go/internal/middleware"
	"piper_go/internal/svc"
	"piper_go/pkg/db/es"
	"piper_go/pkg/db/meta"
	"piper_go/pkg/distributor"
	piperjson "piper_go/pkg/json"
	"piper_go/pkg/notification"
	"piper_go/pkg/persistence"
	"piper_go/pkg/tpl"
	"piper_go/pkg/websocket"
)

func queryTimeRange(r *http.Request) (st, et int64, page, size int64) {
	now := time.Now().UnixMilli()
	st, _ = strconv.ParseInt(r.URL.Query().Get("st"), 10, 64)
	et, _ = strconv.ParseInt(r.URL.Query().Get("et"), 10, 64)
	page, size = pageSize(r)
	if st == 0 {
		st = now - 86400000
	}
	if et == 0 {
		et = now
	}
	return st, et, page, size
}

func TemplateRunHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		tplDoc, err := svcCtx.Meta.Get(meta.TableTemplates, id)
		if err != nil {
			writeFailure(w, err)
			return
		}
		vars := map[string]any{}
		body, _ := readBody(r)
		if len(body) > 0 {
			_ = json.Unmarshal(body, &vars)
		}
		run := r.URL.Query().Get("run") == "true"
		behavior := "TEST"
		if run {
			behavior = "DEFAULT"
		}
		opts := tpl.RunOpts{
			UID:      middleware.UIDFromContext(r),
			NodeID:   svcCtx.Node().InstID,
			AgentID:  r.URL.Query().Get("agent_id"),
			Behavior: behavior,
		}
		tokenID, agentID, err := svcCtx.Dist.RunTemplate(tplDoc, vars, opts)
		if err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(map[string]any{
			"token_id": tokenID,
			"agent_id": agentID,
		}))
	}
}

func TaskRunHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		uid := middleware.UIDFromContext(r)
		nodeID := svcCtx.Node().InstID
		if err := svcCtx.Dist.RunTask(svcCtx.Meta, id, nodeID, uid); err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(nil))
	}
}

func TaskStopHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		if err := svcCtx.Dist.StopTaskRecord(svcCtx.Meta, id); err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(nil))
	}
}

func MiscRunTokenHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := readBody(r)
		if err != nil {
			writeFailure(w, err)
			return
		}
		var token map[string]any
		if err := json.Unmarshal(body, &token); err != nil {
			writeFailure(w, err)
			return
		}
		if err := svcCtx.Dist.RunRawToken(token); err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(nil))
	}
}

func TokenQueryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st, et, page, size := queryTimeRange(r)
		must := []map[string]any{es.RangeQuery("update_time", st, et)}
		if r.URL.Query().Get("success") == "false" {
			must = append(must, es.TermQuery("success", false))
		}
		for _, pair := range [][2]string{
			{"uid", r.URL.Query().Get("uid")},
			{"node_id", r.URL.Query().Get("node_id")},
			{"task_id", r.URL.Query().Get("task_id")},
			{"tpl_id", r.URL.Query().Get("tpl_id")},
		} {
			if pair[1] != "" {
				must = append(must, es.TermQuery(pair[0], pair[1]))
			}
		}
		if q := r.URL.Query().Get("q"); q != "" {
			must = append(must, es.QueryString(q))
		}
		from := int((page - 1) * size)
		body := es.SearchBody(es.BoolMust(must...), from, int(size), "update_time", true)
		res, err := svcCtx.ES.Search(r.Context(), distributor.ESIndexToken, body)
		if err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.SuccessPage(res.Hits, page, size, res.Total))
	}
}

func TokenGetHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		doc, err := svcCtx.ES.Get(r.Context(), distributor.ESIndexToken, id)
		if err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(doc))
	}
}

func TokenNextHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		body := es.SearchBody(es.BoolMust(es.TermQuery("gen_id", id)), 0, 10000, "update_time", true)
		res, err := svcCtx.ES.Search(r.Context(), distributor.ESIndexToken, body)
		if err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(res.Hits))
	}
}

func TokenDataHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		doc, err := svcCtx.ES.Get(r.Context(), distributor.ESIndexToken, id)
		if err != nil {
			writeFailure(w, err)
			return
		}
		fetch := func(tid string) map[string]any {
			d, err := svcCtx.ES.Get(r.Context(), distributor.ESIndexToken, tid)
			if err != nil {
				return nil
			}
			return d
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(persistence.TokenData(doc, fetch)))
	}
}

func TokenDescendantsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		doc, err := svcCtx.ES.Get(r.Context(), distributor.ESIndexToken, id)
		if err != nil {
			writeFailure(w, err)
			return
		}
		fetch := func(tid string) map[string]any {
			d, err := svcCtx.ES.Get(r.Context(), distributor.ESIndexToken, tid)
			if err != nil {
				return nil
			}
			return d
		}
		data := persistence.TokenData(doc, fetch)
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(map[string]any{
			"token":   doc,
			"docs":    data["docs"],
			"sources": data["sources"],
		}))
	}
}

func LogQueryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st, et, page, size := queryTimeRange(r)
		must := []map[string]any{
			es.TermQuery("inst_id", svcCtx.Node().InstID),
			es.RangeQuery("update_time", st, et),
		}
		if kw := r.URL.Query().Get("q"); kw != "" {
			must = append(must, es.MultiMatchPhrase(kw, "t", "src", "func", "msg", "st"))
		}
		if lv := r.URL.Query().Get("lv"); lv != "" {
			must = append(must, es.TermQuery("lv", lv))
		}
		from := int((page - 1) * size)
		body := es.SearchBody(es.BoolMust(must...), from, int(size), "update_time", true)
		res, err := svcCtx.ES.Search(r.Context(), distributor.ESIndexLog, body)
		if err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.SuccessPage(res.Hits, page, size, res.Total))
	}
}

func wsAuthUser(svcCtx *svc.ServiceContext, token string) (string, bool) {
	if svcCtx.Config.WebAPI.NoAuth {
		if u, err := svcCtx.Auth.VerifyAccessToken(token); err == nil {
			return u.ID, true
		}
		return "anonymous", true
	}
	if token == "" {
		return "", false
	}
	u, err := svcCtx.Auth.VerifyAccessToken(token)
	if err != nil {
		return "", false
	}
	return u.ID, true
}

func WSMsgHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		websocket.HandleMsg(w, r, func(token string) (string, bool) {
			return wsAuthUser(svcCtx, token)
		}, func(_ string) {
			doc := notification.NewListDoc("MEDIUM", "Agents list", distributor.AllAgentsJSON())
			if svcCtx.Notify != nil && svcCtx.Notify.Enabled() {
				svcCtx.Notify.SaveAndBroadcast(r.Context(), doc)
			} else {
				websocket.BroadcastAll(doc)
			}
		})
	}
}

func WSTokenHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		websocket.HandleTokenMsg(w, r, func(token string) (string, bool) {
			return wsAuthUser(svcCtx, token)
		})
	}
}
