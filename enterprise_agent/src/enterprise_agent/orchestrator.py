from __future__ import annotations

import logging
from collections.abc import Callable, Iterator
from typing import Any

from enterprise_agent.config.loader import AgentConfig, resolve_sql_db_path
from enterprise_agent.skills.base import TaskContext
from enterprise_agent.skills.registry import SkillRegistry
from enterprise_agent.tools.sql import ensure_demo_database

logger = logging.getLogger(__name__)

SYNTHESIS_SYSTEM = (
    "你是「全能企业运营智囊」，负责将各专业技能包的输出融合为一份 Markdown 业务分析报告。\n"
    "结构建议：执行摘要、数据洞察、市场/竞品观察、风险与建议、附录（关键数字表）。\n"
    "仅依据技能输出作答，缺失部分要说明。使用简体中文，条理清晰。"
)


ProgressEvent = tuple[str, str, str]  # phase, tool_name, message


def run_enterprise_pipeline(
    cfg: AgentConfig,
    llm: Any,
    model: str,
    user_message: str,
    documents: list[tuple[str, bytes]],
    on_progress: Callable[[ProgressEvent], None] | None = None,
) -> tuple[list[str], str, dict[str, str]]:
    """分诊 → 按序执行 Skills → 返回 (skill_ids, route_reason, skill_outputs)。"""
    ensure_demo_database(resolve_sql_db_path(cfg))
    registry = SkillRegistry(cfg)
    skill_ids, route_reason = registry.route_skills(
        llm, model, user_message, document_count=len(documents)
    )
    if on_progress:
        names = ", ".join(registry.get(s).definition.display_name for s in skill_ids)
        on_progress(("route", "skill_registry", f"已加载技能: {names}（{route_reason}）"))

    ctx = TaskContext(user_message=user_message, documents=documents)
    ent = cfg.enterprise_ops

    for sid in skill_ids:
        skill = registry.get(sid)
        if on_progress:
            on_progress(
                ("skill", sid, f"开始执行 {skill.definition.display_name}"),
            )

        def _skill_progress(skill_id: str, msg: str) -> None:
            if on_progress:
                on_progress(("skill", skill_id, msg))

        output = skill.run(
            llm=llm,
            model=model,
            ctx=ctx,
            max_steps=ent.max_skill_steps,
            on_progress=_skill_progress,
        )
        ctx.skill_outputs[sid] = output
        if on_progress:
            on_progress(("skill", sid, "本技能执行完成"))

    return skill_ids, route_reason, ctx.skill_outputs


def build_synthesis_messages(
    user_message: str,
    skill_outputs: dict[str, str],
    route_reason: str,
) -> list[dict[str, str]]:
    blocks: list[str] = [f"路由说明: {route_reason}", "", f"用户原始需求:\n{user_message}", ""]
    for sid, text in skill_outputs.items():
        blocks.append(f"### 技能输出 `{sid}`\n\n{text}\n")
    return [
        {"role": "system", "content": SYNTHESIS_SYSTEM},
        {"role": "user", "content": "\n".join(blocks)},
    ]


def stream_synthesis(
    llm: Any,
    model: str,
    messages: list[dict[str, str]],
) -> Iterator[str]:
    stream = llm.chat.completions.create(model=model, messages=messages, stream=True)
    for chunk in stream:
        delta = chunk.choices[0].delta.content if chunk.choices else None
        if delta:
            yield delta
