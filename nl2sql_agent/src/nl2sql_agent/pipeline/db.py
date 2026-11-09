from __future__ import annotations

from dataclasses import dataclass
from datetime import date, datetime
from decimal import Decimal
from typing import Any

import psycopg2
import psycopg2.extras


def json_safe(value: Any) -> Any:
    """PostgreSQL 单元格 → JSON / LLM 可序列化值。"""
    if isinstance(value, datetime):
        return value.isoformat(sep=" ", timespec="seconds")
    if isinstance(value, date):
        return value.isoformat()
    if isinstance(value, Decimal):
        return float(value)
    return value


@dataclass
class QueryRows:
    columns: list[str]
    rows: list[list[Any]]

    def to_dicts(self) -> list[dict[str, Any]]:
        out: list[dict[str, Any]] = []
        for row in self.rows:
            out.append(
                {self.columns[i]: json_safe(row[i]) for i in range(len(self.columns))}
            )
        return out


def run_select(dsn: str, sql: str, max_rows: int = 500) -> QueryRows:
    conn = psycopg2.connect(dsn)
    try:
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute(f"SET statement_timeout = '30s'")
            cur.execute(sql)
            fetched = cur.fetchmany(max_rows + 1)
            if len(fetched) > max_rows:
                raise ValueError(f"结果行数超过上限 {max_rows}")
            if not fetched:
                return QueryRows(columns=[], rows=[])
            columns = list(fetched[0].keys())
            rows = [[r[c] for c in columns] for r in fetched]
            return QueryRows(columns=columns, rows=rows)
    finally:
        conn.close()


def ping(dsn: str) -> None:
    conn = psycopg2.connect(dsn)
    try:
        with conn.cursor() as cur:
            cur.execute("SELECT 1")
    finally:
        conn.close()
