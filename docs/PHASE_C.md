# Phase C — LLM 写模版与填参（已实现）

## 交付

| 项 | 位置 |
|----|------|
| 工具 `template_author_*` | `agents/tools/handlers.py` |
| 工具 `param_filler_suggest` | 同上 |
| 工具 `runner_*` | 同上 |
| RAG | `agents/rag/template_index.py` + `shared/examples/templates/` |
| Harness 循环 | `agents/harness/loop.py`, `registry.py`, `session.py`, `policies.py` |
| Orchestrator | `agents/agents/orchestrator.py` |
| 强校验 diagnostics | `engine/pkg/agentruntime/validate.go` |
| 示例模版 | `shared/examples/templates/*.json` |

## 使用

```powershell
# 1. 编辑 deploy/config/agent.local.yaml，填写 llm.api_key（OpenAI 或兼容服务的 Key）
# 2. 启动 Runtime
cd piper_agent\runtime
.\piper-runtime.exe -f ..\deploy\config\runtime.local.yaml

# 3. LLM chat（Runtime 地址已在 agent.local.yaml 的 runtime.address）
cd piper_agent\agents
pip install -e ".[llm]"
piper-agent doctor   # 先确认 Runtime 可达
piper-agent chat
```

`llm.api_key` 说明：在 [OpenAI API Keys](https://platform.openai.com/api-keys) 创建的密钥（`sk-...`）；若用 DeepSeek 等 OpenAI 兼容 API，同时设置 `llm.base_url` 与对应 key。也可复制 `agent.secrets.yaml.example` → `agent.secrets.yaml` 仅存放密钥（已 gitignore）。

自然语言示例：「搜索类似 json 采集模版，校验并保存，然后抓取 httpbin 的 author 字段」

## 策略

- `require_validate_before_run: true` 时，`runner_execute` 前必须在本 session 内 `template_author_validate` 成功。
- Validate 返回 `diagnostics[].path` 如 `procedures[0].fields.title.method`，便于 LLM 修正。

## 无 LLM 调试

仍可使用 `piper-agent repl` 手动调用 Runtime。
