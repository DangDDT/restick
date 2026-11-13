from __future__ import annotations

import hashlib
import json
from dataclasses import dataclass

from media_factory.domain.sticker import Sticker

SCHEMA: str = "restick.pack.v1"


@dataclass
class Pack:
    pack_id: str
    slug: str
    name: str
    author: str
    description: str
    tags: list[str]
    version: int
    stickers: list[Sticker]

    def checksum_sha256(self, sticker_bytes: list[bytes]) -> str:
        """Tính checksum SHA-256 của toàn bộ bytes sticker.

        Algorithm:
            1. Khởi tạo hasher sha256.
            2. update lần lượt theo thứ tự sticker.
            3. Trả hexdigest.

        Techniques:
            - Cryptographic hash (SHA-256)

        Keywords: SHA-256, checksum, hexdigest, integrity

        References:
            - https://en.wikipedia.org/wiki/SHA-2
            - https://en.wikipedia.org/wiki/Cryptographic_hash_function
            - https://en.wikipedia.org/wiki/Checksum
        """
        digest = hashlib.sha256()
        for payload in sticker_bytes:
            digest.update(payload)
        return digest.hexdigest()

    def to_json(self, checksum: str) -> str:
        """Serialize pack thành pack.json (schema restick.pack.v1).

        Algorithm:
            1. Dựng dict gồm metadata, checksum và danh sách sticker.
            2. json.dumps với indent=2, sort_keys=True, thêm xuống dòng cuối (đầu ra ổn định).

        Techniques:
            - JSON serialization xác định (deterministic)

        Keywords: pack.json, schema, serialization, deterministic

        References:
            - https://en.wikipedia.org/wiki/JSON
        """
        payload: dict[str, object] = {
            "schema": SCHEMA,
            "id": self.pack_id,
            "slug": self.slug,
            "name": self.name,
            "author": self.author,
            "description": self.description,
            "tags": self.tags,
            "ip_policy": "original_only",
            "version": self.version,
            "checksum_sha256": checksum,
            "stickers": [
                {
                    "file": item.filename,
                    "thumb": item.thumb,
                    "emoji_hint": item.emoji_hint,
                    "width": item.width,
                    "height": item.height,
                }
                for item in self.stickers
            ],
        }
        return json.dumps(payload, indent=2, sort_keys=True) + "\n"
