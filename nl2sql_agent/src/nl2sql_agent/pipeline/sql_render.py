from __future__ import annotations

import re

from nl2sql_agent.knowledge.catalog import SqlTemplate

_DANGEROUS = re.compile(
    r"\b(INSERT|UPDATE|DELETE|DROP|ALTER|TRUNCATE|CREATE|GRANT|REVOKE|COPY)\b",
    re.I,
)


def _sql_string_literal(value: str) -> str:
    return "'" + value.replace("'", "''") + "'"


def render_template(tpl: SqlTemplate, params: dict[str, str]) -> str:
    days = str(params.get("days") or "30").strip()
    if not days.isdigit():
        raise ValueError("days 必须为整数")
    days = str(max(1, min(365, int(days))))

    product = params.get("product_name") or ""
    fab = (params.get("fab_name") or "").strip()
    keyword = params.get("keyword") or product

    fab_filter = ""
    if fab:
        fab_filter = f"AND f.fab_name ILIKE '%' || {_sql_string_literal(fab)} || '%'"

    sql = tpl.sql
    sql = sql.replace("{{days}}", days)
    sql = sql.replace("{{product_name}}", _sql_string_literal(product))
    sql = sql.replace("{{fab_name}}", _sql_string_literal(fab or ""))
    sql = sql.replace("{{keyword}}", _sql_string_literal(keyword))
    sql = sql.replace("{{fab_filter}}", fab_filter)

    if _DANGEROUS.search(sql):
        raise ValueError("仅允许只读 SELECT 查询")
    if ";" in sql.strip().rstrip(";"):
        raise ValueError("不允许多语句 SQL")
    stripped = sql.strip()
    if not stripped.upper().startswith("SELECT"):
        raise ValueError("仅允许 SELECT")
    return stripped
