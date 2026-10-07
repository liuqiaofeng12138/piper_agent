# internal/executor

对 `piper_go` 的薄封装，便于 mock 与单元测试。

## 规划接口

| 方法 | 映射 piper_go |
|------|----------------|
| `ValidateTemplate(doc)` | `tpl` 解析 + builder 检查 |
| `RunTemplate(tplID, vars, opts)` | `distributor.Engine.RunTemplate` |
| `GetTokenData(tokenID)` | `tpl` token data 聚合 |
| `ListTemplates(filter)` | `pkg/db/meta` templates 表 |
| `ListProxies()` | meta proxies 表 |

## 文件规划

- `template.go` — 模版 CRUD 与校验
- `run.go` — 执行与取消
- `data.go` — 查询与导出
- `meta.go` — 索引、账号、函数列表（供 LLM 上下文）
