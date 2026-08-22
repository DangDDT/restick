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
    engine: str = os.getenv("IMG_TOOL_ENGINE", "simple").strip().lower()
    if engine == "rembg":
        return _remove_with_rembg(image)
    return _remove_background(image)


def find_sticker_boxes(
    image: Image.Image,
    min_area: int,
    padding: int,
) -> list[tuple[int, int, int, int]]:
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
    rgba = np.array(image.convert("RGBA"))
    if _has_existing_alpha(rgba):
        return Image.fromarray(rgba, "RGBA")
    alpha = _exterior_flood_alpha(rgba)
    if _is_weak_mask(alpha):
        alpha = _ink_halo_alpha(rgba)
    rgba[:, :, 3] = alpha
    rgba[alpha < 1, :3] = 0
    return Image.fromarray(rgba, "RGBA")


def _has_existing_alpha(rgba: np.ndarray) -> bool:
    return bool(np.mean(rgba[:, :, 3] < 250) > _EXISTING_ALPHA_FRACTION)


def _exterior_flood_alpha(rgba: np.ndarray) -> np.ndarray:
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
    binary = (alpha > 32).astype(np.uint8)
    count, _, stats, _ = cv2.connectedComponentsWithStats(binary, connectivity=8)
    if count <= 1:
        return True
    areas: list[int] = [int(stats[index, cv2.CC_STAT_AREA]) for index in range(1, count)]
    median: int = sorted(areas)[len(areas) // 2]
    return float(median) < _WEAK_MEDIAN_FRACTION * float(alpha.size)


def _ink_halo_alpha(rgba: np.ndarray) -> np.ndarray:
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
    return halo.astype(np.uint8) * 255


def _merge_nearby_ink(
    labels: np.ndarray,
    stats: np.ndarray,
    large_ids: list[int],
    height: int,
    width: int,
) -> np.ndarray:
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
    from rembg import remove

    result = remove(image, alpha_matting=False)
    if isinstance(result, Image.Image):
        return result.convert("RGBA")
    return Image.fromarray(np.array(result)).convert("RGBA")
