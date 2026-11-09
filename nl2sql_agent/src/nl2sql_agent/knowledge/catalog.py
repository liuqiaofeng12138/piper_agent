from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Any

import yaml


@dataclass(frozen=True)
class ColumnMeta:
    name: str
    type: str
    description: str


@dataclass(frozen=True)
class TableMeta:
    name: str
    description: str
    columns: tuple[ColumnMeta, ...]


@dataclass(frozen=True)
class Category:
    id: str
    name: str
    description: str
    template_id: str
    keywords: tuple[str, ...]
    chart: str


@dataclass(frozen=True)
class SqlTemplate:
    id: str
    category_id: str
    description: str
    sql: str


@dataclass
class Catalog:
    tables: dict[str, TableMeta]
    categories: list[Category]
    templates: dict[str, SqlTemplate]

    def template_for_category(self, category_id: str) -> SqlTemplate | None:
        for c in self.categories:
            if c.id == category_id:
                return self.templates.get(c.template_id)
        return None

    def schema_markdown(self) -> str:
        lines = ["## 数据表说明\n"]
        for name, t in self.tables.items():
            lines.append(f"### `{name}` — {t.description}\n")
            for col in t.columns:
                lines.append(f"- **{col.name}** ({col.type}): {col.description}")
            lines.append("")
        return "\n".join(lines)


def load_catalog(path: Path | None = None) -> Catalog:
    if path is None:
        path = Path(__file__).resolve().parent / "catalog.yaml"
    raw: dict[str, Any] = yaml.safe_load(path.read_text(encoding="utf-8")) or {}

    tables: dict[str, TableMeta] = {}
    for tname, tdef in (raw.get("tables") or {}).items():
        cols = tuple(
            ColumnMeta(
                name=str(c["name"]),
                type=str(c.get("type", "")),
                description=str(c.get("description", "")),
            )
            for c in (tdef.get("columns") or [])
        )
        tables[str(tname)] = TableMeta(
            name=str(tname),
            description=str(tdef.get("description", "")),
            columns=cols,
        )

    categories: list[Category] = []
    for c in raw.get("categories") or []:
        categories.append(
            Category(
                id=str(c["id"]),
                name=str(c.get("name", "")),
                description=str(c.get("description", "")),
                template_id=str(c["template_id"]),
                keywords=tuple(str(k) for k in (c.get("keywords") or [])),
                chart=str(c.get("chart", "table")),
            )
        )

    templates: dict[str, SqlTemplate] = {}
    for t in raw.get("templates") or []:
        tid = str(t["id"])
        templates[tid] = SqlTemplate(
            id=tid,
            category_id=str(t.get("category_id", "")),
            description=str(t.get("description", "")),
            sql=str(t.get("sql", "")).strip(),
        )

    return Catalog(tables=tables, categories=categories, templates=templates)
