# Phase D — 代理 / Chrome / 完整 Agent 链路（已实现主功能）

## 能力

| 项 | 说明 |
|----|------|
| **list_proxies / select_proxy** | 从 meta 列代理；会话内选中，`runner_execute` 传 `proxy_id` 绑定 HttpAgent + Token |
| **Chrome** | `builder.type=Chrome` 校验（需 Runtime 已启 Chrome）；`engine: auto\|chrome\|http` |
| **runner_wait_for_complete** | 订阅 Run 至结束，便于 Chat 一次说完 |
| **审计** | `harness.audit_log` JSONL 记录 tool 调用 |
| **速率** | `max_runs_per_session` 限制单会话 Run 次数 |
| **CLI** | `piper-agent chat`（已有）、`piper-agent ask "..."` 单轮 |

## 配置（local.yaml 的 harness 段）

```yaml
harness:
  max_runs_per_session: 10
  audit_log: "../../logs/agent_audit.jsonl"   # 相对本配置文件
```

## 端到端测试话术

1. 「列出可用代理，选 id 为 xxx 的代理。」
2. 「搜索 httpbin json 示例，校验并 save。」
3. 「用选中的代理 run 该模板，wait 完成后 get data。」

Chrome（Runtime 日志有 `chrome agents started` 时）：

「参考 example_chrome_navigate，validate 后 engine chrome run。」

## 重启 Runtime

Phase D 变更在 Go 侧，需重新编译并重启 `piper-runtime.exe`。
