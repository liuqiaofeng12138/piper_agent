package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen            string `yaml:"listen"`
	LogLevel          string `yaml:"log_level"`
	MaxConcurrentRuns int    `yaml:"max_concurrent_runs"`
	Mock              bool   `yaml:"mock"`
	RelaxDeps         bool   `yaml:"relax_deps"`
	SkipStorageWait   bool   `yaml:"skip_storage_wait"`

	ConfigPath string `yaml:"-"`
	engineYAML []byte `yaml:"-"`
}

func DefaultConfigPath() string {
	return filepath.Join("..", "deploy", "config", "local.yaml")
}

func (c *Config) EngineYAML() []byte {
	return c.engineYAML
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		Listen:            ":50051",
		LogLevel:          "info",
		MaxConcurrentRuns: 4,
		Mock:              false,
		RelaxDeps:         true,
		SkipStorageWait:   false,
	}
	if path == "" {
		return cfg, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg.ConfigPath = filepath.Clean(path)

	var doc map[string]any
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	if v, ok := doc["listen"].(string); ok && v != "" {
		cfg.Listen = v
	}
	if v, ok := doc["log_level"].(string); ok && v != "" {
		cfg.LogLevel = v
	}
	if v, ok := intFromYAML(doc["max_concurrent_runs"]); ok && v > 0 {
		cfg.MaxConcurrentRuns = v
	}
	if v, ok := doc["mock"].(bool); ok {
		cfg.Mock = v
	}
	if v, ok := doc["relax_deps"].(bool); ok {
		cfg.RelaxDeps = v
	}
	if v, ok := doc["skip_storage_wait"].(bool); ok {
		cfg.SkipStorageWait = v
	}
	engine, ok := doc["engine"]
	if !ok || engine == nil {
		return cfg, nil
	}
	cfg.engineYAML, err = yaml.Marshal(engine)
	if err != nil {
		return nil, fmt.Errorf("engine section: %w", err)
	}
	return cfg, nil
}

func intFromYAML(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}
