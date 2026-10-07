package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"piper_agent/gateway/internal/config"
)

// LLMClassifier 调用 OpenAI-compatible chat/completions，要求模型输出
// {"agent_id": "...", "confidence": 0-1, "reason": "..."} 的 JSON 路由决策。
type LLMClassifier struct {
	endpoint string
	apiKey   string
	model    string
	timeout  time.Duration
	http     *http.Client
}

// NewLLMClassifier 构建分类器；缺少 api_key 或 model 时返回 nil（调用方回退 rule）。
func NewLLMClassifier(cfg *config.Config) *LLMClassifier {
	if cfg.LLM.APIKey == "" {
		return nil
	}
	model := cfg.Classifier.Model
	if model == "" {
		model = cfg.LLM.Model
	}
	if model == "" {
		return nil
	}
	base := strings.TrimRight(cfg.LLM.BaseURL, "/")
	if base == "" {
		base = "https://api.openai.com"
	}
	timeout := time.Duration(cfg.Classifier.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 4 * time.Second
	}
	return &LLMClassifier{
		endpoint: base + "/chat/completions",
		apiKey:   cfg.LLM.APIKey,
		model:    model,
		timeout:  timeout,
		http:     &http.Client{Timeout: timeout + 2*time.Second},
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	Temperature    float64         `json:"temperature"`
	MaxTokens      int             `json:"max_tokens"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

type routeDecision struct {
	AgentID    string  `json:"agent_id"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

// Classify 返回 (agent_id, reason, ok)；ok=false 时调用方应回退规则路由。
func (c *LLMClassifier) Classify(ctx context.Context, userMessage string, agents []config.AgentSpec) (string, string, bool) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	body := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: buildPrompt(agents)},
			{Role: "user", Content: userMessage},
		},
		Temperature:    0,
		MaxTokens:      200,
		ResponseFormat: &responseFormat{Type: "json_object"},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", "", false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(raw))
	if err != nil {
		return "", "", false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		log.Printf("[router] llm classify request failed: %v", err)
		return "", "", false
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", "", false
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("[router] llm classify status=%d body=%s", resp.StatusCode, truncate(string(payload), 200))
		return "", "", false
	}

	var cr chatResponse
	if err := json.Unmarshal(payload, &cr); err != nil || len(cr.Choices) == 0 {
		log.Printf("[router] llm classify bad response: %v", err)
		return "", "", false
	}
	var decision routeDecision
	if err := json.Unmarshal([]byte(cr.Choices[0].Message.Content), &decision); err != nil {
		log.Printf("[router] llm classify invalid json: %s", truncate(cr.Choices[0].Message.Content, 200))
		return "", "", false
	}
	for _, a := range agents {
		if a.ID == decision.AgentID {
			if decision.Reason == "" {
				decision.Reason = fmt.Sprintf("confidence=%.2f", decision.Confidence)
			}
			return decision.AgentID, decision.Reason, true
		}
	}
	log.Printf("[router] llm classify unknown agent_id=%q", decision.AgentID)
	return "", "", false
}

func buildPrompt(agents []config.AgentSpec) string {
	var sb strings.Builder
	sb.WriteString("你是一个意图分类器。根据用户消息，从下列 Agent 中选择最合适的一个来处理。\n\n可用 Agent：\n")
	for _, a := range agents {
		fmt.Fprintf(&sb, "- id=%q 名称=%q 能力：%s\n", a.ID, a.DisplayName, a.Description)
	}
	sb.WriteString("\n只输出 JSON，格式：{\"agent_id\": \"<id>\", \"confidence\": 0.0-1.0, \"reason\": \"<一句话理由>\"}。agent_id 必须来自上面的列表。")
	return sb.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
