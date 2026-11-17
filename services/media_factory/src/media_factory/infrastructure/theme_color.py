from __future__ import annotations

import colorsys

import numpy as np
from PIL import Image

DEFAULT_THEME_COLOR: str = "#EBE3FF"
SAMPLE_SIDE: int = 64
HUE_BINS: int = 36
MIN_ALPHA: int = 200
MIN_SATURATION: float = 0.2
BRIGHT_VALUE: float = 0.92
BRIGHT_MAX_SATURATION: float = 0.35
MIN_VALUE: float = 0.2
MIN_WEIGHT_RATIO: float = 0.01
PASTEL_LIGHTNESS: float = 0.90
PASTEL_SATURATION: float = 0.85
MAX_PASTEL_SATURATION: float = 0.9


def suggest_theme_color(images: list[Image.Image]) -> str:
    """Gợi ý màu nền pastel #RRGGBB (chữ hoa) từ tông màu chủ đạo của các sticker RGBA.

    Algorithm:
        1. Thu nhỏ từng ảnh về tối đa 64px, chỉ giữ pixel có alpha >= 200.
        2. Đổi RGB sang HSV; bỏ pixel xám (S < 0.2), viền trắng (V > 0.92 và S < 0.35)
           và mực (V < 0.2); màu rực rỡ (V cao nhưng S >= 0.35) được giữ lại.
        3. Gom hue vào 36 thùng, trọng số mỗi pixel = bão hoà (mỗi pixel một đơn vị diện tích).
        4. Tổng trọng số dưới 1% số pixel đặc: trả DEFAULT_THEME_COLOR.
        5. Chọn thùng nặng nhất, lấy hue trung bình có trọng số trong thùng đó.
        6. Trả pastel HSL với hue đó, L = 0.90, S = 0.85 (tối đa 0.9).

    Techniques:
        - Hue histogram (colour quantization theo hue)
        - Weighted mean (trọng số bão hoà x diện tích)
        - Chuyển đổi không gian màu HSV/HSL

    Keywords: theme color, dominant hue, pastel, HSV, HSL, hue histogram

    References:
        - https://en.wikipedia.org/wiki/HSL_and_HSV
        - https://en.wikipedia.org/wiki/Color_quantization
        - https://en.wikipedia.org/wiki/Pastel_(color)
    """
    weights = np.zeros(HUE_BINS)
    hue_sums = np.zeros(HUE_BINS)
    opaque = 0
    for image in images:
        small = image.convert("RGBA")
        small.thumbnail((SAMPLE_SIDE, SAMPLE_SIDE))
        pixels = np.asarray(small, dtype=np.float64).reshape(-1, 4)
        pixels = pixels[pixels[:, 3] >= MIN_ALPHA]
        opaque += len(pixels)
        rgb = pixels[:, :3] / 255.0
        top, bottom = rgb.max(axis=1), rgb.min(axis=1)
        chroma = top - bottom
        saturation = np.divide(chroma, top, out=np.zeros_like(top), where=top > 0)
        whitish = (top > BRIGHT_VALUE) & (saturation < BRIGHT_MAX_SATURATION)
        keep = (saturation >= MIN_SATURATION) & ~whitish & (top >= MIN_VALUE)
        rgb, top, chroma, saturation = rgb[keep], top[keep], chroma[keep], saturation[keep]
        red, green, blue = rgb[:, 0], rgb[:, 1], rgb[:, 2]
        hue = np.where(
            top == red,
            ((green - blue) / chroma) % 6,
            np.where(top == green, (blue - red) / chroma + 2, (red - green) / chroma + 4),
        ) * 60.0
        bins = np.minimum((hue / (360.0 / HUE_BINS)).astype(int), HUE_BINS - 1)
        np.add.at(weights, bins, saturation)
        np.add.at(hue_sums, bins, saturation * hue)
    if opaque == 0 or weights.sum() < MIN_WEIGHT_RATIO * opaque:
        return DEFAULT_THEME_COLOR
    best = int(weights.argmax())
    mean_hue = hue_sums[best] / weights[best]
    saturation = min(PASTEL_SATURATION, MAX_PASTEL_SATURATION)
    red, green, blue = colorsys.hls_to_rgb(mean_hue / 360.0, PASTEL_LIGHTNESS, saturation)
    return f"#{round(red * 255):02X}{round(green * 255):02X}{round(blue * 255):02X}"
