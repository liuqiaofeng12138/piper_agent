# 项目结构说明

与 [TRANSFORMATION_PLAN.md](TRANSFORMATION_PLAN.md) 配套，便于快速定位模块。

```
piper_agent/                      # 本仓库（Monorepo）
├── README.md
├── go.work                       # Go workspace：runtime + engine
├── docs/
│   ├── TRANSFORMATION_PLAN.md
│   └── PROJECT_STRUCTURE.md      # 本文件
├── proto/                        # gRPC 契约（源）
│   ├── common/v1/types.proto
│   └── runtime/v1/*.proto
├── engine/                       # Piper 采集内核（go.mod module: piper_go）
│   └── etc/db/                   # meta SQLite 数据（配置在 deploy/config/local.yaml）
│   ├── pkg/agentruntime/         # Runtime 嵌入入口（Bootstrap）
│   └── pkg/tpl/ pkg/distributor/ # 模版与执行
├── runtime/                      # Agent gRPC Runtime（module: piper_agent/runtime）
│   ├── go.mod                    # replace piper_go => ../engine
│   ├── cmd/piper-runtime/
│   └── internal/
│       ├── grpcserver/
│       ├── piper/                # gRPC → agentruntime
│       ├── executor/
│       └── mock/                 # mock: true 时无 engine 依赖栈
├── agents/                       # Python Agent
│   ├── pyproject.toml
│   └── src/piper_agent/
│       ├── harness/
│       ├── tools/
│       ├── agents/
│       ├── clients/
│       └── cli/
├── shared/
├── deploy/
│   └── config/
│       └── local.yaml            # Runtime + engine + Agent 统一配置
└── scripts/
```

## 模块依赖方向

```
agents (Python Harness)
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
