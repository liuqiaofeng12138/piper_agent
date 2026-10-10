from __future__ import annotations

import io
import tempfile
from pathlib import Path

ALLOWED_SUFFIXES = {".pdf", ".doc", ".docx", ".docm"}


def supported_filename(name: str) -> bool:
    return Path(name).suffix.lower() in ALLOWED_SUFFIXES


def extract_text(filename: str, data: bytes) -> str:
    suffix = Path(filename).suffix.lower()
    if suffix not in ALLOWED_SUFFIXES:
        raise ValueError(f"不支持的文件类型: {suffix}，仅支持 {', '.join(sorted(ALLOWED_SUFFIXES))}")
    if not data:
        raise ValueError(f"文件为空: {filename}")

    try:
        from markitdown import MarkItDown
    except ImportError as e:
        raise RuntimeError("请安装 markitdown: pip install -e '.[dev]'") from e

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
        raise ValueError(f"未能从 {filename} 提取到文字，请确认文件未损坏或尝试另存为 PDF/DOCX")
    return text
