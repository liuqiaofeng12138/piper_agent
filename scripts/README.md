# scripts

| 脚本 | 作用 |
|------|------|
| `start-web.ps1` | 一键启动 Runtime + 全部 Agent Worker + 网关（`piper-serve`） |
| `gen_proto.ps1` | 生成 Go stub（`runtime/pkg/pb`、`gateway/pkg/pb`）与 Python stub（`claw_agent/src/claw_agent/pb` 全量、`agents/src/piper_agent/pb` 仅 agent/v1），并自动运行 `fix_pb_imports.py` |
| `fix_pb_imports.py` | 修正 protoc 生成的 Python 导入为 `claw_agent.pb.*` / `piper_agent.pb.*` |

Windows 开发环境优先提供 `.ps1`；CI 使用 `.sh`。
