from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class PutObjectRequest(_message.Message):
    __slots__ = ("key", "data", "content_type")
    KEY_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    key: str
    data: bytes
    content_type: str
    def __init__(self, key: _Optional[str] = ..., data: _Optional[bytes] = ..., content_type: _Optional[str] = ...) -> None: ...

class PutObjectResponse(_message.Message):
    __slots__ = ("key", "etag")
    KEY_FIELD_NUMBER: _ClassVar[int]
    ETAG_FIELD_NUMBER: _ClassVar[int]
    key: str
    etag: str
    def __init__(self, key: _Optional[str] = ..., etag: _Optional[str] = ...) -> None: ...

class GetObjectRequest(_message.Message):
    __slots__ = ("key",)
    KEY_FIELD_NUMBER: _ClassVar[int]
    key: str
    def __init__(self, key: _Optional[str] = ...) -> None: ...

class GetObjectResponse(_message.Message):
    __slots__ = ("data", "content_type", "etag")
    DATA_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    ETAG_FIELD_NUMBER: _ClassVar[int]
    data: bytes
    content_type: str
    etag: str
    def __init__(self, data: _Optional[bytes] = ..., content_type: _Optional[str] = ..., etag: _Optional[str] = ...) -> None: ...

class StatObjectRequest(_message.Message):
    __slots__ = ("key",)
    KEY_FIELD_NUMBER: _ClassVar[int]
    key: str
    def __init__(self, key: _Optional[str] = ...) -> None: ...

class StatObjectResponse(_message.Message):
    __slots__ = ("size", "etag", "content_type")
    SIZE_FIELD_NUMBER: _ClassVar[int]
    ETAG_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    size: int
    etag: str
    content_type: str
    def __init__(self, size: _Optional[int] = ..., etag: _Optional[str] = ..., content_type: _Optional[str] = ...) -> None: ...

class BuildBundleRequest(_message.Message):
    __slots__ = ("pack_id", "version", "keys")
    PACK_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    KEYS_FIELD_NUMBER: _ClassVar[int]
    pack_id: str
    version: int
    keys: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, pack_id: _Optional[str] = ..., version: _Optional[int] = ..., keys: _Optional[_Iterable[str]] = ...) -> None: ...

class BuildBundleResponse(_message.Message):
    __slots__ = ("bundle_key",)
    BUNDLE_KEY_FIELD_NUMBER: _ClassVar[int]
    bundle_key: str
    def __init__(self, bundle_key: _Optional[str] = ...) -> None: ...
