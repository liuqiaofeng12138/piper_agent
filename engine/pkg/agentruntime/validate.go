package agentruntime

import (
	"fmt"
	"strings"

	"piper_go/pkg/tpl"
)

var allowedMapperTypes = map[string]struct{}{
	"Index": {}, "Template": {}, "Source": {},
}

var allowedFieldMethods = map[string]struct{}{
	"Regex": {}, "Line": {}, "JSONPath": {}, "Selector": {},
}

// ValidateTemplate runs structural checks plus HTTP token build (Phase C diagnostics).
func (s *Service) ValidateTemplate(tplDoc map[string]any, vars map[string]any) []Diagnostic {
	var diags []Diagnostic
	diags = append(diags, validateStructure(tplDoc)...)
	if len(diags) > 0 {
		return diags
	}
	diags = append(diags, s.ValidateBuild(tplDoc, vars)...)
	return diags
}

func validateStructure(tplDoc map[string]any) []Diagnostic {
	var diags []Diagnostic
	if tplDoc == nil {
		return []Diagnostic{{Code: "TEMPLATE_MISSING", Message: "template document is nil", Path: "template"}}
	}
	builder, ok := tplDoc["builder"].(map[string]any)
	if !ok {
		diags = append(diags, Diagnostic{Code: "MISSING_BUILDER", Message: "builder object required", Path: "builder"})
	} else {
		btype, _ := builder["type"].(string)
		if btype == "" {
			btype = "Http"
		}
		if btype != "Http" && btype != "Chrome" {
			diags = append(diags, Diagnostic{
				Code: "INVALID_BUILDER", Message: fmt.Sprintf("unsupported builder type %q", btype), Path: "builder.type",
			})
		}
		if btype == "Http" {
			if url, _ := builder["url_tpl"].(string); strings.TrimSpace(url) == "" {
				diags = append(diags, Diagnostic{
					Code: "MISSING_URL", Message: "http builder requires url_tpl", Path: "builder.url_tpl",
				})
			}
		}
	}
	rawProcs, ok := tplDoc["procedures"].([]any)
	if !ok {
		if _, ok2 := tplDoc["procedures"].([]map[string]any); !ok2 {
			if tplDoc["procedures"] != nil {
				diags = append(diags, Diagnostic{
					Code: "INVALID_PROCEDURES", Message: "procedures must be an array", Path: "procedures",
				})
			}
			return diags
		}
	}
	for i, item := range rawProcs {
		proc, ok := item.(map[string]any)
		if !ok {
			diags = append(diags, Diagnostic{
				Code: "INVALID_PROC", Message: "procedure must be an object", Path: fmt.Sprintf("procedures[%d]", i),
			})
			continue
		}
		diags = append(diags, validateProcedure(proc, i)...)
	}
	return diags
}

func validateProcedure(proc map[string]any, index int) []Diagnostic {
	prefix := fmt.Sprintf("procedures[%d]", index)
	ptype := tpl.ProcTypeFromJSON(proc)
	if ptype == "" {
		if t, _ := proc["type"].(string); t != "" {
			ptype = t
		}
	}
	if ptype == "" {
		return []Diagnostic{{
			Code: "UNKNOWN_PROC", Message: "missing _type or type", Path: prefix + "._type",
		}}
	}
	switch ptype {
	case "Mapper":
		return validateMapper(proc, prefix)
	case "Interceptor":
		return validateInterceptor(proc, prefix)
	case "If", "For", "LoadUrlAction", "IdleAction", "FuncCallAction",
		"ScrollAction", "ClickAction", "SetValueAction", "RedirectAction", "ExecAction",
		"LoginManuallyCheckAction", "LoginAction", "ScreenshotAction":
		return nil
	default:
		return []Diagnostic{{
			Code: "UNKNOWN_PROC_TYPE", Message: fmt.Sprintf("procedure type %q not recognized in validator", ptype), Path: prefix + "._type",
		}}
	}
}

func validateInterceptor(proc map[string]any, prefix string) []Diagnostic {
	regex, _ := proc["regex"].(string)
	if strings.TrimSpace(regex) == "" {
		return []Diagnostic{{
			Code: "MISSING_INTERCEPTOR_REGEX", Message: "Interceptor requires regex", Path: prefix + ".regex",
		}}
	}
	mapper, _ := proc["mapper"].(map[string]any)
	if mapper == nil {
		return nil
	}
	return validateMapper(mapper, prefix+".mapper")
}

func validateMapper(proc map[string]any, prefix string) []Diagnostic {
	var diags []Diagnostic
	mapType, _ := proc["type"].(string)
	if mapType == "" {
		diags = append(diags, Diagnostic{
			Code: "MISSING_MAPPER_TYPE", Message: "Mapper requires type (Index|Template|Source)", Path: prefix + ".type",
		})
	} else if _, ok := allowedMapperTypes[mapType]; !ok {
		diags = append(diags, Diagnostic{
			Code: "INVALID_MAPPER_TYPE", Message: fmt.Sprintf("mapper type %q not supported", mapType), Path: prefix + ".type",
		})
	}
	if mapType == "Index" || mapType == "Template" || mapType == "Source" {
		if ref, _ := proc["ref_id"].(string); strings.TrimSpace(ref) == "" {
			diags = append(diags, Diagnostic{
				Code: "MISSING_REF", Message: "ref_id required for this mapper type", Path: prefix + ".ref_id",
			})
		}
	}
	fields, ok := proc["fields"].(map[string]any)
	if ok {
		for fname, fv := range fields {
			fm, ok := fv.(map[string]any)
			if !ok {
				diags = append(diags, Diagnostic{
					Code: "INVALID_FIELD", Message: "field must be object", Path: prefix + ".fields." + fname,
				})
				continue
			}
			method, _ := fm["method"].(string)
			if method == "" {
				diags = append(diags, Diagnostic{
					Code: "MISSING_METHOD", Message: "field method required (Regex|Line|JSONPath|Selector)", Path: prefix + ".fields." + fname + ".method",
				})
			} else if _, ok := allowedFieldMethods[method]; !ok {
				diags = append(diags, Diagnostic{
					Code: "INVALID_METHOD", Message: fmt.Sprintf("method %q not supported", method), Path: prefix + ".fields." + fname + ".method",
				})
			}
		}
	}
	return diags
}
