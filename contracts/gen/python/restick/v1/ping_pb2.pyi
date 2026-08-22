from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class PingRequest(_message.Message):
    __slots__ = ("from_service",)
    FROM_SERVICE_FIELD_NUMBER: _ClassVar[int]
    from_service: str
    def __init__(self, from_service: _Optional[str] = ...) -> None: ...

class PingResponse(_message.Message):
    __slots__ = ("service_name", "message")
    SERVICE_NAME_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    service_name: str
    message: str
    def __init__(self, service_name: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...
