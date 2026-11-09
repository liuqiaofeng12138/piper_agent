package crawlerharness

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sashabaranov/go-openai"
)

type Loop struct {
	reg     *Registry
	session *Session
	cfg     Config
	llm     *openai.Client
	model   string
}

func NewLoop(cfg Config, reg *Registry, session *Session) (*Loop, error) {
	if cfg.LLM.APIKey == "" {
		return nil, fmt.Errorf("llm.api_key required for embedded web_crawler")
	}
	cliCfg := openai.DefaultConfig(cfg.LLM.APIKey)
	if cfg.LLM.BaseURL != "" {
		cliCfg.BaseURL = cfg.LLM.BaseURL
	}
	model := cfg.LLM.Model
	if model == "" {
		model = "gpt-4o"
	}
	return &Loop{
		reg:     reg,
		session: session,
		cfg:     cfg,
		llm:     openai.NewClientWithConfig(cliCfg),
		model:   model,
	}, nil
}

type Event map[string]any

func (l *Loop) RunTurnIter(ctx context.Context, userText string, cancel func() bool) []Event {
	var out []Event
	l.session.Messages = append(l.session.Messages, map[string]any{"role": "user", "content": userText})
	system := l.systemPrompt()
	out = append(out, Event{"type": "run.started", "message": "agent loop started"})

	tools := make([]openai.Tool, 0)
	for _, raw := range OpenAITools() {
		fn, ok := raw["function"].(map[string]any)
		if !ok {
			continue
		}
		name, _ := fn["name"].(string)
		desc, _ := fn["description"].(string)
		params, _ := fn["parameters"].(map[string]any)
		if name == "" {
			continue
		}
		b, _ := jsonMarshal(params)
		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        name,
				Description: desc,
				Parameters:  json.RawMessage(b),
			},
		})
	}

	for step := 0; step < l.cfg.Harness.MaxSteps; step++ {
		if cancel != nil && cancel() {
			out = append(out, Event{"type": "run.cancelled", "message": "用户取消"})
			return out
		}
		msgs := []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: system},
		}
		for _, m := range l.session.Messages {
			role, _ := m["role"].(string)
			content, _ := m["content"].(string)
			msg := openai.ChatCompletionMessage{Role: role, Content: content}
			if role == "tool" {
				msg.ToolCallID, _ = m["tool_call_id"].(string)
			}
			if tcs, ok := m["tool_calls"].([]map[string]any); ok && len(tcs) > 0 {
				for _, tc := range tcs {
					fn, _ := tc["function"].(map[string]any)
					msg.ToolCalls = append(msg.ToolCalls, openai.ToolCall{
						ID:   fmt.Sprint(tc["id"]),
						Type: openai.ToolTypeFunction,
						Function: openai.FunctionCall{
							Name:      fmt.Sprint(fn["name"]),
							Arguments: fmt.Sprint(fn["arguments"]),
						},
					})
				}
			}
			msgs = append(msgs, msg)
		}

		resp, err := l.llm.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:    l.model,
			Messages: msgs,
			Tools:    tools,
		})
		if err != nil {
			out = append(out, Event{"type": "error", "message": err.Error()})
			return out
		}
		if len(resp.Choices) == 0 {
			out = append(out, Event{"type": "error", "message": "empty llm response"})
			return out
		}
		choice := resp.Choices[0].Message
		assistant := map[string]any{"role": "assistant", "content": choice.Content}
		if len(choice.ToolCalls) > 0 {
			tcs := make([]map[string]any, 0, len(choice.ToolCalls))
			for _, tc := range choice.ToolCalls {
				tcs = append(tcs, map[string]any{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]any{
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					},
				})
			}
			assistant["tool_calls"] = tcs
		}
		l.session.Messages = append(l.session.Messages, assistant)

		if text := strings.TrimSpace(choice.Content); text != "" {
			for _, piece := range chunkText(text, 8) {
				if cancel != nil && cancel() {
					out = append(out, Event{"type": "run.cancelled", "message": "用户取消"})
					return out
				}
				out = append(out, Event{"type": "message.delta", "delta": piece})
			}
		}

		if len(choice.ToolCalls) == 0 {
			final := strings.TrimSpace(choice.Content)
			if final == "" {
				final = "(empty response)"
			}
			out = append(out, Event{"type": "message.done", "content": final})
			return out
		}

		for _, tc := range choice.ToolCalls {
			if cancel != nil && cancel() {
				out = append(out, Event{"type": "run.cancelled", "message": "用户取消"})
				return out
			}
			name := tc.Function.Name
			args := tc.Function.Arguments
			out = append(out, Event{
				"type":        "tool.call",
				"tool_name":   name,
				"tool_detail": truncate(args, 800),
				"message":     "calling",
			})
			result := l.reg.Execute(name, args)
			out = append(out, Event{
				"type":      "run.progress",
				"tool_name": name,
				"message":   truncate(result, 1200),
			})
			l.session.Messages = append(l.session.Messages, map[string]any{
				"role":         "tool",
				"tool_call_id": tc.ID,
				"content":      result,
			})
		}
	}
	final := "已达到最大步数，请缩小任务或继续对话。"
	out = append(out, Event{"type": "message.delta", "delta": final})
	out = append(out, Event{"type": "message.done", "content": final})
	return out
}

func (l *Loop) systemPrompt() string {
	path := l.cfg.PromptsDir + string(os.PathSeparator) + "orchestrator_system.md"
	b, err := os.ReadFile(path)
	if err != nil {
		return "You are Piper Agent. Use tools to validate templates before run."
	}
	return string(b)
}

func chunkText(text string, maxRunes int) []string {
	if text == "" || maxRunes < 1 {
		if text == "" {
			return nil
		}
		return []string{text}
	}
	var chunks []string
	for i := 0; i < len(text); {
		j := i + maxRunes
		if j > len(text) {
			j = len(text)
		}
		chunks = append(chunks, text[i:j])
		i = j
	}
	return chunks
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}
