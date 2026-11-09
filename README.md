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
| [gateway/internal/crawlerharness/](gateway/internal/crawlerharness/) | Go | Web 采集 Harness（内嵌于 `piper-serve`，直连 Runtime） |
| [general_agent/](general_agent/) | Python | 平台通用子 Agent Workers（`general_chat` 等，包名 `piper_agent`） |
| [rag_agent/](rag_agent/) | Python | 文档 RAG 子 Agent（`doc_rag`，LangGraph） |
| [paper_agent/](paper_agent/) | Python | 学术论文检索子 Agent（`paper_search`，arXiv + LangGraph） |
| [nl2sql_agent/](nl2sql_agent/) | Python | 半导体 NL2SQL（模版化 SQL + PostgreSQL + 前端图表） |
| [gateway/](gateway/) | Go | Web API 网关（SSE 对话、会话，Phase W1+） |
| [web/](web/) | Vue + TS | 对话前端 |
| [shared/](shared/) | 中性 | JSON Schema、示例模版、Prompt 片段 |
| [deploy/](deploy/) | Ops | 本地/容器编排 |
| [scripts/](scripts/) | Shell/Python | `protoc` 生成、联调启动 |

根目录 [go.work](go.work) 同时管理 `runtime` 与 `engine` 两个 Go 模块。

## 快速概念

- **Harness（Python）**：会话状态 + LLM + ToolRegistry，把自然语言变成结构化 Run 请求。
- **Runtime（Go）**：校验模版、调度 Token、管理 Proxy/Chrome Agent、写 ES/S3。

## Web 对话（推荐一键后端）

**先为各 Python 子 Agent 创建独立虚拟环境**（每个含 `pyproject.toml` 的项目一个 `.venv`）：

```powershell
powershell -File scripts/setup-python-venvs.ps1
```

**一条命令拉起内嵌 Runtime/采集 + 其余 Python Worker + 网关**：

```powershell
cd gateway
go run ./cmd/piper-serve -f ../deploy/config/local.yaml
```

或：`powershell -File scripts/start-web.ps1`

再开前端：`cd web && npm run dev`

仅单独调试网关时仍可用 `go run ./cmd/piper-gateway`（会内嵌采集 Runtime；其余 Worker 需已运行或仅用非采集 Agent）。

详见 [web/README.md](web/README.md) 与 [web设计方案.md](web设计方案.md)。

## 本地联调（采集与依赖）

1. 复制 [deploy/config/local.yaml.example](deploy/config/local.yaml.example) 为 `local.yaml`，填写 `llm.api_key`（见 [Phase C](docs/PHASE_C.md)）。
2. 采集执行依赖 `engine` 段配置的外部服务（ES/S3 等，见 [Phase B](docs/PHASE_B.md)），由 `piper-serve` **内嵌** Piper Runtime 使用，无需单独启动 `piper-runtime`。

**Agent 对话请使用 Web 界面**（见上文「Web 对话」）。
