from __future__ import annotations

from typing import Any

from enterprise_agent.config.loader import AgentConfig
from enterprise_agent.skills.base import Skill, TaskContext
from enterprise_agent.skills.loader import load_tools_schema
from enterprise_agent.skills.manifest import SkillManifest
from enterprise_agent.skills.manifest_skill import ManifestSkill
from enterprise_agent.tools.documents import (
    extract_layout_table,
    paddleocr_vl_parse,
    split_tables_from_plain,
)

_DEFAULT_MAX_PARSE_CHARS = 48000


class DocumentOCRSkill(ManifestSkill):
    def __init__(self, manifest: SkillManifest, max_parse_chars: int) -> None:
        super().__init__(manifest, load_tools_schema(manifest.skill_dir))
        self._max_parse_chars = max_parse_chars

    def execute_tool(self, name: str, arguments: dict[str, Any], ctx: TaskContext) -> str:
        if name == "paddleocr_vl_parse":
            fname = str(arguments.get("filename") or "").strip()
            for doc_name, data in ctx.documents:
                if doc_name == fname or not fname:
                    return paddleocr_vl_parse(doc_name, data, max_chars=self._max_parse_chars)
            names = ", ".join(n for n, _ in ctx.documents) or "（无）"
            return f"未找到文件 {fname}，当前上传: {names}"
        if name == "extract_layout_table":
            md = str(arguments.get("markdown_text") or "")
            out = extract_layout_table(md)
            if out.startswith("（未检测"):
                out = split_tables_from_plain(md)
            return out
        return f"未知工具: {name}"


def create_skill(manifest: SkillManifest, cfg: AgentConfig) -> Skill:
    _ = cfg
    return DocumentOCRSkill(manifest, _DEFAULT_MAX_PARSE_CHARS)
