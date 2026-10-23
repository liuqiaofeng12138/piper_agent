from __future__ import annotations

from dataclasses import dataclass

from pptx.dml.color import RGBColor
from pptx.enum.shapes import MSO_SHAPE
from pptx.enum.text import MSO_ANCHOR, PP_ALIGN
from pptx.util import Inches, Pt


@dataclass(frozen=True)
class Theme:
    name: str
    background: RGBColor
    card_surface: RGBColor
    title_bar: RGBColor
    accent: RGBColor
    title_text: RGBColor
    body_text: RGBColor
    muted_text: RGBColor
    section_bg: RGBColor
    section_text: RGBColor


THEMES: dict[str, Theme] = {
    "corporate_blue": Theme(
        name="corporate_blue",
        background=RGBColor(0xEE, 0xF2, 0xF6),
        card_surface=RGBColor(0xFF, 0xFF, 0xFF),
        title_bar=RGBColor(0x1E, 0x3A, 0x5F),
        accent=RGBColor(0x25, 0x6E, 0xEB),
        title_text=RGBColor(0x1E, 0x3A, 0x5F),
        body_text=RGBColor(0x33, 0x41, 0x55),
        muted_text=RGBColor(0x64, 0x74, 0x8B),
        section_bg=RGBColor(0x1E, 0x3A, 0x5F),
        section_text=RGBColor(0xFF, 0xFF, 0xFF),
    ),
    "ocean": Theme(
        name="ocean",
        background=RGBColor(0xEC, 0xFD, 0xF5),
        card_surface=RGBColor(0xFF, 0xFF, 0xFF),
        title_bar=RGBColor(0x0F, 0x76, 0x6E),
        accent=RGBColor(0x14, 0xB8, 0xA6),
        title_text=RGBColor(0x0F, 0x76, 0x6E),
        body_text=RGBColor(0x13, 0x4E, 0x4A),
        muted_text=RGBColor(0x5E, 0xA8, 0xA0),
        section_bg=RGBColor(0x0D, 0x94, 0x88),
        section_text=RGBColor(0xFF, 0xFF, 0xFF),
    ),
    "slate": Theme(
        name="slate",
        background=RGBColor(0xF1, 0xF5, 0xF9),
        card_surface=RGBColor(0xFF, 0xFF, 0xFF),
        title_bar=RGBColor(0x33, 0x41, 0x55),
        accent=RGBColor(0x63, 0x66, 0xF1),
        title_text=RGBColor(0x1E, 0x29, 0x3B),
        body_text=RGBColor(0x33, 0x41, 0x55),
        muted_text=RGBColor(0x94, 0xA3, 0xB8),
        section_bg=RGBColor(0x47, 0x55, 0x69),
        section_text=RGBColor(0xFF, 0xFF, 0xFF),
    ),
    "warm": Theme(
        name="warm",
        background=RGBColor(0xFF, 0xF7, 0xED),
        card_surface=RGBColor(0xFF, 0xFF, 0xFF),
        title_bar=RGBColor(0x9A, 0x34, 0x12),
        accent=RGBColor(0xEA, 0x58, 0x0C),
        title_text=RGBColor(0x7C, 0x2D, 0x12),
        body_text=RGBColor(0x44, 0x2A, 0x1A),
        muted_text=RGBColor(0xA8, 0x6E, 0x4B),
        section_bg=RGBColor(0xC2, 0x41, 0x0C),
        section_text=RGBColor(0xFF, 0xFF, 0xFF),
    ),
}


def get_theme(name: str) -> Theme:
    return THEMES.get(name) or THEMES["corporate_blue"]


def set_slide_background(slide, color: RGBColor) -> None:
    fill = slide.background.fill
    fill.solid()
    fill.fore_color.rgb = color


