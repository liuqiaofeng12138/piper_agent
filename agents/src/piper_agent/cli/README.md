# cli

命令行入口。

## 命令（规划）

- `piper-agent chat` — 交互式 Harness
- `piper-agent run "..."` — 单指令非交互
- `piper-agent tools list` — 调试 Tool schema

入口模块：`__main__.py` 或 `click` / `typer` 在 `cli/main.py`。
