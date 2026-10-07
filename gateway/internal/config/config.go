package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen              string `yaml:"listen"`
	ConfigPath          string `yaml:"-"`
	Auth                Auth   `yaml:"auth"`
	Session             SessionConfig `yaml:"session"`
	Classifier          ClassifierConfig `yaml:"classifier"`
	ChatTimeoutSeconds  int    `yaml:"chat_timeout_seconds"`
	RuntimeAddress      string `yaml:"-"`
	Agents              AgentsConfig `yaml:"agents"`
}

type Auth struct {
	Mode  string `yaml:"mode"`
	Token string `yaml:"token"`
}

type SessionConfig struct {
	Driver     string `yaml:"driver"`
	SQLitePath string `yaml:"sqlite_path"`
}

type ClassifierConfig struct {
	Mode string `yaml:"mode"`
}

type AgentsConfig struct {
	WebCrawler AgentEndpoint `yaml:"web_crawler"`
}

type AgentEndpoint struct {
	Address string `yaml:"address"`
	Enabled bool   `yaml:"enabled"`
}

func DefaultConfigPath() string {
	return filepath.Join("..", "deploy", "config", "local.yaml")
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		Listen:             ":8080",
		Auth:               Auth{Mode: "none"},
		Session:            SessionConfig{Driver: "sqlite", SQLitePath: "../../data/gateway.db"},
		Classifier:         ClassifierConfig{Mode: "rule"},
		ChatTimeoutSeconds: 600,
		Agents: AgentsConfig{
			WebCrawler: AgentEndpoint{Address: "127.0.0.1:50061", Enabled: true},
		},
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
	gw, _ := doc["gateway"].(map[string]any)
	if gw != nil {
		if v, ok := gw["listen"].(string); ok && v != "" {
			cfg.Listen = v
		}
		if v, ok := intFromYAML(gw["chat_timeout_seconds"]); ok && v > 0 {
			cfg.ChatTimeoutSeconds = v
		}
		auth, _ := gw["auth"].(map[string]any)
		if auth != nil {
			if v, ok := auth["mode"].(string); ok && v != "" {
				cfg.Auth.Mode = v
			}
			if v, ok := auth["token"].(string); ok {
				cfg.Auth.Token = v
			}
		}
		sess, _ := gw["session"].(map[string]any)
		if sess != nil {
			if v, ok := sess["driver"].(string); ok && v != "" {
				cfg.Session.Driver = v
			}
			if v, ok := sess["sqlite_path"].(string); ok && v != "" {
				cfg.Session.SQLitePath = v
			}
		}
		cls, _ := gw["classifier"].(map[string]any)
		if cls != nil {
			if v, ok := cls["mode"].(string); ok && v != "" {
				cfg.Classifier.Mode = v
			}
		}
	}
	rt, _ := doc["runtime"].(map[string]any)
	if rt != nil {
		if v, ok := rt["address"].(string); ok && v != "" {
			cfg.RuntimeAddress = v
		}
	}
	agents, _ := doc["agents"].(map[string]any)
	if agents != nil {
		wc, _ := agents["web_crawler"].(map[string]any)
		if wc != nil {
			if v, ok := wc["address"].(string); ok && v != "" {
				cfg.Agents.WebCrawler.Address = v
			}
			if v, ok := wc["listen"].(string); ok && v != "" {
				cfg.Agents.WebCrawler.Address = v
			}
			if v, ok := wc["enabled"].(bool); ok {
				cfg.Agents.WebCrawler.Enabled = v
			}
		}
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
