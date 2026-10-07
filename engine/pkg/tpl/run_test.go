package tpl

import (
	"testing"
)

func TestRunHTTPProceduresMapperIndex(t *testing.T) {
	token := map[string]any{
		"id":     "tok1",
		"tpl_id": "tpl1",
		"vars":   map[string]any{},
		"logs":   map[string]any{},
		"r": map[string]any{
			"text": "hello world",
		},
	}
	tplDoc := map[string]any{
		"procedures": []any{
			map[string]any{
				"_type":  "one.rewind.nio.tpl.Mapper",
				"id":     "m1",
				"type":   "Index",
				"ref_id": "idx1",
				"fields": map[string]any{
					"title": map[string]any{
						"name":   "title",
						"path":   `(?<T>hello \w+)`,
						"method": "Regex",
					},
				},
			},
		},
	}
	var queued []map[string]any
	ctx := &RunContext{
		GetTemplate: func(string) map[string]any { return nil },
		QueueToken:  func(t map[string]any) { queued = append(queued, t) },
		ReadTimeout: 5000,
	}
	if err := RunHTTPProcedures(ctx, tplDoc, token); err != nil {
		t.Fatal(err)
	}
	logs, _ := token["logs"].(map[string]any)
	list, _ := logs["m1"].([]map[string]any)
	if len(list) == 0 {
		t.Fatal("expected proc log")
	}
	docs, _ := list[0]["docs"].([]map[string]any)
	if len(docs) != 1 || docs[0]["title"] != "hello world" {
		t.Fatalf("unexpected docs %#v", docs)
	}
}
