from __future__ import annotations

import json
import re
from typing import Any

from web_crawler_agent.clients.runtime_client import RuntimeClient
from web_crawler_agent.config.loader import AgentConfig
from web_crawler_agent.harness.policies import (
    PolicyError,
    ensure_run_quota,
    ensure_validated_before_run,
)
from web_crawler_agent.pb.common.v1 import types_pb2
from web_crawler_agent.harness.session import SessionState
from web_crawler_agent.probe.site_probe import probe_url
from web_crawler_agent.rag.template_index import TemplateIndex
from web_crawler_agent.templates.store import persist_validated_template


class ToolHandlers:
    def __init__(
        self,
        client: RuntimeClient,
        session: SessionState,
        config: AgentConfig,
        index: TemplateIndex,
    ) -> None:
        self.client = client
        self.session = session
        self.config = config
        self.index = index

    def dispatch(self, name: str, arguments: dict[str, Any]) -> str:
        handlers = {
            "search_similar_templates": self.search_similar_templates,
            "template_author_validate": self.template_author_validate,
            "template_author_save": self.template_author_save,
            "param_filler_suggest": self.param_filler_suggest,
            "list_proxies": self.list_proxies,
            "select_proxy": self.select_proxy,
            "runner_execute": self.runner_execute,
            "runner_wait_for_complete": self.runner_wait_for_complete,
            "runner_get_data": self.runner_get_data,
            "load_shared_example_template": self.load_shared_example_template,
            "site_probe": self.site_probe,
        }
        fn = handlers.get(name)
        if fn is None:
            return json.dumps({"error": f"unknown tool {name}"})
        try:
            return fn(arguments)
        except PolicyError as e:
            return json.dumps({"error": str(e), "code": "POLICY"})
        except Exception as e:  # noqa: BLE001 — tool surface for LLM
            return json.dumps({"error": str(e), "code": "INTERNAL"})

    def site_probe(self, args: dict[str, Any]) -> str:
        url = str(args.get("url") or args.get("target") or "").strip()
        if not url:
            return json.dumps({"ok": False, "error": "url required"})
        max_bytes = int(args.get("max_body_bytes") or self.config.harness.site_probe_max_bytes)
        timeout = float(args.get("timeout_seconds") or self.config.harness.site_probe_timeout_seconds)
        result = probe_url(url, max_body_bytes=max_bytes, timeout_seconds=timeout)
        return json.dumps(result, ensure_ascii=False)

    def search_similar_templates(self, args: dict[str, Any]) -> str:
        query = str(args.get("query", ""))
        top_k = int(args.get("top_k") or 5)
        prefer_shared = bool(args.get("prefer_shared"))
        hits = self.index.search(query, top_k=top_k, prefer_shared=prefer_shared)
        rows = []
        for h in hits:
            row: dict[str, Any] = {
                "template_id": h.template_id,
                "name": h.name,
                "domain": h.domain,
                "score": h.score,
                "source": h.source,
                "snippet": h.snippet,
            }
            if h.template_json:
                row["template_json"] = h.template_json
                row["use_verbatim"] = True
            rows.append(row)
        return json.dumps(rows, ensure_ascii=False)

    def load_shared_example_template(self, args: dict[str, Any]) -> str:
        key = str(args.get("example_id") or args.get("filename") or args.get("name") or "")
        if not key:
            return json.dumps({"error": "example_id or filename required"})
        loaded = self.index.load_shared_example(key)
        if not loaded:
            return json.dumps({"error": f"shared example not found: {key}"})
        doc, filename = loaded
        self.session.pending_template = doc
        tid = str(doc.get("id") or doc.get("name") or "")
        body = json.dumps(doc, ensure_ascii=False)
        return json.dumps(
            {
                "template_id": tid,
                "source": f"shared/examples/templates/{filename}",
                "template_json": body,
                "use_verbatim": True,
            },
            ensure_ascii=False,
        )

    def template_author_validate(self, args: dict[str, Any]) -> str:
        vars_map = {str(k): str(v) for k, v in (args.get("vars") or {}).items()}
        tid = str(args.get("template_id") or "")
        raw = args.get("template_json")
        if raw:
            doc = json.loads(raw) if isinstance(raw, str) else raw
            tid = str(doc.get("id") or tid or doc.get("name") or "")
            payload = json.dumps(doc, ensure_ascii=False).encode("utf-8")
            resp = self.client.validate_template(
                template_id=tid,
                name=str(doc.get("name") or tid),
                domain=str(doc.get("domain") or ""),
                json_payload=payload,
                session_id=self.session.session_id,
                vars=vars_map,
            )
        elif tid:
            resp = self.client.validate_template(
                template_id=tid,
                session_id=self.session.session_id,
                vars=vars_map,
                meta_only=True,
            )
        else:
            return json.dumps({"ok": False, "error": "template_json or template_id required"})
        saved_info: dict[str, Any] = {}
        if resp.ok and tid:
            self.session.validated_template_ids.add(tid)
            self.session.last_template_id = tid
            if raw:
                doc = json.loads(raw) if isinstance(raw, str) else raw
                self.session.pending_template = doc
                if self.config.harness.auto_save_templates:
                    try:
                        saved_info = persist_validated_template(
                            client=self.client,
                            index=self.index,
                            session=self.session,
                            cfg=self.config,
                            doc=doc,
                            template_id=tid,
                        )
                    except Exception as e:  # noqa: BLE001
                        saved_info = {"save_error": str(e)}
        diags = [{"code": d.code, "path": d.path, "message": d.message} for d in resp.diagnostics]
        payload: dict[str, Any] = {"ok": resp.ok, "template_id": tid, "diagnostics": diags}
        if saved_info:
            payload["persist"] = saved_info
        return json.dumps(payload, ensure_ascii=False)

    def template_author_save(self, args: dict[str, Any]) -> str:
        raw = args.get("template_json")
        if not raw:
            return json.dumps({"error": "template_json required"})
        doc = json.loads(raw) if isinstance(raw, str) else raw
        tid = str(args.get("template_id") or doc.get("id") or "")
        try:
            info = persist_validated_template(
                client=self.client,
                index=self.index,
                session=self.session,
                cfg=self.config,
                doc=doc,
                template_id=tid,
            )
        except Exception as e:  # noqa: BLE001
            return json.dumps({"error": str(e), "code": "INTERNAL"})
        return json.dumps(info, ensure_ascii=False)

    def param_filler_suggest(self, args: dict[str, Any]) -> str:
        text = str(args.get("user_text") or "")
        vars_out: dict[str, str] = {}
        url_m = re.search(r"https?://[^\s\"']+", text)
        if url_m:
            vars_out["url"] = url_m.group(0)
        for m in re.finditer(r"[\{\{]([a-zA-Z_][\w]*)[\}\}]", text):
            vars_out.setdefault(m.group(1), "")
        raw = args.get("template_json")
        if raw:
            doc = json.loads(raw) if isinstance(raw, str) else raw
            builder = doc.get("builder") or {}
            url_tpl = str(builder.get("url_tpl") or "")
            for m in re.finditer(r"\{\{(\w+)\}\}", url_tpl):
                name = m.group(1)
                if name not in vars_out and name == "url" and "url" in vars_out:
                    vars_out[name] = vars_out["url"]
                else:
                    vars_out.setdefault(name, "")
        return json.dumps({"vars": vars_out}, ensure_ascii=False)

    def list_proxies(self, args: dict[str, Any]) -> str:
        resp = self.client.list_proxies(
            size=int(args.get("size") or 20),
            status=str(args.get("status") or ""),
            session_id=self.session.session_id,
        )
        rows = [
            {"id": p.id, "name": p.name, "domain": p.domain, "status": p.status}
            for p in resp.proxies
        ]
        return json.dumps({"proxies": rows, "total": resp.total}, ensure_ascii=False)

    def select_proxy(self, args: dict[str, Any]) -> str:
        pid = str(args.get("proxy_id") or "")
        if not pid:
            return json.dumps({"error": "proxy_id required"})
        self.session.selected_proxy_id = pid
        return json.dumps({"selected_proxy_id": pid})

    def runner_execute(self, args: dict[str, Any]) -> str:
        tid = str(args.get("template_id") or self.session.last_template_id or "")
        if not tid:
            return json.dumps({"error": "template_id required"})
        ensure_validated_before_run(
            self.session,
            tid,
            required=self.config.harness.require_validate_before_run,
        )
        ensure_run_quota(self.session, self.config.harness.max_runs_per_session)
        vars_map = {str(k): str(v) for k, v in (args.get("vars") or {}).items()}
        engine = str(args.get("engine") or "auto")
        if engine == "auto":
            engine = self._infer_engine(tid)
        proxy_id = str(args.get("proxy_id") or self.session.selected_proxy_id or "")
        resp = self.client.run_template(
            template_id=tid,
            session_id=self.session.session_id,
            vars=vars_map,
            engine=engine,
            proxy_id=proxy_id,
        )
        self.session.run_count += 1
        self.session.last_run_id = resp.run_id
        self.session.last_token_id = resp.status.token_id
        return json.dumps(
            {
                "run_id": resp.run_id,
                "token_id": resp.status.token_id,
                "phase": int(resp.status.phase),
                "engine": engine,
                "proxy_id": proxy_id,
            }
        )

    def runner_wait_for_complete(self, args: dict[str, Any]) -> str:
        run_id = str(args.get("run_id") or self.session.last_run_id or "")
        if not run_id:
            return json.dumps({"error": "no run_id"})
        phases: list[str] = []
        messages: list[str] = []
        for ev in self.client.subscribe_run(run_id=run_id, session_id=self.session.session_id):
            phases.append(types_pb2.RunPhase.Name(ev.status.phase))
            if ev.message:
                messages.append(ev.message)
        return json.dumps(
            {
                "run_id": run_id,
                "phases": phases,
                "messages": messages[-5:],
                "final": phases[-1] if phases else "",
                "hint": "Chrome 仅在检测到登录墙时才会等待 manualLoginWaitSeconds；已登录时会直接抓取。等待期间 runner_wait_for_complete 会保持 RUNNING。",
            },
            ensure_ascii=False,
        )

    def _infer_engine(self, template_id: str) -> str:
        for doc in (
            self.session.pending_template,
            self._template_doc_from_meta(template_id),
        ):
            if not doc:
                continue
            b = doc.get("builder")
            if isinstance(b, dict) and str(b.get("type") or "").lower() == "chrome":
                return "chrome"
        return "http"

    def _template_doc_from_meta(self, template_id: str) -> dict[str, Any] | None:
        try:
            resp = self.client.get_template(template_id=template_id, session_id=self.session.session_id)
            if resp.template and resp.template.json_payload:
                return json.loads(resp.template.json_payload.decode("utf-8"))
        except Exception:
            return None
        return None

    def runner_get_data(self, args: dict[str, Any]) -> str:
        token_id = str(args.get("token_id") or self.session.last_token_id or self.session.last_run_id or "")
        if not token_id:
            return json.dumps({"error": "no token_id"})
        resp = self.client.get_token_data(token_id=token_id, session_id=self.session.session_id)
        try:
            payload = json.loads(resp.payload_json.decode("utf-8"))
        except json.JSONDecodeError:
            payload = {"raw": resp.payload_json.decode("utf-8", errors="replace")}
        return json.dumps(payload, ensure_ascii=False)[:8000]
