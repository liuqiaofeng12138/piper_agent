# internal/adapter

将 Piper 现有配置与进程模型适配为 Runtime 单进程。

## 内容

- 读取 `deploy/config/local.yaml` 中的 `engine:` 段
- 构造与 `engine/internal/svc.ServiceContext` 等价的依赖图（Go import 仍为 `piper_go/...`）
- Chrome / ES / S3 就绪检查（可复用 `pkg/bootstrap`）

## 目标

Runtime 可独立启动，无需再跑完整 go-zero HTTP API（HTTP 仍可选作运维面）。
