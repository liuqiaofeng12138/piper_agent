package crawlerharness

import (
	_ "embed"
	"encoding/json"
)

//go:embed tool_schemas.json
var toolSchemasRaw []byte

func OpenAITools() []map[string]any {
	var out []map[string]any
	_ = json.Unmarshal(toolSchemasRaw, &out)
	return out
}

type Registry struct {
	handlers *ToolHandlers
}

func NewRegistry(h *ToolHandlers) *Registry {
	return &Registry{handlers: h}
}

func (r *Registry) Execute(name string, argumentsJSON string) string {
	args := map[string]any{}
	if argumentsJSON != "" {
		_ = json.Unmarshal([]byte(argumentsJSON), &args)
	}
	return r.handlers.Dispatch(name, args)
}
