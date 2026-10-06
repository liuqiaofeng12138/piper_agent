package tpl

import "time"

func newProcLog(procID string) map[string]any {
	now := time.Now().UnixMilli()
	return map[string]any{
		"id":          procID + "-" + randomSuffix(),
		"proc_id":     procID,
		"success":     false,
		"docs":        []map[string]any{},
		"tokens":      []map[string]any{},
		"sources":     []map[string]any{},
		"create_time": now,
		"done_time":   now,
	}
}

func procLogDone(log map[string]any, err error) {
	log["done_time"] = time.Now().UnixMilli()
	if err != nil {
		log["success"] = false
		log["error"] = err.Error()
	} else {
		log["success"] = true
	}
}

func appendProcLog(token map[string]any, procID string, log map[string]any) {
	logs := tokenLogs(token)
	list, _ := logs[procID].([]map[string]any)
	list = append(list, log)
	logs[procID] = list
}

func tokenLogs(token map[string]any) map[string]any {
	logs, ok := token["logs"].(map[string]any)
	if !ok || logs == nil {
		logs = map[string]any{}
		token["logs"] = logs
	}
	return logs
}

// NewReqLogForToken creates a request log entry keyed by token id (Java Log.Req).
func NewReqLogForToken(tokenID string) map[string]any {
	return newReqLog(tokenID)
}

// EnsureReqLog attaches req log under logs[tokenID].
func EnsureReqLog(token map[string]any, reqLog map[string]any) {
	id, _ := token["id"].(string)
	logs := tokenLogs(token)
	logs[id] = []map[string]any{reqLog}
}

// ReqLogDone marks request log and token success flag.
func ReqLogDone(token map[string]any, err error) {
	reqLogDone(token, err)
}

func newReqLog(tokenID string) map[string]any {
	now := time.Now().UnixMilli()
	return map[string]any{
		"id":          tokenID + randomSuffix(),
		"token_id":    tokenID,
		"success":     false,
		"create_time": now,
		"done_time":   now,
	}
}

func reqLogDone(token map[string]any, err error) {
	id, _ := token["id"].(string)
	logs := tokenLogs(token)
	var reqLog map[string]any
	if list, ok := logs[id].([]map[string]any); ok && len(list) > 0 {
		reqLog = list[len(list)-1]
	} else {
		reqLog = newReqLog(id)
		logs[id] = []map[string]any{reqLog}
	}
	reqLog["done_time"] = time.Now().UnixMilli()
	if err != nil {
		reqLog["success"] = false
		reqLog["error"] = err.Error()
		token["success"] = false
	} else {
		reqLog["success"] = true
	}
}

func randomSuffix() string {
	return time.Now().Format("150405999")
}
