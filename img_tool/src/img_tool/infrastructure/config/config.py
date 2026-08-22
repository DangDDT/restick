from __future__ import annotations

import os


class Config:
    def __init__(self) -> None:
        self.service_name: str = env_or("SERVICE_NAME", "img_tool")
        self.http_addr: str = env_or("HTTP_ADDR", ":8080")
        self.grpc_addr: str = env_or("GRPC_ADDR", ":50053")
        self.api_svc_grpc_addr: str = os.getenv("API_SVC_GRPC_ADDR", "").strip()
        self.ftp_svc_grpc_addr: str = os.getenv("FTP_SVC_GRPC_ADDR", "").strip()
        self.img_tool_grpc_addr: str = os.getenv("IMG_TOOL_GRPC_ADDR", "").strip()
        self.internal_token: str = env_or("INTERNAL_TOKEN", "")
        self.ftp_http_url: str = env_or("FTP_HTTP_URL", "http://ftp_svc:8080")
        self.engine: str = env_or("IMG_TOOL_ENGINE", "simple")


def env_or(key: str, fallback: str) -> str:
    value: str = os.getenv(key, "").strip()
    if value == "":
        return fallback
    return value
