# scripts

| 脚本 | 作用 |
|------|------|
| `setup-python-venvs.ps1` / `setup-python-venvs.sh` | 为仓库内每个含 `pyproject.toml` 的 Python 子项目创建 **独立** `.venv` 并安装依赖（默认 **清华 PyPI 镜像**；`PIPER_PIP_INDEX` / `PIP_INDEX_URL` 可换源，`PIPER_PIP_NO_MIRROR=1` 用官方源） |
| `start-web.ps1` | 一键启动 Runtime + 全部 Agent Worker + 网关（`piper-serve`） |
| `gen_proto.ps1` | 生成 Go stub（`runtime/pkg/pb`、`gateway/pkg/pb`）与 Python `agent/v1` stub（各 Python 子 Agent），并运行 `fix_pb_imports.py` |
| `fix_pb_imports.py` | 修正 protoc 生成的 Python 导入为 `piper_agent.pb.*` / `rag_agent.pb.*` 等 |

Windows 开发环境优先提供 `.ps1`；CI 使用 `.sh`。
