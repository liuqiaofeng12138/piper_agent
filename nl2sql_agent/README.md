# nl2sql_agent — 半导体制造 NL2SQL

基于 **知识库（表结构 + 问题类别 + SQL 模版 id 关联）** 的自然语言查数 Agent，连接 PostgreSQL 演示库，经网关 SSE 将 **表格 + ECharts 图表 + 文字解读** 推到 Web 前端。

## 知识库

见 [`src/nl2sql_agent/knowledge/catalog.yaml`](src/nl2sql_agent/knowledge/catalog.yaml)：

- **tables**：`dim_fab`、`dim_product`、`fact_yield_daily`、`fact_wip_lot`、`fact_equipment_oee`
- **categories**：良率趋势 / 多厂对比 / WIP 工序 / OEE / 产品检索
- **templates**：每类通过 `template_id` 绑定 SQL 模版与占位符说明

## 启动数据库（Docker）

```powershell
cd nl2sql_agent/docker
docker compose up -d
docker compose ps
```

默认 DSN：`postgresql://piper:piper_nl2sql@127.0.0.1:5433/semi_dw`

## Python Worker

```powershell
# 仓库根目录
powershell -File scripts/setup-python-venvs.ps1
cd nl2sql_agent
..\.venv\Scripts\python.exe -m nl2sql_agent.workers.nl2sql --config ../deploy/config/local.yaml
```

在 `deploy/config/local.yaml` 的 `agents` 中启用 `nl2sql` 后，由 `piper-serve` 一并拉起。

## 示例问题

- 「查询近一个月 N28 车规 MCU 的良率」
- 「N28 车规 MCU 各工厂良率对比」
- 「N28 车规 MCU 在制品各工序片数」
- 「上海 Fab 近 30 天设备 OEE」

## 配置

`deploy/config/local.yaml`：

```yaml
nl2sql:
  dsn: "postgresql://piper:piper_nl2sql@127.0.0.1:5433/semi_dw"
  use_llm_extract: false   # true 时用 LLM 选类别/抽槽（需 llm.api_key）
  use_llm_explain: true    # 有 api_key 时用 LLM 写解读；否则规则说明
```

## 测试

```powershell
cd nl2sql_agent
.\.venv\Scripts\python.exe -m pytest tests/ -q
```
