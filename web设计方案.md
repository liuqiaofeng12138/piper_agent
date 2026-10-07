# Piper Agent Web 系统设计方案

本文档描述如何将当前 **CLI 对话式 Agent**（`piper-agent chat`）演进为类似 DeepSeek / OpenAI 网页版的 **前后端 Web 服务**：用户在前端提问，后端完成意图识别、子 Agent 路由、任务执行与流式结果回传。

与仓库内 [docs/TRANSFORMATION_PLAN.md](docs/TRANSFORMATION_PLAN.md) 的关系：改造方案已定义 **Python Harness + Go Runtime（gRPC）** 的采集执行链路；本方案在其之上增加 **Web 网关层、会话 API、多子 Agent 路由** 与 **Vue 前端**，不改变 Runtime 与 `engine/` 的核心职责。

---

## 1. 目标与范围

### 1.1 产品目标

| 能力 | 说明 |
|------|------|
| Web 对话 | 多轮会话、流式输出（打字机效果）、停止生成、历史会话列表 |
| 统一入口 | 单一聊天界面；后端根据用户问题做 **意图识别**，路由到对应 **子 Agent** |
| 可扩展 Agent | 子 Agent 以插件方式注册；**首期仅落地「Web 采集 / 爬虫」子 Agent**（复用现有 `agents` + `runtime`） |
| 高性能网关 | **Go** 承担 HTTP/WebSocket/SSE、并发连接、限流、取消传播、任务调度 |
| 丰富执行生态 | **Python** 承担 LLM 循环、工具调用、各领域 Agent 逻辑 |

### 1.2 现状（CLI）与目标（Web）对照

```
现状：
  用户终端 → piper-agent chat (Python CLI)
           → AgentLoop + Orchestrator (Python Harness)
           → gRPC → piper-runtime (Go) → engine (piper_go)

目标：
  浏览器 (Vue/TS) → HTTP/SSE → gateway (Go 新增)
                  → 意图路由 → Python 子 Agent Worker（gRPC/HTTP 流式）
                  → 采集类问题 → 现有 Harness + piper-runtime → engine
```

CLI（`agents/src/piper_agent/cli/main.py`）在 Web 化后保留，用于运维联调与自动化脚本；**主入口改为 Web**。

### 1.3 非目标（首期）

- 完整多租户 RBAC、计费、公开注册（可预留接口与配置项）
- 除「Web 采集」外的子 Agent 完整实现（仅预留注册协议与占位服务）
- 替换 `engine` 内已有 Java 风格 WebAPI（8888）；网关与之解耦，仅通过 Runtime gRPC 使用采集能力

---

## 2. 架构总览

### 2.1 分层职责

| 层级 | 技术 | 职责 |
|------|------|------|
| **Web 前端** | Vue 3 + TypeScript + Vite | 会话 UI、消息列表、流式渲染、任务状态/进度展示、停止按钮 |
| **API 网关（新增）** | Go | REST + SSE（必选）；WebSocket（可选，与 SSE 二选一为主通道）；鉴权、限流、TraceID、会话 CRUD、取消 `context` |
| **路由 / 编排（网关内）** | Go | 轻量 **意图识别** → 选择 `agent_id`；维护 `conversation_id` / `run_id`；向 Python 发起流式调用 |
| **子 Agent Worker** | Python | 各子 Agent 独立进程/容器；实现统一 **Agent Execute API**（建议 gRPC 流式，与现有 proto 风格一致） |
| **采集 Runtime（已有）** | Go `runtime/` | 模版校验、Token 执行、数据查询；**仅 Web 采集子 Agent 使用** |
| **采集内核（已有）** | Go `engine/` | Distributor、HTTP/Chrome、ES/S3 持久化 |

### 2.2 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│  web/  Vue 3 + TS                                                │
│  - 对话页 / 会话列表 / 设置(API Key 由服务端配置，前端不存密钥)      │
└────────────────────────────┬────────────────────────────────────┘
                             │  HTTPS
                             │  POST /api/v1/chat/completions (SSE)
                             │  GET  /api/v1/conversations
                             │  POST /api/v1/runs/{id}/cancel
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  gateway/  (Go 新建模块，建议纳入 go.work)                         │
│  - SessionStore (Redis 或 SQLite/Postgres 二期)                    │
│  - Router: IntentClassifier → agent_id                           │
│  - AgentPool: 连接各 Python Agent Worker                         │
│  - StreamMultiplexer: 合并 token / tool / run 事件 → SSE           │
└──────────────┬──────────────────────────────┬───────────────────┘
               │ gRPC stream                    │ gRPC (已有)
               ▼                                ▼
