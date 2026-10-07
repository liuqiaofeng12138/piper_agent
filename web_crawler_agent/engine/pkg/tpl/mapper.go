package tpl

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"piper_go/pkg/util"
)

// ExecMapper runs a Mapper proc document against HTTP response (Phase 3 HTTP).
func ExecMapper(ctx *RunContext, proc map[string]any, agent ExecAgent, token map[string]any, log map[string]any) error {
	src := agent.GetSrc()
	src = strings.TrimPrefix(src, "<html><head></head><body><pre style=\"word-wrap: break-word; white-space: pre-wrap;\">")
	src = regexp.MustCompile(`</pre></body></html>$`).ReplaceAllString(src, "")

	dom := agent.GetDom()
	selectMethod, _ := proc["select_method"].(string)
	selectRule, _ := proc["select_rule"].(string)
	multi, _ := proc["multi"].(bool)
	fields := fieldMap(proc["fields"])

	varsList, err := mapperSelectAndParse(src, dom, selectMethod, selectRule, multi, fields, tokenVars(token))
	if err != nil {
		essential, _ := proc["essential"].(bool)
		if essential || proc["essential"] == nil {
			return err
		}
		return nil
	}

	mapperType, _ := proc["type"].(string)
	refID, _ := proc["ref_id"].(string)
	essential, _ := proc["essential"].(bool)
	if proc["essential"] == nil {
		essential = true
	}

	for _, row := range varsList {
		switch mapperType {
		case "Index":
			if refID == "" {
				continue
			}
			doc := map[string]any{"_index_id": refID}
			for k, v := range row {
				doc[k] = v
			}
			setDocID(doc)
			appendDoc(log, doc)
		case "Template":
			if err := spawnTemplateToken(ctx, proc, refID, row, token, log, essential); err != nil {
				return err
			}
		case "Source":
			appendSource(log, row)
		default:
			// YouTubeVideo, BilibiliVideo — Phase 3b
		}
	}
	if len(varsList) > 0 {
		mergeVars(token, varsList[0])
	}
	return nil
}

func mapperSelectAndParse(src string, dom *goquery.Document, selectMethod, selectRule string, multi bool, fields map[string]map[string]any, vars map[string]any) ([]map[string]any, error) {
	if src == "" {
		return nil, fmt.Errorf("source not set")
	}
	var varsList []map[string]any

	switch selectMethod {
	case "Regex":
		if selectRule != "" {
			selected := RegMatch(src, selectRule, false)
			if len(selected) == 0 {
				return nil, fmt.Errorf("mapper path invalid")
			}
			keys := sortedKeys(selected)
			src = selected[keys[0]]
		}
		list, err := parseFields(src, dom, fields, vars, multi)
		if err != nil {
			return nil, err
		}
		varsList = append(varsList, list...)
	case "Selector":
		if dom == nil {
			return nil, fmt.Errorf("doc not set")
		}
		// An empty mapper select_rule means “use the full response document”;
		// individual fields still apply their own CSS selectors in parseFields.
		// This matches the Java Mapper behavior and is needed for Index mappers
		// that extract multiple records directly from the response page.
		if selectRule != "" {
			sel := dom.Find(selectRule)
			if sel.Length() == 0 {
				return nil, fmt.Errorf("mapper path invalid")
			}
			src, _ = sel.First().Html()
		}
		list, err := parseFields(src, dom, fields, vars, multi)
		if err != nil {
			return nil, err
		}
		varsList = append(varsList, list...)
	case "Line":
		for _, line := range regexp.MustCompile(`\r?\n`).Split(src, -1) {
			if selectRule != "" {
				matched, _ := regexp.MatchString(selectRule, line)
				if !matched {
					continue
				}
			}
			list, err := parseFields(line, nil, fields, vars, false)
			if err != nil {
				return nil, err
			}
			varsList = append(varsList, list...)
		}
	case "JSONPath":
		if selectRule == "" {
			return nil, fmt.Errorf("mapper path invalid")
		}
		matches := JSONPathMatch(src, selectRule, multi)
		for _, k := range sortedKeys(matches) {
			list, err := parseFields(matches[k], nil, fields, vars, false)
			if err != nil {
				return nil, err
			}
			varsList = append(varsList, list...)
		}
	default:
		list, err := parseFields(src, dom, fields, vars, multi)
		if err != nil {
			return nil, err
		}
		varsList = append(varsList, list...)
	}
	return varsList, nil
}

