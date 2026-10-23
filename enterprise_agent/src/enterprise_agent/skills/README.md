# Enterprise Agent Skills 标准目录

每个 Skill 为**独立子目录**，运行时由 `loader.py` 自动发现（目录内须包含 `SKILL.md`）。

## 目录布局

```
skills/<skill_id>/
├── SKILL.md      # YAML frontmatter + Markdown 指令（作为 LLM system prompt 正文）
├── tools.json    # OpenAI function calling 工具 schema 数组
└── skill.py      # create_skill(manifest, cfg) -> Skill 工厂与工具实现
```

## SKILL.md frontmatter 字段

| 字段 | 必填 | 说明 |
|------|------|------|
| `id` | 是 | 技能唯一 ID（路由、日志） |
| `name` | 是 | 展示名称 |
| `description` | 是 | 路由中枢与 UI 简述 |
| `route_keywords` | 建议 | 规则分诊关键词列表 |
| `order` | 否 | 注册排序，默认 100 |
| `tools` | 否 | 文档用，列出工具名（实现以 tools.json 为准） |

`---` 之后的 Markdown 正文会作为该 Skill 的 **system_prompt**。

## 新增 Skill

1. 复制任一现有目录为模板；
2. 修改 `SKILL.md` / `tools.json`；
3. 在 `skill.py` 实现 `create_skill` 与 `execute_tool`；
4. 无需改 `registry.py`（自动扫描）。
