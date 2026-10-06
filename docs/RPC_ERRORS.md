# gRPC 错误码约定（Phase A）

## 映射

| `common.v1.ErrorCode` | gRPC `codes` | 典型场景 |
|----------------------|--------------|----------|
| `INVALID_ARGUMENT` | `InvalidArgument` | 缺 `request_id`、模版字段非法 |
| `NOT_FOUND` | `NotFound` | `run_id` / `template_id` 不存在（Phase B+） |
| `FAILED_PRECONDITION` | `FailedPrecondition` | 未 Validate 就 Run（Phase B+ 策略） |
| `ALREADY_EXISTS` | `AlreadyExists` | 幂等键冲突且 Run 仍在进行 |
| `INTERNAL` | `Internal` | Runtime 内部错误 |
| `UNAVAILABLE` | `Unavailable` | piper_go / ES 未就绪 |
| `DEADLINE_EXCEEDED` | `DeadlineExceeded` | 客户端或 Run 超时 |
| `CANCELLED` | `Canceled` | 用户 CancelRun |

## Metadata

客户端应设置（可选）：

- `x-request-id`
- `x-session-id`

服务端日志与 Mock Run 状态应回显 `request_id` / `session_id`。

## 业务诊断 vs 传输错误

- **传输层**：标准 gRPC status + `ErrorCode`（上表）。
- **业务层**：RPC 响应体内的 `repeated Diagnostic`（如 `ValidateTemplateResponse.diagnostics`），HTTP 200 式「软失败」，便于 LLM 修正模版。

Phase A Mock 仅使用 `Diagnostic` + `InvalidArgument` 示例。
