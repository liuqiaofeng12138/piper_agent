# Piper Gateway (Phase W4)

HTTP API 网关：会话 SQLite 持久化、**多子 Agent 注册表 + 意图路由（rule / LLM JSON）**、经 gRPC 调用各 Python Worker，SSE 转发 Agent 事件。

## 启动

### 推荐：`piper-serve`（Runtime + 全部 Worker + 网关）

```powershell
cd gateway
go run ./cmd/piper-serve -f ../deploy/config/local.yaml
```

`web_crawler` 由 **piper-serve 进程内 Go Worker** 提供，并 **内嵌 Piper Runtime**（无 `piper-runtime` 子进程、无 Python 采集 Worker）。其余已启用 Agent 仍通过 `python -m <module>` 拉起 gRPC Worker。

每个 Python Worker **仅**使用其独占的 `.venv`（由 `module` 首包名推断项目目录；也可用 `agents[].python_project` 覆盖）。缺少 `.venv` 时 `piper-serve` 启动失败，请先运行 `scripts/setup-python-venvs.ps1`。

`PYTHONPATH` 自动包含仓库根下所有含 `pyproject.toml` 的项目的 `src/`（新增 `*_agent` 目录会被自动发现）。

环境变量 `PIPER_WORKER_PYTHON` 可覆盖解释器（容器/调试，会跳过上述 .venv 检查）。

环境变量 `PIPER_WORKER_PYTHON` 可覆盖 Python Worker 解释器。

### 仅网关：`piper-gateway`

单独调试 HTTP 层时使用；需自行启动 Runtime 与 Worker。

```powershell
go run ./cmd/piper-gateway -f ../deploy/config/local.yaml
```

## 子 Agent 注册表（Phase W4）

`deploy/config/local.yaml` 的 `agents` 段为列表（兼容旧 map 格式）：

```yaml
agents:
  - id: web_crawler
    display_name: "网页采集"
    description: "自然语言描述网页抓取/数据采集任务，生成 Piper 模版并调用 Runtime 执行"
    listen: "127.0.0.1:15061"
    enabled: true
  - id: general_chat
    display_name: "通用对话"
    description: "日常问答、知识咨询、闲聊等非采集类对话"
    listen: "127.0.0.1:15062"
    enabled: true
    default: true       # 意图未命中时的兜底 Agent
```

新增子 Agent = 一个实现 `agent.v1.AgentWorkerService` 的 Python 模块 + 注册表一项。

## 意图路由

`gateway.classifier.mode`：

- `rule`：命中采集关键词（抓/采集/爬/url/api 等）→ `web_crawler`；否则 → `default` Agent。
- `llm`：用 `llm` 段配置的 OpenAI-compatible 模型做 JSON 结构化路由（`response_format: json_object`，输出 `agent_id/confidence/reason`）；超时（`timeout_ms`，默认 4s）或结果非法时自动回退 `rule`。可用 `classifier.model` 指定更小的分类模型。

路由决策（方式 + 理由）写入网关日志：`[chat] start ... agent=... route="..."`。

## API

- `GET /api/v1/health`（含各 worker / runtime 探活）
- `GET /api/v1/agents`（注册表全量，含 `healthy`、`address`）
- `POST /api/v1/conversations`
- `GET /api/v1/conversations`
- `GET /api/v1/conversations/{id}/messages`
- `DELETE /api/v1/conversations/{id}`（删除会话及消息）
- `POST /api/v1/chat/completions`（`stream: true` 时返回 SSE）
- `POST /api/v1/runs/{id}/cancel`

## 防护（Phase W3）

- `gateway.max_request_body_bytes`：请求体上限（默认 1MB）
- `gateway.rate_limit_per_minute`：按 IP 每分钟请求数（默认 60）
