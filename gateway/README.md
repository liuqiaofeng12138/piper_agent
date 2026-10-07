# Piper Gateway (Phase W2)

HTTP API 网关：会话 SQLite 持久化、规则路由、经 gRPC 调用 Python `web-crawler` Worker，SSE 转发 Agent 事件。

## 启动

### 推荐：`piper-serve`（Runtime + Worker + 网关）

```powershell
cd gateway
go run ./cmd/piper-serve -f ../deploy/config/local.yaml
```

按顺序子进程启动 `piper-runtime`、`python -m piper_agent.workers.web_crawler`，再在本进程监听 HTTP（默认 `:8080`）。  
优先使用 `runtime/piper-runtime.exe`（若已编译），否则 `go run ./cmd/piper-runtime`。  
Python 优先 `agents/.venv`，否则系统 `python` / `python3`。

环境变量：`PIPER_RUNTIME_CMD` 覆盖 Runtime 可执行文件；`PIPER_WORKER_PYTHON` 覆盖 Worker 用的 Python 解释器。

### 仅网关：`piper-gateway`

单独调试 HTTP 层时使用；需自行启动 Runtime 与 Worker。

```powershell
go run ./cmd/piper-gateway -f ../deploy/config/local.yaml
```

## API

- `GET /api/v1/health`（含 worker / runtime 探活）
- `GET /api/v1/agents`（含 `healthy`、`address`）
- `POST /api/v1/conversations`
- `GET /api/v1/conversations`
- `GET /api/v1/conversations/{id}/messages`
- `DELETE /api/v1/conversations/{id}`（删除会话及消息）
- `POST /api/v1/chat/completions`（`stream: true` 时返回 SSE）
- `POST /api/v1/runs/{id}/cancel`

## 防护（Phase W3）

- `gateway.max_request_body_bytes`：请求体上限（默认 1MB）
- `gateway.rate_limit_per_minute`：按 IP 每分钟请求数（默认 60）
