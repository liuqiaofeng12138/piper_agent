from __future__ import annotations

import json
import re
from dataclasses import dataclass
from typing import Any

from nl2sql_agent.knowledge.catalog import Catalog, Category


@dataclass
class ExtractedParams:
    category_id: str
    category_name: str
    template_id: str
    params: dict[str, str]
    route_reason: str


_DAYS_RE = re.compile(r"近\s*(\d+)\s*天|最近\s*(\d+)\s*天|(\d+)\s*天")
_MONTH_RE = re.compile(r"近\s*一\s*个?\s*月|最近\s*一\s*个?\s*月|近\s*30\s*天|一个月")


def _parse_days(question: str, default: str = "30") -> str:
    if _MONTH_RE.search(question):
        return "30"
    m = _DAYS_RE.search(question)
    if m:
        for g in m.groups():
            if g:
                return str(max(1, min(365, int(g))))
    return default


def _guess_product_name(question: str) -> str:
    known = [
        "N28 车规 MCU",
        "N28 电源管理 IC",
        "N14 AI 推理芯片",
        "N40 射频前端",
    ]
    for name in known:
        if name.replace(" ", "") in question.replace(" ", "") or name in question:
            return name
    if "车规" in question or "MCU" in question.upper():
        return "N28 车规 MCU"
    if "电源" in question or "PMIC" in question.upper():
        return "N28 电源管理 IC"
    if "AI" in question.upper() or "推理" in question:
        return "N14 AI 推理芯片"
    if "射频" in question or "RF" in question.upper():
        return "N40 射频前端"
    m = re.search(r"[N]\d{2}[^，。?？\s]{0,12}", question, re.I)
    if m:
        return m.group(0).strip()
    return "N28 车规 MCU"


def _guess_fab_name(question: str) -> str:
    if "上海" in question:
        return "上海"
    if "成都" in question:
        return "成都"
    if "合肥" in question:
        return "合肥"
    if "华东" in question:
        return "Fab12"
    return ""


def _guess_keyword(question: str) -> str:
    m = re.search(r"产品[^，。?？]{0,8}([^\s，。?？]+)", question)
    if m:
        return m.group(1).strip()
    return ""


def classify_category(catalog: Catalog, question: str) -> tuple[Category | None, str]:
    lower = question.lower()
    best: Category | None = None
    best_score = 0
    for cat in catalog.categories:
        score = 0
        for kw in cat.keywords:
            if kw.lower() in lower or kw in question:
                score += 2
        if cat.id == "cat_yield_trend" and ("良率" in question or "yield" in lower):
            score += 3
        if cat.id == "cat_yield_by_fab" and ("对比" in question or "各厂" in question):
            score += 3
        if score > best_score:
            best_score = score
            best = cat
    if best is None:
        best = next((c for c in catalog.categories if c.id == "cat_yield_trend"), None)
        return best, "default: 未命中关键词，默认良率趋势"
    return best, f"rule: 命中类别 {best.name} (score={best_score})"


def extract_params_rule(catalog: Catalog, question: str) -> ExtractedParams:
    cat, reason = classify_category(catalog, question)
    if cat is None:
        raise ValueError("无可用问题类别")
    tpl = catalog.templates.get(cat.template_id)
    if tpl is None:
        raise ValueError(f"模版不存在: {cat.template_id}")

    params: dict[str, str] = {
        "days": _parse_days(question),
        "product_name": _guess_product_name(question),
        "fab_name": _guess_fab_name(question),
        "keyword": _guess_keyword(question) or _guess_product_name(question),
    }
    return ExtractedParams(
        category_id=cat.id,
        category_name=cat.name,
        template_id=cat.template_id,
        params=params,
        route_reason=reason,
    )


def extract_params_llm(
    catalog: Catalog,
    question: str,
    llm_client: Any,
    model: str,
) -> ExtractedParams:
    cats = [
        {
            "id": c.id,
            "name": c.name,
            "description": c.description,
            "template_id": c.template_id,
        }
        for c in catalog.categories
    ]
    prompt = (
        "你是半导体制造 NL2SQL 助手。根据用户问题选择 category_id，并抽取 SQL 模版参数。\n"
        "输出 JSON："
        '{"category_id":"...","params":{"days":"30","product_name":"...","fab_name":"...","keyword":"..."},'
        '"reason":"..."}\n'
        f"类别列表：{json.dumps(cats, ensure_ascii=False)}\n"
        f"用户问题：{question}"
    )
    resp = llm_client.chat.completions.create(
        model=model,
        messages=[{"role": "user", "content": prompt}],
        response_format={"type": "json_object"},
        temperature=0,
    )
    raw = json.loads(resp.choices[0].message.content or "{}")
    cid = str(raw.get("category_id") or "")
    cat = next((c for c in catalog.categories if c.id == cid), None)
    if cat is None:
        return extract_params_rule(catalog, question)
    params = {k: str(v) for k, v in (raw.get("params") or {}).items()}
    params.setdefault("days", _parse_days(question))
    params.setdefault("product_name", _guess_product_name(question))
    params.setdefault("fab_name", _guess_fab_name(question))
    params.setdefault("keyword", _guess_keyword(question))
    return ExtractedParams(
        category_id=cat.id,
        category_name=cat.name,
        template_id=cat.template_id,
        params=params,
        route_reason="llm: " + str(raw.get("reason") or ""),
    )
