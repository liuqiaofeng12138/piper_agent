package tpl

import (
	"fmt"
	"regexp"
)

// CollectInterceptors returns enabled Interceptor procs (Java Template.getInterceptors).
func CollectInterceptors(tplDoc map[string]any) []map[string]any {
	procs, _ := tplDoc["procedures"].([]any)
	var out []map[string]any
	for _, p := range procs {
		proc, ok := p.(map[string]any)
		if !ok {
			continue
		}
		if ProcTypeFromJSON(proc) != "Interceptor" {
			continue
		}
		if disabled, _ := proc["disabled"].(bool); disabled {
			continue
		}
		out = append(out, proc)
	}
	return out
}

// CapturedResponse is a network response captured during Chrome navigation.
type CapturedResponse struct {
	URL     string
	Body    []byte
	Headers map[string]string
}

// RunInterceptorMatch applies Java Interceptor.matchCallback for one captured response.
func RunInterceptorMatch(ctx *RunContext, agent ExecAgent, token map[string]any, icpt map[string]any, cap CapturedResponse) {
	regex, _ := icpt["regex"].(string)
	if regex == "" {
		return
	}
	matched, err := regexp.MatchString(regex, cap.URL)
	if err != nil || !matched {
		return
	}
	procID, _ := icpt["id"].(string)
	if procID == "" {
		procID = "interceptor"
	}
	log := newProcLog(procID)
	defer func() {
		procLogDone(log, nil)
		appendProcLog(token, procID, log)
	}()

	icptType, _ := icpt["type"].(string)
	if icptType == "" {
		icptType = "Source"
	}
	switch icptType {
	case "Source":
		multi, _ := icpt["multi"].(bool)
		sources, _ := log["sources"].([]map[string]any)
		if len(sources) > 0 && !multi {
			return
		}
		src := map[string]any{
			"url": cap.URL,
		}
		log["sources"] = append(sources, src)
	case "Mapper":
		mapper, _ := icpt["mapper"].(map[string]any)
		if mapper == nil {
			return
		}
		agent.SetResponseText(string(cap.Body))
		if err := ExecMapper(ctx, mapper, agent, token, log); err != nil {
			procLogDone(log, err)
		}
	default:
		_ = fmt.Errorf("unknown interceptor type")
	}
}
