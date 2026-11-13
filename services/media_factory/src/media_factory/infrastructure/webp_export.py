from __future__ import annotations

from io import BytesIO

from PIL import Image


def encode_webp(image: Image.Image, lossless: bool) -> bytes:
    """Mã hóa ảnh PIL thành bytes WebP.

    Algorithm:
        1. Lossless: save WEBP lossless=True (VP8L), quality=100.
        2. Lossy: save WEBP quality=100.
        3. method=6 là mức nỗ lực nén cao nhất (chậm hơn, file nhỏ hơn).

    Techniques:
        - WebP lossless (VP8L) và lossy (VP8)
        - Compression effort `method`

    Keywords: WebP, VP8L, lossless, method, encode, Pillow

    References:
        - https://en.wikipedia.org/wiki/WebP
        - https://developers.google.com/speed/webp/docs/riff_container
        - https://pillow.readthedocs.io/en/stable/handbook/image-file-formats.html
    """
    buffer = BytesIO()
    if lossless:
        image.save(buffer, format="WEBP", lossless=True, quality=100, method=6)
    else:
        image.save(buffer, format="WEBP", quality=100, method=6)
    return buffer.getvalue()
