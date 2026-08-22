from __future__ import annotations

from img_tool.infrastructure.config.config import Config


def peer_map(cfg: Config) -> dict[str, str]:
    return {
        "api_svc": cfg.api_svc_grpc_addr,
        "ftp_svc": cfg.ftp_svc_grpc_addr,
    }