┌──────────────────────────┐      ┌──────────────────────────────┐
│  agents/ (Python)         │      │  runtime/ piper-runtime       │
│  - worker: web_crawler    │─────▶│  Validate / Run / GetData     │
│    (封装 orchestrator)    │      │  → engine/                    │
│  - worker: (future) RAG   │      └──────────────────────────────┘
│  - worker: (future) ...   │
└──────────────────────────┘
```

### 2.3 为什么仍用 Go + Python

- **Go**：万级 SSE 连接、请求超时与取消、路由与限流；与现有 `runtime` 同语言，便于同进程或 sidecar 部署。
- **Python**：现有 `AgentLoop`、`ToolHandlers`、`orchestrator` 已在 `agents/`；LLM 与爬虫工具生态在 Python；IO 密集型 Agent 用 `asyncio` + 多 Worker 进程水平扩展。

Python **GIL** 问题：网关不跑 LLM 主循环；每个子 Agent 以 **多进程 Worker** 或 **多副本 Pod** 扩展，网关做负载均衡。

---

## 3. 子 Agent 与意图路由

### 3.1 子 Agent 注册模型（网关侧）

建议在 `gateway` 维护静态 + 配置化注册表（`deploy/config/local.yaml` 扩展段）：

```yaml
agents:
  - id: web_crawler
    display_name: "网页采集"
    description: "自然语言描述采集任务，生成 Piper 模版并执行抓取"
    endpoint: "127.0.0.1:15061"   # Python web_crawler worker gRPC（避开 Windows 保留端口段）
    enabled: true
    intents: ["web_scrape", "crawl", "piper_collect", "default"]  # default 表示兜底
  # 预留
  - id: doc_rag
    enabled: false
    endpoint: "127.0.0.1:50062"
```

**首期**：仅 `web_crawler` 为 `enabled: true`；其余条目用于文档与后续接入。

### 3.2 意图识别策略（分阶段）

| 阶段 | 方式 | 延迟目标 | 说明 |
|------|------|----------|------|
| **MVP** | 规则 + 关键词 + 默认路由 | &lt; 50ms | 含「抓取、采集、爬、模板、URL、index」等 → `web_crawler`；无法判断 → `web_crawler`（当前唯一可用 Agent） |
| **V2** | 小模型结构化输出 | &lt; 200ms | Go 调用 OpenAI-compatible `chat/completions`，`response_format: json`，输出 `{ "agent_id", "confidence", "reason" }` |
| **V3** | 独立分类服务 | 可缓存 | 高频意图本地 BERT / 1B 模型；与业务 LLM 分离 |

路由结果写入审计日志：`trace_id`, `conversation_id`, `chosen_agent_id`, `classifier_version`。

### 3.3 Web 采集子 Agent（首期实现）

**职责**：将现有 CLI 能力封装为 **无 REPL 的流式服务**。

- 代码复用：`agents/src/piper_agent/agents/orchestrator.py` 的 `create_loop()`、`AgentLoop.run_turn()`（需改造为 **逐事件 yield**：assistant token、tool_start、tool_end、run_progress）。
- 对 Runtime：继续使用 `RuntimeClient` → `runtime.address`（与 today 的 `local.yaml` 一致）。
- 进程形态：由 `piper-serve` 子进程启动 `python -m piper_agent.workers.web_crawler`（gRPC `AgentWorkerService.Execute`），**不提供 CLI 入口**。

**用户可见事件类型**（经网关转为 SSE）：

| event | 含义 |
|-------|------|
| `message.delta` | 助手文本增量 |
| `message.done` | 本轮结束 |
| `tool.call` | 调用 list_templates / run_template 等 |
| `run.progress` | 订阅 Runtime SubscribeRun 的摘要 |
| `error` | 可展示错误 |
| `run.cancelled` | 用户点击停止 |

---

## 4. 网关 API 设计（面向 Vue）

### 4.1 约定

- Base path：`/api/v1`
- 认证（首期）：`deploy` 配置 `gateway.auth` 为 `none` 或固定 `Bearer` token；与 `engine` WebAPI `noAuth` 本地联调一致。
- 所有响应带 `X-Trace-Id`；SSE 每条 `data:` JSON 含 `trace_id`。

### 4.2 核心接口

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/conversations` | 创建会话，返回 `conversation_id` |
| `GET` | `/conversations` | 列表（分页） |
| `GET` | `/conversations/{id}/messages` | 历史消息 |
| `POST` | `/chat/completions` | Body: `{ conversation_id?, message, stream: true }`；**响应 SSE** |
| `POST` | `/runs/{run_id}/cancel` | 取消当前生成/后台 Run |
| `GET` | `/agents` | 返回已启用子 Agent 元数据（前端可展示「由 xxx Agent 处理」） |
| `GET` | `/health` | 网关 + runtime + worker 探活 |

### 4.3 SSE 载荷示例

