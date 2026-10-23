from __future__ import annotations

import re
from dataclasses import dataclass
from pathlib import Path

from pptx import Presentation
from pptx.enum.shapes import MSO_SHAPE
from pptx.enum.text import PP_ALIGN
from pptx.util import Inches, Pt

from word2ppt_agent.config.loader import Word2PPTConfig
from word2ppt_agent.ppt.outline import DeckOutline, MetricItem, SlideSpec
from word2ppt_agent.ppt.themes import (
    add_left_accent_bar,
    add_rounded_card,
    add_slide_footer,
    add_slide_header,
    add_textbox,
    fill_bullets,
    fill_numbered_list,
    get_theme,
    set_slide_background,
)


@dataclass
class BuildResult:
    slide_count: int
    output_path: Path
    used_llm_outline: bool = False


def _normalize_lines(text: str) -> list[str]:
    lines: list[str] = []
    for raw in text.splitlines():
        line = re.sub(r"\s+", " ", raw).strip()
        if line:
            lines.append(line)
    if not lines:
        chunks = re.split(r"\n{2,}", text.strip())
        lines = [c.strip() for c in chunks if c.strip()]
    return lines


def _chunk_bullets(lines: list[str], cfg: Word2PPTConfig) -> list[list[str]]:
    bullets: list[str] = []
    for line in lines:
        if len(line) <= cfg.max_chars_per_bullet:
            bullets.append(line)
            continue
        start = 0
        while start < len(line):
            bullets.append(line[start : start + cfg.max_chars_per_bullet])
            start += cfg.max_chars_per_bullet

    slides: list[list[str]] = []
    buf: list[str] = []
    for b in bullets:
        buf.append(b)
        if len(buf) >= cfg.max_bullets_per_slide:
            slides.append(buf)
            buf = []
    if buf:
        slides.append(buf)
    return slides


def build_from_outline(outline: DeckOutline, output_path: Path) -> BuildResult:
    theme = get_theme(outline.theme)
    prs = Presentation()
    prs.core_properties.author = "word2ppt-agent"
    prs.core_properties.title = outline.deck_title

    page = 0
    section_idx = 0
    for spec in outline.slides:
        page += 1
        if spec.layout == "title":
            _render_title_slide(prs, spec, theme, outline.deck_title)
        elif spec.layout == "section":
            section_idx += 1
            _render_section_slide(prs, spec, theme, section_idx)
            add_slide_footer(prs.slides[-1], outline.deck_title, page, theme)
        elif spec.layout == "agenda":
            _render_agenda_slide(prs, spec, theme, outline.deck_title, page)
        elif spec.layout == "two_column":
            _render_two_column_slide(prs, spec, theme, outline.deck_title, page)
        elif spec.layout == "closing":
            _render_closing_slide(prs, spec, theme)
        elif spec.layout == "highlight":
            _render_highlight_slide(prs, spec, theme, outline.deck_title, page)
        elif spec.layout == "metrics":
            _render_metrics_slide(prs, spec, theme, outline.deck_title, page)
        else:
            _render_content_slide(prs, spec, theme, outline.deck_title, page)

        if spec.speaker_notes:
            notes = prs.slides[-1].notes_slide.notes_text_frame
            notes.text = spec.speaker_notes

    output_path.parent.mkdir(parents=True, exist_ok=True)
    prs.save(str(output_path))
    return BuildResult(slide_count=page, output_path=output_path, used_llm_outline=True)


def _render_title_slide(prs: Presentation, spec: SlideSpec, theme, deck_title: str) -> None:
    slide = prs.slides.add_slide(prs.slide_layouts[6])
    set_slide_background(slide, theme.title_bar)
    deco = slide.shapes.add_shape(
        MSO_SHAPE.OVAL, Inches(7.2), Inches(-0.8), Inches(3.5), Inches(3.5)
    )
    deco.fill.solid()
    deco.fill.fore_color.rgb = theme.accent
    deco.line.fill.background()
    add_textbox(slide, 0.75, 1.9, 8.5, 1.4, spec.title, 42, theme.section_text, bold=True)
    if spec.subtitle:
        add_textbox(slide, 0.75, 3.35, 8.2, 1.1, spec.subtitle, 20, theme.section_text)
    bar = slide.shapes.add_shape(
        MSO_SHAPE.RECTANGLE, Inches(0.75), Inches(4.85), Inches(2.4), Inches(0.1)
    )
    bar.fill.solid()
    bar.fill.fore_color.rgb = theme.accent
    bar.line.fill.background()
    add_textbox(slide, 0.75, 6.55, 5.0, 0.35, deck_title[:36], 10, theme.muted_text)


