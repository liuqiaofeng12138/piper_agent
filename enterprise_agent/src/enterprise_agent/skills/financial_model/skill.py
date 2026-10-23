from __future__ import annotations

import io
from typing import Any

import pandas as pd

from enterprise_agent.config.loader import AgentConfig
from enterprise_agent.skills.base import Skill, TaskContext
from enterprise_agent.skills.loader import load_tools_schema
from enterprise_agent.skills.manifest import SkillManifest
from enterprise_agent.skills.manifest_skill import ManifestSkill
from enterprise_agent.tools.finance import python_code_interpreter, read_excel_pandas


class FinancialModelSkill(ManifestSkill):
    def __init__(self, manifest: SkillManifest) -> None:
        super().__init__(manifest, load_tools_schema(manifest.skill_dir))
        self._frames: dict[str, pd.DataFrame] = {}

    def _load_frames(self, ctx: TaskContext) -> None:
        if self._frames:
            return
        for name, data in ctx.documents:
            if not name.lower().endswith((".xlsx", ".xls", ".csv")):
                continue
            if name.lower().endswith(".csv"):
                self._frames[name] = pd.read_csv(io.BytesIO(data))
            else:
                self._frames[name] = pd.read_excel(io.BytesIO(data), engine="openpyxl")

    def execute_tool(self, name: str, arguments: dict[str, Any], ctx: TaskContext) -> str:
        self._load_frames(ctx)
        if name == "read_excel_pandas":
            fname = str(arguments.get("filename") or "").strip()
            sheet = arguments.get("sheet_name")
            for doc_name, data in ctx.documents:
                if doc_name == fname or not fname:
                    return read_excel_pandas(
                        doc_name,
                        data,
                        sheet_name=str(sheet) if sheet else None,
                    )
            return "未找到可读的 Excel 文件"
        if name == "python_code_interpreter":
            code = str(arguments.get("code") or "")
            return python_code_interpreter(code, frames=self._frames)
        return f"未知工具: {name}"


def create_skill(manifest: SkillManifest, cfg: AgentConfig) -> Skill:
    _ = cfg
    return FinancialModelSkill(manifest)
