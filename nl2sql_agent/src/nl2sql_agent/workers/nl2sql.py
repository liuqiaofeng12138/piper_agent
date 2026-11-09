"""NL2SQL 子 Agent gRPC Worker（半导体演示数仓）。"""

from __future__ import annotations

import logging
import threading
from concurrent import futures
from pathlib import Path
from typing import Any

import grpc

from nl2sql_agent.config.loader import load_agent_config, worker_listen_address
from nl2sql_agent.knowledge.catalog import load_catalog
from nl2sql_agent.pipeline.orchestrator import NL2SQLOrchestrator
from nl2sql_agent.pb.agent.v1 import execute_pb2, execute_pb2_grpc

logger = logging.getLogger(__name__)

AGENT_ID = "nl2sql"


class _RunCancels:
    def __init__(self) -> None:
        self._events: dict[str, threading.Event] = {}
        self._lock = threading.Lock()

    def register(self, run_id: str) -> threading.Event:
        ev = threading.Event()
        with self._lock:
            self._events[run_id] = ev
        return ev

    def cancel(self, run_id: str) -> bool:
        with self._lock:
            ev = self._events.get(run_id)
        if ev is None:
            return False
        ev.set()
        return True

    def remove(self, run_id: str) -> None:
        with self._lock:
            self._events.pop(run_id, None)


def _event_to_pb(ev: dict[str, Any]) -> execute_pb2.ExecuteEvent:
    return execute_pb2.ExecuteEvent(
        type=str(ev.get("type", "")),
        delta=str(ev.get("delta", "")),
        content=str(ev.get("content", "")),
        tool_name=str(ev.get("tool_name", "")),
        tool_detail=str(ev.get("tool_detail", "")),
        message=str(ev.get("message", "")),
    )


class NL2SQLServicer(execute_pb2_grpc.AgentWorkerServiceServicer):
    def __init__(self, config_path: str | None) -> None:
        cfg = load_agent_config(config_path)
        catalog_path = Path(cfg.nl2sql.catalog_path) if cfg.nl2sql.catalog_path else None
        catalog = load_catalog(catalog_path)

        llm = None
        if cfg.llm.api_key and (cfg.nl2sql.use_llm_extract or cfg.nl2sql.use_llm_explain):
            from openai import OpenAI

            kwargs: dict[str, Any] = {"api_key": cfg.llm.api_key}
            if cfg.llm.base_url:
                kwargs["base_url"] = cfg.llm.base_url
            llm = OpenAI(**kwargs)

        self._orch = NL2SQLOrchestrator(
            dsn=cfg.nl2sql.dsn,
            catalog=catalog,
            llm_client=llm,
            llm_model=cfg.llm.model,
            use_llm_extract=cfg.nl2sql.use_llm_extract,
            use_llm_explain=cfg.nl2sql.use_llm_explain and bool(cfg.llm.api_key),
        )
        self._cancels = _RunCancels()
        try:
            self._orch.ensure_db()
        except Exception as e:  # noqa: BLE001
            logger.warning("database ping failed at startup: %s", e)

    def Execute(self, request: execute_pb2.ExecuteRequest, context: grpc.ServicerContext):
        run_id = request.run_id or "run-unknown"
        logger.info(
            "execute start run=%s conv=%s trace=%s msg_len=%d",
            run_id,
            request.conversation_id,
            request.trace_id,
            len(request.user_message),
        )
        cancel_ev = self._cancels.register(run_id)

        def cancelled() -> bool:
            return cancel_ev.is_set() or not context.is_active()

        try:
            for ev in self._orch.run_turn(request.user_message, cancel=cancelled):
                yield _event_to_pb(ev)
                if ev.get("type") in ("error", "run.cancelled", "message.done"):
                    break
        except Exception as e:  # noqa: BLE001
            logger.exception("Execute failed run_id=%s", run_id)
            yield execute_pb2.ExecuteEvent(type="error", message=str(e))
        finally:
            self._cancels.remove(run_id)
            logger.info("execute end run=%s", run_id)

    def Cancel(self, request: execute_pb2.CancelRequest, context: grpc.ServicerContext):
        ok = self._cancels.cancel(request.run_id)
        return execute_pb2.CancelResponse(ok=ok, message="cancelled" if ok else "run not found")

    def Health(self, request: execute_pb2.HealthRequest, context: grpc.ServicerContext):
        try:
            self._orch.ensure_db()
            status = "ok"
        except Exception as e:  # noqa: BLE001
            status = f"degraded: {e}"
        return execute_pb2.HealthResponse(status=status, agent_id=AGENT_ID)


def _bind_grpc_port(server: grpc.Server, addr: str) -> None:
    bound = server.add_insecure_port(addr)
    if bound == 0:
        raise RuntimeError(f"无法在 {addr} 绑定 gRPC Worker")


def serve(config_path: str | None, listen: str | None = None) -> None:
    addr = listen or worker_listen_address(config_path, AGENT_ID)
    servicer = NL2SQLServicer(config_path)
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=8))
    execute_pb2_grpc.add_AgentWorkerServiceServicer_to_server(servicer, server)
    _bind_grpc_port(server, addr)
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    logger.info("nl2sql worker listening on %s", addr)
    server.start()
    server.wait_for_termination()


def main() -> None:
    import argparse

    p = argparse.ArgumentParser(description="NL2SQL Agent Worker")
    p.add_argument("--config", default=None)
    p.add_argument("--listen", default=None)
    args = p.parse_args()
    serve(args.config, args.listen)


if __name__ == "__main__":
    main()
