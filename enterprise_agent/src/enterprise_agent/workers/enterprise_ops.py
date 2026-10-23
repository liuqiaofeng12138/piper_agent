"""企业运营与 BI 智囊 gRPC Worker（Skills 架构主控）。"""

from __future__ import annotations

import logging
import threading
from concurrent import futures

import grpc

from enterprise_agent.config.loader import load_agent_config, worker_listen_address
from enterprise_agent.llm_errors import format_llm_error, is_expected_llm_client_error
from enterprise_agent.orchestrator import (
    build_synthesis_messages,
    run_enterprise_pipeline,
    stream_synthesis,
)
from enterprise_agent.pb.agent.v1 import execute_pb2, execute_pb2_grpc

logger = logging.getLogger(__name__)

AGENT_ID = "enterprise_ops"


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


class _ConversationHistories:
    def __init__(self, max_messages: int) -> None:
        self._max = max_messages
        self._histories: dict[str, list[dict[str, str]]] = {}
        self._lock = threading.Lock()

    def append_user(self, conversation_id: str, content: str) -> list[dict[str, str]]:
        with self._lock:
            history = self._histories.setdefault(conversation_id, [])
            history.append({"role": "user", "content": content})
            if len(history) > self._max:
                del history[: len(history) - self._max]
            return list(history)

    def append_assistant(self, conversation_id: str, content: str) -> None:
        if not content:
            return
        with self._lock:
            history = self._histories.setdefault(conversation_id, [])
            history.append({"role": "assistant", "content": content})
            if len(history) > self._max:
                del history[: len(history) - self._max]


class EnterpriseOpsServicer(execute_pb2_grpc.AgentWorkerServiceServicer):
    def __init__(self, config_path: str | None) -> None:
        self._cfg = load_agent_config(config_path)
        if not self._cfg.llm.api_key:
            hint = self._cfg.config_path or "deploy/config/local.yaml"
            raise RuntimeError(f"请在配置文件中设置 llm.api_key: {hint}")
        try:
            from openai import OpenAI
        except ImportError as e:
            raise RuntimeError("Install LLM: pip install openai") from e

        kwargs: dict = {"api_key": self._cfg.llm.api_key}
        if self._cfg.llm.base_url:
            kwargs["base_url"] = self._cfg.llm.base_url
        self._llm = OpenAI(**kwargs)
        self._model = self._cfg.llm.model
        self._cancels = _RunCancels()
        self._histories = _ConversationHistories(self._cfg.enterprise_ops.max_history_messages)

    def Execute(self, request: execute_pb2.ExecuteRequest, context: grpc.ServicerContext):
        run_id = request.run_id or "run-unknown"
        conv_id = request.conversation_id or "default"
        cancel_ev = self._cancels.register(run_id)

        def cancelled() -> bool:
            return cancel_ev.is_set() or not context.is_active()

        question = (request.user_message or "").strip()
        docs: list[tuple[str, bytes]] = [
            (d.filename, bytes(d.data)) for d in request.documents if d.filename and d.data
        ]

        logger.info(
            "execute start run=%s conv=%s trace=%s msg_len=%d docs=%d",
            run_id,
            conv_id,
            request.trace_id,
            len(question),
            len(docs),
        )

        try:
            if not question:
                yield execute_pb2.ExecuteEvent(type="error", message="请输入企业运营或数据分析相关的问题")
                return

            yield execute_pb2.ExecuteEvent(
                type="run.progress",
                tool_name="triage",
                message="技能路由中枢正在分诊…",
            )

            def on_progress(event: tuple[str, str, str]) -> None:
                _phase, tool, msg = event
                logger.info("progress run=%s %s: %s", run_id, tool, msg[:200])

            skill_ids, route_reason, outputs = run_enterprise_pipeline(
                self._cfg,
                self._llm,
                self._model,
                question,
                docs,
                on_progress=on_progress,
            )

            if cancelled():
                yield execute_pb2.ExecuteEvent(type="run.cancelled", message="已取消")
                return

            yield execute_pb2.ExecuteEvent(
                type="run.progress",
                tool_name="skills",
                message=f"已完成 {len(skill_ids)} 个技能包，正在生成综合报告…",
            )

            history = self._histories.append_user(conv_id, question)
            base = build_synthesis_messages(question, outputs, route_reason)
            prior = history[:-1]
            messages = [base[0], *prior, base[1]]

            full: list[str] = []
            for delta in stream_synthesis(self._llm, self._model, messages):
                if cancelled():
                    yield execute_pb2.ExecuteEvent(type="run.cancelled", message="用户停止生成")
                    return
                full.append(delta)
                yield execute_pb2.ExecuteEvent(type="message.delta", delta=delta)

            content = "".join(full)
            self._histories.append_assistant(conv_id, content)
            yield execute_pb2.ExecuteEvent(type="message.done", content=content)
        except Exception as e:  # noqa: BLE001
            msg = format_llm_error(e)
            if is_expected_llm_client_error(e):
                logger.warning("Execute LLM error run_id=%s: %s", run_id, msg)
            else:
                logger.exception("Execute failed run_id=%s", run_id)
            yield execute_pb2.ExecuteEvent(type="error", message=msg)
        finally:
            self._cancels.remove(run_id)
            logger.info("execute end run=%s conv=%s", run_id, conv_id)

    def Cancel(self, request: execute_pb2.CancelRequest, context: grpc.ServicerContext):
        ok = self._cancels.cancel(request.run_id)
        return execute_pb2.CancelResponse(ok=ok, message="cancelled" if ok else "run not found")

    def Health(self, request: execute_pb2.HealthRequest, context: grpc.ServicerContext):
        return execute_pb2.HealthResponse(status="ok", agent_id=AGENT_ID)


def _bind_grpc_port(server: grpc.Server, addr: str) -> None:
    try:
        bound = server.add_insecure_port(addr)
    except RuntimeError as e:
        raise RuntimeError(
            f"无法在 {addr} 绑定 gRPC Worker：{e}\n"
            "请在 deploy/config/local.yaml 中设置 agents.enterprise_ops.listen，例如 127.0.0.1:15066。"
        ) from e
    if bound == 0:
        raise RuntimeError(f"无法在 {addr} 绑定 gRPC Worker（add_insecure_port 返回 0）")


def serve(config_path: str | None, listen: str | None = None) -> None:
    _ = load_agent_config(config_path)
    addr = listen or worker_listen_address(config_path, AGENT_ID)
    servicer = EnterpriseOpsServicer(config_path)
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=8))
    execute_pb2_grpc.add_AgentWorkerServiceServicer_to_server(servicer, server)
    _bind_grpc_port(server, addr)
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    logger.info("enterprise_ops worker listening on %s", addr)
    server.start()
    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        server.stop(grace=3)


def _cli() -> None:
    import argparse
    from pathlib import Path

    from enterprise_agent.config.loader import default_config_path

    parser = argparse.ArgumentParser(prog="enterprise-agent")
    parser.add_argument("--config", default=str(default_config_path()))
    parser.add_argument("--listen", default=None)
    args = parser.parse_args()
    cfg = args.config if Path(args.config).is_file() else None
    serve(cfg, args.listen)


if __name__ == "__main__":
    _cli()
