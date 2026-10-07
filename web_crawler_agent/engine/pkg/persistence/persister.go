package persistence

import (
	"context"
	"encoding/base64"

	"piper_go/pkg/db/es"
	"piper_go/pkg/storage"

	"github.com/zeromicro/go-zero/core/logx"
)

// Persister writes collected docs/sources after token execution (Java Persister.handle).
type Persister struct {
	es *es.Client
	s3 *storage.S3
}

func NewPersister(esClient *es.Client, s3 *storage.S3) *Persister {
	return &Persister{es: esClient, s3: s3}
}

func (p *Persister) FromToken(ctx context.Context, token map[string]any) {
	if p == nil || p.es == nil || token == nil {
		return
	}
	success, _ := token["success"].(bool)
	if !success {
		return
	}
	docs := CollectDocsFromToken(token)
	sources := CollectSourcesFromToken(token)
	p.Persist(ctx, docs, sources)
}

func (p *Persister) Persist(ctx context.Context, docs []map[string]any, sources []map[string]any) {
	for _, d := range docs {
		indexName := ESIndexNameForDoc(d)
		id, _ := d["id"].(string)
		if id == "" {
			continue
		}
		if err := p.es.Index(ctx, indexName, id, d); err != nil {
			logx.Errorf("persist doc %s/%s: %v", indexName, id, err)
		}
	}
	if len(sources) == 0 {
		return
	}
	bucket := storage.BucketSource()
	if p.s3 != nil {
		_ = p.s3.EnsureBucket(ctx, bucket)
	}
	for _, s := range sources {
		id, _ := s["id"].(string)
		if id == "" {
			continue
		}
		if p.s3 != nil {
			body := decodeSourceBody(s)
			mime, _ := s["mime"].(string)
			if err := p.s3.Put(ctx, bucket, id, body, mime); err != nil {
				logx.Errorf("persist source s3 %s: %v", id, err)
				continue
			}
			if size, ok := s["size"].(int64); ok {
				s["size"] = size
			} else {
				s["size"] = int64(len(body))
			}
		}
		delete(s, "src")
		if err := p.es.Index(ctx, IndexSource, id, s); err != nil {
			logx.Errorf("persist source es %s: %v", id, err)
		}
	}
}

func decodeSourceBody(s map[string]any) []byte {
	switch v := s["src"].(type) {
	case []byte:
		return v
	case string:
		if b, err := base64.StdEncoding.DecodeString(v); err == nil {
			return b
		}
		return []byte(v)
	default:
		return nil
	}
}
