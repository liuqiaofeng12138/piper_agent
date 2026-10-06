package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen            string `yaml:"listen"`
	LogLevel          string `yaml:"log_level"`
	MaxConcurrentRuns int    `yaml:"max_concurrent_runs"`
	Mock              bool   `yaml:"mock"`
	PiperGoConfig     string `yaml:"piper_go_config"`
	RelaxDeps         bool   `yaml:"relax_deps"`
	SkipStorageWait   bool   `yaml:"skip_storage_wait"`
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
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return nil, err
	}
	if cfg.Listen == "" {
		cfg.Listen = ":50051"
	}
	if cfg.PiperGoConfig != "" && !filepath.IsAbs(cfg.PiperGoConfig) {
		cfg.PiperGoConfig = filepath.Clean(filepath.Join(filepath.Dir(path), cfg.PiperGoConfig))
	}
	return cfg, nil
}
