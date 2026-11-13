from __future__ import annotations

from bisect import bisect_left
from pathlib import Path

import cv2
from PIL import Image

from media_factory.application.process_job import ProcessingError


def read_frames(
    path: Path,
    fps: int,
    max_frames: int,
    max_side: int = 1920,
    max_seconds: float = 4.0,
) -> list[Image.Image]:
    """Đọc clip video thành danh sách khung RGBA, lấy mẫu theo timestamp.

    Algorithm:
        1. Mở file bằng cv2.VideoCapture; không mở được -> decode_failed.
        2. Pass 1: grab() từng khung (không giải mã sang ảnh), ghi POS_MSEC.
        3. duration = ts cuối + một khoảng khung; > max_seconds -> clip_too_long.
        4. Số khung ra = min(max_frames, round(duration * fps)); mốc i/fps giây.
        5. Với mỗi mốc chọn khung có timestamp gần nhất (bisect trên ts đã sắp xếp).
        6. Pass 2: mở lại, retrieve() đúng các khung đã chọn, BGR -> RGB -> RGBA.

    Techniques:
        - Lấy mẫu theo thời gian (không theo chỉ số) để video VFR của điện
          thoại vẫn cho đúng số khung
        - Nearest-neighbour temporal resampling
        - Hai pass để không giữ toàn bộ khung chưa nén trong RAM

    Keywords: OpenCV, VideoCapture, POS_MSEC, variable frame rate, resampling

    References:
        - https://en.wikipedia.org/wiki/Variable_frame_rate
        - https://en.wikipedia.org/wiki/Nearest-neighbor_interpolation
    """
    capture = cv2.VideoCapture(path.as_posix())
    if not capture.isOpened():
        raise ProcessingError("decode_failed", "Không đọc được video")
    try:
        width = int(capture.get(cv2.CAP_PROP_FRAME_WIDTH))
        height = int(capture.get(cv2.CAP_PROP_FRAME_HEIGHT))
        if max(width, height) > max_side:
            raise ProcessingError("clip_too_large", f"Video lớn hơn {max_side}px")
        times: list[float] = []
        while capture.grab():
            times.append(capture.get(cv2.CAP_PROP_POS_MSEC))
            if times[-1] / 1000 > max_seconds:
                raise ProcessingError("clip_too_long", f"Video dài hơn {max_seconds:g}s")
    finally:
        capture.release()
    if not times:
        raise ProcessingError("decode_failed", "Video không có khung hình")
    step = times[-1] - times[-2] if len(times) > 1 else 0.0
    duration = (times[-1] + step) / 1000
    if duration > max_seconds:
        raise ProcessingError("clip_too_long", f"Video dài hơn {max_seconds:g}s")

    count = max(1, min(max_frames, round(duration * fps)))
    picks: list[int] = []
    for i in range(count):
        t = i * 1000 / fps
        j = bisect_left(times, t)
        if j > 0 and (j == len(times) or t - times[j - 1] <= times[j] - t):
            j -= 1
        picks.append(j)
    wanted = set(picks)

    capture = cv2.VideoCapture(path.as_posix())
    decoded: dict[int, Image.Image] = {}
    try:
        index = 0
        while index <= max(wanted) and capture.grab():
            if index in wanted:
                ok, bgr = capture.retrieve()
                if not ok:
                    raise ProcessingError("decode_failed", "Không giải mã được khung hình")
                rgb = cv2.cvtColor(bgr, cv2.COLOR_BGR2RGB)
                decoded[index] = Image.fromarray(rgb).convert("RGBA")
            index += 1
    finally:
        capture.release()
    if len(decoded) != len(wanted):
        raise ProcessingError("decode_failed", "Video bị cắt giữa chừng")
    return [decoded[j].copy() for j in picks]
