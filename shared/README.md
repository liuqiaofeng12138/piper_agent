# shared

跨 Python / Go 共享的**非代码**资产。

## 子目录

| 路径 | 内容 |
|------|------|
| [schema/](schema/) | Piper 模版 JSON Schema、Tool JSON Schema |
| [examples/templates/](examples/templates/) | 最小 HTTP/Chrome 模版样例，供 LLM few-shot |
| [prompts/](prompts/) | 系统提示、安全边界、输出格式说明 |

Runtime 校验可同时使用 `schema/template.schema.json` 与 piper_go 内置解析双重保障。
