from enterprise_agent.config.loader import AgentConfig
from enterprise_agent.skills.registry import SkillRegistry


def test_rule_route_composite_task() -> None:
    reg = SkillRegistry(AgentConfig())
    ids, reason = reg.route_skills(
        None,
        "gpt-4o",
        "查上个月华东区各产品销售额，并分析竞品网上定价策略",
        document_count=0,
    )
    assert "data_sql" in ids
    assert "web_research" in ids
    assert reason.startswith("rule")
