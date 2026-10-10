"""文档 RAG 子 Agent gRPC Worker（LangGraph 检索 + LLM 流式回答）。"""

from __future__ import annotations

import logging
import threading
from concurrent import futures

import grpc

from rag_agent.config.loader import load_agent_config, worker_listen_address
from rag_agent.ingest.extract import extract_text, supported_filename
from rag_agent.pb.agent.v1 import execute_pb2, execute_pb2_grpc
from rag_agent.llm_errors import format_llm_error, is_expected_llm_client_error
from rag_agent.rag.graph import build_rag_graph
from rag_agent.rag.store import DocumentStore

logger = logging.getLogger(__name__)

AGENT_ID = "doc_rag"
SYSTEM_PROMPT = (
    "你是 Piper Agent 的文档问答助手。用户会上传 PDF、Word 等文档，你需要严格依据提供的「文档片段」回答问题。"
    "若片段中没有相关信息，请明确说明文档中未找到，不要编造。"
    "回答使用简体中文，条理清晰，必要时引用文档中的要点。"
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


class DocRAGServicer(execute_pb2_grpc.AgentWorkerServiceServicer):
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
        self._histories = _ConversationHistories(cfg.rag.max_history_messages)
        self._store = DocumentStore(cfg.rag.chunk_size, cfg.rag.chunk_overlap)
        self._graph = build_rag_graph(self._store, cfg.rag.top_k)

    def Execute(self, request: execute_pb2.ExecuteRequest, context: grpc.ServicerContext):
        run_id = request.run_id or "run-unknown"
        conv_id = request.conversation_id or "default"
        cancel_ev = self._cancels.register(run_id)

        def cancelled() -> bool:
            return cancel_ev.is_set() or not context.is_active()

        logger.info(
            "execute start run=%s conv=%s trace=%s docs=%d msg_len=%d",
            run_id,
            conv_id,
            request.trace_id,
            len(request.documents),
            len(request.user_message),
        )

        try:
            new_docs: list[tuple[str, str]] = []
            for doc in request.documents:
                name = doc.filename or "upload"
                if not supported_filename(name):
                    yield execute_pb2.ExecuteEvent(
                        type="error",
                        message=f"不支持的文件: {name}",
                    )
                    return
                try:
                    text = extract_text(name, bytes(doc.data))
                    new_docs.append((name, text))
                except Exception as e:  # noqa: BLE001
                    yield execute_pb2.ExecuteEvent(type="error", message=f"解析 {name} 失败: {e}")
                    return

            question = (request.user_message or "").strip()
            if not question and new_docs:
                question = "请概括上传文档的主要内容，并列出关键要点。"
            if not question:
                yield execute_pb2.ExecuteEvent(type="error", message="请输入问题或上传文档")
                return

            if new_docs:
                names = ", ".join(n for n, _ in new_docs)
                yield execute_pb2.ExecuteEvent(
                    type="run.progress",
                    tool_name="ingest",
                    message=f"已接收文档: {names}",
                )

            state = self._graph.invoke(
                {
                    "conversation_id": conv_id,
                    "question": question,
                    "new_documents": new_docs,
                }
            )
            ingested = int(state.get("ingested_count") or 0)
            if ingested > 0:
                yield execute_pb2.ExecuteEvent(
                    type="run.progress",
                    tool_name="index",
                    message=f"文档已切分为 {ingested} 个片段并加入知识库",
                )

            context_text = str(state.get("retrieved_context") or "")
            history = self._histories.append_user(conv_id, question)
            messages = [
                {"role": "system", "content": SYSTEM_PROMPT},
                {
                    "role": "system",
                    "content": f"以下是与问题相关的文档片段：\n\n{context_text}",
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
            "请在 deploy/config/local.yaml 中设置 agents.doc_rag.listen，例如 127.0.0.1:15063。"
        ) from e
    if bound == 0:
        raise RuntimeError(f"无法在 {addr} 绑定 gRPC Worker（add_insecure_port 返回 0）")


def serve(config_path: str | None, listen: str | None = None) -> None:
    _ = load_agent_config(config_path)
    addr = listen or worker_listen_address(config_path, AGENT_ID)
    servicer = DocRAGServicer(config_path)
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=8))
    execute_pb2_grpc.add_AgentWorkerServiceServicer_to_server(servicer, server)
    _bind_grpc_port(server, addr)
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    logger.info("doc-rag worker listening on %s", addr)
    server.start()
    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        server.stop(grace=3)


def _cli() -> None:
    import argparse
    from pathlib import Path

    from rag_agent.config.loader import default_config_path

    parser = argparse.ArgumentParser(prog="rag-agent-doc-rag")
    parser.add_argument("--config", default=str(default_config_path()))
    parser.add_argument("--listen", default=None)
    args = parser.parse_args()
    cfg = args.config if Path(args.config).is_file() else None
    serve(cfg, args.listen)


if __name__ == "__main__":
    _cli()
