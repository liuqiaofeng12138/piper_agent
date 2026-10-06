package misc

import (
	"net/http"
	"strings"

	"piper_go/internal/svc"
	piperjson "piper_go/pkg/json"
	"piper_go/pkg/monitor"
)

func OverviewMetricsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		promBase := strings.TrimSpace(svcCtx.Config.WebAPI.PrometheusHost)
		node := r.URL.Query().Get("node_instance")
		piperInst := r.URL.Query().Get("piper_instance")

		if n := svcCtx.Node(); n != nil && piperInst == "" {
			piperInst = n.PrometheusInstance
		}
		if piperInst == "" {
			piperInst = "127.0.0.1:8888"
		}

		candidates := []string{
			node,
			"127.0.0.1:9100",
			"node-exporter:9100",
		}
		if node == "" {
			node = monitor.ResolveNodeInstance(r.Context(), promBase, candidates[1:])
		} else {
			node = monitor.ResolveNodeInstance(r.Context(), promBase, candidates)
		}

		stats := monitor.OverviewStats(r.Context(), promBase, node, piperInst)
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(stats))
	}
}
