from __future__ import annotations

import logging
import threading
import time

import grpc
from restick.v1 import ping_pb2, ping_pb2_grpc

from img_tool.infrastructure.config.config import Config
from img_tool.infrastructure.grpcclient.ping import peer_map
from img_tool.interfaces.grpc_server import build_server
from img_tool.interfaces.http_health import listen
from img_tool.interfaces.worker import run_worker


def _split_http_addr(addr: str) -> tuple[str, int]:
    trimmed: str = addr.strip()
    if trimmed.startswith(":"):
        return "0.0.0.0", int(trimmed[1:])
    host, _, port = trimmed.rpartition(":")
    if host == "":
        host = "0.0.0.0"
    return host, int(port)


def _ping_peers(cfg: Config) -> None:
    pending: dict[str, str] = {name: addr for name, addr in peer_map(cfg).items() if addr != ""}
    for attempt in range(1, 13):
        time.sleep(1)
        done: list[str] = []
        for name, addr in pending.items():
            try:
                channel = grpc.insecure_channel(addr)
                stub = ping_pb2_grpc.PingServiceStub(channel)
                stub.Ping(ping_pb2.PingRequest(from_service=cfg.service_name), timeout=5)
                logging.info("grpc peer ok peer=%s addr=%s", name, addr)
                done.append(name)
            except grpc.RpcError as exc:
                logging.warning(
                    "grpc peer unreachable peer=%s addr=%s attempt=%s err=%s",
                    name,
                    addr,
                    attempt,
                    exc,
                )
        for name in done:
            del pending[name]
        if len(pending) == 0:
            return


def _run_worker(cfg: Config) -> None:
    if cfg.api_svc_grpc_addr == "" or cfg.ftp_svc_grpc_addr == "":
        logging.warning("worker disabled: missing gRPC addresses")
        return
    run_worker(cfg)


def serve() -> None:
    logging.basicConfig(level=logging.INFO, format="%(message)s")
    cfg: Config = Config()
    host, port = _split_http_addr(cfg.http_addr)
    http_server = listen(host, port, cfg.service_name)
    http_thread = threading.Thread(target=http_server.serve_forever, daemon=True)
    http_thread.start()
    ping_thread = threading.Thread(target=_ping_peers, args=(cfg,), daemon=True)
    ping_thread.start()
    worker_thread = threading.Thread(target=_run_worker, args=(cfg,), daemon=True)
    worker_thread.start()
    grpc_bind: str = cfg.grpc_addr
    if grpc_bind.startswith(":"):
        grpc_bind = f"0.0.0.0{grpc_bind}"
    server = build_server(grpc_bind, cfg.service_name)
    logging.info("img_tool listening http=%s grpc=%s", cfg.http_addr, grpc_bind)
    server.start()
    server.wait_for_termination()
