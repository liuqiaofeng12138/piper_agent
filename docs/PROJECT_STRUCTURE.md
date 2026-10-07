# 项目结构说明

与 [TRANSFORMATION_PLAN.md](TRANSFORMATION_PLAN.md) 配套，便于快速定位模块。

```
piper_agent/                      # 本仓库（Monorepo）
├── README.md
├── go.work                       # Go workspace：runtime + engine + gateway
├── docs/
│   ├── TRANSFORMATION_PLAN.md
│   └── PROJECT_STRUCTURE.md      # 本文件
├── proto/                        # gRPC 契约（源）
│   ├── common/v1/*.proto
│   ├── runtime/v1/*.proto
│   └── agent/v1/*.proto          # Worker 契约（gateway ↔ Python Agent）
├── engine/                       # Piper 采集内核（go.mod module: piper_go，全 Agent 共享）
│   ├── etc/db/                   # meta SQLite 数据（配置在 deploy/config/local.yaml）
│   ├── pkg/agentruntime/         # Runtime 嵌入入口（Bootstrap）
│   └── pkg/tpl/ pkg/distributor/ # 模版与执行
├── runtime/                      # Agent gRPC Runtime（module: piper_agent/runtime，共享执行侧）
│   ├── go.mod                    # replace piper_go => ../engine
│   ├── cmd/piper-runtime/
│   └── internal/
│       ├── grpcserver/
│       ├── piper/                # gRPC → agentruntime
│       ├── executor/
│       └── mock/                 # mock: true 时无 engine 依赖栈
├── gateway/                      # Web API 网关（module: piper_agent/gateway）
│   ├── cmd/piper-gateway/        # 仅网关
│   └── cmd/piper-serve/          # 一键拉起 Runtime + Workers + 网关
├── web_crawler_agent/            # Web 采集子 Agent（Python，目录名即包名）
│   ├── pyproject.toml
│   ├── src/web_crawler_agent/
│   │   ├── harness/              # 会话、LLM 循环、ToolRegistry
│   │   ├── tools/                # 模版生成/校验/执行/取数
│   │   ├── agents/               # Orchestrator
│   │   ├── clients/              # Runtime gRPC 客户端
│   │   ├── rag/                  # 模版检索
│   │   ├── workers/              # web_crawler gRPC Worker
│   │   └── cli/                  # web-crawler-agent doctor
│   └── tests/
├── general_agent/                # 平台通用子 Agent（Python 包名 piper_agent）
│   ├── pyproject.toml
│   └── src/piper_agent/
│       ├── workers/              # general_chat gRPC Worker
│       ├── config/
│       └── pb/                   # 仅 agent/v1 stub
├── shared/
├── deploy/
│   └── config/
│       └── local.yaml            # Runtime + engine + Agent 统一配置
└── scripts/
```

> 约定：`engine/` 与 `runtime/` 是**所有子 Agent 共享**的采集内核与执行侧，不随单个 Agent 移动；
> 每个子 Agent 一个独立目录（`<name>_agent/`），内部统一为 `pyproject.toml + src/<pkg>/ + tests/` 布局。

## 模块依赖方向

```
gateway (Go HTTP/SSE)
    → gRPC agent.v1 → web_crawler_agent / general_agent (Python Workers)
                          → gRPC → runtime (Go gRPC + Run 状态机)
                                      → engine / piper_go (Template / Distributor / Persistence)
                                      → ES / S3 / SQLite meta
```

**禁止** Python 直接 import Go；**禁止** LLM 直连 ES。所有执行必须经过 Runtime gRPC。

## 配置路径约定

| 配置项 | 解析方式 |
|--------|----------|
| `deploy/config/local.yaml` 的 `engine.H2.path` 等 | 相对**该 yaml 文件**所在目录 |
| `runtime/go.mod` 的 `replace piper_go` | 相对 `runtime/` → `../engine` |
