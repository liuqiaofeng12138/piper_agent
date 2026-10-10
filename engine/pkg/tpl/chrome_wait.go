package tpl

import "strings"

// ChromeBuilderRequireLogin marks templates that need a logged-in browser session.
func ChromeBuilderRequireLogin(tplDoc map[string]any) bool {
	if tplDoc == nil {
		return false
	}
	builder, _ := tplDoc["builder"].(map[string]any)
	if builder == nil {
		return false
	}
	v, _ := builder["require_login"].(bool)
	return v
}

// ChromeBuilderWait reads optional post-login wait from template builder.
func ChromeBuilderWait(tplDoc map[string]any) (selector string, waitSeconds int) {
	if tplDoc == nil {
		return "", 0
	}
	builder, _ := tplDoc["builder"].(map[string]any)
	if builder == nil {
		return "", 0
	}
	selector, _ = builder["wait_selector"].(string)
	selector = strings.TrimSpace(selector)
	waitSeconds = int(toInt64(builder["wait_seconds"], 0))
	return selector, waitSeconds
}

// TemplateHasManualLoginProcedure reports LoginManuallyCheckAction in top-level procedures.
func TemplateHasManualLoginProcedure(tplDoc map[string]any) bool {
	if tplDoc == nil {
		return false
	}
	procs, _ := tplDoc["procedures"].([]any)
	for _, p := range procs {
		proc, ok := p.(map[string]any)
		if !ok {
			continue
		}
		if ProcTypeFromJSON(proc) == "LoginManuallyCheckAction" {
			return true
		}
	}
	return false
}
