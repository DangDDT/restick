from __future__ import annotations

import json
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any


def listen(host: str, port: int, service_name: str) -> ThreadingHTTPServer:
    class HealthHandler(BaseHTTPRequestHandler):
        def log_message(self, format: str, *args: Any) -> None:
            return

        def do_GET(self) -> None:
            if self.path != "/healthz":
                self.send_response(404)
                self.end_headers()
                return
            body: bytes = json.dumps({"status": "ok", "service": service_name}).encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

    return ThreadingHTTPServer((host, port), HealthHandler)
