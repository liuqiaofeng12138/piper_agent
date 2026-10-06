package metrics

import (
	"net/http"

	"piper_go/internal/svc"
	"piper_go/pkg/distributor"
)

func Handler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svcCtx.Ready() {
			distributor.Stats.SetUp(1)
			distributor.SyncMetricSeeds(svcCtx.Meta)
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(distributor.Stats.PrometheusText()))
	}
}
