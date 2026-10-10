package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"piper_agent/gateway/internal/config"
)

// discoverAgentSrcDirs 扫描仓库根下所有含 pyproject.toml 的 Python 子项目，将其 src 加入 PYTHONPATH。
func discoverAgentSrcDirs(repoRoot string) []string {
	entries, err := os.ReadDir(repoRoot)
	if err != nil {
		return legacyPythonSrcDirs(repoRoot)
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		proj := filepath.Join(repoRoot, name)
		if !fileExists(filepath.Join(proj, "pyproject.toml")) {
			continue
		}
		src := filepath.Join(proj, "src")
		if fileExists(src) {
			dirs = append(dirs, src)
		}
	}
	if len(dirs) == 0 {
		return legacyPythonSrcDirs(repoRoot)
	}
	return dirs
}

func legacyPythonSrcDirs(repoRoot string) []string {
	return []string{
		filepath.Join(repoRoot, "web_crawler_agent", "src"),
		filepath.Join(repoRoot, "general_agent", "src"),
		filepath.Join(repoRoot, "rag_agent", "src"),
	}
}

func venvPython(repoRoot string, proj string) string {
	venvPy := filepath.Join(repoRoot, proj, ".venv", "Scripts", "python.exe")
	if runtime.GOOS != "windows" {
		venvPy = filepath.Join(repoRoot, proj, ".venv", "bin", "python")
	}
	if fileExists(venvPy) {
		return venvPy
	}
	return ""
}

func resolveWorkerPython(repoRoot string, agent config.AgentSpec) (python string, project string, err error) {
	proj, err := agent.PythonProjectDir(repoRoot)
	if err != nil {
		return "", "", err
	}
	py := venvPython(repoRoot, proj)
	if py == "" {
		return "", proj, fmt.Errorf(
			"缺少 %s/.venv（Agent %q 须使用独立虚拟环境）。请执行: scripts\\setup-python-venvs.ps1",
			proj, agent.ID,
		)
	}
	return py, proj, nil
}
