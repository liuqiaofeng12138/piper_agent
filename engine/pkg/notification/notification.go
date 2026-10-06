package notification

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"piper_go/pkg/db/es"
	"piper_go/pkg/util"
	"piper_go/pkg/websocket"
)

const ESIndex = "notification"

// Service persists notifications to ES and broadcasts on /msg (Java Notification.save).
type Service struct {
	es      *es.Client
	instID  string
	enabled bool
}

func NewService(esClient *es.Client, instID string, enabled bool) *Service {
	return &Service{es: esClient, instID: instID, enabled: enabled}
}

func (s *Service) Enabled() bool {
	return s != nil && s.enabled
}

func (s *Service) SaveAndBroadcast(ctx context.Context, doc map[string]any) {
	if s == nil || s.es == nil || !s.enabled {
		return
	}
	now := time.Now().UnixMilli()
	if _, ok := doc["create_time"]; !ok {
		doc["create_time"] = now
	}
	doc["update_time"] = now
	doc["inst_id"] = s.instID
	if _, ok := doc["read"]; !ok {
		doc["read"] = false
	}
	id, _ := doc["id"].(string)
	if id == "" {
		typ, _ := doc["type"].(string)
		objID, _ := doc["obj_id"].(string)
		if objID == "" {
			objID = fmt.Sprint(time.Now().UnixNano())
		}
		id = util.MD5Hex(s.instID + "::" + typ + "::" + objID)
		doc["id"] = id
	}
	_ = s.es.Index(ctx, ESIndex, id, doc)
	websocket.BroadcastAll(doc)
}

func (s *Service) FromLog(ctx context.Context, logDoc map[string]any) {
	lv, _ := logDoc["lv"].(string)
	if lv != "WARN" && lv != "ERROR" && lv != "FATAL" {
		return
	}
	priority := "HIGH"
	switch lv {
	case "ERROR":
		priority = "HIGHER"
	case "FATAL":
		priority = "HIGHEST"
	}
	msg, _ := logDoc["msg"].(string)
	id, _ := logDoc["id"].(string)
	if id == "" {
		id = util.MD5Hex(s.instID + "::Node::" + msg)
	}
	s.SaveAndBroadcast(ctx, map[string]any{
		"id":       id,
		"type":     "Node",
		"priority": priority,
		"msg":      msg,
	})
}

func (s *Service) List(ctx context.Context, instID string, unreadOnly bool, q string, st, et int64, page, size int64) ([]map[string]any, int64, error) {
	must := []map[string]any{
		es.TermQuery("inst_id", instID),
		es.RangeQuery("update_time", st, et),
	}
	if unreadOnly {
		must = append(must, es.TermQuery("read", false))
	}
	if q != "" {
		must = append(must, es.QueryString(url.QueryEscape(q)))
	}
	from := int((page - 1) * size)
	body := es.SearchBody(es.BoolMust(must...), from, int(size), "update_time", true)
	res, err := s.es.Search(ctx, ESIndex, body)
	if err != nil {
		return nil, 0, err
	}
	return res.Hits, res.Total, nil
}

func (s *Service) MarkRead(ctx context.Context, id string) error {
	return s.es.UpdateFields(ctx, ESIndex, id, map[string]any{"read": true})
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.es.DeleteDoc(ctx, ESIndex, id)
}

func NewListDoc(priority, msg string, refObj any) map[string]any {
	return map[string]any{
		"type":     "List",
		"priority": priority,
		"msg":      msg,
		"ref_obj":  refObj,
	}
}
