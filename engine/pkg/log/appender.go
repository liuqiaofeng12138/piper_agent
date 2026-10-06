package log

import (
	"context"
	"sync"
	"time"

	"piper_go/pkg/db/es"

	"github.com/zeromicro/go-zero/core/logx"
)

const ESIndexLog = "log"

// Appender batches application logs to Elasticsearch (Java log.ESAppender subset).
type Appender struct {
	es     *es.Client
	instID string
	mu     sync.Mutex
	buf    []map[string]any
	onDoc  func(map[string]any)
}

func (a *Appender) SetDocHook(fn func(map[string]any)) {
	a.onDoc = fn
}

func NewAppender(esClient *es.Client, instID string) *Appender {
	a := &Appender{es: esClient, instID: instID}
	go a.flushLoop()
	return a
}

func (a *Appender) Append(level, thread, src, fn, msg string) {
	if a == nil || a.es == nil {
		return
	}
	now := time.Now().UnixMilli()
	doc := map[string]any{
		"inst_id":     a.instID,
		"lv":          level,
		"t":           thread,
		"src":         src,
		"func":        fn,
		"msg":         msg,
		"create_time": now,
		"update_time": now,
	}
	a.mu.Lock()
	a.buf = append(a.buf, doc)
	a.mu.Unlock()
}

func (a *Appender) flushLoop() {
	ticker := time.NewTicker(2 * time.Second)
	for range ticker.C {
		a.Flush(context.Background())
	}
}

func (a *Appender) Flush(ctx context.Context) {
	a.mu.Lock()
 batch := a.buf
	a.buf = nil
	a.mu.Unlock()
	for _, doc := range batch {
		id := time.Now().Format("20060102150405.000") + randomSuffix()
		doc["id"] = id
		if err := a.es.Index(ctx, ESIndexLog, id, doc); err != nil {
			logx.Errorf("log appender: %v", err)
			continue
		}
		if a.onDoc != nil {
			a.onDoc(doc)
		}
	}
}

func randomSuffix() string {
	return time.Now().Format("000000999")
}
