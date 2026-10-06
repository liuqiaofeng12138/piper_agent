package misc

import (
	"net/http"

	"piper_go/internal/svc"
	piperjson "piper_go/pkg/json"
)

func InfoHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(svcCtx.NodeInfoForAPI()))
	}
}
