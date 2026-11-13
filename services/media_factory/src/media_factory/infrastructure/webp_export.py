from __future__ import annotations

from io import BytesIO

from PIL import Image


def encode_webp(image: Image.Image, lossless: bool) -> bytes:
    buffer = BytesIO()
    if lossless:
        image.save(buffer, format="WEBP", lossless=True, quality=100, method=6)
    else:
        image.save(buffer, format="WEBP", quality=100, method=6)
    return buffer.getvalue()
