import json
from unittest.mock import MagicMock

from web_crawler_agent.config.loader import AgentConfig
from web_crawler_agent.harness.session import SessionState
from web_crawler_agent.rag.template_index import TemplateIndex
from web_crawler_agent.tools.handlers import ToolHandlers


def test_load_shared_example_template(tmp_path):
    ex = tmp_path / "http_jsonpath.json"
    ex.write_text(
        '{"id":"http_jsonpath","name":"example","fields":["a"]}',
        encoding="utf-8",
    )
    client = MagicMock()
    session = SessionState()
    cfg = AgentConfig()
    idx = TemplateIndex(client, examples_dir=tmp_path)
    h = ToolHandlers(client, session, cfg, idx)
    out = json.loads(h.load_shared_example_template({"example_id": "http_jsonpath"}))
    assert out["use_verbatim"] is True
    assert json.loads(out["template_json"])["fields"] == ["a"]
    assert session.pending_template["id"] == "http_jsonpath"


def test_search_prefers_shared_over_meta(tmp_path):
    ex = tmp_path / "http_jsonpath.json"
    ex.write_text('{"id":"http_jsonpath","name":"from_file"}', encoding="utf-8")
    client = MagicMock()
    tpl = MagicMock()
    tpl.id = "http_jsonpath"
    tpl.name = "from_meta"
    tpl.domain = ""
    tpl.json_payload = b'{"id":"http_jsonpath","name":"from_meta"}'
    resp = MagicMock()
    resp.templates = [tpl]
    client.list_templates.return_value = resp
    idx = TemplateIndex(client, examples_dir=tmp_path)
    idx.refresh()
    hits = idx.search("shared http_jsonpath")
    assert hits[0].source.startswith("example:")
    assert hits[0].template_json is not None


def test_template_index_search_examples(tmp_path):
    ex = tmp_path / "demo.json"
    ex.write_text(
        '{"name":"news_list","domain":"news.com","builder":{"type":"Http","url_tpl":"https://news.com"}}',
        encoding="utf-8",
    )
    client = MagicMock()
    client.list_templates.side_effect = Exception("offline")
    idx = TemplateIndex(client, examples_dir=tmp_path)
    idx.refresh()
    hits = idx.search("news list http")
    assert hits and hits[0].name == "news_list"


def test_param_filler_extract_url():
    client = MagicMock()
    session = SessionState()
    cfg = AgentConfig()
    h = ToolHandlers(client, session, cfg, TemplateIndex(client))
    out = json.loads(
        h.param_filler_suggest(
            {"user_text": "请抓取 https://example.com/page 的标题", "template_json": ""}
        )
    )
    assert out["vars"].get("url") == "https://example.com/page"