def _render_section_slide(prs: Presentation, spec: SlideSpec, theme, section_idx: int) -> None:
    slide = prs.slides.add_slide(prs.slide_layouts[6])
    set_slide_background(slide, theme.section_bg)
    num = slide.shapes.add_shape(
        MSO_SHAPE.OVAL, Inches(0.85), Inches(2.35), Inches(1.1), Inches(1.1)
    )
    num.fill.solid()
    num.fill.fore_color.rgb = theme.accent
    num.line.fill.background()
    add_textbox(
        slide,
        0.85,
        2.55,
        1.1,
        0.7,
        f"{section_idx:02d}",
        28,
        theme.section_text,
        bold=True,
        align=PP_ALIGN.CENTER,
    )
    add_textbox(slide, 2.2, 2.45, 7.0, 1.2, spec.title, 34, theme.section_text, bold=True)
    if spec.subtitle:
        add_textbox(slide, 2.2, 3.55, 7.0, 0.9, spec.subtitle, 18, theme.section_text)


def _render_agenda_slide(
    prs: Presentation, spec: SlideSpec, theme, deck_title: str, page: int
) -> None:
    slide = prs.slides.add_slide(prs.slide_layouts[6])
    set_slide_background(slide, theme.background)
    add_slide_header(slide, spec.title, theme)
    add_left_accent_bar(slide, theme, height=4.8, top=1.45)
    add_rounded_card(slide, 0.75, 1.4, 8.5, 5.35, theme)
    body = slide.shapes.add_textbox(Inches(1.05), Inches(1.75), Inches(7.9), Inches(4.8))
    items = spec.bullets or ["（暂无目录）"]
    fill_numbered_list(body.text_frame, items, theme.body_text, size=18)
    add_slide_footer(slide, deck_title, page, theme)


def _render_content_slide(
    prs: Presentation, spec: SlideSpec, theme, deck_title: str, page: int
) -> None:
    slide = prs.slides.add_slide(prs.slide_layouts[6])
    set_slide_background(slide, theme.background)
    add_slide_header(slide, spec.title, theme)
    add_left_accent_bar(slide, theme)
    add_rounded_card(slide, 0.72, 1.38, 8.55, 5.55, theme)
    if spec.subtitle:
        add_textbox(slide, 1.0, 1.55, 8.0, 0.55, spec.subtitle, 14, theme.accent, bold=True)
        body_top = 2.15
    else:
        body_top = 1.65
    body = slide.shapes.add_textbox(Inches(1.0), Inches(body_top), Inches(7.95), Inches(4.5))
    bullets = spec.bullets or ["（本页暂无要点）"]
    fill_bullets(body.text_frame, bullets, theme.body_text, size=17, accent=theme.title_text)
    add_slide_footer(slide, deck_title, page, theme)


def _render_highlight_slide(
    prs: Presentation, spec: SlideSpec, theme, deck_title: str, page: int
) -> None:
    slide = prs.slides.add_slide(prs.slide_layouts[6])
    set_slide_background(slide, theme.background)
    add_slide_header(slide, spec.title, theme)
    quote_card = slide.shapes.add_shape(
        MSO_SHAPE.ROUNDED_RECTANGLE, Inches(0.7), Inches(1.45), Inches(8.6), Inches(2.35)
    )
    quote_card.fill.solid()
    quote_card.fill.fore_color.rgb = theme.title_bar
    quote_card.line.fill.background()
    quote = spec.subtitle or (spec.bullets[0] if spec.bullets else "核心观点")
    add_textbox(slide, 1.0, 1.85, 8.0, 1.6, f"「{quote}」", 22, theme.section_text, bold=True)
    if spec.bullets:
        add_rounded_card(slide, 0.72, 4.05, 8.55, 2.55, theme)
        body = slide.shapes.add_textbox(Inches(1.0), Inches(4.25), Inches(7.9), Inches(2.2))
        fill_bullets(body.text_frame, spec.bullets, theme.body_text, size=16)
    add_slide_footer(slide, deck_title, page, theme)


def _render_metrics_slide(
    prs: Presentation, spec: SlideSpec, theme, deck_title: str, page: int
) -> None:
    slide = prs.slides.add_slide(prs.slide_layouts[6])
    set_slide_background(slide, theme.background)
    add_slide_header(slide, spec.title, theme)
    metrics = spec.metrics or [
        MetricItem("要点", v) for v in (spec.bullets[:3] if spec.bullets else ["—"])
    ]
    cols = min(len(metrics), 4)
    card_w = 8.6 / cols - 0.15
    for i, m in enumerate(metrics[:4]):
        left = 0.7 + i * (card_w + 0.15)
        add_rounded_card(slide, left, 1.55, card_w, 2.0, theme)
        accent_bar = slide.shapes.add_shape(
            MSO_SHAPE.RECTANGLE, Inches(left), Inches(1.55), Inches(card_w), Inches(0.12)
        )
        accent_bar.fill.solid()
        accent_bar.fill.fore_color.rgb = theme.accent
        accent_bar.line.fill.background()
        add_textbox(slide, left + 0.15, 1.85, card_w - 0.3, 0.9, m.value, 26, theme.title_text, bold=True)
        add_textbox(slide, left + 0.15, 2.75, card_w - 0.3, 0.6, m.label, 12, theme.muted_text)
    if spec.bullets:
        add_rounded_card(slide, 0.72, 3.85, 8.55, 2.75, theme)
        body = slide.shapes.add_textbox(Inches(1.0), Inches(4.05), Inches(7.9), Inches(2.4))
        fill_bullets(body.text_frame, spec.bullets, theme.body_text, size=15)
    add_slide_footer(slide, deck_title, page, theme)


