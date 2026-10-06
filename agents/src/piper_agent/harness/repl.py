"""Phase A minimal REPL — exercises mock Runtime over gRPC (no LLM)."""

from __future__ import annotations

import shlex
import uuid
from typing import Optional

from piper_agent.clients.runtime_client import RuntimeClient
from piper_agent.pb.common.v1 import types_pb2


class HarnessREPL:
    """Interactive shell mimicking future agent tool sequence: validate → run → stream."""

    def __init__(self, client: RuntimeClient) -> None:
        self.client = client
        self.session_id = f"sess-{uuid.uuid4().hex[:8]}"
        self.last_template_id: Optional[str] = None
        self.last_run_id: Optional[str] = None

    def run(self) -> None:
        print(f"Piper Agent Harness REPL (Phase A) session={self.session_id}")
        print("Commands: help | list | validate [id] | run [id] | watch | data | quit")
        while True:
            try:
                line = input("harness> ").strip()
            except (EOFError, KeyboardInterrupt):
                print()
                break
            if not line:
                continue
            parts = shlex.split(line)
            cmd = parts[0].lower()
            if cmd in ("quit", "exit", "q"):
                break
            if cmd == "help":
                self._help()
            elif cmd == "list":
                self._list()
            elif cmd == "validate":
                tid = parts[1] if len(parts) > 1 else "demo-tpl"
                self._validate(tid)
            elif cmd == "run":
                if len(parts) < 2:
                    tid = self.last_template_id or "demo-tpl"
                else:
                    tid = parts[1]
                self._run(tid)
            elif cmd == "watch":
                self._watch()
            elif cmd == "data":
                self._data()
            else:
                print(f"unknown command: {cmd}")

    def _help(self) -> None:
        print(
            "  list                    — ListTemplates from meta"
            "\n  validate [template_id]  — ValidateTemplate (meta id or demo-tpl)"
            "\n  run [template_id]       — RunTemplate after validate"
            "\n  watch                   — SubscribeRun for last run"
            "\n  data                    — GetTokenData for last run"
        )

    def _list(self) -> None:
        resp = self.client.list_templates(session_id=self.session_id, size=10)
        for t in resp.templates:
            print(f"  {t.id or t.name}  name={t.name} domain={t.domain}")

    def _validate(self, template_id: str) -> None:
        if template_id != "demo-tpl":
            resp = self.client.validate_template(
                template_id=template_id,
                session_id=self.session_id,
                meta_only=True,
            )
        else:
            resp = self.client.validate_template(
                template_id=template_id,
                name=f"demo-{template_id}",
                domain="example.com",
                json_payload=b'{"builder":{"type":"Http","url_tpl":"https://example.com"},"procedures":[]}',
                session_id=self.session_id,
            )
        self.last_template_id = template_id
        if resp.ok:
            print(f"validate ok template_id={template_id}")
        else:
            for d in resp.diagnostics:
                print(f"  [{d.code}] {d.path}: {d.message}")

    def _run(self, template_id: str) -> None:
        try:
            resp = self.client.run_template(
                template_id=template_id,
                session_id=self.session_id,
                engine="http",
            )
        except Exception as e:  # noqa: BLE001 — REPL shows gRPC errors to user
            print(f"run failed: {e}")
            return
        self.last_run_id = resp.run_id
        phase = types_pb2.RunPhase.Name(resp.status.phase)
        print(f"run started run_id={resp.run_id} phase={phase} token={resp.status.token_id}")

    def _watch(self) -> None:
        if not self.last_run_id:
            print("no run yet; use run first")
            return
        for ev in self.client.subscribe_run(
            run_id=self.last_run_id, session_id=self.session_id
        ):
            phase = types_pb2.RunPhase.Name(ev.status.phase)
            print(f"  event phase={phase} msg={ev.message}")

    def _data(self) -> None:
        if not self.last_run_id:
            print("no run yet")
            return
        resp = self.client.get_token_data(token_id=self.last_run_id, session_id=self.session_id)
        print(resp.payload_json.decode("utf-8", errors="replace")[:2000])
