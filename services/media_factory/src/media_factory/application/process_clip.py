from __future__ import annotations

import json
from dataclasses import dataclass
from pathlib import Path

import numpy as np
from PIL import Image

from media_factory.application.process_job import (
    COLOR_PATTERN,
    MAX_FOREGROUND_RATIO,
    MAX_REQUEST_BYTES,
    MIN_FOREGROUND_RATIO,
    ProcessingError,
    _absolute_path,
    _integer,
    _text,
    _write_error_result,
    _write_result,
)
from media_factory.infrastructure.cutout import execute_cutout
from media_factory.infrastructure.outline import MAX_OUTLINE_WIDTH, compose_sticker
from media_factory.infrastructure.video import read_frames
from media_factory.infrastructure.webp_export import encode_animated_webp

REQUEST_SCHEMA: str = "restick.clip-request.v1"
RESULT_SCHEMA: str = "restick.clip-result.v1"
CEILING_BYTES: int = 512_000
MAX_CLIP_BYTES: int = 32 << 20
MAX_CLIP_SECONDS: int = 4
MAX_CLIP_FPS: int = 24
FALLBACK_QUALITY: int = 60
FALLBACK_FPS: tuple[int, ...] = (10, 8)
ALPHA_THRESHOLD: int = 32


@dataclass(frozen=True)
class ClipRequest:
    job_id: str
    pack_id: str
    sticker_index: int
    clip_path: Path
    output_dir: Path
    fps: int
    quality: int
    outline_width: int
    outline_color: str


def process_clip(request_path: Path, result_path: Path) -> None:
    """Chạy một job clip theo giao thức restick.clip-request.v1, ghi restick.clip-result.v1.

    Algorithm:
        1. Khởi tạo identity rỗng (schema, job_id, pack_id).
        2. read_clip_request rồi _render_clip.
        3. ProcessingError: ghi kết quả lỗi rồi raise lại.
        4. Lỗi bất ngờ: bọc thành internal_error, ghi kết quả lỗi.
        5. Thành công: ghi kết quả nguyên tử qua _write_result.

    Techniques:
        - Request/result file protocol (JSON)
        - Atomic write (temp file + rename)
        - Coded error handling

    Keywords: machine job, protocol, restick.clip-request.v1, atomic write, error code

    References:
        - https://en.wikipedia.org/wiki/JSON
        - https://en.wikipedia.org/wiki/Atomicity_(database_systems)
        - https://en.wikipedia.org/wiki/Error_code
    """
    identity: dict[str, object] = {"schema": RESULT_SCHEMA, "job_id": "", "pack_id": ""}
    try:
        request = read_clip_request(request_path, identity)
        animation, diagnostics = _render_clip(request)
    except ProcessingError as exc:
        _write_error_result(result_path, identity, exc)
        raise
    except Exception as exc:  # noqa: BLE001 - the protocol must always report a code
        failure = ProcessingError("internal_error", f"unexpected {type(exc).__name__}")
        _write_error_result(result_path, identity, failure)
        raise failure from exc
    _write_result(
        result_path, {**identity, "animation": animation, "diagnostics": diagnostics}
    )


