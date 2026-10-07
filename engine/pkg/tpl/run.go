package tpl

import (
	"fmt"
	"strings"
	"time"
)

// RunContext carries template execution dependencies.
type RunContext struct {
	GetTemplate func(id string) map[string]any
	QueueToken  func(map[string]any)
	CallFunc    func(funcID string, varNames []string, vars map[string]any) (any, error)
	ReadTimeout int
}

// RunHTTPProcedures executes procedures using HTTP agent (Phase 3a).
func RunHTTPProcedures(ctx *RunContext, tplDoc map[string]any, token map[string]any) error {
	agent := NewHTTPAgent(token, ctx.ReadTimeout)
	return RunProcedures(ctx, tplDoc, token, agent)
}

// RunProcedures executes template procedures (Java Template.run).
func RunProcedures(ctx *RunContext, tplDoc map[string]any, token map[string]any, agent ExecAgent) error {
	procs, _ := tplDoc["procedures"].([]any)
	for _, p := range procs {
		proc, ok := p.(map[string]any)
		if !ok {
			continue
		}
		if disabled, _ := proc["disabled"].(bool); disabled {
			continue
		}
		if ProcTypeFromJSON(proc) == "Interceptor" {
			continue
		}
		if err := execProc(ctx, proc, agent, token); err != nil {
			return err
		}
	}
	return nil
}

func execProc(ctx *RunContext, proc map[string]any, agent ExecAgent, token map[string]any) error {
	procID, _ := proc["id"].(string)
	if procID == "" {
		procID = ProcTypeFromJSON(proc)
	}
	log := newProcLog(procID)
	var execErr error
	defer func() {
		procLogDone(log, execErr)
		appendProcLog(token, procID, log)
	}()

	typ := ProcTypeFromJSON(proc)
	switch typ {
	case "Mapper":
		execErr = ExecMapper(ctx, proc, agent, token, log)
	case "If":
		execErr = execIf(ctx, proc, agent, token, log)
	case "For":
		execErr = execFor(ctx, proc, agent, token, log)
	case "IdleAction":
		ms := toInt64(proc["idleTime"], 1000)
		time.Sleep(time.Duration(ms) * time.Millisecond)
	case "FuncCallAction":
		execErr = execFuncCall(ctx, proc, token, log)
	case "LoadUrlAction":
		url := ReplaceVars(strVal(proc, "url"), tokenVars(token), true)
		if err := agent.LoadURL(url); err != nil {
			execErr = err
		} else {
			execErr = fmt.Errorf("scheduled retry")
		}
	case "ScrollAction", "ScrollLoadMoreAction":
		execErr = agent.ChromeScroll(proc, tokenVars(token))
	case "ClickAction", "ClickByContentAction":
		css := strVal(proc, "elementCssPath")
		if css == "" {
			css = strVal(proc, "css")
		}
		execErr = agent.ChromeClick(css)
	case "SetValueAction":
		execErr = agent.ChromeSetValue(strVal(proc, "inputCssPath"), strVal(proc, "value"), tokenVars(token))
	case "RedirectAction":
		url := ReplaceVars(strVal(proc, "url"), tokenVars(token), true)
		execErr = agent.ChromeNavigate(url)
	case "ScreenshotAction", "ExecAction", "ClearCacheAction", "MouseHoverAction", "DragAction":
		// Partial: ExecAction runs JS if script field present
		if script := strVal(proc, "script"); script != "" {
			_, execErr = agent.ChromeEvalJS(script)
		}
	case "LoginAction", "LoginManuallyCheckAction":
		// Account/login flows depend on account DB — handled at chrome token wrapper layer
	default:
		if strings.Contains(typ, "Chrome") || strings.HasPrefix(typ, "Scroll") {
			// unknown chrome action — no-op on HTTP
		}
	}
	return execErr
}