```json
{
  "type": "message.delta",
  "conversation_id": "c_xxx",
  "run_id": "r_xxx",
  "agent_id": "web_crawler",
  "delta": "正在为您生成采集模版…"
}
```

前端使用 `fetch` + `ReadableStream` 或 `@microsoft/fetch-event-source` 解析；**停止生成**时 `abort()` fetch，网关将取消传播到 Python gRPC 与 Runtime `CancelRun`（若已发起执行）。

### 4.4 Go 与 Python 之间的契约（建议新增 proto）

在 `proto/` 下新增（实施时落地）：

- `proto/gateway/v1/session.proto`：会话消息存储（可选，若不全走 DB）
- `proto/agent/v1/execute.proto`：`ExecuteRequest`（conversation_id, user_message, history[], config_ref）、`ExecuteEvent`（oneof：delta / tool / progress / done / error）

Go gateway 作为 **gRPC client** 调用 Python worker；与现有 `proto/runtime/v1/execute.proto` **职责分离**：前者是「对话级 Agent」，后者是「采集 Runtime」。

---

## 5. 前端（`web/`）设计

### 5.1 技术栈

| 项 | 选型 |
|----|------|
| 框架 | Vue 3（Composition API） |
| 语言 | TypeScript |
| 构建 | Vite |
| 路由 | Vue Router |
| 状态 | Pinia（会话列表、当前流式状态） |
| HTTP | `fetch` + SSE；开发期 Vite proxy 到网关 |
| UI | 任选 Element Plus / Naive UI（实施时定一种即可） |

### 5.2 目录结构（新建）

```
web/
├── package.json
├── vite.config.ts          # proxy: /api -> http://127.0.0.1:8080
├── index.html
├── src/
│   ├── main.ts
│   ├── App.vue
│   ├── router/index.ts
│   ├── stores/chat.ts
│   ├── api/client.ts       # REST + SSE 封装
│   ├── views/
│   │   ├── ChatView.vue    # 主对话（类似 ChatGPT）
│   │   └── SettingsView.vue
│   └── components/
│       ├── MessageList.vue
│       ├── Composer.vue
│       └── AgentBadge.vue  # 显示当前路由到的 agent_id
└── README.md
```

### 5.3 页面行为（对标 OpenAI / DeepSeek）

- 左侧（可折叠）：会话列表；新建会话。
- 中间：消息气泡；助手消息流式追加；展示 tool / 采集进度为可折叠「执行步骤」。
- 输入框：发送、停止；Enter 发送 / Shift+Enter 换行。
- 空状态：示例提示语（如「帮我抓取某 API 的 title 和 url 字段」）。

### 5.4 环境变量

- `VITE_API_BASE`：默认 `/api/v1`（生产由 Nginx 反代网关）。

---

## 6. 仓库目录演进（Monorepo）

在现有 [README.md](README.md) 结构上 **新增**：

```
piper_agent/
├── gateway/                 # Go：HTTP/SSE API、意图路由、Agent 客户端
│   ├── cmd/piper-gateway/
│   └── internal/
│       ├── router/          # 意图识别
│       ├── session/         # 会话存储
│       ├── stream/          # SSE
│       └── agentclient/     # 调 Python worker
├── web/                     # Vue 3 + TS 前端
├── agents/
│   └── src/piper_agent/
│       ├── workers/         # 新增：web_crawler_worker.py 等
│       └── ...              # 现有 harness / orchestrator 不变
├── runtime/                 # 已有，配置 listen :50051
├── engine/                  # 已有
├── proto/                   # 扩展 agent/v1、gateway/v1
└── deploy/
    ├── config/local.yaml    # 增加 gateway、agents workers 端口
    └── docker-compose.web.yaml  # 可选：一键起 gateway + worker + runtime + web
```

`go.work` 增加 `./gateway` 模块。

---

## 7. 配置与本地启动顺序

### 7.1 配置扩展（`deploy/config/local.yaml`）

在现有 `runtime`、`llm` 段之外增加：

```yaml
gateway:
  listen: ":8080"
  auth:
    mode: none          # none | bearer
    token: ""
  session:
    driver: memory      # memory | redis | postgres
  classifier:
    mode: rule          # rule | llm

agents:
  web_crawler:
    listen: "127.0.0.1:15061"
```

### 7.2 推荐启动顺序（开发）

1. 依赖：ES / MinIO 等（见 [docs/PHASE_B.md](docs/PHASE_B.md)）。
2. **一键后端**：`cd gateway && go run ./cmd/piper-serve -f ../deploy/config/local.yaml`（内部拉起 Runtime、Worker、网关）。
3. `cd web && npm run dev` → 浏览器访问 Vite 端口。

