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
| [gateway/](gateway/) | Go | Web API 网关（SSE 对话、会话，Phase W1+） |
| [web/](web/) | Vue + TS | 对话前端 |
| [shared/](shared/) | 中性 | JSON Schema、示例模版、Prompt 片段 |
| [deploy/](deploy/) | Ops | 本地/容器编排 |
| [scripts/](scripts/) | Shell/Python | `protoc` 生成、联调启动 |

根目录 [go.work](go.work) 同时管理 `runtime` 与 `engine` 两个 Go 模块。

## 快速概念

- **Harness（Python）**：会话状态 + LLM + ToolRegistry，把自然语言变成结构化 Run 请求。
- **Runtime（Go）**：校验模版、调度 Token、管理 Proxy/Chrome Agent、写 ES/S3。

## Web 对话（Phase W2）

1. 启动 Runtime：`cd runtime && go run ./cmd/piper-runtime -f ../deploy/config/local.yaml`
2. 启动采集 Worker：`cd agents && pip install -e ".[llm]" && piper-agent worker web-crawler -f ../deploy/config/local.yaml`
3. 启动网关：`cd gateway && go run ./cmd/piper-gateway -f ../deploy/config/local.yaml`
4. 启动前端：`cd web && npm install && npm run dev` → 浏览器打开 Vite 地址

详见 [web/README.md](web/README.md) 与 [web设计方案.md](web设计方案.md)。

## 本地联调（CLI + Runtime）

1. 复制 [deploy/config/local.yaml.example](deploy/config/local.yaml.example) 为 `local.yaml`，填写 `llm.api_key`（见 [Phase C](docs/PHASE_C.md)）。
2. 编译并启动 Runtime（需 ES/S3 等，见 [Phase B](docs/PHASE_B.md)）：
   ```powershell
   cd runtime
   go build -o piper-runtime.exe ./cmd/piper-runtime
   .\piper-runtime.exe -f ..\deploy\config\local.yaml
   ```
   默认 `-f` 即 `../deploy/config/local.yaml`，在 `runtime` 目录下也可省略。
3. Agent：`cd agents && pip install -e ".[llm]" && piper-agent chat`（同样读取 `local.yaml`）

无需设置 `OPENAI_API_KEY` 等环境变量。
