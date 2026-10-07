# agents（Python）

**平台通用子 Agent Workers**：与采集无关的通用对话类 Agent。
Web 采集 Agent 已独立为 [../claw_agent/](../claw_agent/) 项目。

## 子目录

| 路径 | 模块 |
|------|------|
| [src/piper_agent/workers/](src/piper_agent/workers/) | `general_chat`：通用对话 gRPC Worker（无工具，纯 LLM 流式问答） |
| [src/piper_agent/config/](src/piper_agent/config/) | 配置加载（`llm` 段、Worker 监听地址） |
| [src/piper_agent/pb/](src/piper_agent/pb/) | `agent.v1` Worker 契约 stub（由 `scripts/gen_proto.ps1` 生成） |

## 说明

- 由 `piper-serve` 按 `deploy/config/local.yaml` 的 `agents` 注册表自动拉起（模块 `piper_agent.workers.general_chat`）。
- 新增通用类子 Agent：在 `workers/` 下实现 `agent.v1.AgentWorkerService`，并在注册表加一项即可。
