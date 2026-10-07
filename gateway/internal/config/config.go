package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen              string           `yaml:"listen"`
	ConfigPath          string           `yaml:"-"`
	Auth                Auth             `yaml:"auth"`
	Session             SessionConfig    `yaml:"session"`
	Classifier          ClassifierConfig `yaml:"classifier"`
	ChatTimeoutSeconds  int              `yaml:"chat_timeout_seconds"`
	MaxRequestBodyBytes int64            `yaml:"max_request_body_bytes"`
	RateLimitPerMinute  int              `yaml:"rate_limit_per_minute"`
	RuntimeAddress      string           `yaml:"-"`
	LLM                 LLMConfig        `yaml:"-"`
	Agents              []AgentSpec      `yaml:"agents"`
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
	Mode      string `yaml:"mode"`       // rule | llm
	Model     string `yaml:"model"`      // 默认复用 llm.model
	TimeoutMs int    `yaml:"timeout_ms"` // LLM 分类超时，超时回退规则路由
}

// LLMConfig 网关侧分类器使用的 LLM 配置（复用 deploy/config/local.yaml 的 llm 段）。
type LLMConfig struct {
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	APIKey   string `yaml:"api_key"`
	BaseURL  string `yaml:"base_url"`
}

// AgentSpec 子 Agent 注册表项（配置化，支持多 Worker）。
type AgentSpec struct {
	ID          string `yaml:"id"`
	DisplayName string `yaml:"display_name"`
	Description string `yaml:"description"`
	Address     string `yaml:"address"`
	Enabled     bool   `yaml:"enabled"`
	Default     bool   `yaml:"default"`
	Module      string `yaml:"module"` // Python worker 模块，默认 piper_agent.workers.<id>
}

// WorkerModule 返回启动该 Agent 的 Python 模块路径。
func (a AgentSpec) WorkerModule() string {
	if a.Module != "" {
		return a.Module
	}
	return "piper_agent.workers." + a.ID
}

// FindAgent 按 id 查找已启用 Agent。
func (c *Config) FindAgent(id string) *AgentSpec {
	for i := range c.Agents {
		if c.Agents[i].ID == id && c.Agents[i].Enabled {
			return &c.Agents[i]
		}
	}
	return nil
}

// DefaultAgent 返回兜底 Agent（显式 default 优先，否则第一个 enabled）。
func (c *Config) DefaultAgent() *AgentSpec {
	for i := range c.Agents {
		if c.Agents[i].Default && c.Agents[i].Enabled {
			return &c.Agents[i]
		}
	}
	for i := range c.Agents {
		if c.Agents[i].Enabled {
			return &c.Agents[i]
		}
	}
	return nil
}

// EnabledAgents 返回所有已启用 Agent。
func (c *Config) EnabledAgents() []AgentSpec {
	out := make([]AgentSpec, 0, len(c.Agents))
	for _, a := range c.Agents {
		if a.Enabled {
			out = append(out, a)
		}
	}
	return out
}

func DefaultConfigPath() string {
	return filepath.Join("..", "deploy", "config", "local.yaml")
}

func defaultAgents() []AgentSpec {
	return []AgentSpec{
		{
			ID:          "web_crawler",
			DisplayName: "网页采集",
			Description: "自然语言描述采集任务，调用 Piper Runtime 执行抓取",
			Address:     "127.0.0.1:15061",
			Enabled:     true,
			Default:     true,
			Module:      "claw_agent.workers.web_crawler",
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		Listen:             ":8080",
		Auth:               Auth{Mode: "none"},
		Session:            SessionConfig{Driver: "sqlite", SQLitePath: "../../data/gateway.db"},
		Classifier:         ClassifierConfig{Mode: "rule", TimeoutMs: 4000},
		ChatTimeoutSeconds: 600,
		MaxRequestBodyBytes: 1 << 20,
		RateLimitPerMinute:  60,
		Agents:             defaultAgents(),
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
		if v, ok := intFromYAML(gw["max_request_body_bytes"]); ok && v > 0 {
			cfg.MaxRequestBodyBytes = int64(v)
		}
		if v, ok := intFromYAML(gw["rate_limit_per_minute"]); ok && v > 0 {
			cfg.RateLimitPerMinute = v
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
			if v, ok := cls["model"].(string); ok && v != "" {
				cfg.Classifier.Model = v
			}
			if v, ok := intFromYAML(cls["timeout_ms"]); ok && v > 0 {
				cfg.Classifier.TimeoutMs = v
			}
		}
	}
	rt, _ := doc["runtime"].(map[string]any)
	if rt != nil {
		if v, ok := rt["address"].(string); ok && v != "" {
			cfg.RuntimeAddress = v
		}
	}
	if llm, _ := doc["llm"].(map[string]any); llm != nil {
		if v, ok := llm["provider"].(string); ok && v != "" {
			cfg.LLM.Provider = v
		}
		if v, ok := llm["model"].(string); ok && v != "" {
			cfg.LLM.Model = v
		}
		if v, ok := llm["api_key"].(string); ok {
			cfg.LLM.APIKey = v
		}
		if v, ok := llm["base_url"].(string); ok && v != "" {
			cfg.LLM.BaseURL = v
		}
	}
	if raw, ok := doc["agents"]; ok && raw != nil {
		agents, err := parseAgents(raw)
		if err != nil {
			return nil, fmt.Errorf("agents config: %w", err)
		}
		if len(agents) > 0 {
			cfg.Agents = agents
		}
	}
	return cfg, nil
}

// parseAgents 同时支持新的列表格式与旧的 map 格式：
//
//	agents:                          agents:
//	  - id: web_crawler                web_crawler:
//	    address: ...                     listen: ...
//	    enabled: true                    enabled: true
func parseAgents(raw any) ([]AgentSpec, error) {
	switch node := raw.(type) {
	case []any:
		out := make([]AgentSpec, 0, len(node))
		for _, item := range node {
			m, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("agent entry must be a map, got %T", item)
			}
			out = append(out, agentFromMap("", m))
		}
		return out, nil
	case map[string]any:
		out := make([]AgentSpec, 0, len(node))
		for id, item := range node {
			m, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("agent %q entry must be a map, got %T", id, item)
			}
			out = append(out, agentFromMap(id, m))
		}
		return out, nil
	default:
		return nil, fmt.Errorf("agents must be a list or map, got %T", raw)
	}
}

func agentFromMap(id string, m map[string]any) AgentSpec {
	spec := AgentSpec{ID: id, Enabled: true}
	if v, ok := m["id"].(string); ok && v != "" {
		spec.ID = v
	}
	if v, ok := m["display_name"].(string); ok {
		spec.DisplayName = v
	}
	if v, ok := m["description"].(string); ok {
		spec.Description = v
	}
	if v, ok := m["address"].(string); ok && v != "" {
		spec.Address = v
	}
	if v, ok := m["listen"].(string); ok && v != "" {
		spec.Address = v
	}
	if v, ok := m["enabled"].(bool); ok {
		spec.Enabled = v
	}
	if v, ok := m["default"].(bool); ok {
		spec.Default = v
	}
	if v, ok := m["module"].(string); ok {
		spec.Module = v
	}
	if spec.DisplayName == "" {
		spec.DisplayName = spec.ID
	}
	return spec
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
