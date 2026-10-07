package persistence

import "testing"

func TestDocForES(t *testing.T) {
	doc := map[string]any{
		"_index_id": "idx1",
		"title":     "hello",
	}
	out := DocForES(doc, "task1", "tpl1", "tok1")
	if out["__task_id"] != "task1" || out["__tpl_id"] != "tpl1" || out["__token_id"] != "tok1" {
		t.Fatalf("lineage fields missing: %#v", out)
	}
	if out["__index"] != "idx1" {
		t.Fatalf("expected __index idx1")
	}
	if out["id"] == nil {
		t.Fatal("expected id")
	}
}

func TestDocForES_uniquePerRow(t *testing.T) {
	d1 := map[string]any{"_index_id": "articles", "quote_text": "a", "author": "1"}
	d2 := map[string]any{"_index_id": "articles", "quote_text": "b", "author": "2"}
	id1 := DocForES(d1, "", "", "tok")["id"]
	id2 := DocForES(d2, "", "", "tok")["id"]
	if id1 == id2 {
		t.Fatalf("expected distinct ids, both %v", id1)
	}
}