单独调试时可使用 `piper-runtime`、`python -m piper_agent.workers.web_crawler`、`piper-gateway` 分进程启动。

生产：Nginx 托管 `web` 静态资源，`/api` 反代 `gateway`；TLS 在 Nginx 终止。

---

## 8. 通信、并发与可靠性

### 8.1 推荐：gRPC 流式（网关 ↔ Python）

- 与现有 Runtime 一致，便于 `scripts/gen_proto` 统一生成 Go/Python stub。
- 支持 **server streaming**：LLM token、工具事件、Run 进度逐条回传。
- 客户端断开或 `cancel`：Go `context.Cancel` → Python 停止 LLM 请求并调用 Runtime `CancelRun`。

### 8.2 长任务

采集任务可能持续数分钟：SSE 保持连接；`run.progress` 事件推送 Token 状态；结果摘要通过 `message.delta` 或最终 `message.done` 附带 structured payload（如 `index`, `doc_count`）。

### 8.3 状态存储

| 数据 | MVP | 生产建议 |
|------|-----|----------|
| 会话与消息 | 网关内存或 SQLite 文件 | PostgreSQL |
| Agent 执行上下文 | Python `SessionState` + 可选 Redis | Redis（多副本 worker 共享） |
| 采集结果 | 已有 ES/S3 | 不变 |

### 8.4 观测

- 全链路 `trace_id`：浏览器响应头 → 网关日志 → Python worker → Runtime gRPC metadata。
- 可选 OpenTelemetry；Python 侧延续 `harness/audit.py` JSONL。

---

## 9. 实施阶段（可验收里程碑）

### Phase W1 — 网关骨架 + 前端壳子（1–2 周）

- [x] 新建 `gateway/`，实现 `GET /health`、`POST /conversations`、`POST /chat/completions`（**假流**或固定 echo SSE）
- [x] 新建 `web/`，完成 Chat UI + SSE 消费
- [x] Vite 代理联调

**验收**：浏览器发一句话，能看到流式假回复。

### Phase W2 — 接入 Web 采集子 Agent（2–3 周）

- [x] 定义 `proto/agent/v1/execute.proto` 并生成代码
- [x] Python `worker web-crawler`：包装 `AgentLoop`，流式事件
- [x] 网关规则路由到 `web_crawler`；取消与超时
- [x] 会话历史持久化（至少 SQLite）

**验收**：浏览器输入与 CLI 等价的采集描述，能触发工具调用并完成一次 Run（或明确错误展示）。

### Phase W3 — 产品体验与运维（1–2 周）

- [x] 停止生成、run 进度 UI、Agent 徽章
- [x] `GET /agents`、探活聚合
- [x] `deploy/docker-compose.web.yaml`、文档更新
- [x] 限流与请求体大小限制

### Phase W4 — 多 Agent 扩展（持续）

- [ ] 分类器升级为 LLM JSON 路由
- [ ] 新子 Agent：独立 Python worker + 注册表一项 + 前端可选展示
- [ ] 鉴权对接企业 SSO（可选）

---

## 10. 安全与合规（Web 场景补充）

- **LLM API Key** 仅存在于服务端配置（`local.yaml` / 密钥管理），禁止写入前端或版本库。
- **CORS**：开发 Vite 源白名单；生产仅允许站点域名。
- **SSRF / 内网**：延续 Runtime 对目标 URL 的策略（见改造方案安全章节）。
- **采集合规**：系统提示与用户协议中声明合法使用边界；审计日志保留指令与 `agent_id`。

---

## 11. 与现有文档的对照

| 已有能力 | Web 系统中的位置 |
|----------|------------------|
| `piper-agent chat` / `ask` | 已移除；仅保留 `piper-agent doctor` 运维检查 |
| `AgentLoop` + `ToolHandlers` | Python worker 内核 |
| `piper-runtime` gRPC | 仅 web_crawler 调用 |
| `engine` WebSocket 进度 | 可选：网关订阅并转为 `run.progress` SSE；或 worker 轮询 `SubscribeRun` |
| [TRANSFORMATION_PLAN.md](docs/TRANSFORMATION_PLAN.md) Phase A–D | 采集链路的实现依据，Web 不重复实现 Runtime 语义 |

---

## 12. 小结

本 Web 系统通过 **Go 网关 + Vue 前端** 提供统一对话入口，用 **意图路由** 将问题分发给 **Python 子 Agent Worker**；首期仅实现 **Web 采集（爬虫）子 Agent**，直接复用现有 `agents` Harness 与 `runtime`/`engine` 采集栈。CLI 降级为开发与自动化入口，用户主路径为浏览器会话与流式反馈。

**文档版本**：v1.0（Web 系统版）  
**维护**：与 `proto/agent/v1`、`gateway/`、`web/` 实现同步更新。
