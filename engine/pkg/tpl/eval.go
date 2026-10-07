package tpl

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dop251/goja"
	"piper_go/pkg/util"
)

// EvalRule replaces {{var}} in rule then evaluates as JavaScript expression (Java Evaluator.serialEval).
func EvalRule(rule string, row map[string]any) (string, error) {
	for k, v := range row {
		repl := fmt.Sprint(v)
		rule = strings.ReplaceAll(rule, "{{"+k+"}}", repl)
	}
	if strings.Contains(rule, "{{") || strings.Contains(rule, "}}") {
		return "", fmt.Errorf("eval rule not fully constructed: %s", rule)
	}
	vm := goja.New()
	val, err := vm.RunString(rule)
	if err != nil {
		return "", err
	}
	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return "", nil
	}
	return val.String(), nil
}

// EvalBoolExpr replaces var names in expr with values (Java If CheckVars).
func EvalBoolExpr(expr string, vars map[string]any) (bool, error) {
	expr_ := expr
	for k, v := range vars {
		expr_ = strings.ReplaceAll(expr_, k, regexp.QuoteMeta(fmt.Sprint(v)))
	}
	vm := goja.New()
	val, err := vm.RunString(expr_)
	if err != nil {
		return false, err
	}
	return val.ToBoolean(), nil
}

func evalGenID(rule string, row map[string]any) (string, error) {
	s, err := EvalRule(rule, row)
	if err != nil {
		return "", err
	}
	return util.MD5Hex(s), nil
}
