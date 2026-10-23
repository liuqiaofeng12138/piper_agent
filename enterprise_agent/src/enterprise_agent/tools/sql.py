"""结构化业务数据查询（演示库 SQLite，只读 SELECT）。"""

from __future__ import annotations

import re
import sqlite3
from datetime import date, timedelta
from pathlib import Path

_FORBIDDEN = re.compile(
    r"\b(INSERT|UPDATE|DELETE|DROP|ALTER|CREATE|REPLACE|ATTACH|DETACH|PRAGMA|VACUUM)\b",
    re.IGNORECASE,
)


def ensure_demo_database(db_path: Path) -> None:
    if db_path.is_file():
        return
    db_path.parent.mkdir(parents=True, exist_ok=True)
    conn = sqlite3.connect(db_path)
    try:
        conn.executescript(
            """
            CREATE TABLE products (
                product_id INTEGER PRIMARY KEY,
                product_name TEXT NOT NULL,
                category TEXT NOT NULL
            );
            CREATE TABLE sales (
                sale_id INTEGER PRIMARY KEY,
                sale_date TEXT NOT NULL,
                region TEXT NOT NULL,
                product_id INTEGER NOT NULL,
                amount REAL NOT NULL,
                units INTEGER NOT NULL,
                FOREIGN KEY (product_id) REFERENCES products(product_id)
            );
            """
        )
        products = [
            (1, "云协作套件 Pro", "SaaS"),
            (2, "数据洞察平台", "Analytics"),
            (3, "智能客服机器人", "AI"),
            (4, "供应链协同", "ERP"),
        ]
        conn.executemany(
            "INSERT INTO products (product_id, product_name, category) VALUES (?,?,?)",
            products,
        )
        today = date.today()
        first_this_month = today.replace(day=1)
        last_month_end = first_this_month - timedelta(days=1)
        last_month_start = last_month_end.replace(day=1)
        regions = ["华东", "华北", "华南", "西部"]
        rows: list[tuple] = []
        sid = 1
        d = last_month_start
        while d <= last_month_end:
            ds = d.isoformat()
            for region in regions:
                for pid, amount_base, units in [
                    (1, 82000, 12),
                    (2, 125000, 8),
                    (3, 56000, 20),
                    (4, 98000, 6),
                ]:
                    factor = 1.0 + (hash(f"{region}{ds}{pid}") % 17) / 100.0
                    rows.append((sid, ds, region, pid, round(amount_base * factor, 2), units))
                    sid += 1
            d += timedelta(days=1)
        conn.executemany(
            "INSERT INTO sales (sale_id, sale_date, region, product_id, amount, units) "
            "VALUES (?,?,?,?,?,?)",
            rows,
        )
        conn.commit()
    finally:
        conn.close()


def get_schema_metadata(db_path: Path) -> str:
    ensure_demo_database(db_path)
    conn = sqlite3.connect(db_path)
    try:
        cur = conn.execute(
            "SELECT name, sql FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' "
            "ORDER BY name"
        )
        parts: list[str] = []
        for name, ddl in cur.fetchall():
            parts.append(f"表 {name}:\n{ddl}")
            cols = conn.execute(f"PRAGMA table_info({name})").fetchall()
            col_lines = ", ".join(f"{c[1]} ({c[2]})" for c in cols)
            parts.append(f"  列: {col_lines}")
        parts.append(
            "\n说明: sale_date 为 ISO 日期；region 含 华东/华北/华南/西部；"
            "amount 为人民币销售额；可与 products 表 JOIN 获取 product_name。"
        )
        return "\n".join(parts)
    finally:
        conn.close()


def _validate_readonly_sql(query: str) -> str:
    q = query.strip().rstrip(";").strip()
    if not q:
        raise ValueError("SQL 不能为空")
    if ";" in q:
        raise ValueError("仅允许单条 SQL 语句")
    if _FORBIDDEN.search(q):
        raise ValueError("仅允许 SELECT 查询，禁止写操作或 PRAGMA")
    if not re.match(r"^\s*SELECT\b", q, re.IGNORECASE):
        raise ValueError("仅允许 SELECT 语句")
    return q


def run_sql_query(db_path: Path, query: str, max_rows: int = 200) -> str:
    ensure_demo_database(db_path)
    sql = _validate_readonly_sql(query)
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    try:
        cur = conn.execute(sql)
        rows = cur.fetchmany(max_rows + 1)
        if len(rows) > max_rows:
            rows = rows[:max_rows]
            truncated = True
        else:
            truncated = False
        if not rows:
            return "（查询无结果）"
        headers = rows[0].keys()
        lines = ["\t".join(headers)]
        for row in rows:
            lines.append("\t".join(str(row[h]) for h in headers))
        out = "\n".join(lines)
        if truncated:
            out += f"\n\n（结果已截断，仅展示前 {max_rows} 行）"
        return out
    finally:
        conn.close()
