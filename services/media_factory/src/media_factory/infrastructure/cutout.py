from __future__ import annotations

import os

import cv2
import numpy as np
from PIL import Image

_BG_THRESHOLD: int = 10
_INK_THRESHOLD: int = 22
_EXISTING_ALPHA_FRACTION: float = 0.02
_WEAK_MEDIAN_FRACTION: float = 0.008
_SMALL_ASSIGN_FRACTION: int = 20


def execute_cutout(image: Image.Image) -> Image.Image:
    """Tách nền khỏi ảnh, chọn engine theo biến môi trường.

    Algorithm:
        1. Đọc IMG_TOOL_ENGINE (mặc định 'simple').
        2. Nếu là 'rembg' thì gọi mô hình rembg, ngược lại dùng thuật toán thuần OpenCV.

    Techniques:
        - Strategy selection bằng biến môi trường
        - Background removal (cutout)

    Keywords: background removal, cutout, alpha matte, rembg, engine switch

    References:
        - https://en.wikipedia.org/wiki/Image_segmentation
    """
    engine: str = os.getenv("IMG_TOOL_ENGINE", "simple").strip().lower()
    if engine == "rembg":
        return _remove_with_rembg(image)
    return _remove_background(image)


def find_sticker_boxes(
    image: Image.Image,
    min_area: int,
    padding: int,
) -> list[tuple[int, int, int, int]]:
    """Tìm bounding box của từng sticker trong ảnh đã tách nền.

    Algorithm:
        1. Lấy kênh alpha, nhị phân hóa với ngưỡng alpha > 32.
        2. Morphological closing 3x3 để nối các nét đứt.
        3. findContours (RETR_EXTERNAL) lấy đường bao ngoài; bỏ contour nhỏ hơn min_area.
        4. Bỏ contour chạm viền ảnh và chiếm hơn 40% diện tích tờ (nền sót).
        5. Tính boundingRect rồi nới thêm padding, kẹp trong biên ảnh.

    Techniques:
        - Thresholding
        - Morphological closing
        - Contour extraction / bounding rectangle

    Keywords: contour, bounding box, morphological closing, binary threshold, connected region

    References:
        - https://en.wikipedia.org/wiki/Thresholding_(image_processing)
        - https://en.wikipedia.org/wiki/Closing_(morphology)
        - https://en.wikipedia.org/wiki/Minimum_bounding_box
    """
    rgba = np.array(image.convert("RGBA"))
    alpha = rgba[:, :, 3]
    binary = (alpha > 32).astype(np.uint8) * 255
    kernel = np.ones((3, 3), np.uint8)
    closed = cv2.morphologyEx(binary, cv2.MORPH_CLOSE, kernel, iterations=1)
    contours, _ = cv2.findContours(closed, cv2.RETR_EXTERNAL, cv2.CHAIN_APPROX_SIMPLE)
    height, width = alpha.shape
    sheet_area: int = width * height
    boxes: list[tuple[int, int, int, int]] = []
    for contour in contours:
        area = float(cv2.contourArea(contour))
        if area < float(min_area):
            continue
        x, y, w, h = cv2.boundingRect(contour)
        touches_border = x <= 0 or y <= 0 or x + w >= width - 1 or y + h >= height - 1
        if touches_border and area > 0.4 * float(sheet_area):
            continue
        left = max(0, x - padding)
        top = max(0, y - padding)
        right = min(width, x + w + padding)
        bottom = min(height, y + h + padding)
        boxes.append((left, top, right, bottom))
    return boxes


def _remove_background(image: Image.Image) -> Image.Image:
    """Cắt nền trắng bằng thuật toán thuần OpenCV/NumPy.

    Algorithm:
        1. Nếu ảnh đã có alpha (hơn 2% pixel trong suốt) thì giữ nguyên.
        2. Tính mặt nạ bằng exterior flood fill trên nền gần trắng.
        3. Nếu mặt nạ yếu (quá ít/nhỏ thành phần) thì chuyển sang ink-halo fallback.
        4. Làm mềm biên alpha sinh ra bằng Gaussian 3x3; giữ nguyên alpha có sẵn.
        5. Ghi alpha, xóa RGB của pixel trong suốt hoàn toàn.

    Techniques:
        - Exterior flood fill
        - Connected components
        - Ink-halo fallback

    Keywords: background removal, near-white, flood fill, alpha mask, fallback

    References:
        - https://en.wikipedia.org/wiki/Flood_fill
        - https://en.wikipedia.org/wiki/Connected-component_labeling
        - https://en.wikipedia.org/wiki/Image_segmentation
    """
    rgba = np.array(image.convert("RGBA"))
    if _has_existing_alpha(rgba):
        return Image.fromarray(rgba, "RGBA")
    alpha = _exterior_flood_alpha(rgba)
    if _is_weak_mask(alpha):
        alpha = _ink_halo_alpha(rgba)
    alpha = cv2.GaussianBlur(alpha, (3, 3), 0.7)
    rgba[:, :, 3] = alpha
    rgba[alpha < 1, :3] = 0
    return Image.fromarray(rgba, "RGBA")


