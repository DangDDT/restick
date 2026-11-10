from __future__ import annotations

from pathlib import Path

import cv2
import numpy as np
from PIL import Image, ImageFilter

CANVAS_SIZE: int = 512
THUMB_SIZE: int = 128
MAX_SHEET_SIDE: int = 4096


def sort_boxes(boxes: list[tuple[int, int, int, int]]) -> list[tuple[int, int, int, int]]:
    return sorted(boxes, key=lambda box: (box[1], box[0]))


def filter_outlier_boxes(
    boxes: list[tuple[int, int, int, int]],
) -> list[tuple[int, int, int, int]]:
    if len(boxes) < 2:
        return boxes
    areas: list[int] = [(box[2] - box[0]) * (box[3] - box[1]) for box in boxes]
    ranked: list[int] = sorted(areas)
    median: int = ranked[len(ranked) // 2]
    floor: float = 0.25 * float(median)
    return [box for box, area in zip(boxes, areas, strict=True) if float(area) >= floor]


def upscale_source(image: Image.Image) -> Image.Image:
    width, height = image.size
    is_strip: bool = width > int(2.5 * height) or height > int(2.5 * width)
    short: int = min(width, height)
    if not is_strip or short >= CANVAS_SIZE:
        return image
    scale: float = min(
        float(CANVAS_SIZE) / float(short),
        MAX_SHEET_SIDE / float(max(width, height)),
    )
    new_width: int = max(1, int(round(width * scale)))
    new_height: int = max(1, int(round(height * scale)))
    return _resize_rgba(image.convert("RGBA"), new_width, new_height)


def fit_in_square(image: Image.Image, size: int, margin: int = 0) -> Image.Image:
    """Center the image in a transparent square, keeping `margin` px free on each side.

    alpha_composite keeps alpha truthful; pasting with the image as its own mask
    squares the alpha channel of semi-transparent pixels.
    """
    inner: int = max(1, size - 2 * margin)
    scaled: Image.Image = _scale_to_fit(image.convert("RGBA"), inner)
    canvas = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    left: int = (size - scaled.width) // 2
    top: int = (size - scaled.height) // 2
    canvas.alpha_composite(scaled, (left, top))
    return canvas


def fit_on_canvas(image: Image.Image) -> Image.Image:
    return fit_in_square(image, CANVAS_SIZE)


def make_thumb(image: Image.Image) -> Image.Image:
    return fit_in_square(image, THUMB_SIZE)


def assign_boxes(
    boxes: list[tuple[int, int, int, int]],
    rows: int,
    columns: int,
    size: tuple[int, int],
) -> list[tuple[int, int, int, int]]:
    """Map each box to the grid cell containing its centroid; return them row-major.

    A cell must hold exactly one box and a box may not bleed into another cell:
    an off-grid sheet fails loudly instead of being silently regrouped.
    """
    width, height = size
    cells: dict[int, tuple[int, int, int, int]] = {}
    for item in boxes:
        left, top, right, bottom = item
        column: int = min(columns - 1, max(0, int((left + right) * columns / (2 * width))))
        row: int = min(rows - 1, max(0, int((top + bottom) * rows / (2 * height))))
        if row * columns + column in cells:
            raise ValueError(f"two candidate boxes map to the same cell row={row} column={column}")
        for other_column in range(columns):
            if other_column == column:
                continue
            cell_left, _, cell_right, _ = _cell_bounds(row, other_column, rows, columns, size)
            if min(right, cell_right) - max(left, cell_left) > 0:
                raise ValueError(f"candidate box crosses into cell row={row} column={other_column}")
        for other_row in range(rows):
            if other_row == row:
                continue
            _, cell_top, _, cell_bottom = _cell_bounds(other_row, column, rows, columns, size)
            if min(bottom, cell_bottom) - max(top, cell_top) > 0:
                raise ValueError(f"candidate box crosses into cell row={other_row} column={column}")
        cells[row * columns + column] = item
    missing: int = rows * columns - len(cells)
    if missing > 0:
        raise ValueError(f"missing candidate boxes for {missing} of {rows * columns} cells")
    return [cells[index] for index in range(rows * columns)]


def _cell_bounds(
    row: int,
    column: int,
    rows: int,
    columns: int,
    size: tuple[int, int],
) -> tuple[float, float, float, float]:
    width, height = size
    return (
        column * width / columns,
        row * height / rows,
        (column + 1) * width / columns,
        (row + 1) * height / rows,
    )


def load_image(path: Path) -> Image.Image:
    image = Image.open(path)
    image.load()
    if max(image.size) > MAX_SHEET_SIDE:
        raise ValueError("sheet side exceeds 4096")
    return image.convert("RGBA")


def _scale_to_fit(image: Image.Image, size: int) -> Image.Image:
    width, height = image.size
    if width <= 0 or height <= 0:
        return image
    scale: float = min(float(size) / float(width), float(size) / float(height))
    new_width: int = max(1, int(round(width * scale)))
    new_height: int = max(1, int(round(height * scale)))
    if new_width == width and new_height == height:
        return image
    return _resize_rgba(image, new_width, new_height)


def _resize_rgba(image: Image.Image, width: int, height: int) -> Image.Image:
    source = np.array(image.convert("RGBA")).astype(np.float32)
    alpha = np.clip(source[:, :, 3:4] / 255.0, 0.0, 1.0)
    source[:, :, :3] *= alpha
    interpolation = (
        cv2.INTER_AREA if width * height < image.width * image.height else cv2.INTER_LANCZOS4
    )
    resized = cv2.resize(source, (width, height), interpolation=interpolation)
    out_alpha = np.clip(resized[:, :, 3:4], 0.0, 255.0)
    safe = np.maximum(out_alpha / 255.0, 1.0 / 255.0)
    resized[:, :, :3] = np.clip(resized[:, :, :3] / safe, 0.0, 255.0)
    resized[:, :, 3:4] = out_alpha
    empty = out_alpha[:, :, 0] < 1.0
    resized[empty, :3] = 0.0
    image_out = Image.fromarray(np.clip(np.round(resized), 0, 255).astype(np.uint8), "RGBA")
    if width * height > image.width * image.height:
        return _sharpen_rgb(image_out)
    return image_out


def _sharpen_rgb(image: Image.Image) -> Image.Image:
    rgb = image.convert("RGB").filter(ImageFilter.UnsharpMask(radius=1.0, percent=55, threshold=2))
    sharpened = rgb.convert("RGBA")
    sharpened.putalpha(image.getchannel("A"))
    return sharpened
