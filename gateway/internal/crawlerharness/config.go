package crawlerharness

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ConfigPath   string
	RepoRoot     string
	ExamplesDir  string
	PromptsDir   string
	LLM          LLMConfig
	Harness      HarnessConfig
}

type LLMConfig struct {
	Model   string
	BaseURL string
	APIKey  string
}

type HarnessConfig struct {
	MaxSteps                 int
	RequireValidateBeforeRun bool
	MaxRunsPerSession        int
	SiteProbeMaxBytes        int
	SiteProbeTimeoutSeconds  float64
	AutoSaveTemplates        bool
}

func LoadConfig(configPath, repoRoot string) Config {
	cfg := Config{
		ConfigPath: configPath,
		RepoRoot:   repoRoot,
		ExamplesDir: filepath.Join(repoRoot, "shared", "examples", "templates"),
		PromptsDir:  filepath.Join(repoRoot, "shared", "prompts"),
		Harness: HarnessConfig{
			MaxSteps:                 20,
			RequireValidateBeforeRun: true,
			MaxRunsPerSession:        10,
			SiteProbeMaxBytes:        65536,
			SiteProbeTimeoutSeconds:  25,
			AutoSaveTemplates:        true,
		},
	}
	b, err := os.ReadFile(configPath)
	if err != nil {
		return cfg
	}
	var doc map[string]any
	if yaml.Unmarshal(b, &doc) != nil {
		return cfg
	}
	if llm, _ := doc["llm"].(map[string]any); llm != nil {
		if v, ok := llm["model"].(string); ok {
			cfg.LLM.Model = v
		}
		if v, ok := llm["api_key"].(string); ok {
			cfg.LLM.APIKey = v
		}
		if v, ok := llm["base_url"].(string); ok {
			cfg.LLM.BaseURL = v
		}
	}
	if har, _ := doc["harness"].(map[string]any); har != nil {
		if v, ok := intFrom(har["max_steps"]); ok {
			cfg.Harness.MaxSteps = v
		}
		if v, ok := har["require_validate_before_run"].(bool); ok {
			cfg.Harness.RequireValidateBeforeRun = v
		}
		if v, ok := intFrom(har["max_runs_per_session"]); ok {
			cfg.Harness.MaxRunsPerSession = v
		}
	}
	return cfg
}

func intFrom(v any) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case int64:
		return int(t), true
	case float64:
		return int(t), true
	default:
		return 0, false
	}
}
