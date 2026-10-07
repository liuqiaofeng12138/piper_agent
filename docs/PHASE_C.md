# Phase C — LLM 写模版与填参（已实现）

## 交付

| 项 | 位置 |
|----|------|
| 工具 `template_author_*` | `web_crawler_agent/src/web_crawler_agent/tools/handlers.py` |
| 工具 `param_filler_suggest` | 同上 |
| 工具 `runner_*` | 同上 |
| RAG | `web_crawler_agent/src/web_crawler_agent/rag/template_index.py` + `shared/examples/templates/` |
| Harness 循环 | `web_crawler_agent/src/web_crawler_agent/harness/loop.py`, `registry.py`, `session.py`, `policies.py` |
| Orchestrator | `web_crawler_agent/src/web_crawler_agent/agents/orchestrator.py` |
| 强校验 diagnostics | `engine/pkg/agentruntime/validate.go` |
| 示例模版 | `shared/examples/templates/*.json` |

## 使用

```powershell
# 1. 编辑 deploy/config/local.yaml，填写 llm.api_key（OpenAI 或兼容服务的 Key）
# 2. 启动 Runtime
cd piper_agent\runtime
.\piper-runtime.exe -f ..\deploy\config\local.yaml

# 3. LLM chat（Runtime 地址在同文件 runtime.address）
cd piper_agent\web_crawler_agent
pip install -e ".[llm]"
web-crawler-agent doctor   # 先确认 Runtime 可达；对话入口已改为 Web（见根 README）
```

`llm.api_key` 说明：在 [OpenAI API Keys](https://platform.openai.com/api-keys) 创建的密钥（`sk-...`）；若用 DeepSeek 等 OpenAI 兼容 API，同时设置 `llm.base_url` 与对应 key。含密钥的 `local.yaml` 已 gitignore。

自然语言示例：「搜索类似 json 采集模版，校验并保存，然后抓取 httpbin 的 author 字段」

## 策略

- `require_validate_before_run: true` 时，`runner_execute` 前必须在本 session 内 `template_author_validate` 成功。
- Validate 返回 `diagnostics[].path` 如 `procedures[0].fields.title.method`，便于 LLM 修正。

## 无 LLM 调试

可使用 `web-crawler-agent doctor` 检查 Runtime 连通性；手动对话调试请走 Web 界面。
