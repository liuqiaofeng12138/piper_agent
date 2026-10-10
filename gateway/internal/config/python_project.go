package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PythonProjectDir 返回该 Agent 独占的 Python 项目目录名（与仓库根下含 pyproject.toml 的文件夹对应）。
// 可通过 agents[].python_project 显式覆盖；否则从 module 首段推断（piper_agent → general_agent）。
func (a AgentSpec) PythonProjectDir(repoRoot string) (string, error) {
	if p := strings.TrimSpace(a.PythonProject); p != "" {
		if err := validatePythonProject(repoRoot, p); err != nil {
			return "", err
		}
		return p, nil
	}
	mod := a.WorkerModule()
	top := strings.Split(mod, ".")[0]
	if top == "" {
		return "", fmt.Errorf("agent %q: empty worker module", a.ID)
	}
	if top == "piper_agent" {
		const dir = "general_agent"
		if err := validatePythonProject(repoRoot, dir); err != nil {
			return "", fmt.Errorf("agent %q: %w", a.ID, err)
		}
		return dir, nil
	}
	if err := validatePythonProject(repoRoot, top); err != nil {
		return "", fmt.Errorf(
			"agent %q module %q: %w（可在 agents 配置中设置 python_project 指向含 pyproject.toml 的目录）",
			a.ID, mod, err,
		)
	}
	return top, nil
}

func validatePythonProject(repoRoot, dir string) error {
	root := filepath.Join(repoRoot, dir)
	if _, err := os.Stat(filepath.Join(root, "pyproject.toml")); err != nil {
		return fmt.Errorf("未找到 %s/pyproject.toml", dir)
	}
	return nil
}
