# internal/grpcserver

实现 `proto/runtime/v1` 中各 Service。

## 拦截器链（建议）

1. `recovery` — panic → Internal error
2. `logging` — request_id, session_id, latency
3. `auth` — 可选 mTLS 或 bearer（与 piper_go `noAuth` 对齐策略）

## 服务实现映射

| gRPC Service | 实现 struct 位置 |
|--------------|------------------|
| RuntimeMeta | `meta_service.go` → executor |
| RuntimeTemplate | `template_service.go` → harness + executor |
| RuntimeExecute | `execute_service.go` → harness |
| RuntimeData | `data_service.go` → executor |
