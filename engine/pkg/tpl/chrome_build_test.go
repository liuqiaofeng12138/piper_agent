package tpl

import "testing"

func TestBuildChromeToken(t *testing.T) {
	tplDoc := map[string]any{
		"id": "tpl-chrome-1",
		"builder": map[string]any{
			"type":    "Chrome",
			"url_tpl": "https://example.com/search?w={{w}}",
			"domain":  "example.com",
		},
	}
	tok, err := BuildChromeToken(tplDoc, map[string]any{"w": "hello"}, RunOpts{Behavior: "TEST"})
	if err != nil {
		t.Fatal(err)
	}
	if tok["type"] != "Chrome" {
		t.Fatalf("type %v", tok["type"])
	}
	if tok["url"] != "https://example.com/search?w=hello" {
		t.Fatalf("url %v", tok["url"])
	}
}

func TestCollectInterceptors(t *testing.T) {
	tplDoc := map[string]any{
		"procedures": []any{
			map[string]any{"_type": "one.rewind.nio.tpl.Mapper", "id": "m1"},
			map[string]any{"_type": "one.rewind.nio.tpl.Interceptor", "id": "i1", "regex": ".*"},
		},
	}
	list := CollectInterceptors(tplDoc)
	if len(list) != 1 {
		t.Fatalf("expected 1 interceptor, got %d", len(list))
	}
}
