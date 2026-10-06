# internal

go-zero **应用内私有代码**，不可被外部 module import。

## 子目录

| 目录 | 说明 |
|------|------|
| [config](config/README.md) | YAML 配置结构体 |
| [bootstrap](bootstrap/README.md) | 启动编排（对齐 `WebAPI.main`） |
| [middleware](middleware/README.md) | CORS、鉴权、Ready |
| [svc](svc/README.md) | ServiceContext |
| [types](types/README.md) | goctl 生成的 API 类型 |
| [handler](handler/README.md) | HTTP handlers（按域分子目录） |
| [logic](logic/README.md) | 用例 logic（按域分子目录） |

## 与 pkg 的边界

- `internal/*` 可以 import `pkg/*`
- `pkg/*` **不得** import `internal/*`
