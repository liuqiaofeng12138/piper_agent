# cmd/piper-runtime

Runtime 主程序。

## 职责

- 解析 flags：`-f config.yaml`、`-listen :50051`
- 初始化：`internal/adapter` 加载 piper_go 依赖（ES、meta SQLite、Chrome 开关）
- 启动 `internal/grpcserver`
- 信号处理：SIGTERM 时 drain 进行中的 Run

## 待实现

`main.go` 在 Phase A 添加；Phase B 接入真实 executor。