def _has_existing_alpha(rgba: np.ndarray) -> bool:
    """Kiểm tra ảnh đã có nền trong suốt chưa.

    Algorithm:
        1. Tính tỉ lệ pixel có alpha < 250.
        2. So với ngưỡng 2%.

    Techniques:
        - Thresholding trên kênh alpha

    Keywords: alpha channel, transparency check, threshold

    References:
        - https://en.wikipedia.org/wiki/Alpha_compositing
        - https://en.wikipedia.org/wiki/Thresholding_(image_processing)
    """
    return bool(np.mean(rgba[:, :, 3] < 250) > _EXISTING_ALPHA_FRACTION)


def _exterior_flood_alpha(rgba: np.ndarray) -> np.ndarray:
    """Tạo alpha từ vùng nền ngoài: nền gần trắng nối với viền ảnh bị loại.

    Algorithm:
        1. Đo khoảng cách max-channel tới trắng; pixel <= _BG_THRESHOLD là nền ứng viên.
        2. Erode 3x3 để nền không dính vào nét vẽ mảnh.
        3. connectedComponents (connectivity=4) gán nhãn các vùng nền.
        4. Chỉ vùng nền chạm viền ảnh là 'exterior'; vùng nền kín bên trong sticker được giữ lại.
        5. Morphological closing 3x3 trên mặt nạ giữ để lấp lỗ nhỏ.

    Techniques:
        - Exterior flood fill bằng connected components
        - Morphological erosion/closing

    Keywords: flood fill, connected components, exterior background, erosion, closing

    References:
        - https://en.wikipedia.org/wiki/Flood_fill
        - https://en.wikipedia.org/wiki/Connected-component_labeling
        - https://en.wikipedia.org/wiki/Erosion_(morphology)
        - https://en.wikipedia.org/wiki/Closing_(morphology)
    """
    rgb = rgba[:, :, :3].astype(np.int16)
    diff = np.max(np.abs(rgb - 255), axis=2)
    is_bg = (diff <= _BG_THRESHOLD).astype(np.uint8)
    is_bg = cv2.erode(is_bg, np.ones((3, 3), np.uint8), iterations=1)
    _, labels = cv2.connectedComponents(is_bg, connectivity=4)
    border = np.unique(
        np.concatenate([labels[0, :], labels[-1, :], labels[:, 0], labels[:, -1]])
    )
    border = border[border != 0]
    exterior = np.isin(labels, border)
    keep = np.where(exterior, 0, 255).astype(np.uint8)
    return cv2.morphologyEx(keep, cv2.MORPH_CLOSE, np.ones((3, 3), np.uint8), iterations=1)


