"""通用对话子 Agent gRPC Worker（无工具，纯 LLM 流式问答）。

与 web_crawler worker 实现同一 AgentWorkerService 契约；
不连接 Runtime，按 conversation_id 维护多轮消息历史。
"""

from __future__ import annotations

import logging
import threading
from concurrent import futures

import grpc

from piper_agent.config.loader import load_agent_config, worker_listen_address
from piper_agent.pb.agent.v1 import execute_pb2, execute_pb2_grpc

logger = logging.getLogger(__name__)

AGENT_ID = "general_chat"
SYSTEM_PROMPT = (
    "你是 Piper Agent 平台的通用对话助手，负责日常问答、知识咨询与闲聊。"
    "如果用户提出网页抓取/数据采集类需求，请提示他描述采集目标，"
    "系统会自动路由到网页采集 Agent。回答使用简体中文，简洁清晰。"
)
MAX_HISTORY_MESSAGES = 40


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
    def __init__(self) -> None:
        self._histories: dict[str, list[dict[str, str]]] = {}
        self._lock = threading.Lock()

    def append_and_get(self, conversation_id: str, user_message: str) -> list[dict[str, str]]:
        with self._lock:
            history = self._histories.setdefault(conversation_id, [])
            history.append({"role": "user", "content": user_message})
            if len(history) > MAX_HISTORY_MESSAGES:
                del history[: len(history) - MAX_HISTORY_MESSAGES]
            return list(history)

    def append_assistant(self, conversation_id: str, content: str) -> None:
        if not content:
            return
        with self._lock:
            history = self._histories.setdefault(conversation_id, [])
            history.append({"role": "assistant", "content": content})
            if len(history) > MAX_HISTORY_MESSAGES:
                del history[: len(history) - MAX_HISTORY_MESSAGES]


class GeneralChatServicer(execute_pb2_grpc.AgentWorkerServiceServicer):
    def __init__(self, config_path: str | None) -> None:
        cfg = load_agent_config(config_path)
        if not cfg.llm.api_key:
            hint = cfg.config_path or "deploy/config/local.yaml"
            raise RuntimeError(
                f"请在配置文件中设置 llm.api_key（OpenAI 或兼容服务的 API Key）: {hint}"
            )
        try:
            from openai import OpenAI
        except ImportError as e:
            raise RuntimeError("Install LLM extras: pip install -e '.[llm]'") from e

        kwargs: dict = {"api_key": cfg.llm.api_key}
        if cfg.llm.base_url:
            kwargs["base_url"] = cfg.llm.base_url
        self._llm = OpenAI(**kwargs)
        self._model = cfg.llm.model
        self._cancels = _RunCancels()
        self._histories = _ConversationHistories()

    def Execute(self, request: execute_pb2.ExecuteRequest, context: grpc.ServicerContext):
        run_id = request.run_id or "run-unknown"
        conv_id = request.conversation_id or "default"
        logger.info(
            "execute start run=%s conv=%s trace=%s msg_len=%d",
            run_id,
            conv_id,
            request.trace_id,
            len(request.user_message),
        )
        cancel_ev = self._cancels.register(run_id)

        def cancelled() -> bool:
            return cancel_ev.is_set() or not context.is_active()

        full: list[str] = []
        try:
            history = self._histories.append_and_get(conv_id, request.user_message)
            messages = [{"role": "system", "content": SYSTEM_PROMPT}, *history]
            stream = self._llm.chat.completions.create(
                model=self._model,
                messages=messages,
                stream=True,
            )
            for chunk in stream:
                if cancelled():
                    logger.info("run=%s cancelled", run_id)
                    yield execute_pb2.ExecuteEvent(
                        type="run.cancelled", message="用户停止生成"
                    )
                    break
                delta = chunk.choices[0].delta.content if chunk.choices else None
                if delta:
                    full.append(delta)
                    yield execute_pb2.ExecuteEvent(type="message.delta", delta=delta)
            else:
                content = "".join(full)
                self._histories.append_assistant(conv_id, content)
                yield execute_pb2.ExecuteEvent(type="message.done", content=content)
        except Exception as e:  # noqa: BLE001 — surface to gateway
            logger.exception("Execute failed run_id=%s", run_id)
            yield execute_pb2.ExecuteEvent(type="error", message=str(e))
        finally:
            self._cancels.remove(run_id)
            logger.info("execute end run=%s conv=%s", run_id, conv_id)

    def Cancel(self, request: execute_pb2.CancelRequest, context: grpc.ServicerContext):
        ok = self._cancels.cancel(request.run_id)
        logger.info("cancel run=%s ok=%s", request.run_id, ok)
        return execute_pb2.CancelResponse(ok=ok, message="cancelled" if ok else "run not found")

    def Health(self, request: execute_pb2.HealthRequest, context: grpc.ServicerContext):
        return execute_pb2.HealthResponse(status="ok", agent_id=AGENT_ID)


def _bind_grpc_port(server: grpc.Server, addr: str) -> None:
    try:
        bound = server.add_insecure_port(addr)
    except RuntimeError as e:
        raise RuntimeError(
            f"无法在 {addr} 绑定 gRPC Worker：{e}\n"
            "在 Windows 上 50000–50100 等端口常被系统保留（错误 10013）。"
            "请在 deploy/config/local.yaml 中设置 agents.general_chat.listen，"
            "例如 127.0.0.1:15062，并与 gateway 使用同一地址。"
        ) from e
    if bound == 0:
        raise RuntimeError(f"无法在 {addr} 绑定 gRPC Worker（add_insecure_port 返回 0）")


def serve(config_path: str | None, listen: str | None = None) -> None:
    _ = load_agent_config(config_path)
    addr = listen or worker_listen_address(config_path, AGENT_ID)
    servicer = GeneralChatServicer(config_path)
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=8))
    execute_pb2_grpc.add_AgentWorkerServiceServicer_to_server(servicer, server)
    _bind_grpc_port(server, addr)
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    logger.info("general-chat worker listening on %s", addr)
    server.start()
    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        server.stop(grace=3)


def main(config_path: str | None, listen: str | None) -> None:
    serve(config_path, listen)


def _cli() -> None:
    import argparse
    from pathlib import Path

    from piper_agent.config.loader import default_config_path

    parser = argparse.ArgumentParser(
        prog="piper-agent-general-chat",
        description="通用对话子 Agent gRPC Worker（由 piper-serve 拉起，非 CLI 入口）",
    )
    parser.add_argument(
        "--config",
        default=str(default_config_path()),
        help="deploy/config/local.yaml",
    )
    parser.add_argument(
        "--listen",
        default=None,
        help="override agents.general_chat.listen",
    )
    args = parser.parse_args()
    cfg = args.config if Path(args.config).is_file() else None
    main(cfg, args.listen)


if __name__ == "__main__":
    _cli()
