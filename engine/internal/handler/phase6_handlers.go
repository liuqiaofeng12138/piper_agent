package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/pathvar"

	"piper_go/internal/svc"
	piperjson "piper_go/pkg/json"
)

func NotificationQueryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svcCtx.Notify == nil || svcCtx.ES == nil {
			piperjson.WriteMsg(w, http.StatusOK, piperjson.SuccessPage([]any{}, 1, 10, 0))
			return
		}
		st, et, page, size := queryTimeRange(r)
		unreadOnly := r.URL.Query().Get("read") == "false"
		instID := ""
		if n := svcCtx.Node(); n != nil {
			instID = n.InstID
		}
		list, total, err := svcCtx.Notify.List(r.Context(), instID, unreadOnly, r.URL.Query().Get("q"), st, et, page, size)
		if err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.SuccessPage(list, page, size, total))
	}
}

func NotificationReadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		if svcCtx.Notify == nil {
			piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(nil))
			return
		}
		if err := svcCtx.Notify.MarkRead(r.Context(), id); err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(nil))
	}
}

func NotificationDeleteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := pathvar.Vars(r)["id"]
		if svcCtx.Notify == nil {
			piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(nil))
			return
		}
		if err := svcCtx.Notify.Delete(r.Context(), id); err != nil {
			writeFailure(w, err)
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(nil))
	}
}