def _is_weak_mask(alpha: np.ndarray) -> bool:
    """Đánh giá mặt nạ flood fill có đáng tin không.

    Algorithm:
        1. Nhị phân hóa alpha (> 32) và đếm connected components (connectivity=8).
        2. Không có thành phần nào thì yếu.
        3. Lấy median diện tích các thành phần; nhỏ hơn _WEAK_MEDIAN_FRACTION diện tích ảnh thì yếu.

    Techniques:
        - Connected components với stats
        - Median làm thước đo chống nhiễu

    Keywords: connected components, median area, mask quality, heuristic

    References:
        - https://en.wikipedia.org/wiki/Connected-component_labeling
        - https://en.wikipedia.org/wiki/Median
    """
    binary = (alpha > 32).astype(np.uint8)
    count, _, stats, _ = cv2.connectedComponentsWithStats(binary, connectivity=8)
    if count <= 1:
        return True
    areas: list[int] = [int(stats[index, cv2.CC_STAT_AREA]) for index in range(1, count)]
    median: int = sorted(areas)[len(areas) // 2]
    return float(median) < _WEAK_MEDIAN_FRACTION * float(alpha.size)


def _ink_halo_alpha(rgba: np.ndarray) -> np.ndarray:
    """Fallback khi nền không sạch: tạo alpha từ nét mực và quầng (halo) quanh chúng.

    Algorithm:
        1. Ink = pixel cách trắng hơn _INK_THRESHOLD.
        2. Gán nhãn connected components; giữ thành phần lớn (hoặc tối đa 9 thành phần lớn nhất).
        3. Gộp mảnh nhỏ gần thành phần lớn (_merge_nearby_ink).
        4. Với mỗi thành phần lớn tính distance transform; mỗi pixel thuộc thành phần gần nhất.
        5. Giữ pixel trong bán kính halo và gần chủ sở hữu hơn thành phần thứ hai.
        6. Tô kín contour ngoài để vùng trắng trong nhân vật không bị thủng.

    Techniques:
        - Distance transform (L2)
        - Connected components
        - Nearest-owner assignment

    Keywords: ink mask, halo, distance transform, connected components, voronoi-like ownership

    References:
        - https://en.wikipedia.org/wiki/Distance_transform
        - https://en.wikipedia.org/wiki/Connected-component_labeling
        - https://en.wikipedia.org/wiki/Euclidean_distance
    """
    rgb = rgba[:, :, :3].astype(np.int16)
    diff = np.max(np.abs(rgb - 255), axis=2)
    ink = (diff > _INK_THRESHOLD).astype(np.uint8)
    count, labels, stats, _ = cv2.connectedComponentsWithStats(ink, connectivity=8)
    height, width = ink.shape
    min_large: int = max(800, int(_WEAK_MEDIAN_FRACTION * float(height * width)))
    large_ids: list[int] = [
        index
        for index in range(1, count)
        if int(stats[index, cv2.CC_STAT_AREA]) >= min_large
    ]
    if len(large_ids) == 0:
        ranked: list[int] = sorted(
            range(1, count),
            key=lambda index: int(stats[index, cv2.CC_STAT_AREA]),
            reverse=True,
        )
        large_ids = ranked[:9]
    if len(large_ids) == 0:
        return (ink * 255).astype(np.uint8)
    merged = _merge_nearby_ink(labels, stats, large_ids, height, width)
    radius: float = float(max(8, min(height, width) // 64))
    nearest = np.full((height, width), np.inf, dtype=np.float32)
    second = np.full((height, width), np.inf, dtype=np.float32)
    owner = np.zeros((height, width), dtype=np.int32)
    for label in large_ids:
        dist = cv2.distanceTransform(
            np.where(merged == label, 0, 255).astype(np.uint8),
            cv2.DIST_L2,
            5,
        )
        closer = dist < nearest
        second[closer] = nearest[closer]
        nearest[closer] = dist[closer]
        owner[closer] = label
        between = (~closer) & (dist < second)
        second[between] = dist[between]
    halo = (owner > 0) & (nearest <= radius) & (nearest < second)
    alpha = halo.astype(np.uint8) * 255
    contours, _ = cv2.findContours(alpha, cv2.RETR_EXTERNAL, cv2.CHAIN_APPROX_SIMPLE)
    cv2.drawContours(alpha, contours, -1, 255, cv2.FILLED)
    return alpha


def _merge_nearby_ink(
    labels: np.ndarray,
    stats: np.ndarray,
    large_ids: list[int],
    height: int,
    width: int,
) -> np.ndarray:
    """Gán các mảnh mực nhỏ vào thành phần lớn gần nhất để một sticker không bị vỡ.

    Algorithm:
        1. Khởi tạo bản đồ merged chứa các thành phần lớn.
        2. Tính distance transform tới tập thành phần lớn và tới từng thành phần.
        3. Bỏ mảnh diện tích < 20 hoặc xa hơn assign_limit.
        4. Gán mảnh vào thành phần lớn có khoảng cách nhỏ nhất.

    Techniques:
        - Distance transform (L2)
        - Nearest-neighbour assignment

    Keywords: distance transform, merge fragments, nearest component, connected components

    References:
        - https://en.wikipedia.org/wiki/Distance_transform
        - https://en.wikipedia.org/wiki/Connected-component_labeling
    """
    large_set: set[int] = set(large_ids)
    merged = np.zeros_like(labels)
    for label in large_ids:
        merged[labels == label] = label
    assign_limit: float = float(max(24, min(height, width) // _SMALL_ASSIGN_FRACTION))
    large_mask = np.isin(labels, large_ids).astype(np.uint8)
    dist_large = cv2.distanceTransform(
        np.where(large_mask > 0, 0, 255).astype(np.uint8),
        cv2.DIST_L2,
        5,
    )
    dist_by_label: dict[int, np.ndarray] = {
        label: cv2.distanceTransform(
            np.where(labels == label, 0, 255).astype(np.uint8),
            cv2.DIST_L2,
            5,
        )
        for label in large_ids
    }
    for small_id in range(1, int(stats.shape[0])):
        if small_id in large_set:
            continue
        if int(stats[small_id, cv2.CC_STAT_AREA]) < 20:
            continue
        members = labels == small_id
        if float(dist_large[members].min()) > assign_limit:
            continue
        best_id: int = large_ids[0]
        best_dist: float = 1e9
        for label in large_ids:
            current: float = float(dist_by_label[label][members].min())
            if current < best_dist:
                best_dist = current
                best_id = label
        merged[members] = best_id
    return merged


def _remove_with_rembg(image: Image.Image) -> Image.Image:
    """Tách nền bằng thư viện rembg (mô hình học sâu).

    Algorithm:
        1. Import rembg khi cần (lazy import).
        2. Gọi remove với alpha_matting=False.
        3. Chuẩn hóa kết quả về PIL RGBA.

    Techniques:
        - Lazy import
        - Deep-learning salient object segmentation

    Keywords: rembg, background removal, segmentation, lazy import

    References:
        - https://en.wikipedia.org/wiki/Image_segmentation
    """
    from rembg import remove

    result = remove(image, alpha_matting=False)
    if isinstance(result, Image.Image):
        return result.convert("RGBA")
    return Image.fromarray(np.array(result)).convert("RGBA")
