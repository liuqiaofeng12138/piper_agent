from enterprise_agent.skills.loader import discover_skill_dirs, load_manifest, load_tools_schema


def test_discover_four_standard_skills() -> None:
    dirs = discover_skill_dirs()
    ids = {load_manifest(d).skill_id for d in dirs}
    assert ids == {"data_sql", "document_ocr", "financial_model", "web_research"}


def test_data_sql_tools_json() -> None:
    root = next(d for d in discover_skill_dirs() if d.name == "data_sql")
    tools = load_tools_schema(root)
    names = {t["function"]["name"] for t in tools}
    assert names == {"get_schema_metadata", "run_sql_query"}
