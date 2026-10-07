# proto

gRPC 与 protobuf 契约层，**Harness（Python）与 Runtime（Go）的唯一跨语言 API**。

## 建议布局

```
proto/
├── buf.yaml                 # 可选：buf  lint/breaking
├── runtime/v1/
│   ├── meta.proto           # ListTemplates, ListProxies, ListIndices
│   ├── template.proto       # ValidateTemplate, UpsertTemplate, BuildToken
│   ├── execute.proto        # RunTemplate, CancelRun, SubscribeRun (stream)
│   ├── data.proto           # GetTokenData, SearchDocs
│   └── session.proto        # CreateSession, AppendMessage（可选）
└── common/v1/
    ├── types.proto          # TemplateDoc, Vars, RunStatus, Diagnostic
    └── errors.proto         # ErrorCode enum
```

## 生成代码

- Go：`scripts/gen_proto_go.sh` → `runtime/pkg/pb/`
- Python：`scripts/gen_proto.ps1` → `web_crawler_agent/src/web_crawler_agent/pb/`（全部 proto）与 `general_agent/src/piper_agent/pb/`（仅 `agent/v1`），随后自动运行 `scripts/fix_pb_imports.py` 修正 `web_crawler_agent.pb.*` / `piper_agent.pb.*` 导入

## 约定

- 所有 RPC 请求携带 `request_id`；Run 类携带 `session_id`（Harness 生成）。
- 大 payload（完整模版 JSON）走 `bytes json_payload` 或 `google.protobuf.Struct`，与 Piper meta 表形态一致。
