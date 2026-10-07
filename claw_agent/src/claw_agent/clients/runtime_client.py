"""gRPC client for Phase A mock Runtime."""

from __future__ import annotations

import uuid
from dataclasses import dataclass
from typing import Iterator

import grpc

from claw_agent.pb.common.v1 import types_pb2
from claw_agent.pb.runtime.v1 import (
    data_pb2,
    data_pb2_grpc,
    execute_pb2,
    execute_pb2_grpc,
    meta_pb2,
    meta_pb2_grpc,
    template_pb2,
    template_pb2_grpc,
)


def _request_id() -> str:
    return str(uuid.uuid4())


@dataclass
class RuntimeClient:
    address: str = "127.0.0.1:50051"

    def __post_init__(self) -> None:
        self._channel = grpc.insecure_channel(self.address)
        self._template = template_pb2_grpc.RuntimeTemplateStub(self._channel)
        self._execute = execute_pb2_grpc.RuntimeExecuteStub(self._channel)
        self._meta = meta_pb2_grpc.RuntimeMetaStub(self._channel)
        self._data = data_pb2_grpc.RuntimeDataStub(self._channel)

    def close(self) -> None:
        self._channel.close()

    def validate_template(
        self,
        *,
        template_id: str = "",
        name: str = "",
        domain: str = "",
        json_payload: bytes = b"{}",
        session_id: str = "",
        vars: dict[str, str] | None = None,
        meta_only: bool = False,
    ) -> template_pb2.ValidateTemplateResponse:
        doc = None
        if not meta_only:
            doc = types_pb2.TemplateDoc(
                id=template_id,
                name=name,
                domain=domain,
                json_payload=json_payload,
            )
        req = template_pb2.ValidateTemplateRequest(
            request_id=_request_id(),
            template=doc,
            template_id=template_id if meta_only else "",
            vars=types_pb2.Vars(values=vars or {}),
        )
        return self._template.ValidateTemplate(req, metadata=_metadata(session_id))

    def upsert_template(
        self,
        *,
        template_id: str = "",
        name: str = "",
        json_payload: bytes = b"{}",
        session_id: str = "",
    ) -> template_pb2.UpsertTemplateResponse:
        doc = types_pb2.TemplateDoc(
            id=template_id,
            name=name,
            json_payload=json_payload,
        )
        req = template_pb2.UpsertTemplateRequest(
            request_id=_request_id(),
            template=doc,
        )
        return self._template.UpsertTemplate(req, metadata=_metadata(session_id))

    def run_template(
        self,
        *,
        template_id: str,
        session_id: str = "",
        engine: str = "http",
        proxy_id: str = "",
        idempotency_key: str = "",
        vars: dict[str, str] | None = None,
    ) -> execute_pb2.RunTemplateResponse:
        req = execute_pb2.RunTemplateRequest(
            request_id=_request_id(),
            session_id=session_id,
            idempotency_key=idempotency_key,
            template_id=template_id,
            vars=types_pb2.Vars(values=vars or {}),
            engine=engine,
            proxy_id=proxy_id,
        )
        return self._execute.RunTemplate(req, metadata=_metadata(session_id))

    def subscribe_run(
        self, *, run_id: str, session_id: str = ""
    ) -> Iterator[execute_pb2.RunEvent]:
        req = execute_pb2.SubscribeRunRequest(
            request_id=_request_id(),
            run_id=run_id,
        )
        return self._execute.SubscribeRun(req, metadata=_metadata(session_id))

    def cancel_run(self, *, run_id: str, session_id: str = "") -> None:
        req = execute_pb2.CancelRunRequest(request_id=_request_id(), run_id=run_id)
        self._execute.CancelRun(req, metadata=_metadata(session_id))

    def get_template(
        self, *, template_id: str, session_id: str = ""
    ) -> meta_pb2.GetTemplateResponse:
        req = meta_pb2.GetTemplateRequest(request_id=_request_id(), template_id=template_id)
        return self._meta.GetTemplate(req, metadata=_metadata(session_id))

    def list_templates(
        self, *, page: int = 1, size: int = 20, query: str = "", session_id: str = ""
    ) -> meta_pb2.ListTemplatesResponse:
        req = meta_pb2.ListTemplatesRequest(
            request_id=_request_id(), page=page, size=size, query=query
        )
        return self._meta.ListTemplates(req, metadata=_metadata(session_id))

    def get_token_data(
        self, *, token_id: str, session_id: str = ""
    ) -> data_pb2.GetTokenDataResponse:
        req = data_pb2.GetTokenDataRequest(request_id=_request_id(), token_id=token_id)
        return self._data.GetTokenData(req, metadata=_metadata(session_id))

    def list_proxies(
        self, *, page: int = 1, size: int = 50, status: str = "", session_id: str = ""
    ) -> meta_pb2.ListProxiesResponse:
        req = meta_pb2.ListProxiesRequest(
            request_id=_request_id(), page=page, size=size, status=status
        )
        return self._meta.ListProxies(req, metadata=_metadata(session_id))


def _metadata(session_id: str) -> list[tuple[str, str]]:
    md: list[tuple[str, str]] = []
    if session_id:
        md.append(("x-session-id", session_id))
    return md
