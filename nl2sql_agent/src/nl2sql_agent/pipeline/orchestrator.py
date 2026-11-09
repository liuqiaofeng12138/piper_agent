from __future__ import annotations

import json
from dataclasses import dataclass
from typing import Any, Callable, Iterator

from nl2sql_agent.knowledge.catalog import Catalog, load_catalog
from nl2sql_agent.pipeline.chart import build_chart_spec
from nl2sql_agent.pipeline.db import QueryRows, ping, run_select
from nl2sql_agent.pipeline.explain import explain_llm, explain_rule
from nl2sql_agent.pipeline.params import ExtractedParams, extract_params_llm, extract_params_rule
from nl2sql_agent.pipeline.sql_render import render_template

NL2SQL_PAYLOAD_TAG = "piper-nl2sql-data"


@dataclass
class QueryResult:
    extracted: ExtractedParams
    sql: str
    data: QueryRows
    chart: dict[str, Any]
    explanation: str

    def embed_payload(self, markdown: str) -> str:
        payload = {
            "chart": self.chart,
            "columns": self.data.columns,
            "rows": self.data.rows,
            "sql": self.sql,
            "category": self.extracted.category_name,
        }
        blob = json.dumps(payload, ensure_ascii=False, default=str)
        return f"{markdown}\n\n<!-- {NL2SQL_PAYLOAD_TAG}:{blob} -->\n"


class NL2SQLOrchestrator:
    def __init__(
        self,
        dsn: str,
        catalog: Catalog | None = None,
        llm_client: Any | None = None,
        llm_model: str = "gpt-4o",
        use_llm_extract: bool = False,
        use_llm_explain: bool = False,
    ) -> None:
        self._dsn = dsn
        self._catalog = catalog or load_catalog()
        self._llm = llm_client
        self._model = llm_model
        self._use_llm_extract = use_llm_extract and llm_client is not None
        self._use_llm_explain = use_llm_explain and llm_client is not None

    def ensure_db(self) -> None:
        ping(self._dsn)

    def run_turn(
        self,
        question: str,
        cancel: Callable[[], bool] | None = None,
    ) -> Iterator[dict[str, Any]]:
        if cancel and cancel():
            yield {"type": "run.cancelled", "message": "已取消"}
            return

        yield {
            "type": "run.progress",
            "tool_name": "nl2sql.classify",
            "message": "识别问题类别与 SQL 模版…",
        }

        if self._use_llm_extract:
            extracted = extract_params_llm(self._catalog, question, self._llm, self._model)
        else:
            extracted = extract_params_rule(self._catalog, question)

        tpl = self._catalog.templates.get(extracted.template_id)
        if tpl is None:
            yield {"type": "error", "message": f"模版 {extracted.template_id} 不存在"}
            return

        yield {
            "type": "tool.call",
            "tool_name": "nl2sql.template",
            "tool_detail": f"category={extracted.category_id} template={extracted.template_id}\n"
            f"params={json.dumps(extracted.params, ensure_ascii=False)}",
        }

        try:
            sql = render_template(tpl, extracted.params)
        except ValueError as e:
            yield {"type": "error", "message": str(e)}
            return

        yield {
            "type": "tool.call",
            "tool_name": "nl2sql.sql",
            "tool_detail": sql,
        }

        if cancel and cancel():
            yield {"type": "run.cancelled", "message": "已取消"}
            return

        yield {
            "type": "run.progress",
            "tool_name": "nl2sql.query",
            "message": "连接 PostgreSQL 并执行只读查询…",
        }

        try:
            data = run_select(self._dsn, sql)
        except Exception as e:  # noqa: BLE001
            yield {"type": "error", "message": f"数据库查询失败: {e}"}
            return

        cat = next((c for c in self._catalog.categories if c.id == extracted.category_id), None)
        chart = build_chart_spec(cat, data.columns, data.rows) if cat else {
            "chartType": "table",
            "title": "查询结果",
            "columns": data.columns,
            "rows": data.to_dicts(),
        }

        yield {
            "type": "query.result",
            "tool_name": "nl2sql.result",
            "tool_detail": json.dumps(
                {"columns": data.columns, "rows": data.rows},
                ensure_ascii=False,
                default=str,
            ),
        }
        yield {
            "type": "chart.data",
            "tool_name": "nl2sql.chart",
            "tool_detail": json.dumps(chart, ensure_ascii=False, default=str),
        }

        if self._use_llm_explain:
            explanation = explain_llm(
                self._llm,
                self._model,
                question,
                extracted,
                sql,
                data.to_dicts(),
            )
        else:
            explanation = explain_rule(self._catalog, extracted, sql, data)

        result = QueryResult(
            extracted=extracted,
            sql=sql,
            data=data,
            chart=chart,
            explanation=explanation,
        )
        final_md = result.embed_payload(explanation)
        yield {"type": "message.done", "content": final_md}
