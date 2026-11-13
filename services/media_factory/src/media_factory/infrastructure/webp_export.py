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


def encode_animated_webp(frames: list[Image.Image], fps: int, quality: int) -> bytes:
    """Mã hóa danh sách khung RGBA thành một WebP động (lossy + alpha).

    Algorithm:
        1. duration mỗi khung = round(1000 / fps) mili giây.
        2. Lưu khung đầu với save_all=True, append_images = các khung còn lại.
        3. Pillow/libwebp gộp các khung liên tiếp giống hệt nhau thành một ANMF có thời
           lượng cộng dồn (đã đo: 6 khung -> 3 khung, tổng thời lượng giữ nguyên), nên
           n_frames đọc lại có thể nhỏ hơn số khung truyền vào.
        4. method=4 (đường lưu animated của Pillow mặc định method=0, đo được file lớn
           hơn ~34% so với method=4; ảnh tĩnh mặc định đã là 4), alpha_quality=100 giữ
           viền trong suốt sắc, lossless=False, loop=0 (lặp vô hạn).

    Techniques:
        - Animated WebP (ANIM/ANMF chunks trong RIFF)
        - VP8 lossy + kênh alpha ALPH nén riêng
        - Compression effort `method`

    Keywords: animated WebP, ANMF, alpha_quality, method, loop, Pillow

    References:
        - https://en.wikipedia.org/wiki/WebP
        - https://developers.google.com/speed/webp/docs/riff_container
        - https://pillow.readthedocs.io/en/stable/handbook/image-file-formats.html
    """
    buffer = BytesIO()
    frames[0].save(
        buffer,
        format="WEBP",
        save_all=True,
        append_images=frames[1:],
        duration=round(1000 / fps),
        loop=0,
        quality=quality,
        alpha_quality=100,
        lossless=False,
        method=4,
    )
    return buffer.getvalue()
