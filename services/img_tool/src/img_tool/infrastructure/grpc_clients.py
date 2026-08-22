from __future__ import annotations

import json
import sys
from pathlib import Path
from urllib.parse import urlencode
from urllib.request import Request, urlopen

import grpc

try:
    from restick.v1 import job_pb2, job_pb2_grpc, object_pb2, object_pb2_grpc
except ModuleNotFoundError:  # pragma: no cover - fallback for local/CI editable installs
    proto_root: Path = Path(__file__).resolve().parents[5] / "contracts" / "gen" / "python"
    sys.path.insert(0, str(proto_root))
    from restick.v1 import job_pb2, job_pb2_grpc, object_pb2, object_pb2_grpc

MAX_MSG: int = 16 * 1024 * 1024


def _channel(addr: str) -> grpc.Channel:
    return grpc.insecure_channel(
        addr,
        options=[
            ("grpc.max_receive_message_length", MAX_MSG),
            ("grpc.max_send_message_length", MAX_MSG),
        ],
    )


class ObjectClient:
    def __init__(self, addr: str, token: str) -> None:
        self._stub = object_pb2_grpc.ObjectServiceStub(_channel(addr))
        self._meta: tuple[tuple[str, str], ...] = (("x-internal-token", token),)

    def get(self, key: str) -> bytes:
        response = self._stub.GetObject(
            object_pb2.GetObjectRequest(key=key),
            metadata=self._meta,
            timeout=60,
        )
        return bytes(response.data)

    def put(self, key: str, data: bytes, content_type: str) -> None:
        self._stub.PutObject(
            object_pb2.PutObjectRequest(key=key, data=data, content_type=content_type),
            metadata=self._meta,
            timeout=60,
        )

    def build_bundle(self, pack_id: str, version: int, keys: list[str]) -> str:
        response = self._stub.BuildBundle(
            object_pb2.BuildBundleRequest(pack_id=pack_id, version=version, keys=keys),
            metadata=self._meta,
            timeout=60,
        )
        return str(response.bundle_key)


class JobClient:
    def __init__(self, addr: str, token: str) -> None:
        self._stub = job_pb2_grpc.JobServiceStub(_channel(addr))
        self._meta: tuple[tuple[str, str], ...] = (("x-internal-token", token),)

    def claim(self) -> job_pb2.ClaimJobResponse:
        return self._stub.ClaimJob(job_pb2.ClaimJobRequest(), metadata=self._meta, timeout=15)

    def complete(
        self,
        job_id: str,
        checksum: str,
        thumbnail_key: str,
        bundle_key: str,
        version: int,
        stickers: list[dict[str, object]],
    ) -> None:
        metas = [
            job_pb2.StickerMeta(
                file=str(item["file"]),
                thumb=str(item["thumb"]),
                emoji_hint=str(item["emoji_hint"]),
                width=int(item["width"]),
                height=int(item["height"]),
                object_key=str(item["object_key"]),
                thumb_key=str(item["thumb_key"]),
            )
            for item in stickers
        ]
        self._stub.CompleteJob(
            job_pb2.CompleteJobRequest(
                job_id=job_id,
                checksum_sha256=checksum,
                thumbnail_key=thumbnail_key,
                bundle_key=bundle_key,
                version=version,
                stickers=metas,
            ),
            metadata=self._meta,
            timeout=30,
        )

    def fail(self, job_id: str, message: str) -> None:
        self._stub.FailJob(
            job_pb2.FailJobRequest(job_id=job_id, error=message),
            metadata=self._meta,
            timeout=15,
        )

    def upsert_cli(
        self,
        pack_id: str,
        slug: str,
        name: str,
        author: str,
        tags: list[str],
        checksum: str,
        thumbnail_key: str,
        bundle_key: str,
        version: int,
        stickers: list[dict[str, object]],
    ) -> None:
        metas = [
            job_pb2.StickerMeta(
                file=str(item["file"]),
                thumb=str(item["thumb"]),
                emoji_hint=str(item["emoji_hint"]),
                width=int(item["width"]),
                height=int(item["height"]),
                object_key=str(item["object_key"]),
                thumb_key=str(item["thumb_key"]),
            )
            for item in stickers
        ]
        self._stub.UpsertCLIPack(
            job_pb2.UpsertCLIPackRequest(
                pack_id=pack_id,
                slug=slug,
                name=name,
                author=author,
                tags=tags,
                checksum_sha256=checksum,
                thumbnail_key=thumbnail_key,
                bundle_key=bundle_key,
                version=version,
                stickers=metas,
            ),
            metadata=self._meta,
            timeout=30,
        )


def list_prefix(ftp_http_url: str, token: str, prefix: str) -> list[str]:
    query = urlencode({"prefix": prefix})
    request = Request(
        f"{ftp_http_url.rstrip('/')}/internal/v1/objects?{query}",
        headers={"X-Internal-Token": token},
        method="GET",
    )
    with urlopen(request, timeout=30) as response:
        payload = json.loads(response.read().decode("utf-8"))
    keys = payload.get("keys", [])
    return [str(key) for key in keys]
