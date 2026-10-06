# Piper Agent

基于自然语言的 Piper 采集 Agent：**Python 负责 Agent Harness（理解、规划、工具调用）**，**Go 负责 Runtime（模版执行、代理、持久化）**，通过 **gRPC** 通信。

## 文档

- [改造方案（可行性 + 架构 + 阶段计划）](docs/TRANSFORMATION_PLAN.md)
- [Phase A 已实现：如何运行](docs/PHASE_A.md)
- [Phase B：真实 piper_go 执行](docs/PHASE_B.md)
- [Phase C：LLM + 工具 + RAG](docs/PHASE_C.md)
- [Phase D：代理 / Chrome / 完整链路](docs/PHASE_D.md)

## 目录结构（Monorepo）

| 路径 | 语言 | 职责 |
|------|------|------|
| [proto/](proto/) | IDL | gRPC / protobuf 契约 |
| [engine/](engine/) | Go | Piper 采集内核（Go 模块名 `piper_go`：模版、分发、ES/S3） |
| [runtime/](runtime/) | Go | Agent Runtime gRPC Server，内嵌调用 `engine` |
| [agents/](agents/) | Python | LLM Agent、Harness 循环、Tool 客户端 |
| [shared/](shared/) | 中性 | JSON Schema、示例模版、Prompt 片段 |
| [deploy/](deploy/) | Ops | 本地/容器编排 |
| [scripts/](scripts/) | Shell/Python | `protoc` 生成、联调启动 |

根目录 [go.work](go.work) 同时管理 `runtime` 与 `engine` 两个 Go 模块。

## 快速概念

- **Harness（Python）**：会话状态 + LLM + ToolRegistry，把自然语言变成结构化 Run 请求。
- **Runtime（Go）**：校验模版、调度 Token、管理 Proxy/Chrome Agent、写 ES/S3。

## 本地联调

1. 编辑 [deploy/config/agent.local.yaml](deploy/config/agent.local.yaml)：`runtime.address`、`llm.api_key`（OpenAI 或兼容 API 的 Key，见 [Phase C](docs/PHASE_C.md)）。
2. 编译并启动 Runtime（需 ES/S3 等，见 [Phase B](docs/PHASE_B.md)）：
   ```powershell
   cd runtime
   go build -o piper-runtime.exe ./cmd/piper-runtime
   .\piper-runtime.exe -f ..\deploy\config\runtime.local.yaml
   ```
3. Agent：`cd agents && pip install -e ".[llm]" && piper-agent chat`（或 `piper-agent repl`）

可选：单独启动 Piper REST API → `cd engine && go build -o pipergo.exe . && .\pipergo.exe -f etc\pipergo-api.yaml`

无需设置 `OPENAI_API_KEY` 等环境变量；可选 `agent.secrets.yaml` 单独存密钥。
