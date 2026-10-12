from pathlib import Path

from word2ppt_agent.config.loader import Word2PPTConfig
from word2ppt_agent.ppt.builder import build_presentation, default_ppt_title


def test_build_presentation_minimal(tmp_path: Path) -> None:
    out = tmp_path / "out.pptx"
    result = build_presentation(
        [("demo.txt", "第一行\n第二行\n第三行")],
        title="测试",
        output_path=out,
        cfg=Word2PPTConfig(),
    )
    assert out.is_file()
    assert result.slide_count >= 2


def test_default_title() -> None:
    assert "报告" in default_ppt_title("季度报告", ["a.docx"])
    assert default_ppt_title("", ["readme.pdf"]) == "readme — 内容汇总"
