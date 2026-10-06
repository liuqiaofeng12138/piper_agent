# clients

gRPC 客户端与连接管理。

## 规划

- `runtime_client.py`：`grpc.aio` channel，`RUNTIME_ADDR` 环境变量
- `interceptors.py`：metadata 注入 `request_id`, `session_id`
- 重试：仅 idempotent RPC（List*）；Run 用 idempotency_key

生成 stub 来自 `agents/src/piper_agent/pb/`（protoc grpc_python）。
