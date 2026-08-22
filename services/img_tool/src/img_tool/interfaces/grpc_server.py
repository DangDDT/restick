from __future__ import annotations

from concurrent import futures

import grpc
from restick.v1 import ping_pb2, ping_pb2_grpc


class PingServicer(ping_pb2_grpc.PingServiceServicer):
    def __init__(self, service_name: str) -> None:
        self._service_name: str = service_name

    def Ping(
        self,
        request: ping_pb2.PingRequest,
        context: grpc.ServicerContext,
    ) -> ping_pb2.PingResponse:
        return ping_pb2.PingResponse(service_name=self._service_name, message="pong")


def build_server(grpc_bind: str, service_name: str) -> grpc.Server:
    server: grpc.Server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=4),
        options=[
            ("grpc.max_receive_message_length", 16 * 1024 * 1024),
            ("grpc.max_send_message_length", 16 * 1024 * 1024),
        ],
    )
    ping_pb2_grpc.add_PingServiceServicer_to_server(PingServicer(service_name), server)
    server.add_insecure_port(grpc_bind)
    return server
