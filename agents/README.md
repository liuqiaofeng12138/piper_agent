# agents（Python）

**认知层**：自然语言理解、模版草稿、参数推断、与用户对话。

## 子目录

| 路径 | 模块 |
|------|------|
| [src/piper_agent/harness/](src/piper_agent/harness/) | Agent Harness：会话、LLM 循环、ToolRegistry |
| [src/piper_agent/tools/](src/piper_agent/tools/) | 工具实现（gRPC 客户端封装） |
| [src/piper_agent/agents/](src/piper_agent/agents/) | 具体 Agent：Orchestrator、TemplateAuthor、Runner |
| [src/piper_agent/clients/](src/piper_agent/clients/) | gRPC channel、重试、超时 |
| [src/piper_agent/cli/](src/piper_agent/cli/) | `piper-agent chat` / `run` |
| [tests/](tests/) | Harness 与 tool 的单元/集成测试 |

## 安装（规划）

```bash
cd agents
pip install -e ".[dev]"
```

## Harness 数据流

```
User → cli → harness.session → LLM → tools.* → grpc → Go Runtime
                     ↑___________________________|
                           tool results
```
