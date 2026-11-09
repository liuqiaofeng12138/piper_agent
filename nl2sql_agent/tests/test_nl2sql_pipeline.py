from nl2sql_agent.knowledge.catalog import load_catalog
from nl2sql_agent.pipeline.params import extract_params_rule
from nl2sql_agent.pipeline.sql_render import render_template


def test_yield_trend_template_render():
    catalog = load_catalog()
    extracted = extract_params_rule(catalog, "查询近一个月 N28 车规 MCU 的良率")
    assert extracted.template_id == "tpl_yield_daily_trend"
    tpl = catalog.templates[extracted.template_id]
    sql = render_template(tpl, extracted.params)
    assert "fact_yield_daily" in sql
    assert "N28 车规 MCU" in sql
    assert "30" in sql


def test_wip_category():
    catalog = load_catalog()
    extracted = extract_params_rule(catalog, "N28 车规 MCU 在制品各工序有多少片")
    assert extracted.category_id == "cat_wip_stage"
    tpl = catalog.templates[extracted.template_id]
    sql = render_template(tpl, extracted.params)
    assert "fact_wip_lot" in sql
