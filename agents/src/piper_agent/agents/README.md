# agents（具体 Agent 角色）

可选多 Agent；MVP 可用单 Orchestrator + 工具集。

## 角色

| Agent | 职责 |
|-------|------|
| `orchestrator` | 读用户意图，分派子任务，汇总结果 |
| `template_author` | 输出 procedures / Mapper 结构（强 schema） |
| `param_filler` | 从话术提取 vars、proxy、index |
| `runner` | 只负责调用 run + 轮询 + 报告 |

## 文件

- `orchestrator.py` — 系统 prompt + harness 绑定
- `prompts/` — YAML 或 markdown 片段（可 symlink 到 `shared/prompts`）
