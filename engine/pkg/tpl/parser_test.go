package tpl

import "testing"

func TestRegMatchNamedGroup(t *testing.T) {
	src := "Current IP Address: 1.2.3.4\n"
	m := RegMatch(src, `(?<T>Current IP Address.+?)$`, false)
	if len(m) != 1 {
		t.Fatalf("expected 1 match, got %d", len(m))
	}
	for _, v := range m {
		if v != "Current IP Address: 1.2.3.4" {
			t.Fatalf("unexpected value %q", v)
		}
	}
}

func TestEvalRuleIncrement(t *testing.T) {
	row := map[string]any{"page": "1"}
	out, err := EvalRule("{{page}}+1", row)
	if err != nil {
		t.Fatal(err)
	}
	if out != "2" {
		t.Fatalf("expected 2, got %q", out)
	}
}

func TestParseFieldsRegex(t *testing.T) {
	fields := map[string]map[string]any{
		"title": {
			"name":   "title",
			"path":   `(?<T>Current IP Address.+?)$`,
			"method": "Regex",
			"replacements": []any{
				map[string]any{"find": "Current IP Address: ", "replace": ""},
			},
		},
	}
	rows, err := parseFields("Current IP Address: 9.9.9.9\n", nil, fields, map[string]any{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["title"] != "9.9.9.9" {
		t.Fatalf("unexpected rows: %#v", rows)
	}
}