func execIf(ctx *RunContext, proc map[string]any, agent ExecAgent, token map[string]any, log map[string]any) error {
	_ = log
	errStr := evalIf(agent, token, proc)
	if errStr == "" {
		return nil
	}
	if e, ok := proc["e"].(string); ok && e != "" {
		return fmt.Errorf("%s", e)
	}
	if action, ok := proc["action"].(map[string]any); ok {
		return execProc(ctx, action, agent, token)
	}
	return fmt.Errorf("if not satisfy: %s", errStr)
}

func evalIf(agent ExecAgent, token map[string]any, proc map[string]any) string {
	ifType, _ := proc["type"].(string)
	if ifType == "" {
		ifType = "CheckSource"
	}
	switch ifType {
	case "CheckSource":
		src := agent.GetSrc()
		if contains, ok := proc["contains"].([]any); ok {
			for _, w := range contains {
				if !strings.Contains(src, fmt.Sprint(w)) {
					return fmt.Sprint(w) + " not present"
				}
			}
		}
		if notContain, ok := proc["not_contain"].([]any); ok {
			for _, w := range notContain {
				if strings.Contains(src, fmt.Sprint(w)) {
					return "Should not contain " + fmt.Sprint(w)
				}
			}
		}
	case "CheckElement":
		path := strVal(proc, "elementPath")
		if path == "" {
			path = strVal(proc, "elementCssPath")
		}
		if agent.IsChrome() && path != "" {
			n, err := agent.ElementCount(path)
			if err != nil {
				return err.Error()
			}
			if n == 0 {
				return path + " not present"
			}
		}
	case "CheckVars":
		expr, _ := proc["expr"].(string)
		if expr != "" {
			ok, err := EvalBoolExpr(expr, tokenVars(token))
			if err != nil {
				return err.Error()
			}
			if !ok {
				return expr + " not true"
			}
		}
	}
	return ""
}

func execFor(ctx *RunContext, proc map[string]any, agent ExecAgent, token map[string]any, log map[string]any) error {
	_ = log
	repeat := toInt(proc["repeat"], 100)
	idle := toInt64(proc["idleTime"], 1000)
	inner, _ := proc["procedures"].([]any)
	prevSize := domSize(agent)
	sameCount := 0
	for c := 0; c < repeat; c++ {
		for _, p := range inner {
			child, ok := p.(map[string]any)
			if !ok {
				continue
			}
			typ := ProcTypeFromJSON(child)
			if typ == "For" || typ == "Interceptor" {
				continue
			}
			if err := execProc(ctx, child, agent, token); err != nil {
				return err
			}
		}
		if agent.IsChrome() {
			newSize := domSize(agent)
			if newSize == prevSize {
				sameCount++
				if sameCount > 3 {
					break
				}
			} else {
				prevSize = newSize
				sameCount = 0
			}
		}
		time.Sleep(time.Duration(idle) * time.Millisecond)
	}
	return nil
}

func domSize(agent ExecAgent) int {
	doc := agent.GetDom()
	if doc == nil {
		return 0
	}
	return doc.Find("*").Length()
}

func execFuncCall(ctx *RunContext, proc map[string]any, token map[string]any, log map[string]any) error {
	_ = log
	if ctx.CallFunc == nil {
		return fmt.Errorf("func caller not configured")
	}
	funcID, _ := proc["func_id"].(string)
	varNames := []string{}
	switch vn := proc["var_names"].(type) {
	case []any:
		for _, n := range vn {
			varNames = append(varNames, fmt.Sprint(n))
		}
	case []string:
		varNames = vn
	}
	vars := map[string]any{}
	for _, name := range varNames {
		v := tokenVars(token)[name]
		if v == nil {
			return fmt.Errorf("missing var %s", name)
		}
		vars[name] = v
	}
	res, err := ctx.CallFunc(funcID, varNames, vars)
	if err != nil {
		return err
	}
	tokenVars(token)[funcID] = res
	return nil
}