def read_clip_request(request_path: Path, identity: dict[str, object]) -> ClipRequest:
    """Đọc và kiểm tra request JSON của clip job, trả ClipRequest.

    Algorithm:
        1. Kiểm tra file tồn tại, <= 1 MiB, parse JSON object.
        2. Ghi sớm job_id/pack_id vào identity để Go loại kết quả cũ.
        3. Kiểm tra schema, sticker_index, fps (6-24, mặc định 12), quality (30-95,
           mặc định 75), outline_width, outline_color.
        4. clip_path (file <= 32 MiB) và output_dir (thư mục) phải tuyệt đối, tồn tại,
           không '..', không symlink.

    Techniques:
        - Input validation tại trust boundary
        - JSON parsing
        - Regular expression

    Keywords: request validation, JSON, schema, path validation, defaults

    References:
        - https://en.wikipedia.org/wiki/Input_validation
        - https://en.wikipedia.org/wiki/JSON
        - https://en.wikipedia.org/wiki/Regular_expression
    """
    if not request_path.is_file():
        raise ProcessingError("invalid_request", "request file is missing")
    if request_path.stat().st_size > MAX_REQUEST_BYTES:
        raise ProcessingError("invalid_request", "request exceeds 1 MiB")
    try:
        payload = json.loads(request_path.read_text(encoding="utf-8"))
    except (OSError, UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise ProcessingError("invalid_request", f"request is not readable JSON: {exc}") from exc
    if not isinstance(payload, dict):
        raise ProcessingError("invalid_request", "request must be a JSON object")
    for key in ("job_id", "pack_id"):
        value = payload.get(key)
        if isinstance(value, str):
            identity[key] = value
    if payload.get("schema") != REQUEST_SCHEMA:
        raise ProcessingError("invalid_schema", f"schema must be {REQUEST_SCHEMA}")
    payload.setdefault("fps", 12)
    payload.setdefault("quality", 75)
    outline_color = _text(payload, "outline_color")
    if not COLOR_PATTERN.match(outline_color):
        raise ProcessingError("invalid_field", "outline_color must be #RRGGBB")
    clip_path = _absolute_path(payload, "clip_path")
    if not clip_path.is_file():
        raise ProcessingError("invalid_path", "clip_path must be an existing file")
    if clip_path.stat().st_size > MAX_CLIP_BYTES:
        raise ProcessingError("clip_too_large", "clip exceeds 32 MiB")
    output_dir = _absolute_path(payload, "output_dir")
    if not output_dir.is_dir():
        raise ProcessingError("invalid_path", "output_dir must be an existing directory")
    return ClipRequest(
        job_id=_text(payload, "job_id"),
        pack_id=_text(payload, "pack_id"),
        sticker_index=_integer(payload, "sticker_index", 0, 999),
        clip_path=clip_path,
        output_dir=output_dir,
        fps=_integer(payload, "fps", 6, MAX_CLIP_FPS),
        quality=_integer(payload, "quality", 30, 95),
        outline_width=_integer(payload, "outline_width", 0, MAX_OUTLINE_WIDTH),
        outline_color=outline_color,
    )


def shared_box(alphas: list[np.ndarray]) -> tuple[int, int, int, int]:
    """Hợp bounding box của vùng alpha > 32 trên mọi khung.

    Algorithm:
        1. Với mỗi khung lấy mask alpha > 32, tìm các hàng/cột có pixel.
        2. Lấy min/max trên toàn bộ khung; trả (left, top, right, bottom), right/bottom
           là biên ngoài (exclusive) như Image.crop.
        3. Không khung nào có pixel thì ValueError.

    Techniques:
        - Bounding box union
        - Thresholded alpha mask

    Keywords: bounding box, union, crop, alpha mask, stable framing

    References:
        - https://en.wikipedia.org/wiki/Minimum_bounding_box
    """
    boxes: list[tuple[int, int, int, int]] = []
    for alpha in alphas:
        mask = alpha > ALPHA_THRESHOLD
        rows = np.flatnonzero(mask.any(axis=1))
        cols = np.flatnonzero(mask.any(axis=0))
        if rows.size:
            boxes.append((int(cols[0]), int(rows[0]), int(cols[-1]) + 1, int(rows[-1]) + 1))
    if not boxes:
        raise ValueError("no foreground in any frame")
    return (
        min(b[0] for b in boxes),
        min(b[1] for b in boxes),
        max(b[2] for b in boxes),
        max(b[3] for b in boxes),
    )


def fit_under_ceiling(
    frames: list[Image.Image], fps: int, quality: int, ceiling: int = CEILING_BYTES
) -> tuple[bytes, int, int]:
    """Mã hóa animated WebP, hạ dần chất lượng/fps tới khi <= ceiling.

    Algorithm:
        1. Thang thử: (fps, quality), (fps, q60), (10, q60), (8, q60); bỏ bước không
           giảm thật sự so với bước trước (quality/fps đã thấp hơn).
        2. Hạ fps bằng cách lấy mẫu lại chỉ số khung i * fps_cũ / fps_mới (giữ thời lượng).
        3. Bước đầu tiên có dung lượng <= ceiling thì trả (data, fps_used, quality_used).
        4. Hết thang vẫn lớn thì báo output_too_large.

    Techniques:
        - Rate-quality trade-off ladder
        - Temporal subsampling

    Keywords: size ceiling, quality ladder, frame rate reduction, subsampling, WebP

    References:
        - https://en.wikipedia.org/wiki/Rate%E2%80%93distortion_theory
        - https://en.wikipedia.org/wiki/Downsampling_(signal_processing)
    """
    steps: list[tuple[int, int]] = [(fps, quality)]
    low_quality = min(quality, FALLBACK_QUALITY)
    steps.append((fps, low_quality))
    steps.extend((f, low_quality) for f in FALLBACK_FPS if f < fps)
    last = (0, 0)
    for step_fps, step_quality in steps:
        if (step_fps, step_quality) == last:
            continue
        last = (step_fps, step_quality)
        count = max(1, round(len(frames) * step_fps / fps))
        picked = [frames[min(len(frames) - 1, round(i * fps / step_fps))] for i in range(count)]
        data = encode_animated_webp(picked, step_fps, step_quality)
        if len(data) <= ceiling:
            return data, step_fps, step_quality
    raise ProcessingError("output_too_large", f"animation exceeds {ceiling} bytes")


def _render_clip(request: ClipRequest) -> tuple[dict[str, object], dict[str, object]]:
    """Đọc clip, tách nền từng khung, cắt chung, dựng sticker, nén và ghi animated/NNN.webp.

    Algorithm:
        1. read_frames (tối đa 4 s * 24 fps khung), execute_cutout mỗi khung.
        2. Tỉ lệ foreground mỗi khung phải trong [0.005, 0.995], ngược lại no_foreground
           (libwebp bỏ kênh alpha khi khung gần như trống).
        3. shared_box trên mọi khung, crop như nhau để vật thể không nhảy khung.
        4. compose_sticker từng khung (cùng kích thước crop nên cùng tỉ lệ co).
        5. fit_under_ceiling; ghi file, OSError thành output_failed.
        6. mask_iou_min = IoU nhỏ nhất của mask alpha hai khung liên tiếp.

    Techniques:
        - Per-frame foreground-ratio sanity check
        - Shared crop window
        - Intersection over Union (IoU)

    Keywords: animated sticker, cutout, shared box, IoU, temporal stability

    References:
        - https://en.wikipedia.org/wiki/Jaccard_index
        - https://en.wikipedia.org/wiki/Minimum_bounding_box
        - https://en.wikipedia.org/wiki/Exception_handling
    """
    frames = read_frames(
        request.clip_path, request.fps, MAX_CLIP_SECONDS * MAX_CLIP_FPS,
        max_seconds=float(MAX_CLIP_SECONDS),
    )
    cutouts = [execute_cutout(frame) for frame in frames]
    alphas = [np.array(c.getchannel("A")) for c in cutouts]
    ratios = [float(np.mean(a > ALPHA_THRESHOLD)) for a in alphas]
    if any(not MIN_FOREGROUND_RATIO <= r <= MAX_FOREGROUND_RATIO for r in ratios):
        raise ProcessingError(
            "no_foreground",
            "a frame has no usable foreground; use a plain near-white background",
        )
    box = shared_box(alphas)
    cropped = [c.crop(box) for c in cutouts]
    masks = [np.array(c.getchannel("A")) > ALPHA_THRESHOLD for c in cropped]
    ious = [
        float(np.logical_and(a, b).sum() / max(1, np.logical_or(a, b).sum()))
        for a, b in zip(masks, masks[1:], strict=False)
    ]
    canvases = [
        compose_sticker(c, request.outline_width, request.outline_color) for c in cropped
    ]
    data, fps_used, quality_used = fit_under_ceiling(canvases, request.fps, request.quality)
    filename = f"{request.sticker_index:03d}.webp"
    target = request.output_dir / "animated" / filename
    try:
        target.parent.mkdir(exist_ok=True)
        target.write_bytes(data)
    except OSError as exc:
        target.unlink(missing_ok=True)
        raise ProcessingError("output_failed", f"cannot write artifacts: {exc.strerror}") from exc
    with Image.open(target) as written:
        count = getattr(written, "n_frames", 1)
    animation: dict[str, object] = {
        "filename": f"animated/{filename}",
        "frames": count,
        "fps": fps_used,
        "duration_ms": round(count * 1000 / fps_used),
        "byte_size": len(data),
        "quality": quality_used,
        "width": canvases[0].width,
        "height": canvases[0].height,
    }
    diagnostics: dict[str, object] = {
        "foreground_ratio_min": round(min(ratios), 4),
        "mask_iou_min": round(min(ious, default=1.0), 4),
    }
    return animation, diagnostics
