package handler

import (
	"net/http"
	"strings"

	"piper_go/internal/svc"
	"piper_go/pkg/db/es"
	"piper_go/pkg/persistence"
	piperjson "piper_go/pkg/json"
)

func DataSearchHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st, et, page, size := queryTimeRange(r)
		params := persistence.ParseDataQuery(r, st, et, page, size)
		if len(params.Indices) == 0 {
			writeFailureMsg(w, "index query param required")
			return
		}
		for i, idx := range params.Indices {
			params.Indices[i] = strings.ToLower(idx)
		}
		body := es.SearchBody(params.BoolQuery(), params.From(), int(params.Size), "update_time", true)
		res, err := svcCtx.ES.SearchIndices(r.Context(), params.Indices, body)
		if err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.SuccessPage(res.Hits, page, size, res.Total))
	}
}

func DataSearchDateHistogramHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st, et, _, _ := queryTimeRange(r)
		params := persistence.ParseDataQuery(r, st, et, 1, 1)
		if len(params.Indices) == 0 {
			piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(map[string]int64{}))
			return
		}
		for i, idx := range params.Indices {
			params.Indices[i] = strings.ToLower(idx)
		}
		interval := persistence.HistogramIntervalMinutes(st, et)
		agg, err := svcCtx.ES.SearchDateHistogram(r.Context(), params.Indices, params.BoolQuery(), "update_time", interval, st, et)
		if err != nil {
			if strings.Contains(err.Error(), "index_not_found") {
				piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(map[string]int64{}))
				return
			}
			writeFailure(w, err)
			return
		}
		if agg == nil {
			agg = map[string]int64{}
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(agg))
	}
}
