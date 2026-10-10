import json
from unittest.mock import MagicMock

from web_crawler_agent.config.loader import AgentConfig, HarnessConfig
from web_crawler_agent.harness.session import SessionState
from web_crawler_agent.rag.template_index import TemplateIndex
from web_crawler_agent.templates.store import persist_validated_template, save_template_file


def test_save_template_file_writes_json_and_manifest(tmp_path):
    cfg = AgentConfig(config_path=tmp_path / "local.yaml")
    cfg.harness = HarnessConfig(saved_templates_dir=tmp_path / "saved")
    doc = {"id": "dog_api", "name": "dog", "builder": {"type": "Http", "url_tpl": "https://dog.ceo"}}
    path = save_template_file(cfg, "dog_api", doc)
    assert path is not None and path.is_file()
    loaded = json.loads(path.read_text(encoding="utf-8"))
    assert loaded["id"] == "dog_api"
    manifest = (tmp_path / "saved" / "_manifest.jsonl").read_text(encoding="utf-8")
    assert "dog_api" in manifest


def test_persist_calls_upsert_and_file(tmp_path):
    client = MagicMock()
    resp = MagicMock()
    resp.template_id = "dog_api"
    client.upsert_template.return_value = resp
    index = MagicMock()
    session = SessionState()
    cfg = AgentConfig(config_path=tmp_path / "local.yaml")
    cfg.harness = HarnessConfig(saved_templates_dir=tmp_path / "saved", auto_save_templates=True)
    doc = {"id": "dog_api", "name": "dog"}
    out = persist_validated_template(
        client=client,
        index=index,
        session=session,
        cfg=cfg,
        doc=doc,
        template_id="dog_api",
    )
    assert out["saved_to_meta"] is True
    assert out["saved_to_file"]
    client.upsert_template.assert_called_once()
    index.refresh.assert_called_once()
