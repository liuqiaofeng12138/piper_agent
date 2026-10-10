"""学术论文搜索子 Agent gRPC Worker（LangGraph 检索 arXiv + LLM 流式汇总）。"""

from __future__ import annotations

import logging
import threading
from concurrent import futures

import grpc

from paper_agent.config.loader import load_agent_config, worker_listen_address
from paper_agent.llm_errors import format_llm_error, is_expected_llm_client_error
from paper_agent.papers.graph import build_paper_search_graph
from paper_agent.pb.agent.v1 import execute_pb2, execute_pb2_grpc

logger = logging.getLogger(__name__)

AGENT_ID = "paper_search"
SYSTEM_PROMPT = (
    "你是 Piper Agent 的学术论文检索助手。系统已从 arXiv 联网检索论文，并在上下文中提供标题、"
    "作者、日期、链接与摘要。请根据用户问题用简体中文汇总展示：\n"
    "1. 用简短引言说明检索主题与篇数；\n"
    "2. 按条目列出每篇论文（标题、作者、日期、arXiv 链接、一两句核心贡献）；\n"
    "3. 最后给出 2–4 条整体趋势或阅读建议。\n"
    "只能依据提供的论文信息作答，不要编造未出现的论文或结论。若未检索到结果，说明可能原因并建议用户换关键词。"
)


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


class PaperSearchServicer(execute_pb2_grpc.AgentWorkerServiceServicer):
    def __init__(self, config_path: str | None) -> None:
        cfg = load_agent_config(config_path)
        if not cfg.llm.api_key:
            hint = cfg.config_path or "deploy/config/local.yaml"
            raise RuntimeError(f"请在配置文件中设置 llm.api_key: {hint}")
        try:
            from openai import OpenAI
        except ImportError as e:
            raise RuntimeError("Install LLM: pip install openai") from e

        kwargs: dict = {"api_key": cfg.llm.api_key}
        if cfg.llm.base_url:
            kwargs["base_url"] = cfg.llm.base_url
        self._llm = OpenAI(**kwargs)
        self._model = cfg.llm.model
        self._cancels = _RunCancels()
        self._histories = _ConversationHistories(cfg.paper.max_history_messages)
        self._graph = build_paper_search_graph(
            cfg.paper.max_results_cap,
            cfg.paper.arxiv_timeout_seconds,
        )

    def Execute(self, request: execute_pb2.ExecuteRequest, context: grpc.ServicerContext):
        run_id = request.run_id or "run-unknown"
        conv_id = request.conversation_id or "default"
        cancel_ev = self._cancels.register(run_id)

        def cancelled() -> bool:
            return cancel_ev.is_set() or not context.is_active()

        question = (request.user_message or "").strip()
        logger.info(
            "execute start run=%s conv=%s trace=%s msg_len=%d",
            run_id,
            conv_id,
            request.trace_id,
            len(question),
        )

        try:
            if not question:
                yield execute_pb2.ExecuteEvent(type="error", message="请输入要检索的论文主题或问题")
                return

            yield execute_pb2.ExecuteEvent(
                type="run.progress",
                tool_name="parse",
                message="正在解析检索意图…",
            )

            state = self._graph.invoke({"user_message": question})
            topic = str(state.get("search_topic") or "")
            count = int(state.get("max_results") or 0)
            papers = state.get("papers") or []
            err = (state.get("search_error") or "").strip()

            if err:
                yield execute_pb2.ExecuteEvent(
                    type="run.progress",
                    tool_name="arxiv",
                    message=f"arXiv 检索异常: {err}",
                )
            else:
                yield execute_pb2.ExecuteEvent(
                    type="run.progress",
                    tool_name="arxiv",
                    message=f"已从 arXiv 检索「{topic}」相关论文 {len(papers)} 篇（请求 {count} 篇）",
                )

            context_text = str(state.get("papers_context") or "")
            history = self._histories.append_user(conv_id, question)
            messages = [
                {"role": "system", "content": SYSTEM_PROMPT},
                {
                    "role": "system",
                    "content": (
                        f"检索主题: {topic}\n请求篇数: {count}\n\n"
                        f"以下为检索到的论文资料：\n\n{context_text}"
                    ),
                },
                *history,
            ]

            full: list[str] = []
            stream = self._llm.chat.completions.create(
                model=self._model,
                messages=messages,
                stream=True,
            )
            for chunk in stream:
                if cancelled():
                    yield execute_pb2.ExecuteEvent(type="run.cancelled", message="用户停止生成")
                    break
                delta = chunk.choices[0].delta.content if chunk.choices else None
                if delta:
                    full.append(delta)
                    yield execute_pb2.ExecuteEvent(type="message.delta", delta=delta)
            else:
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
            "请在 deploy/config/local.yaml 中设置 agents.paper_search.listen，例如 127.0.0.1:15064。"
        ) from e
    if bound == 0:
        raise RuntimeError(f"无法在 {addr} 绑定 gRPC Worker（add_insecure_port 返回 0）")


def serve(config_path: str | None, listen: str | None = None) -> None:
    _ = load_agent_config(config_path)
    addr = listen or worker_listen_address(config_path, AGENT_ID)
    servicer = PaperSearchServicer(config_path)
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=8))
    execute_pb2_grpc.add_AgentWorkerServiceServicer_to_server(servicer, server)
    _bind_grpc_port(server, addr)
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    logger.info("paper-search worker listening on %s", addr)
    server.start()
    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        server.stop(grace=3)


def _cli() -> None:
    import argparse
    from pathlib import Path

    from paper_agent.config.loader import default_config_path

    parser = argparse.ArgumentParser(prog="paper-agent-search")
    parser.add_argument("--config", default=str(default_config_path()))
    parser.add_argument("--listen", default=None)
    args = parser.parse_args()
    cfg = args.config if Path(args.config).is_file() else None
    serve(cfg, args.listen)


if __name__ == "__main__":
    _cli()
