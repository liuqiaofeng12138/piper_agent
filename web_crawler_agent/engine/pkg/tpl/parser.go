package tpl

import (
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/oliveagle/jsonpath"
)

// RegMatch finds regex matches; prefers named group T like Java Parser.regMatch.
func RegMatch(src, path string, multi bool) map[int]string {
	out := map[int]string{}
	if path == "" {
		return out
	}
	path = strings.ReplaceAll(path, "(?<T>", "(?P<T>")
	if !strings.Contains(path, "(?m)") {
		path = "(?m)" + path
	}
	re, err := regexp.Compile(path)
	if err != nil {
		return out
	}
	tIdx := re.SubexpIndex("T")
	matches := re.FindAllStringSubmatchIndex(src, -1)
	for _, loc := range matches {
		if len(loc) < 2 {
			continue
		}
		start, end := loc[0], loc[1]
		val := src[start:end]
		if tIdx > 0 && 2*tIdx+1 < len(loc) {
			gs, ge := loc[2*tIdx], loc[2*tIdx+1]
			if gs >= 0 && ge > gs {
				val = src[gs:ge]
				start = gs
			}
		}
		if val != "" {
			out[start] = val
		}
		if !multi {
			break
		}
	}
	return out
}

func sortedKeys(m map[int]string) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

// CSSMatch extracts text/html from DOM via CSS selector (Java Parser.cssMatch simplified).
func CSSMatch(doc *goquery.Document, path, attribute string, multi bool) map[int]string {
	out := map[int]string{}
	if doc == nil || path == "" {
		return out
	}
	sel := doc.Selection.Find(path)
	count := 0
	sel.Each(func(i int, s *goquery.Selection) {
		if !multi && count > 0 {
			return
		}
		key := s.Text()
		if attribute != "" {
			key, _ = s.Attr(attribute)
		} else if attribute == "" && s.Length() > 0 {
			key = s.Text()
		}
		out[i*1000] = key
		count++
	})
	return out
}

// JSONPathMatch reads jsonpath from src JSON string.
func JSONPathMatch(src, path string, multi bool) map[int]string {
	out := map[int]string{}
	if path == "" {
		return out
	}
	var data any
	if err := json.Unmarshal([]byte(src), &data); err != nil {
		return out
	}
	found, err := jsonpath.JsonPathLookup(data, path)
	if err != nil {
		return out
	}
	switch arr := found.(type) {
	case []any:
		for i, o := range arr {
			if !multi && i > 0 {
				break
			}
			out[i] = jsonString(o)
		}
	default:
		out[0] = jsonString(found)
	}
	return out
}

func jsonString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

func applyReplacements(src string, replacements []map[string]any) string {
	out := src
	for _, r := range replacements {
		find, _ := r["find"].(string)
		repl, _ := r["replace"].(string)
		if find == "" {
			continue
		}
		re, err := regexp.Compile(find)
		if err != nil {
			continue
		}
		out = re.ReplaceAllString(out, repl)
	}
	return out
}

func fieldMap(fields any) map[string]map[string]any {
	result := map[string]map[string]any{}
	switch f := fields.(type) {
	case map[string]any:
		for k, v := range f {
			if m, ok := v.(map[string]any); ok {
				if _, hasName := m["name"]; !hasName {
					m["name"] = k
				}
				result[k] = m
			}
		}
	case []any:
		for _, item := range f {
			if m, ok := item.(map[string]any); ok {
				name, _ := m["name"].(string)
				if name != "" {
					result[name] = m
				}
			}
		}
	}
	return result
}

func toInt(v any, def int) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case string:
		i, _ := strconv.Atoi(t)
		return i
	default:
		return def
	}
}

func toInt64(v any, def int64) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	default:
		return def
	}
}

func cloneVars(v map[string]any) map[string]any {
	out := map[string]any{}
	for k, val := range v {
		out[k] = val
	}
	return out
}
