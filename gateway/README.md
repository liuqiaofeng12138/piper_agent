# Piper Gateway (Phase W2)

HTTP API 网关：会话 SQLite 持久化、规则路由、经 gRPC 调用 Python `web-crawler` Worker，SSE 转发 Agent 事件。

## 启动

在仓库根目录或 `gateway` 目录：

```powershell
cd gateway
go run ./cmd/piper-gateway -f ../deploy/config/local.yaml
```

默认监听 `:8080`；Worker 地址见 `agents.web_crawler.listen`（与 Python Worker 一致）。

需先启动 `piper-runtime` 与 `piper-agent worker web-crawler`。

## API

- `GET /api/v1/health`
- `GET /api/v1/agents`
- `POST /api/v1/conversations`
- `GET /api/v1/conversations`
- `GET /api/v1/conversations/{id}/messages`
- `POST /api/v1/chat/completions`（`stream: true` 时返回 SSE）
