"""文档解析与版面表格提取（MarkItDown 实现，工具名对齐 OCR Skill 契约）。"""

from __future__ import annotations

import re
import tempfile
from pathlib import Path

ALLOWED_SUFFIXES = {".pdf", ".doc", ".docx", ".docm", ".png", ".jpg", ".jpeg", ".webp"}


def supported_filename(name: str) -> bool:
    return Path(name).suffix.lower() in ALLOWED_SUFFIXES


def paddleocr_vl_parse(filename: str, data: bytes, max_chars: int = 48000) -> str:
    """将 PDF/图片/Word 转为 Markdown 结构化文本（生产环境可替换为 PaddleOCR-VL）。"""
    suffix = Path(filename).suffix.lower()
    if suffix not in ALLOWED_SUFFIXES:
        raise ValueError(f"不支持的文件类型: {suffix}")
    if not data:
        raise ValueError(f"文件为空: {filename}")

    try:
        from markitdown import MarkItDown
    except ImportError as e:
        raise RuntimeError("请安装 markitdown: pip install -e enterprise_agent") from e

    converter = MarkItDown()
    with tempfile.NamedTemporaryFile(suffix=suffix, delete=False) as tmp:
        tmp.write(data)
        tmp_path = tmp.name
    try:
        result = converter.convert(tmp_path)
        text = (result.text_content or "").strip()
    finally:
        Path(tmp_path).unlink(missing_ok=True)

    if not text:
        raise ValueError(f"未能从 {filename} 提取到文字")
    if len(text) > max_chars:
        text = text[:max_chars] + "\n\n…（内容已截断）"
    return f"# 文档: {filename}\n\n{text}"


def extract_layout_table(markdown_text: str, max_tables: int = 8) -> str:
    """从 Markdown 中抽取管道表/简单表格块并对齐输出。"""
    lines = markdown_text.splitlines()
    blocks: list[list[str]] = []
    current: list[str] = []
    for line in lines:
        if "|" in line and line.strip().startswith("|"):
            current.append(line)
        else:
            if len(current) >= 2:
                blocks.append(current)
            current = []
    if len(current) >= 2:
        blocks.append(current)

    if not blocks:
        return "（未检测到 Markdown 表格）"

    out: list[str] = []
    for i, block in enumerate(blocks[:max_tables], start=1):
        out.append(f"## 表格 {i}\n" + "\n".join(block))
    if len(blocks) > max_tables:
        out.append(f"\n（另有 {len(blocks) - max_tables} 个表格未展示）")
    return "\n\n".join(out)


def split_tables_from_plain(text: str) -> str:
    """兜底：连续数字/制表符行视为表格候选。"""
    pattern = re.compile(r"(?:^[\d\.\t|].+\n){3,}", re.MULTILINE)
    matches = pattern.findall(text)
    if not matches:
        return "（未检测到表格样式文本）"
    return "\n\n---\n\n".join(m[:4000] for m in matches[:5])
