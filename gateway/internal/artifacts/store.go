package artifacts

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var idPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

// ResolveDir 将配置中的相对路径解析为绝对目录（相对 yaml 所在目录）。
func ResolveDir(configPath, dir string) (string, error) {
	if strings.TrimSpace(dir) == "" {
		dir = "../../data/artifacts"
	}
	p := dir
	if !filepath.IsAbs(p) {
		base := filepath.Dir(configPath)
		if base == "" || base == "." {
			base = "."
		}
		p = filepath.Join(base, dir)
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", err
	}
	return abs, nil
}

// PPTXPath 返回 artifact 在磁盘上的路径（仅允许 32 位十六进制 id）。
func PPTXPath(artifactsDir, id string) (string, error) {
	if !idPattern.MatchString(id) {
		return "", fmt.Errorf("invalid artifact id")
	}
	return filepath.Join(artifactsDir, id+".pptx"), nil
}

// SanitizeFilename 用于 Content-Disposition。
func SanitizeFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." {
		return "presentation.pptx"
	}
	var b strings.Builder
	for _, r := range name {
		if r < 32 || r == '"' || r == '\\' {
			continue
		}
		b.WriteRune(r)
	}
	out := b.String()
	if out == "" {
		return "presentation.pptx"
	}
	if !strings.HasSuffix(strings.ToLower(out), ".pptx") {
		out += ".pptx"
	}
	return out
}
