# harness（Python Agent Harness）

LLM 驱动的控制循环，是用户与 Go Runtime 之间的**唯一编排入口**。

## 模块文件（规划）

| 文件 | 职责 |
|------|------|
| `session.py` | `SessionState`：messages、artifacts、run_id |
| `loop.py` | `run_turn()`：调用 LLM → 解析 tool_calls → 执行 → 追加结果 |
| `registry.py` | `ToolRegistry`：名称、schema、handler、权限 |
| `policies.py` | 必须先 validate 再 run；最大步数；敏感确认 |
| `events.py` | 结构化事件供 Web / Worker 订阅 |

## 与 Go Runtime Harness 的关系

| Python Harness | Go Harness (`runtime/internal/harness`) |
|----------------|----------------------------------------|
| 自然语言、规划 | 结构化 Run 状态机 |
| 生成模版 JSON | 校验 + 执行 Token |
| `session_id` | `run_id` / Token id |

两者通过 gRPC 的 `session_id` / `run_id` 关联，OpenTelemetry trace 可选贯穿。
