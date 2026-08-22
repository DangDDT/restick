from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class ValidateAPIKeyRequest(_message.Message):
    __slots__ = ("api_key",)
    API_KEY_FIELD_NUMBER: _ClassVar[int]
    api_key: str
    def __init__(self, api_key: _Optional[str] = ...) -> None: ...

class ValidateAPIKeyResponse(_message.Message):
    __slots__ = ("valid", "user_id", "key_id", "name")
    VALID_FIELD_NUMBER: _ClassVar[int]
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    KEY_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    valid: bool
    user_id: str
    key_id: str
    name: str
    def __init__(self, valid: _Optional[bool] = ..., user_id: _Optional[str] = ..., key_id: _Optional[str] = ..., name: _Optional[str] = ...) -> None: ...
