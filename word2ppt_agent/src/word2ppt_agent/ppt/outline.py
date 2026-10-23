from __future__ import annotations

import json
import re
from dataclasses import dataclass, field
from typing import Any

VALID_LAYOUTS = frozenset(
    {"title", "section", "content", "two_column", "closing", "agenda", "highlight", "metrics"}
)
VALID_THEMES = frozenset({"corporate_blue", "ocean", "slate", "warm"})


@dataclass
class MetricItem:
    label: str
    value: str


@dataclass
class SlideSpec:
    layout: str
    title: str
    subtitle: str = ""
    bullets: list[str] = field(default_factory=list)
    left_title: str = ""
    right_title: str = ""
    left_bullets: list[str] = field(default_factory=list)
    right_bullets: list[str] = field(default_factory=list)
    metrics: list[MetricItem] = field(default_factory=list)
    speaker_notes: str = ""


@dataclass
class DeckOutline:
    deck_title: str
    deck_subtitle: str
    theme: str
    slides: list[SlideSpec]


def _clean_str(v: Any, max_len: int = 500) -> str:
    s = re.sub(r"\s+", " ", str(v or "")).strip()
    return s[:max_len]


def _clean_metrics(raw: Any, max_items: int = 4) -> list[MetricItem]:
    if not isinstance(raw, list):
        return []
    out: list[MetricItem] = []
    for item in raw:
        if isinstance(item, dict):
            label = _clean_str(item.get("label"), 40)
            value = _clean_str(item.get("value"), 32)
        elif isinstance(item, str) and "|" in item:
            left, right = item.split("|", 1)
            label, value = _clean_str(left, 40), _clean_str(right, 32)
        else:
            continue
        if label and value:
            out.append(MetricItem(label=label, value=value))
        if len(out) >= max_items:
            break
    return out


def _clean_bullets(raw: Any, max_items: int = 8, max_item_len: int = 280) -> list[str]:
    if not isinstance(raw, list):
        return []
    out: list[str] = []
    for item in raw:
        line = _clean_str(item, max_item_len)
        if line:
            out.append(line)
        if len(out) >= max_items:
            break
    return out


def parse_deck_outline(data: dict[str, Any], max_slides: int) -> DeckOutline:
    theme = _clean_str(data.get("theme"), 32).lower() or "corporate_blue"
    if theme not in VALID_THEMES:
        theme = "corporate_blue"

    deck_title = _clean_str(data.get("deck_title") or data.get("title"), 120) or "演示文稿"
    deck_subtitle = _clean_str(data.get("deck_subtitle") or data.get("subtitle"), 200)

    slides_raw = data.get("slides")
    if not isinstance(slides_raw, list) or not slides_raw:
        raise ValueError("slides 必须为非空数组")

    slides: list[SlideSpec] = []
    for item in slides_raw[:max_slides]:
        if not isinstance(item, dict):
            continue
        layout = _clean_str(item.get("layout"), 32).lower() or "content"
        if layout not in VALID_LAYOUTS:
            layout = "content"
        title = _clean_str(item.get("title"), 120) or "未命名"
        subtitle = _clean_str(item.get("subtitle"), 200)
        bullets = _clean_bullets(item.get("bullets"))
        left = _clean_bullets(item.get("left_bullets"))
        right = _clean_bullets(item.get("right_bullets"))
        metrics = _clean_metrics(item.get("metrics"))
        notes = _clean_str(item.get("speaker_notes"), 800)
        slides.append(
            SlideSpec(
                layout=layout,
                title=title,
                subtitle=subtitle,
                bullets=bullets,
                left_title=_clean_str(item.get("left_title"), 60),
                right_title=_clean_str(item.get("right_title"), 60),
                left_bullets=left,
                right_bullets=right,
                metrics=metrics,
                speaker_notes=notes,
            )
        )

    if not slides:
        raise ValueError("未解析到有效 slides")

    if slides[0].layout != "title":
        slides.insert(
            0,
            SlideSpec(layout="title", title=deck_title, subtitle=deck_subtitle),
        )
    else:
        slides[0].title = deck_title or slides[0].title
        if deck_subtitle:
            slides[0].subtitle = deck_subtitle

    return DeckOutline(
        deck_title=deck_title,
        deck_subtitle=deck_subtitle,
        theme=theme,
        slides=slides,
    )


def parse_outline_json(text: str, max_slides: int) -> DeckOutline:
    raw = text.strip()
    if raw.startswith("```"):
        raw = re.sub(r"^```(?:json)?\s*", "", raw)
        raw = re.sub(r"\s*```$", "", raw)
    data = json.loads(raw)
    if not isinstance(data, dict):
        raise ValueError("outline 根节点必须为 JSON 对象")
    return parse_deck_outline(data, max_slides)
