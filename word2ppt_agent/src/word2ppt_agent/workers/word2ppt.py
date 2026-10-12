"""文档转 PPT 子 Agent gRPC Worker。"""

from __future__ import annotations

import logging
import re
import threading
import uuid
from concurrent import futures
from pathlib import Path

import grpc

from word2ppt_agent.config.loader import load_agent_config, resolve_artifacts_dir, worker_listen_address
from word2ppt_agent.ingest.extract import extract_text, supported_filename
from word2ppt_agent.ppt.builder import build_presentation, default_ppt_title
from word2ppt_agent.pb.agent.v1 import execute_pb2, execute_pb2_grpc

logger = logging.getLogger(__name__)

AGENT_ID = "word2ppt"


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


def _safe_download_name(name: str) -> str:
    base = Path(name).name
    base = re.sub(r'[<>:"/\\|?*]', "_", base)
    return base or "presentation.pptx"


class Word2PPTServicer(execute_pb2_grpc.AgentWorkerServiceServicer):
    def __init__(self, config_path: str | None) -> None:
        self._cfg = load_agent_config(config_path)
        self._artifacts_dir = resolve_artifacts_dir(self._cfg)
        self._cancels = _RunCancels()

    def Execute(self, request: execute_pb2.ExecuteRequest, context: grpc.ServicerContext):
        run_id = request.run_id or "run-unknown"
        cancel_ev = self._cancels.register(run_id)

        def cancelled() -> bool:
            return cancel_ev.is_set() or not context.is_active()

        logger.info(
            "execute start run=%s trace=%s docs=%d",
            run_id,
            request.trace_id,
            len(request.documents),
        )

        try:
            if not request.documents:
                yield execute_pb2.ExecuteEvent(
                    type="error",
                    message="请上传 Word、PDF、Excel 等文件，我将根据内容生成汇总 PPT",
                )
                return

            parsed: list[tuple[str, str]] = []
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
                    parsed.append((name, text))
                except Exception as e:  # noqa: BLE001
                    yield execute_pb2.ExecuteEvent(type="error", message=f"解析 {name} 失败: {e}")
                    return

            if cancelled():
                yield execute_pb2.ExecuteEvent(type="run.cancelled", message="已取消")
                return

            names = ", ".join(n for n, _ in parsed)
            yield execute_pb2.ExecuteEvent(
                type="run.progress",
                tool_name="extract",
                message=f"已解析 {len(parsed)} 个文件: {names}",
            )

            title = default_ppt_title(request.user_message, [n for n, _ in parsed])
            artifact_id = uuid.uuid4().hex
            download_name = _safe_download_name(f"{title}.pptx")
            out_path = self._artifacts_dir / f"{artifact_id}.pptx"

            yield execute_pb2.ExecuteEvent(
                type="run.progress",
                tool_name="ppt",
                message="正在生成演示文稿…",
            )

            result = build_presentation(
                parsed,
                title=title,
                output_path=out_path,
                cfg=self._cfg.word2ppt,
            )

            if cancelled():
                out_path.unlink(missing_ok=True)
                yield execute_pb2.ExecuteEvent(type="run.cancelled", message="已取消")
                return

            yield execute_pb2.ExecuteEvent(
                type="artifact.ready",
                tool_name="ppt",
                tool_detail=artifact_id,
                message=download_name,
            )

            download_link = f"/api/v1/artifacts/{artifact_id}?filename={download_name}"
            summary = (
                f"已根据上传的 **{len(parsed)}** 个文件生成汇总 PPT（约 **{result.slide_count}** 页）。\n\n"
                f"[下载 {download_name}]({download_link})"
            )
            yield execute_pb2.ExecuteEvent(type="message.done", content=summary)
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
        return execute_pb2.HealthResponse(status="ok", agent_id=AGENT_ID)


def _bind_grpc_port(server: grpc.Server, addr: str) -> None:
    try:
        bound = server.add_insecure_port(addr)
    except RuntimeError as e:
        raise RuntimeError(
            f"无法在 {addr} 绑定 gRPC Worker：{e}\n"
            "请在 deploy/config/local.yaml 中设置 agents.word2ppt.listen，例如 127.0.0.1:15065。"
        ) from e
    if bound == 0:
        raise RuntimeError(f"无法在 {addr} 绑定 gRPC Worker（add_insecure_port 返回 0）")


def serve(config_path: str | None, listen: str | None = None) -> None:
    _ = load_agent_config(config_path)
    addr = listen or worker_listen_address(config_path, AGENT_ID)
    servicer = Word2PPTServicer(config_path)
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
    execute_pb2_grpc.add_AgentWorkerServiceServicer_to_server(servicer, server)
    _bind_grpc_port(server, addr)
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    logger.info("word2ppt worker listening on %s", addr)
    server.start()
    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        server.stop(grace=3)


def _cli() -> None:
    import argparse
    from pathlib import Path

    from word2ppt_agent.config.loader import default_config_path

    parser = argparse.ArgumentParser(prog="word2ppt-agent")
    parser.add_argument("--config", default=str(default_config_path()))
    parser.add_argument("--listen", default=None)
    args = parser.parse_args()
    cfg = args.config if Path(args.config).is_file() else None
    serve(cfg, args.listen)


if __name__ == "__main__":
    _cli()
