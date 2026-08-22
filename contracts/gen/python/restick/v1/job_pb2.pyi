from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ClaimJobRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ClaimJobResponse(_message.Message):
    __slots__ = ("empty", "job_id", "pack_id", "source_kind", "source_prefix", "name", "slug", "author", "tags")
    EMPTY_FIELD_NUMBER: _ClassVar[int]
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    PACK_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_KIND_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PREFIX_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    AUTHOR_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    empty: bool
    job_id: str
    pack_id: str
    source_kind: str
    source_prefix: str
    name: str
    slug: str
    author: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, empty: _Optional[bool] = ..., job_id: _Optional[str] = ..., pack_id: _Optional[str] = ..., source_kind: _Optional[str] = ..., source_prefix: _Optional[str] = ..., name: _Optional[str] = ..., slug: _Optional[str] = ..., author: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ...) -> None: ...

class StickerMeta(_message.Message):
    __slots__ = ("file", "thumb", "emoji_hint", "width", "height", "object_key", "thumb_key")
    FILE_FIELD_NUMBER: _ClassVar[int]
    THUMB_FIELD_NUMBER: _ClassVar[int]
    EMOJI_HINT_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    OBJECT_KEY_FIELD_NUMBER: _ClassVar[int]
    THUMB_KEY_FIELD_NUMBER: _ClassVar[int]
    file: str
    thumb: str
    emoji_hint: str
    width: int
    height: int
    object_key: str
    thumb_key: str
    def __init__(self, file: _Optional[str] = ..., thumb: _Optional[str] = ..., emoji_hint: _Optional[str] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., object_key: _Optional[str] = ..., thumb_key: _Optional[str] = ...) -> None: ...

class CompleteJobRequest(_message.Message):
    __slots__ = ("job_id", "checksum_sha256", "thumbnail_key", "bundle_key", "version", "stickers")
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    CHECKSUM_SHA256_FIELD_NUMBER: _ClassVar[int]
    THUMBNAIL_KEY_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_KEY_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    STICKERS_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    checksum_sha256: str
    thumbnail_key: str
    bundle_key: str
    version: int
    stickers: _containers.RepeatedCompositeFieldContainer[StickerMeta]
    def __init__(self, job_id: _Optional[str] = ..., checksum_sha256: _Optional[str] = ..., thumbnail_key: _Optional[str] = ..., bundle_key: _Optional[str] = ..., version: _Optional[int] = ..., stickers: _Optional[_Iterable[_Union[StickerMeta, _Mapping]]] = ...) -> None: ...

class CompleteJobResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class FailJobRequest(_message.Message):
    __slots__ = ("job_id", "error")
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    error: str
    def __init__(self, job_id: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...

class FailJobResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class UpsertCLIPackRequest(_message.Message):
    __slots__ = ("pack_id", "slug", "name", "author", "tags", "checksum_sha256", "thumbnail_key", "bundle_key", "version", "stickers")
    PACK_ID_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    AUTHOR_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    CHECKSUM_SHA256_FIELD_NUMBER: _ClassVar[int]
    THUMBNAIL_KEY_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_KEY_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    STICKERS_FIELD_NUMBER: _ClassVar[int]
    pack_id: str
    slug: str
    name: str
    author: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    checksum_sha256: str
    thumbnail_key: str
    bundle_key: str
    version: int
    stickers: _containers.RepeatedCompositeFieldContainer[StickerMeta]
    def __init__(self, pack_id: _Optional[str] = ..., slug: _Optional[str] = ..., name: _Optional[str] = ..., author: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., checksum_sha256: _Optional[str] = ..., thumbnail_key: _Optional[str] = ..., bundle_key: _Optional[str] = ..., version: _Optional[int] = ..., stickers: _Optional[_Iterable[_Union[StickerMeta, _Mapping]]] = ...) -> None: ...

class UpsertCLIPackResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
