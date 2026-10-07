# pkg/distributor

采集 **调度内核**：Token 队列、Agent 绑定、指标。

## 核心文件（规划）

- `distributor.go` — 对齐 `Distributor.java`
- `token.go` — `Token.java`
- `cache.go` — `Cache.java`
- `stats.go` — `Stats.java`（Prometheus）
- `agent_selector.go` — `AgentSelector.java`
- `proc.go` — 引用 `pkg/tpl/proc` 或内联调度

## 子目录

- [callback](callback/README.md) — Token/Agent/Proxy 回调
- [exception](exception/README.md) — 业务错误类型
- [processing](processing/README.md) — Persister、TokenPersister、DependencyResolver
- [msg](msg/README.md) — KafkaMsg 等

