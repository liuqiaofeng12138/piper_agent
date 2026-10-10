import json

from web_crawler_agent.probe.site_probe import probe_url


def test_probe_json():
    payload = {"slideshow": {"title": "T", "author": "A", "slides": [{"title": "s1"}]}}

    def fetch(_url: str, _timeout: float, _limit: int):
        raw = json.dumps(payload).encode()
        return 200, "https://example.com/api", {"content-type": "application/json"}, raw

    out = probe_url("https://example.com/api", fetch=fetch)
    assert out["ok"] is True
    assert out["format"] == "json"
    assert "slideshow" in out.get("json_top_level_keys", [])
    assert any("slideshow" in p for p in out.get("json_sample_paths", []))


def test_probe_html_title():
    html = b"""<!DOCTYPE html><html><head><title>Hello Site</title></head>
    <body><p>content</p></body></html>"""

    def fetch(_url: str, _timeout: float, _limit: int):
        return 200, "https://example.com/", {"content-type": "text/html"}, html

    out = probe_url("example.com", fetch=fetch)
    assert out["ok"] is True
    assert out["format"] == "html"
    assert out["title"] == "Hello Site"
    assert out["suggested_engine"] == "http"
