from __future__ import annotations

from datetime import date, datetime
from decimal import Decimal
from typing import Any

from nl2sql_agent.knowledge.catalog import Category


def _serialize(v: Any) -> Any:
    if isinstance(v, (datetime, date)):
        return v.isoformat()
    if isinstance(v, Decimal):
        return float(v)
    return v


def build_chart_spec(category: Category, columns: list[str], rows: list[list[Any]]) -> dict[str, Any]:
    if not columns or not rows:
        return {"chartType": "table", "title": category.name, "columns": columns, "rows": []}

    dict_rows = [
        {columns[i]: _serialize(row[i]) for i in range(len(columns))}
        for row in rows
    ]

    date_col = next((c for c in columns if "date" in c.lower()), None)
    pct_cols = [c for c in columns if "pct" in c.lower() or "yield" in c.lower() or c.endswith("_pct")]
    name_col = next(
        (c for c in columns if c in ("fab_name", "stage", "product_name", "equipment_id")),
        None,
    )
    num_col = pct_cols[0] if pct_cols else next(
        (c for c in columns if c not in (date_col, name_col)),
        columns[-1],
    )

    chart_type = category.chart
    if chart_type == "line" and date_col and num_col:
        return {
            "chartType": "line",
            "title": category.name,
            "xField": date_col,
            "yFields": [num_col],
            "seriesField": "equipment_id" if "equipment_id" in columns else None,
            "columns": columns,
            "rows": dict_rows,
        }
    if chart_type == "bar" and name_col and num_col:
        return {
            "chartType": "bar",
            "title": category.name,
            "xField": name_col,
            "yFields": [num_col],
            "columns": columns,
            "rows": dict_rows,
        }
    return {
        "chartType": "table",
        "title": category.name,
        "columns": columns,
        "rows": dict_rows,
    }
