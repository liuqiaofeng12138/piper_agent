package agentruntime

import "testing"

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
