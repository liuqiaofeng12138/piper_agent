# internal/harness（Go 执行 Harness）

与 Python Agent Harness **配对**的概念：Python 负责「想」，Go 负责本侧「做」的编排。

## 核心类型（规划）

```go
// RunHandle 一次自然语言任务在 Runtime 内的执行句柄
type RunHandle struct {
    RunID, SessionID string
    Phase            RunPhase // Validated, Running, Succeeded, Failed
}

type Orchestrator interface {
    StartRun(ctx context.Context, req *pb.RunTemplateRequest) (*RunHandle, error)
    CancelRun(ctx context.Context, runID string) error
    StreamEvents(ctx context.Context, runID string) (<-chan Event, error)
}
```

## 职责

1. 串行保证：未 Validate 的模版不得进入 distributor
2. 超时与 Cancel：context 传播到 `piper_go` Token
3. 事件聚合：Token 状态、队列、Proc 日志摘要 → gRPC stream
4. 幂等：`idempotency_key` 防止 Harness 重试双跑

## 非职责

- LLM 调用、Prompt 管理 → `agents/harness`
