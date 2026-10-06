package persistence

import (
	"net/http"
	"net/url"
	"strconv"

	"piper_go/pkg/db/es"
)

// DataQueryParams mirrors DataRoute.getQueryParams.
type DataQueryParams struct {
	Indices  []string
	ST       int64
	ET       int64
	Page     int64
	Size     int64
	Query    string
	TaskID   string
	TplID    string
	Meta     string
	Mime     string
	FileSize int64
}

func ParseDataQuery(r *http.Request, st, et, page, size int64) DataQueryParams {
	p := DataQueryParams{
		Indices: r.URL.Query()["index"],
		ST:      st,
		ET:      et,
		Page:    page,
		Size:    size,
		Query:   r.URL.Query().Get("q"),
		TaskID:  r.URL.Query().Get("task_id"),
		TplID:   r.URL.Query().Get("tpl_id"),
		Meta:    r.URL.Query().Get("meta"),
		Mime:    r.URL.Query().Get("mime"),
	}
	fs, _ := strconv.ParseInt(r.URL.Query().Get("file_size"), 10, 64)
	p.FileSize = fs
	return p
}

func (p DataQueryParams) BoolQuery() map[string]any {
	must := []map[string]any{
		es.RangeQuery("update_time", p.ST, p.ET),
	}
	if p.Query != "" {
		must = append(must, es.QueryString(url.QueryEscape(p.Query)))
	}
	if p.FileSize > 0 {
		must = append(must, map[string]any{
			"range": map[string]any{"size": map[string]any{"gte": p.FileSize}},
		})
	}
	if p.TaskID != "" {
		must = append(must, es.TermQuery("__task_id", p.TaskID))
	}
	if p.TplID != "" {
		must = append(must, es.TermQuery("__tpl_id", p.TplID))
	}
	if p.Meta != "" {
		must = append(must, map[string]any{"match": map[string]any{"meta": p.Meta}})
	}
	if p.Mime != "" {
		must = append(must, map[string]any{"prefix": map[string]any{"mime": p.Mime}})
	}
	return es.BoolMust(must...)
}

func (p DataQueryParams) From() int {
	if p.Page < 1 {
		return 0
	}
	return int((p.Page - 1) * p.Size)
}

func HistogramIntervalMinutes(st, et int64) int {
	const eightHoursMs = 8 * 60 * 60 * 1000
	if et <= st {
		return 1
	}
	m := int((et - st) / eightHoursMs)
	if m < 1 {
		m = 1
	}
	return m + 1
}
