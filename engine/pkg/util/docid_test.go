package util

import "testing"

func TestAssignDocumentID_contentFingerprint(t *testing.T) {
	a := map[string]any{"quote_text": "hello", "author": "A"}
	b := map[string]any{"quote_text": "world", "author": "B"}
	idA := AssignDocumentID(a, "", "tok1", "articles")
	idB := AssignDocumentID(b, "", "tok1", "articles")
	if idA == idB {
		t.Fatalf("expected different ids for different rows, got %s", idA)
	}
	idA2 := AssignDocumentID(a, "", "tok1", "articles")
	if idA != idA2 {
		t.Fatalf("expected stable id for same row")
	}
}

func TestAssignDocumentID_title(t *testing.T) {
	fields := map[string]any{"title": "My Book", "price": "1"}
	id := AssignDocumentID(fields, "", "tok", "idx")
	if id != MD5Hex("My Book") {
		t.Fatalf("expected title hash, got %s", id)
	}
}
