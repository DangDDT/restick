from __future__ import annotations

import re

import cv2
import numpy as np
from PIL import Image

from media_factory.infrastructure.layout import CANVAS_SIZE, fit_in_square

MAX_OUTLINE_WIDTH: int = 16
# Slack around the sticker so a dilated outline never touches the canvas edge.
OUTLINE_SLACK: int = 4
_COLOR_PATTERN: re.Pattern[str] = re.compile(r"^#[0-9A-Fa-f]{6}$")


def compose_sticker(image: Image.Image, outline_width: int, outline_color: str) -> Image.Image:
    """Đặt cutout vào canvas 512 rồi vẽ viền đặc quanh vùng alpha.

    Algorithm:
        1. Kiểm tra outline_width (0-16) và outline_color (#RRGGBB).
        2. fit_in_square với lề outline_width + OUTLINE_SLACK để viền không chạm mép.
        3. Nếu width = 0 trả ngay canvas.
        4. Dilate kênh alpha bằng elliptical structuring element (2w+1).
        5. Dải viền = pixel alpha 0 mà sau dilation > 0 (chỉ nằm ngoài vật thể).
        6. Tô màu viền, alpha_composite viền xuống dưới ảnh gốc.

    Techniques:
        - Morphological dilation với elliptical structuring element
        - Alpha compositing (over)

    Keywords: outline, stroke, dilation, structuring element, alpha composite, sticker border

    References:
        - https://en.wikipedia.org/wiki/Dilation_(morphology)
        - https://en.wikipedia.org/wiki/Structuring_element
        - https://en.wikipedia.org/wiki/Alpha_compositing
    """
    if outline_width < 0 or outline_width > MAX_OUTLINE_WIDTH:
        raise ValueError(f"outline_width must be 0-{MAX_OUTLINE_WIDTH}")
    if not _COLOR_PATTERN.match(outline_color):
        raise ValueError("outline_color must be #RRGGBB")
    canvas: Image.Image = fit_in_square(
        image.convert("RGBA"), CANVAS_SIZE, outline_width + OUTLINE_SLACK
    )
    if outline_width == 0:
        return canvas
    alpha = np.array(canvas.getchannel("A"))
    kernel = cv2.getStructuringElement(
        cv2.MORPH_ELLIPSE, (2 * outline_width + 1, 2 * outline_width + 1)
    )
    # Keep a fully opaque band strictly outside the foreground; interior alpha stays exact.
    dilated = cv2.dilate(alpha, kernel)
    band = np.where((alpha == 0) & (dilated > 0), 255, 0).astype(np.uint8)
    layer = Image.new(
        "RGBA",
        canvas.size,
        (
            int(outline_color[1:3], 16),
            int(outline_color[3:5], 16),
            int(outline_color[5:7], 16),
            255,
        ),
    )
    layer.putalpha(Image.fromarray(band, "L"))
    return Image.alpha_composite(layer, canvas)
