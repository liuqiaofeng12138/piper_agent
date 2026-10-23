from __future__ import annotations

import json
import re

from enterprise_agent.config.loader import AgentConfig
from enterprise_agent.skills.base import Skill
from enterprise_agent.skills.loader import build_registry_skills, load_all_manifests


class SkillRegistry:
    def __init__(self, cfg: AgentConfig) -> None:
        self._manifests = load_all_manifests()
        self._skill_ids = tuple(m.skill_id for m in self._manifests)
        self._rule_hints = {m.skill_id: list(m.route_keywords) for m in self._manifests}
        self._skills = build_registry_skills(cfg)

    def get(self, skill_id: str) -> Skill:
        if skill_id not in self._skills:
            raise KeyError(skill_id)
        return self._skills[skill_id]

    def all_definitions(self) -> list[dict[str, str]]:
        return [
            {
                "id": s.definition.skill_id,
                "name": s.definition.display_name,
                "description": s.definition.description,
            }
            for s in self._skills.values()
        ]

    def route_skills(
        self,
        llm: object | None,
        model: str,
        user_message: str,
        document_count: int,
    ) -> tuple[list[str], str]:
        if document_count > 0:
            ids = _rule_route(
                user_message,
                document_count,
                force_docs=True,
                skill_ids=self._skill_ids,
                rule_hints=self._rule_hints,
            )
            return ids, "rule: 附带文档，加载文档/财务相关技能"

        if llm is not None:
            try:
                ids, reason = _llm_route(
                    llm,
                    model,
                    user_message,
                    self.all_definitions(),
                    skill_ids=self._skill_ids,
                )
                if ids:
                    return ids, f"llm: {reason}"
            except Exception:
                pass

        ids = _rule_route(
            user_message,
            document_count,
            force_docs=False,
            skill_ids=self._skill_ids,
            rule_hints=self._rule_hints,
        )
        return ids, "rule: 关键词分诊"


def _rule_route(
    message: str,
    document_count: int,
    force_docs: bool,
    skill_ids: tuple[str, ...],
    rule_hints: dict[str, list[str]],
) -> list[str]:
    lower = message.lower()
    scores: dict[str, int] = {k: 0 for k in skill_ids}
    for sid, words in rule_hints.items():
        for w in words:
            if w in lower or w in message:
                scores[sid] += 1

    if document_count > 0 or force_docs:
        if "document_ocr" in scores:
            scores["document_ocr"] += 2
        if any(ext in lower for ext in (".xlsx", ".xls", "excel", "报表")):
            if "financial_model" in scores:
                scores["financial_model"] += 2

    picked = [sid for sid, sc in scores.items() if sc > 0]
    if not picked:
        picked = ["data_sql"] if "data_sql" in skill_ids else [skill_ids[0]]
    order = [sid for sid in skill_ids if sid in picked]
    return order[:4]


def _llm_route(
    llm: object,
    model: str,
    user_message: str,
    definitions: list[dict[str, str]],
    skill_ids: tuple[str, ...],
) -> tuple[list[str], str]:
    catalog = json.dumps(definitions, ensure_ascii=False)
    system = (
        "你是企业运营 Agent 的技能路由中枢。根据用户任务从技能目录中选择 1~4 个 skill id，"
        '仅输出 JSON：{"skills":["data_sql",...],"reason":"..."}。'
        f"可选 id: {', '.join(skill_ids)}。"
    )
    user = f"技能目录:\n{catalog}\n\n用户任务:\n{user_message}"
    resp = llm.chat.completions.create(
        model=model,
        messages=[{"role": "system", "content": system}, {"role": "user", "content": user}],
        temperature=0,
    )
    text = (resp.choices[0].message.content or "").strip()
    m = re.search(r"\{[\s\S]*\}", text)
    if not m:
        return [], ""
    data = json.loads(m.group())
    skills = [str(s) for s in data.get("skills") or [] if str(s) in skill_ids]
    reason = str(data.get("reason") or "")
    return skills, reason
