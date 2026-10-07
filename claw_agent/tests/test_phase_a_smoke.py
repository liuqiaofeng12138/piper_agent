"""Smoke test against Phase A mock Runtime (server must be listening)."""

import os
import sys

import grpc

from claw_agent.clients.runtime_client import RuntimeClient
from claw_agent.pb.common.v1 import types_pb2

ADDR = os.environ.get("PIPER_RUNTIME_ADDR", "localhost:50051")


def main() -> int:
    client = RuntimeClient(address=ADDR)
    try:
        bad = client.validate_template(name="", json_payload=b"")
        assert not bad.ok and bad.diagnostics

        ok = client.validate_template(
            template_id="demo-tpl",
            name="demo",
            json_payload=b'{"procedures":[]}',
            session_id="test-session",
        )
        assert ok.ok, ok.diagnostics

        run = client.run_template(template_id="demo-tpl", session_id="test-session")
        assert run.run_id

        phases = []
        for ev in client.subscribe_run(run_id=run.run_id, session_id="test-session"):
            phases.append(types_pb2.RunPhase.Name(ev.status.phase))
        assert phases[-1] == "RUN_PHASE_SUCCEEDED", phases
        print("phase_a_smoke ok", run.run_id, phases)
        return 0
    except grpc.RpcError as e:
        print("gRPC error (is piper-runtime running?):", e, file=sys.stderr)
        return 1
    finally:
        client.close()


if __name__ == "__main__":
    raise SystemExit(main())
