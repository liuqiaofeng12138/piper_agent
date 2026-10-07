# claw_agent（Python）

**Web 采集子 Agent**：自然语言理解、Piper 模版草稿、参数推断、采集执行编排。
由 Web 网关（`gateway/`）通过 gRPC `agent.v1.AgentWorkerService` 调用。

## 子目录

| 路径 | 模块 |
|------|------|
| [src/claw_agent/harness/](src/claw_agent/harness/) | Agent Harness：会话、LLM 循环、ToolRegistry |
| [src/claw_agent/tools/](src/claw_agent/tools/) | 工具实现（模版生成/校验/执行/取数） |
| [src/claw_agent/agents/](src/claw_agent/agents/) | Orchestrator：`create_loop()` 组装 |
| [src/claw_agent/clients/](src/claw_agent/clients/) | Runtime gRPC 客户端（重试、超时） |
| [src/claw_agent/rag/](src/claw_agent/rag/) | 模版检索（示例库索引） |
| [src/claw_agent/workers/](src/claw_agent/workers/) | `web_crawler` gRPC Worker（网关调用入口） |
| [src/claw_agent/cli/](src/claw_agent/cli/) | 仅 `claw-agent doctor`（Runtime 连通性检查） |
| [tests/](tests/) | Harness 与 tool 的单元/集成测试 |

## 安装

```bash
cd claw_agent
uv pip install -e ".[dev,llm]"   # 或 pip install -e ".[dev,llm]"
```

## 运行

通常由 `piper-serve` 自动拉起（模块 `claw_agent.workers.web_crawler`）。单独调试：

```powershell
python -m claw_agent.workers.web_crawler --config ..\deploy\config\local.yaml --listen 127.0.0.1:15061
```

## Harness 数据流

```
User → gateway → workers.web_crawler → harness.loop → LLM → tools.* → grpc → Go Runtime → engine
                          ↑__________________________|
                                tool results / run progress
```
