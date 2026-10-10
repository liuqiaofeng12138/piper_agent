package agentruntime

import "testing"

func TestValidateInterceptorProcedure(t *testing.T) {
	tplDoc := map[string]any{
		"builder": map[string]any{"type": "Chrome", "url_tpl": "https://weibo.com/u/1"},
		"procedures": []any{
			map[string]any{
				"_type": "one.rewind.nio.tpl.Interceptor",
				"id":    "ic",
				"regex": "mymblog",
				"type":  "Mapper",
				"mapper": map[string]any{
					"_type":  "one.rewind.nio.tpl.Mapper",
					"type":   "Index",
					"ref_id": "page",
					"fields": map[string]any{
						"t": map[string]any{"method": "JSONPath", "path": "$.data.list[0].text_raw"},
					},
				},
			},
		},
	}
	diags := validateStructure(tplDoc)
	for _, d := range diags {
		if d.Code == "UNKNOWN_PROC_TYPE" {
			t.Fatalf("interceptor should validate: %#v", diags)
		}
	}
}

func TestValidateStructureMapperPath(t *testing.T) {
	tplDoc := map[string]any{
		"builder": map[string]any{"type": "Http", "url_tpl": "https://example.com"},
		"procedures": []any{
			map[string]any{
				"_type": "one.rewind.nio.tpl.Mapper",
				"type":  "Index",
				"fields": map[string]any{
					"title": map[string]any{"method": "BadMethod"},
				},
			},
		},
	}
	diags := validateStructure(tplDoc)
	if len(diags) == 0 {
		t.Fatal("expected diagnostics")
	}
	found := false
	for _, d := range diags {
		if d.Path == "procedures[0].fields.title.method" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected field method path, got %#v", diags)
	}
}
