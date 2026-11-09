from __future__ import annotations

import json
from typing import Any

from nl2sql_agent.knowledge.catalog import Catalog
from nl2sql_agent.pipeline.params import ExtractedParams
from nl2sql_agent.pipeline.db import QueryRows


def explain_rule(
    catalog: Catalog,
    extracted: ExtractedParams,
    sql: str,
    data: QueryRows,
) -> str:
    tpl = catalog.templates.get(extracted.template_id)
    tpl_desc = tpl.description.strip() if tpl else ""
    lines = [
        f"### 分析说明（{extracted.category_name}）",
        "",
        f"- **路由**：{extracted.route_reason}",
        f"- **模版 ID**：`{extracted.template_id}`",
        f"- **参数**：`{json.dumps(extracted.params, ensure_ascii=False)}`",
        "",
        "#### 模版说明",
        tpl_desc,
        "",
        "#### 执行的 SQL",
        "```sql",
        sql,
        "```",
        "",
        f"#### 结果概览",
        f"共 **{len(data.rows)}** 行。",
    ]
    if data.rows and "yield_pct" in data.columns:
        idx = data.columns.index("yield_pct")
        vals = [float(r[idx]) for r in data.rows if r[idx] is not None]
        if vals:
            lines.append(
                f"- 良率范围 **{min(vals):.3f}%** ~ **{max(vals):.3f}%**，"
                f"平均 **{sum(vals)/len(vals):.3f}%**。"
            )
    lines.append("")
    lines.append("#### 涉及数据表")
    lines.append("- `dim_product` / `dim_fab` / `fact_yield_daily`（良率类）")
    lines.append("- `fact_wip_lot`（在制品）")
    lines.append("- `fact_equipment_oee`（设备 OEE）")
    return "\n".join(lines)


def explain_llm(
    llm_client: Any,
    model: str,
    question: str,
    extracted: ExtractedParams,
    sql: str,
    sample_rows: list[dict[str, Any]],
) -> str:
    prompt = (
        "你是半导体制造数据分析师。根据用户问题、SQL 与查询结果样本，"
        "用简体中文写 3-5 段说明：业务含义、关键指标、是否异常、建议。"
        "不要编造样本中不存在的数据。\n\n"
        f"用户问题：{question}\n"
        f"类别：{extracted.category_name}\n"
        f"SQL：\n{sql}\n"
        f"样本行（JSON）：{json.dumps(sample_rows[:8], ensure_ascii=False, default=str)}"
    )
    resp = llm_client.chat.completions.create(
        model=model,
        messages=[{"role": "user", "content": prompt}],
        temperature=0.3,
    )
    body = (resp.choices[0].message.content or "").strip()
    return f"### 分析说明\n\n{body}\n\n#### SQL\n```sql\n{sql}\n```\n"
