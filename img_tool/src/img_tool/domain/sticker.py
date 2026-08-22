from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class Sticker:
    filename: str
    thumb: str
    emoji_hint: str
    width: int
    height: int
    rgba: bytes
