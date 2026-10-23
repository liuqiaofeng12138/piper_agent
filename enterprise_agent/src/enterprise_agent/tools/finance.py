"""财务建模：Excel 读取与受限 Python 解释器。"""

from __future__ import annotations

import io
import textwrap
from typing import Any

import pandas as pd


def read_excel_pandas(filename: str, data: bytes, sheet_name: str | None = None) -> str:
    if not data:
        raise ValueError(f"文件为空: {filename}")
    bio = io.BytesIO(data)
    try:
        if sheet_name:
            df = pd.read_excel(bio, sheet_name=sheet_name, engine="openpyxl")
        else:
            df = pd.read_excel(bio, engine="openpyxl")
    except Exception as e:  # noqa: BLE001
        raise ValueError(f"无法读取 Excel {filename}: {e}") from e
    if isinstance(df, dict):
        parts = []
        for name, frame in list(df.items())[:5]:
            parts.append(f"### Sheet: {name}\n{frame.head(30).to_markdown(index=False)}")
        return "\n\n".join(parts)
    preview = df.head(40)
    meta = f"行数≈{len(df)}，列: {', '.join(str(c) for c in df.columns)}"
    return f"{meta}\n\n{preview.to_markdown(index=False)}"


def python_code_interpreter(
    code: str,
    frames: dict[str, pd.DataFrame] | None = None,
    timeout_hint: str = "",
) -> str:
    """在受限命名空间中执行分析代码（无 import、无 open）。返回 stdout 或 result 变量。"""
    code = textwrap.dedent(code).strip()
    if not code:
        return "（代码为空）"
    banned = ("import ", "__", "open(", "exec(", "eval(", "compile(", "os.", "sys.", "subprocess")
    lower = code.lower()
    for token in banned:
        if token in lower:
            raise ValueError(f"代码包含禁止片段: {token}")

    import math

    namespace: dict[str, Any] = {
        "pd": pd,
        "math": math,
        "frames": frames or {},
    }
    stdout: list[str] = []

    def _print(*args: Any, **kwargs: Any) -> None:
        stdout.append(" ".join(str(a) for a in args))

    namespace["print"] = _print
    try:
        exec(code, {"__builtins__": {}}, namespace)  # noqa: S102
    except Exception as e:  # noqa: BLE001
        return f"执行错误: {e}"

    if stdout:
        return "\n".join(stdout)
    if "result" in namespace:
        val = namespace["result"]
        if isinstance(val, pd.DataFrame):
            return val.head(50).to_markdown(index=False)
        return str(val)
    return "（代码已执行，无 print 输出且未设置 result 变量）"
