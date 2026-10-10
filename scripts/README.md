# scripts

| 脚本 | 作用 |
|------|------|
| `setup-python-venvs.ps1` / `setup-python-venvs.sh` | 为仓库内每个含 `pyproject.toml` 的 Python 子项目创建 **独立** `.venv` 并安装依赖（`piper-serve` 启动 Worker 时 **必须** 存在对应 `.venv`） |
| `start-web.ps1` | 一键启动 Runtime + 全部 Agent Worker + 网关（`piper-serve`） |
| `gen_proto.ps1` | 生成 Go stub（`runtime/pkg/pb`、`gateway/pkg/pb`）与 Python stub（`web_crawler_agent` 全量、`general_agent` / `rag_agent` 仅 agent/v1），并自动运行 `fix_pb_imports.py` |
| `fix_pb_imports.py` | 修正 protoc 生成的 Python 导入为 `web_crawler_agent.pb.*` / `piper_agent.pb.*` / `rag_agent.pb.*` |

Windows 开发环境优先提供 `.ps1`；CI 使用 `.sh`。
