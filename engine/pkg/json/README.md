# pkg/json

Piper HTTP JSON 工具（Phase 0）。

| 文件 | 说明 |
|------|------|
| `msg.go` | 与 Java `Msg` 相同的 `code` / `msg` / `data` / `_meta` 信封 |
| `write.go` | `WriteJSON` / `WriteMsg` 写响应 |

后续 Gson 多态适配器放在本子包或 `adapter/` 子目录。