def _render_two_column_slide(
    prs: Presentation, spec: SlideSpec, theme, deck_title: str, page: int
) -> None:
    slide = prs.slides.add_slide(prs.slide_layouts[6])
    set_slide_background(slide, theme.background)
    add_slide_header(slide, spec.title, theme)
    add_rounded_card(slide, 0.55, 1.38, 4.25, 5.55, theme)
    add_rounded_card(slide, 5.2, 1.38, 4.25, 5.55, theme)
    left_title = spec.left_title or "要点 A"
    right_title = spec.right_title or "要点 B"
    add_textbox(slide, 0.75, 1.52, 3.9, 0.45, left_title, 14, theme.accent, bold=True)
    add_textbox(slide, 5.4, 1.52, 3.9, 0.45, right_title, 14, theme.accent, bold=True)
    left = slide.shapes.add_textbox(Inches(0.75), Inches(2.05), Inches(3.85), Inches(4.6))
    right = slide.shapes.add_textbox(Inches(5.4), Inches(2.05), Inches(3.85), Inches(4.6))
    fill_bullets(left.text_frame, spec.left_bullets or ["—"], theme.body_text, size=15)
    fill_bullets(right.text_frame, spec.right_bullets or ["—"], theme.body_text, size=15)
    add_slide_footer(slide, deck_title, page, theme)


def _render_closing_slide(prs: Presentation, spec: SlideSpec, theme) -> None:
    slide = prs.slides.add_slide(prs.slide_layouts[6])
    set_slide_background(slide, theme.title_bar)
    add_textbox(slide, 0.9, 1.5, 8.2, 1.0, spec.title, 36, theme.section_text, bold=True)
    if spec.bullets:
        add_rounded_card(slide, 1.0, 2.75, 8.0, 3.8, theme)
        body = slide.shapes.add_textbox(Inches(1.35), Inches(3.05), Inches(7.3), Inches(3.2))
        fill_bullets(body.text_frame, spec.bullets, theme.title_text, size=18, accent=theme.accent)


def build_presentation(
    documents: list[tuple[str, str]],
    title: str,
    output_path: Path,
    cfg: Word2PPTConfig,
) -> BuildResult:
    """规则排版兜底（大模型不可用或未启用时使用）。"""
    prs = Presentation()
    theme = get_theme("corporate_blue")
    spec = SlideSpec(layout="title", title=title or "文档内容汇总", subtitle="")
    names = "、".join(n for n, _ in documents)
    if names:
        spec.subtitle = f"共 {len(documents)} 个文件：{names}"
    _render_title_slide(prs, spec, theme, title)

    page = 1
    for filename, text in documents:
        if page >= cfg.max_slides_per_doc * len(documents) + 2:
            break
        lines = _normalize_lines(text)
        bullet_groups = _chunk_bullets(lines, cfg)
        stem = Path(filename).stem
        if not bullet_groups:
            _render_content_slide(
                prs,
                SlideSpec(layout="content", title=Path(filename).name, bullets=["（未能提取到可展示的文本要点）"]),
                theme,
                title,
                page,
            )
            page += 1
            continue
        for idx, group in enumerate(bullet_groups[: cfg.max_slides_per_doc]):
            page_title = stem if idx == 0 else f"{stem}（续 {idx}）"
            _render_content_slide(
                prs,
                SlideSpec(layout="content", title=page_title, bullets=group),
                theme,
                title,
                page,
            )
            page += 1

    prs.core_properties.author = "word2ppt-agent"
    prs.core_properties.title = title
    output_path.parent.mkdir(parents=True, exist_ok=True)
    prs.save(str(output_path))
    return BuildResult(slide_count=page, output_path=output_path, used_llm_outline=False)


def default_ppt_title(user_message: str, doc_names: list[str]) -> str:
    msg = (user_message or "").strip()
    if msg and len(msg) <= 60 and "附件" not in msg:
        return msg
    if len(doc_names) == 1:
        return f"{Path(doc_names[0]).stem} — 内容汇总"
    return "多文档内容汇总"