def add_textbox(
    slide,
    left: float,
    top: float,
    width: float,
    height: float,
    text: str,
    font_size: int,
    color: RGBColor,
    bold: bool = False,
    align: PP_ALIGN = PP_ALIGN.LEFT,
) -> None:
    box = slide.shapes.add_textbox(Inches(left), Inches(top), Inches(width), Inches(height))
    tf = box.text_frame
    tf.word_wrap = True
    tf.vertical_anchor = MSO_ANCHOR.TOP
    p = tf.paragraphs[0]
    p.text = text
    p.alignment = align
    p.font.size = Pt(font_size)
    p.font.bold = bold
    p.font.color.rgb = color


def add_rounded_card(slide, left: float, top: float, width: float, height: float, theme: Theme) -> None:
    card = slide.shapes.add_shape(
        MSO_SHAPE.ROUNDED_RECTANGLE,
        Inches(left),
        Inches(top),
        Inches(width),
        Inches(height),
    )
    card.fill.solid()
    card.fill.fore_color.rgb = theme.card_surface
    card.line.color.rgb = RGBColor(0xE2, 0xE8, 0xF0)
    card.line.width = Pt(0.75)


def add_left_accent_bar(slide, theme: Theme, height: float = 5.2, top: float = 1.25) -> None:
    bar = slide.shapes.add_shape(
        MSO_SHAPE.RECTANGLE, Inches(0.45), Inches(top), Inches(0.12), Inches(height)
    )
    bar.fill.solid()
    bar.fill.fore_color.rgb = theme.accent
    bar.line.fill.background()


def add_slide_header(slide, title: str, theme: Theme) -> None:
    header = slide.shapes.add_shape(
        MSO_SHAPE.RECTANGLE, Inches(0), Inches(0), Inches(10), Inches(1.12)
    )
    header.fill.solid()
    header.fill.fore_color.rgb = theme.title_bar
    header.line.fill.background()
    accent = slide.shapes.add_shape(
        MSO_SHAPE.RECTANGLE, Inches(0), Inches(1.12), Inches(10), Inches(0.06)
    )
    accent.fill.solid()
    accent.fill.fore_color.rgb = theme.accent
    accent.line.fill.background()
    add_textbox(slide, 0.55, 0.24, 8.8, 0.75, title, 26, theme.section_text, bold=True)


def add_slide_footer(slide, deck_title: str, page_num: int, theme: Theme) -> None:
    foot = slide.shapes.add_shape(
        MSO_SHAPE.RECTANGLE, Inches(0), Inches(7.15), Inches(10), Inches(0.35)
    )
    foot.fill.solid()
    foot.fill.fore_color.rgb = theme.title_bar
    foot.line.fill.background()
    add_textbox(slide, 0.5, 7.18, 5.5, 0.3, deck_title[:40], 9, theme.section_text)
    add_textbox(
        slide,
        8.8,
        7.18,
        1.0,
        0.3,
        str(page_num),
        9,
        theme.section_text,
        align=PP_ALIGN.RIGHT,
    )


def fill_bullets(
    text_frame,
    bullets: list[str],
    color: RGBColor,
    size: int = 16,
    accent: RGBColor | None = None,
    bullet_char: str = "●",
) -> None:
    text_frame.clear()
    accent = accent or color
    for i, item in enumerate(bullets):
        p = text_frame.paragraphs[0] if i == 0 else text_frame.add_paragraph()
        p.text = f"{bullet_char}  {item}"
        p.level = 0
        p.font.size = Pt(size)
        p.font.color.rgb = color
        p.space_after = Pt(10)
        p.line_spacing = 1.15


def fill_numbered_list(text_frame, items: list[str], color: RGBColor, size: int = 17) -> None:
    text_frame.clear()
    for i, item in enumerate(items, start=1):
        p = text_frame.paragraphs[0] if i == 1 else text_frame.add_paragraph()
        p.text = f"{i:02d}  {item}"
        p.font.size = Pt(size)
        p.font.color.rgb = color
        p.space_after = Pt(12)
        p.font.bold = i == 1
