# runtime（Go）

**Piper Agent Runtime**：稳定、长运行的执行环境，负责模版校验、Token 调度、代理与采集 Agent 池、结果持久化。

## 子目录

| 路径 | 模块 |
|------|------|
| [cmd/piper-runtime/](cmd/piper-runtime/) | 进程入口：读配置、启 gRPC、优雅退出 |
| [internal/grpcserver/](internal/grpcserver/) | gRPC 服务注册、拦截器（auth、logging、otel） |
| [internal/harness/](internal/harness/) | **执行侧 Harness**：Run 生命周期、步骤编排、与 Python Harness 的 run_id 对齐 |
| [internal/executor/](internal/executor/) | 调用 `piper_go`：`tpl.*`、`distributor.Engine`、meta Store |
| [internal/config/](internal/config/) | 解析 `deploy/config/local.yaml`（含 `engine:` 段） |
| [pkg/pb/](pkg/pb/) | `protoc` 生成的 Go stub（勿手改） |
| [pkg/runtime/](pkg/runtime/) | 对外可复用的 Runtime SDK（其他 Go 服务嵌入） |

## Harness vs Executor

- **internal/harness**：编排「一次用户任务」在 Runtime 内的状态机（Validate → BuildToken → Run → Wait → FetchData → 可选 Persist），不负责 LLM。
- **internal/executor**：无状态的 Piper 内核调用，便于单测。

## 依赖

- 模块路径建议：`piper_agent/runtime`（go.mod 在本目录）
- 通过 `go.mod` 的 `replace piper_go => ../engine` 引用同仓库采集内核
