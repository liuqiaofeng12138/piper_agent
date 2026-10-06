package handler

import (
	"io"
	"net/http"
	"strconv"

	piperjson "piper_go/pkg/json"
)

func readBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	return io.ReadAll(r.Body)
}

func pageSize(r *http.Request) (page, size int64) {
	page, _ = strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)
	size, _ = strconv.ParseInt(r.URL.Query().Get("size"), 10, 64)
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	return page, size
}

func writeFailure(w http.ResponseWriter, err error) {
	piperjson.WriteMsg(w, http.StatusOK, piperjson.FailureErr(err))
}

func writeFailureMsg(w http.ResponseWriter, msg string) {
	piperjson.WriteMsg(w, http.StatusOK, piperjson.FailureString(msg))
}

func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) < len(prefix) {
		return ""
	}
	if h[:len(prefix)] != prefix {
		return ""
	}
	return h[len(prefix):]
}
