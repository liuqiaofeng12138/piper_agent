"""Web 采集子 Agent gRPC Worker。"""

from __future__ import annotations

import logging
import threading
from concurrent import futures
from typing import Any

import grpc

from piper_agent.agents.orchestrator import create_loop
from piper_agent.clients.runtime_client import RuntimeClient
from piper_agent.config.loader import load_agent_config, worker_listen_address
from piper_agent.harness.loop import AgentLoop
from piper_agent.pb.agent.v1 import execute_pb2, execute_pb2_grpc

logger = logging.getLogger(__name__)


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


class _ConversationLoops:
    def __init__(self, config_path: str | None) -> None:
        self._config_path = config_path
        self._loops: dict[str, tuple[AgentLoop, RuntimeClient]] = {}
        self._lock = threading.Lock()

    def get(self, conversation_id: str) -> AgentLoop:
        with self._lock:
            if conversation_id in self._loops:
                return self._loops[conversation_id][0]
            loop, client = create_loop(self._config_path, None)
            loop.session.session_id = conversation_id
            self._loops[conversation_id] = (loop, client)
            return loop

    def close_all(self) -> None:
        with self._lock:
            for _, client in self._loops.values():
                client.close()
            self._loops.clear()


def _event_to_pb(ev: dict[str, Any]) -> execute_pb2.ExecuteEvent:
    return execute_pb2.ExecuteEvent(
        type=str(ev.get("type", "")),
        delta=str(ev.get("delta", "")),
        content=str(ev.get("content", "")),
        tool_name=str(ev.get("tool_name", "")),
        tool_detail=str(ev.get("tool_detail", "")),
        message=str(ev.get("message", "")),
    )


class WebCrawlerServicer(execute_pb2_grpc.AgentWorkerServiceServicer):
    def __init__(self, config_path: str | None) -> None:
        self._config_path = config_path
        self._cancels = _RunCancels()
        self._conversations = _ConversationLoops(config_path)

    def Execute(self, request: execute_pb2.ExecuteRequest, context: grpc.ServicerContext):
        run_id = request.run_id or "run-unknown"
        cancel_ev = self._cancels.register(run_id)

        def cancelled() -> bool:
            return cancel_ev.is_set() or not context.is_active()

        try:
            loop = self._conversations.get(request.conversation_id or "default")
            for ev in loop.run_turn_iter(request.user_message, cancel=cancelled):
                yield _event_to_pb(ev)
                if ev.get("type") in ("run.cancelled", "error"):
                    break
        except Exception as e:  # noqa: BLE001 — surface to gateway
            logger.exception("Execute failed run_id=%s", run_id)
            yield execute_pb2.ExecuteEvent(type="error", message=str(e))
        finally:
            self._cancels.remove(run_id)

    def Cancel(self, request: execute_pb2.CancelRequest, context: grpc.ServicerContext):
        ok = self._cancels.cancel(request.run_id)
        return execute_pb2.CancelResponse(ok=ok, message="cancelled" if ok else "run not found")

    def Health(self, request: execute_pb2.HealthRequest, context: grpc.ServicerContext):
        return execute_pb2.HealthResponse(status="ok", agent_id="web_crawler")


def _bind_grpc_port(server: grpc.Server, addr: str) -> None:
    try:
        bound = server.add_insecure_port(addr)
    except RuntimeError as e:
        raise RuntimeError(
            f"无法在 {addr} 绑定 gRPC Worker：{e}\n"
            "在 Windows 上 50000–50100 等端口常被系统保留（错误 10013）。"
            "请在 deploy/config/local.yaml 中设置 agents.web_crawler.listen，"
            "例如 127.0.0.1:15061，并与 gateway 使用同一地址。"
        ) from e
    if bound == 0:
        raise RuntimeError(f"无法在 {addr} 绑定 gRPC Worker（add_insecure_port 返回 0）")


def serve(config_path: str | None, listen: str | None = None) -> None:
    _ = load_agent_config(config_path)
    addr = listen or worker_listen_address(config_path)
    servicer = WebCrawlerServicer(config_path)
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=8))
    execute_pb2_grpc.add_AgentWorkerServiceServicer_to_server(servicer, server)
    _bind_grpc_port(server, addr)
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    logger.info("web-crawler worker listening on %s", addr)
    server.start()
    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        servicer._conversations.close_all()
        server.stop(grace=3)


def main(config_path: str | None, listen: str | None) -> None:
    serve(config_path, listen)


def _cli() -> None:
    import argparse
    from pathlib import Path

    from piper_agent.config.loader import default_config_path

    parser = argparse.ArgumentParser(
        prog="piper-agent-web-crawler",
        description="Web 采集子 Agent gRPC Worker（由 piper-serve 拉起，非 CLI 入口）",
    )
    parser.add_argument(
        "--config",
        default=str(default_config_path()),
        help="deploy/config/local.yaml",
    )
    parser.add_argument(
        "--listen",
        default=None,
        help="override agents.web_crawler.listen",
    )
    args = parser.parse_args()
    cfg = args.config if Path(args.config).is_file() else None
    main(cfg, args.listen)


if __name__ == "__main__":
    _cli()
