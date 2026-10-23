import json
from pathlib import Path

from word2ppt_agent.ppt.builder import build_from_outline
from word2ppt_agent.ppt.outline import DeckOutline, SlideSpec, parse_outline_json


def test_parse_outline_json() -> None:
    raw = {
        "deck_title": "测试演示",
        "deck_subtitle": "副标题",
        "theme": "ocean",
        "slides": [
            {"layout": "title", "title": "测试演示", "subtitle": "副标题"},
            {"layout": "section", "title": "第一章"},
            {
                "layout": "content",
                "title": "要点",
                "bullets": ["一", "二"],
                "speaker_notes": "讲慢一点",
            },
            {"layout": "closing", "title": "谢谢", "bullets": ["Q&A"]},
        ],
    }
    outline = parse_outline_json(json.dumps(raw), max_slides=20)
    assert outline.theme == "ocean"
    assert len(outline.slides) >= 4


def test_build_from_outline(tmp_path: Path) -> None:
    outline = DeckOutline(
        deck_title="Demo",
        deck_subtitle="Sub",
        theme="slate",
        slides=[
            SlideSpec(layout="title", title="Demo", subtitle="Sub"),
            SlideSpec(layout="content", title="Page", bullets=["a", "b"]),
        ],
    )
    out = tmp_path / "deck.pptx"
    result = build_from_outline(outline, out)
    assert out.is_file()
    assert result.slide_count == 2
    assert result.used_llm_outline