func parseFields(src string, dom *goquery.Document, fields map[string]map[string]any, vars map[string]any, multi bool) ([]map[string]any, error) {
	hasSelector := false
	for _, f := range fields {
		if m, _ := f["method"].(string); m == "Selector" {
			hasSelector = true
			break
		}
	}
	if hasSelector && dom == nil {
		return nil, fmt.Errorf("doc not set")
	}

	founds := map[string]map[int]string{}
	for k, v := range vars {
		if k == "id" {
			continue
		}
		founds[k] = map[int]string{0: fmt.Sprint(v)}
	}

	extractNames := map[string]struct{}{}
	for _, f := range fields {
		if path, _ := f["path"].(string); path != "" {
			name, _ := f["name"].(string)
			extractNames[name] = struct{}{}
		}
	}
	for name := range extractNames {
		delete(founds, name)
	}

	for _, f := range fields {
		name, _ := f["name"].(string)
		path, _ := f["path"].(string)
		method, _ := f["method"].(string)
		if method == "" {
			method = "Regex"
		}
		items := map[int]string{}
		if path != "" {
			var raw map[int]string
			switch method {
			case "Selector":
				attr, _ := f["attribute"].(string)
				raw = CSSMatch(dom, path, attr, multi)
			case "JSONPath":
				raw = JSONPathMatch(src, path, multi)
			default:
				raw = RegMatch(src, path, multi)
			}
			repls, _ := f["replacements"].([]any)
			replMaps := []map[string]any{}
			for _, r := range repls {
				if m, ok := r.(map[string]any); ok {
					replMaps = append(replMaps, m)
				}
			}
			for i, val := range raw {
				val = applyReplacements(val, replMaps)
				nullable := true
				if v, ok := f["nullable"].(bool); ok {
					nullable = v
				}
				if nullable || val != "" {
					items[i] = val
				}
			}
		}
		if len(items) == 0 {
			if def, ok := f["defaultString"].(string); ok && def != "" {
				items[0] = def
			}
		}
		if len(items) > 0 {
			founds[name] = items
		}
	}

	for _, f := range fields {
		name, _ := f["name"].(string)
		nullable := true
		if v, ok := f["nullable"].(bool); ok {
			nullable = v
		}
		evalRule, _ := f["evalRule"].(string)
		if evalRule == "" && !nullable {
			if m, ok := founds[name]; !ok || len(m) == 0 {
				return nil, fmt.Errorf("field[%s] not allowed null", name)
			}
		}
	}

	recordSize := 0
	for _, m := range founds {
		if len(m) > recordSize {
			recordSize = len(m)
		}
	}
	if recordSize == 0 {
		recordSize = 1
	}

	normal := map[string]map[int]string{}
	special := map[string]map[int]string{}
	for name, m := range founds {
		if len(m) == recordSize || recordSize == 1 {
			normal[name] = m
		} else {
			special[name] = m
		}
	}

	var data []map[string]any
	for i := 0; i < recordSize; i++ {
		row := map[string]any{}
		for name, m := range normal {
			keys := sortedKeys(m)
			if i < len(keys) {
				row[name] = m[keys[i]]
			} else if len(keys) > 0 {
				row[name] = m[keys[0]]
			}
		}
		data = append(data, row)
	}

	for _, row := range data {
		for name, m := range special {
			if len(m) == 0 {
				continue
			}
			keys := sortedKeys(m)
			row[name] = m[keys[0]]
		}
	}

	for _, row := range data {
		for _, f := range fields {
			name, _ := f["name"].(string)
			evalRule, _ := f["evalRule"].(string)
			if evalRule == "" {
				continue
			}
			genID, _ := f["genId"].(bool)
			if genID {
				id, err := evalGenID(evalRule, row)
				if err != nil {
					return nil, err
				}
				row[name] = id
				continue
			}
			val, err := EvalRule(evalRule, row)
			if err != nil {
				return nil, err
			}
			if val != "" {
				row[name] = val
			}
		}
	}

	if len(data) == 0 {
		data = append(data, map[string]any{})
	}
	return dedupeRows(data), nil
}

func dedupeRows(rows []map[string]any) []map[string]any {
	seen := map[string]struct{}{}
	var out []map[string]any
	for _, r := range rows {
		b, _ := json.Marshal(r)
		key := string(b)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, r)
	}
	return out
}

func spawnTemplateToken(ctx *RunContext, proc map[string]any, refID string, row map[string]any, token map[string]any, log map[string]any, essential bool) error {
	ttl := toInt(token["ttl"], 255)
	if ttl == 1 {
		return fmt.Errorf("token ttl 0")
	}
	tplID, _ := token["tpl_id"].(string)
	if refID == "" {
		refID = tplID
	}
	tplDoc := ctx.GetTemplate(refID)
	if tplDoc == nil {
		if essential {
			return fmt.Errorf("template[%s] not found", refID)
		}
		return nil
	}
	vars := cloneVars(tokenVars(token))
	for k, v := range row {
		vars[k] = v
	}
	opts := RunOpts{
		UID:      strVal(token, "uid"),
		TaskID:   strVal(token, "task_id"),
		NodeID:   strVal(token, "node_id"),
		AgentID:  strVal(token, "agent_id"),
		Behavior: "DEFAULT",
	}
	if refID == tplID {
		opts.Behavior = "NEXT_PAGE"
	}
	child, err := BuildToken(tplDoc, vars, opts)
	if err != nil {
		if essential {
			return err
		}
		return nil
	}
	appendChildToken(log, child)
	ctx.QueueToken(child)
	return nil
}

func appendDoc(log map[string]any, doc map[string]any) {
	docs, _ := log["docs"].([]map[string]any)
	log["docs"] = append(docs, doc)
}

func appendSource(log map[string]any, row map[string]any) {
	url, _ := row["url"].(string)
	src := map[string]any{"url": url}
	for k, v := range row {
		src[k] = v
	}
	sources, _ := log["sources"].([]map[string]any)
	log["sources"] = append(sources, src)
}

func appendChildToken(log map[string]any, tok map[string]any) {
	tokens, _ := log["tokens"].([]map[string]any)
	log["tokens"] = append(tokens, tok)
}

func setDocID(doc map[string]any) {
	indexID, _ := doc["_index_id"].(string)
	explicitID, _ := doc["id"].(string)
	fields := map[string]any{}
	for k, v := range doc {
		if strings.HasPrefix(k, "_") {
			continue
		}
		fields[k] = v
	}
	doc["id"] = util.AssignDocumentID(fields, explicitID, "", indexID)
}

func mergeVars(token map[string]any, row map[string]any) {
	v := tokenVars(token)
	for k, val := range row {
		v[k] = val
	}
}

func tokenVars(token map[string]any) map[string]any {
	v, ok := token["vars"].(map[string]any)
	if !ok || v == nil {
		v = map[string]any{}
		token["vars"] = v
	}
	return v
}

func strVal(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}
