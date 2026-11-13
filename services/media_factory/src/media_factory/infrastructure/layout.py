from __future__ import annotations

from pathlib import Path

import cv2
import numpy as np
from PIL import Image, ImageFilter

CANVAS_SIZE: int = 512
THUMB_SIZE: int = 128
MAX_SHEET_SIDE: int = 4096


def sort_boxes(boxes: list[tuple[int, int, int, int]]) -> list[tuple[int, int, int, int]]:
    """Sắp các box theo thứ tự đọc: trên xuống dưới, trái sang phải.

    Algorithm:
        1. Sắp theo khóa (top, left).

    Techniques:
        - Sorting với khóa tuple

    Keywords: reading order, sort, bounding box

    References:
        - https://en.wikipedia.org/wiki/Sorting_algorithm
        - https://en.wikipedia.org/wiki/Minimum_bounding_box
    """
    return sorted(boxes, key=lambda box: (box[1], box[0]))


def filter_outlier_boxes(
    boxes: list[tuple[int, int, int, int]],
) -> list[tuple[int, int, int, int]]:
    """Loại các box nhỏ bất thường (nhiễu).

    Algorithm:
        1. Tính diện tích từng box.
        2. Lấy median diện tích.
        3. Giữ box có diện tích >= 25% median.

    Techniques:
        - Median-based outlier filter

    Keywords: outlier, median, area filter, noise

    References:
        - https://en.wikipedia.org/wiki/Median
        - https://en.wikipedia.org/wiki/Minimum_bounding_box
    """
    if len(boxes) < 2:
        return boxes
    areas: list[int] = [(box[2] - box[0]) * (box[3] - box[1]) for box in boxes]
    ranked: list[int] = sorted(areas)
    median: int = ranked[len(ranked) // 2]
    floor: float = 0.25 * float(median)
    return [box for box, area in zip(boxes, areas, strict=True) if float(area) >= floor]


def upscale_source(image: Image.Image) -> Image.Image:
    """Phóng to ảnh dạng dải (strip) quá mảnh trước khi tách nền.

    Algorithm:
        1. Dải = cạnh dài > 2.5 lần cạnh ngắn.
        2. Không phải dải hoặc cạnh ngắn >= CANVAS_SIZE thì giữ nguyên.
        3. Ngược lại scale để cạnh ngắn = CANVAS_SIZE, cạnh dài không vượt MAX_SHEET_SIDE.
        4. Resize bằng _resize_rgba (premultiplied alpha).

    Techniques:
        - Aspect-preserving upscaling
        - Image resampling

    Keywords: upscale, strip image, aspect ratio, resampling

    References:
        - https://en.wikipedia.org/wiki/Image_scaling
        - https://en.wikipedia.org/wiki/Aspect_ratio_(image)
    """
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
    """Đặt ảnh vào giữa canvas vuông trong suốt, chừa lề margin mỗi bên.

    Algorithm:
        1. Tính vùng trong = size - 2*margin.
        2. Co ảnh vừa vùng trong, giữ tỉ lệ (_scale_to_fit).
        3. Tạo canvas RGBA trong suốt, tính tọa độ căn giữa.
        4. alpha_composite ảnh lên canvas (không dùng paste với mask vì làm bình phương alpha).

    Techniques:
        - Aspect-preserving fit
        - Alpha compositing (Porter-Duff over)

    Keywords: fit, letterbox, center, aspect ratio, alpha composite

    References:
        - https://en.wikipedia.org/wiki/Aspect_ratio_(image)
        - https://en.wikipedia.org/wiki/Alpha_compositing
        - https://en.wikipedia.org/wiki/Image_scaling
    """
    inner: int = max(1, size - 2 * margin)
    scaled: Image.Image = _scale_to_fit(image.convert("RGBA"), inner)
    canvas = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    left: int = (size - scaled.width) // 2
    top: int = (size - scaled.height) // 2
    canvas.alpha_composite(scaled, (left, top))
    return canvas


def fit_on_canvas(image: Image.Image) -> Image.Image:
    """Đặt ảnh vào canvas sticker 512x512.

    Algorithm:
        1. Gọi fit_in_square với CANVAS_SIZE, không lề.

    Techniques:
        - Aspect-preserving fit

    Keywords: canvas, 512, fit, sticker

    References:
        - https://en.wikipedia.org/wiki/Aspect_ratio_(image)
        - https://en.wikipedia.org/wiki/Image_scaling
    """
    return fit_in_square(image, CANVAS_SIZE)


def make_thumb(image: Image.Image) -> Image.Image:
    """Tạo thumbnail sticker 128x128.

    Algorithm:
        1. Gọi fit_in_square với THUMB_SIZE (resize co ảnh, giữ tỉ lệ).

    Techniques:
        - Thumbnail resize
        - Aspect-preserving fit

    Keywords: thumbnail, resize, 128

    References:
        - https://en.wikipedia.org/wiki/Thumbnail
        - https://en.wikipedia.org/wiki/Image_scaling
    """
    return fit_in_square(image, THUMB_SIZE)


def assign_boxes(
    boxes: list[tuple[int, int, int, int]],
    rows: int,
    columns: int,
    size: tuple[int, int],
) -> list[tuple[int, int, int, int]]:
    """Gán mỗi box vào ô lưới (rows x columns) chứa tâm của nó, trả về theo thứ tự row-major.

    Algorithm:
        1. Với mỗi box tính centroid rồi suy ra cột/hàng trong lưới đều.
        2. Báo lỗi nếu hai box vào cùng một ô.
        3. Báo lỗi nếu box tràn sang ô khác (so với biên ô từ _cell_bounds).
        4. Báo lỗi nếu còn ô thiếu box.
        5. Trả danh sách theo chỉ số row*columns+column.

    Techniques:
        - Grid binning theo centroid
        - Row-major ordering
        - Fail-fast validation

    Keywords: grid, centroid, row-major, cell assignment, layout validation

    References:
        - https://en.wikipedia.org/wiki/Centroid
        - https://en.wikipedia.org/wiki/Row-_and_column-major_order
        - https://en.wikipedia.org/wiki/Fail-fast_system
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
    """Trả biên (left, top, right, bottom) của một ô lưới.

    Algorithm:
        1. Chia đều chiều rộng cho columns và chiều cao cho rows.
        2. Nhân với chỉ số cột/hàng.

    Techniques:
        - Phép chia lưới đều

    Keywords: grid cell, bounds, uniform grid

    References:
        - https://en.wikipedia.org/wiki/Row-_and_column-major_order
        - https://en.wikipedia.org/wiki/Minimum_bounding_box
    """
    width, height = size
    return (
        column * width / columns,
        row * height / rows,
        (column + 1) * width / columns,
        (row + 1) * height / rows,
    )


def load_image(path: Path) -> Image.Image:
    """Đọc ảnh từ đĩa, kiểm tra kích thước, đổi sang RGBA.

    Algorithm:
        1. Image.open rồi load() để giải mã hẳn.
        2. Từ chối nếu cạnh dài > MAX_SHEET_SIDE.
        3. convert('RGBA').

    Techniques:
        - Image decoding bằng Pillow
        - Input validation

    Keywords: load image, decode, size limit, RGBA

    References:
        - https://pillow.readthedocs.io/en/stable/handbook/image-file-formats.html
        - https://en.wikipedia.org/wiki/Input_validation
    """
    image = Image.open(path)
    image.load()
    if max(image.size) > MAX_SHEET_SIDE:
        raise ValueError("sheet side exceeds 4096")
    return image.convert("RGBA")


def _scale_to_fit(image: Image.Image, size: int) -> Image.Image:
    """Co/giãn ảnh vừa hình vuông size, giữ tỉ lệ.

    Algorithm:
        1. scale = min(size/width, size/height).
        2. Làm tròn kích thước mới, tối thiểu 1 px.
        3. Bỏ qua nếu kích thước không đổi; ngược lại _resize_rgba.

    Techniques:
        - Aspect-preserving scaling

    Keywords: scale to fit, aspect ratio, resize

    References:
        - https://en.wikipedia.org/wiki/Image_scaling
        - https://en.wikipedia.org/wiki/Aspect_ratio_(image)
    """
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
    """Resize ảnh RGBA đúng chuẩn alpha (premultiplied) để viền không bị quầng.

    Algorithm:
        1. Chuyển sang float, nhân RGB với alpha (premultiply).
        2. cv2.resize: INTER_AREA khi thu nhỏ, INTER_LANCZOS4 khi phóng to.
        3. Chia lại RGB cho alpha (un-premultiply), xóa RGB nơi alpha = 0.
        4. Phóng to thì làm nét nhẹ bằng _sharpen_rgb.

    Techniques:
        - Premultiplied alpha
        - Area/Lanczos resampling

    Keywords: premultiplied alpha, resize, INTER_AREA, Lanczos, resampling

    References:
        - https://en.wikipedia.org/wiki/Alpha_compositing
        - https://en.wikipedia.org/wiki/Lanczos_resampling
        - https://en.wikipedia.org/wiki/Interpolation
    """
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
    """Làm nét kênh RGB, giữ nguyên alpha.

    Algorithm:
        1. Tách RGB và áp UnsharpMask(radius=1, percent=55, threshold=2).
        2. Ghép lại với kênh alpha gốc.

    Techniques:
        - Unsharp masking

    Keywords: unsharp mask, sharpen, alpha preserved

    References:
        - https://en.wikipedia.org/wiki/Unsharp_masking
        - https://en.wikipedia.org/wiki/Image_scaling
    """
    rgb = image.convert("RGB").filter(ImageFilter.UnsharpMask(radius=1.0, percent=55, threshold=2))
    sharpened = rgb.convert("RGBA")
    sharpened.putalpha(image.getchannel("A"))
    return sharpened
