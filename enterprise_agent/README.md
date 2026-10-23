# enterprise_agent — 企业运营与 BI 智囊

手搓 **Skills 架构**（无 LangGraph / 无通用 Agent 框架）：

| Skill ID | 职责 |
|----------|------|
| `data_sql` | 演示 SQLite 经营库只读查询 |
| `document_ocr` | 上传文档 Markdown 化与表格抽取 |
| `financial_model` | Excel 预览 + 受限 Python/pandas 指标分析 |
| `web_research` | DuckDuckGo 搜索与网页正文抓取 |

主控：`SkillRegistry` 分诊 → 各 Skill 独立工具循环 → `orchestrator` 汇总流式报告。

Skills 采用标准目录布局（见 `src/enterprise_agent/skills/README.md`）：每个技能含 `SKILL.md`、`tools.json`、`skill.py`。

## 本地运行

```powershell
cd enterprise_agent
..\.venv  # 或执行仓库根 scripts\setup-python-venvs.ps1
.\.venv\Scripts\python.exe -m enterprise_agent.workers.enterprise_ops
```

在 `deploy/config/local.yaml` 的 `agents` 中启用 `enterprise_ops` 后，由 `piper-serve` 一并拉起。
