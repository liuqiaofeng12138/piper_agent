from pathlib import Path

import pytest

from enterprise_agent.tools.sql import ensure_demo_database, get_schema_metadata, run_sql_query


def test_readonly_sql_rejects_write(tmp_path: Path) -> None:
    db = tmp_path / "demo.sqlite"
    ensure_demo_database(db)
    with pytest.raises(ValueError, match="SELECT"):
        run_sql_query(db, "DELETE FROM sales")


def test_demo_sales_query(tmp_path: Path) -> None:
    db = tmp_path / "demo.sqlite"
    ensure_demo_database(db)
    meta = get_schema_metadata(db)
    assert "sales" in meta
    out = run_sql_query(
        db,
        "SELECT region, SUM(amount) AS total FROM sales WHERE region='华东' GROUP BY region",
    )
    assert "华东" in out
    assert "total" in out.lower() or "华东" in out
