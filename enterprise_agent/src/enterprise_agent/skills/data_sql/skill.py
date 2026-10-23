from __future__ import annotations

from typing import Any

from enterprise_agent.config.loader import AgentConfig, resolve_sql_db_path
from enterprise_agent.skills.loader import load_tools_schema
from enterprise_agent.skills.manifest import SkillManifest
from enterprise_agent.skills.manifest_skill import ManifestSkill
from enterprise_agent.skills.base import Skill, TaskContext
from enterprise_agent.tools.sql import get_schema_metadata, run_sql_query


class DataSQLSkill(ManifestSkill):
    def __init__(self, manifest: SkillManifest, db_path) -> None:
        super().__init__(manifest, load_tools_schema(manifest.skill_dir))
        self._db_path = db_path

    def execute_tool(self, name: str, arguments: dict[str, Any], ctx: TaskContext) -> str:
        if name == "get_schema_metadata":
            return get_schema_metadata(self._db_path)
        if name == "run_sql_query":
            return run_sql_query(self._db_path, str(arguments.get("query") or ""))
        return f"未知工具: {name}"


def create_skill(manifest: SkillManifest, cfg: AgentConfig) -> Skill:
    return DataSQLSkill(manifest, resolve_sql_db_path(cfg))
