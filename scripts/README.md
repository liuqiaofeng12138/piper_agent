# scripts

| 脚本 | 作用 |
|------|------|
| `gen_proto_go.ps1` / `.sh` | 生成 Go `runtime/pkg/pb` |
| `gen_proto_py.ps1` / `.sh` | 生成 Python `agents/src/piper_agent/pb` |
| `dev_up.ps1` | 启动 Runtime + 可选 piper_go 依赖栈 |

Windows 开发环境优先提供 `.ps1`；CI 使用 `.sh`。
