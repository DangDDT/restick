from __future__ import annotations

import json
import os
import re
from dataclasses import dataclass
from pathlib import Path

import numpy as np
from PIL import Image

from img_tool.infrastructure.cutout import execute_cutout, find_sticker_boxes
from img_tool.infrastructure.layout import MAX_SHEET_SIDE, assign_boxes, make_thumb
from img_tool.infrastructure.outline import MAX_OUTLINE_WIDTH, compose_sticker
from img_tool.infrastructure.webp_export import encode_webp

REQUEST_SCHEMA: str = "restick.job.v1"
RESULT_SCHEMA: str = "restick.job-result.v1"
MAX_REQUEST_BYTES: int = 1 << 20
MAX_RESULT_BYTES: int = 1 << 20
MAX_SHEET_BYTES: int = 8 << 20
MAX_GRID: int = 8
ALLOWED_FORMATS: frozenset[str] = frozenset({"PNG", "JPEG", "WEBP"})
COLOR_PATTERN: re.Pattern[str] = re.compile(r"^#[0-9A-Fa-f]{6}$")
MIN_FOREGROUND_RATIO: float = 0.005
MAX_FOREGROUND_RATIO: float = 0.995
# Noise below this share of one cell's area is not a sticker candidate.
MIN_CELL_AREA_FRACTION: float = 0.005


class ProcessingError(Exception):
    """Coded, manager-facing failure. `code` is stable for the machine protocol."""

    def __init__(self, code: str, message: str) -> None:
        super().__init__(message)
        self.code = code
        self.message = message


@dataclass(frozen=True)
class JobRequest:
    job_id: str
    pack_id: str
    version: int
    sheet_path: Path
    output_dir: Path
    rows: int
    columns: int
    outline_width: int
    outline_color: str


def process_job(request_path: Path, result_path: Path) -> None:
    """Process one sheet job and write `restick.job-result.v1`; raise on failure."""
    identity: dict[str, object] = {
        "schema": RESULT_SCHEMA,
        "job_id": "",
        "pack_id": "",
        "version": 0,
    }
    try:
        request = read_request(request_path, identity)
        stickers, diagnostics = _render(request)
    except ProcessingError as exc:
        _write_error_result(result_path, identity, exc)
        raise
    except Exception as exc:  # noqa: BLE001 - the protocol must always report a code
        failure = ProcessingError("internal_error", f"unexpected {type(exc).__name__}")
        _write_error_result(result_path, identity, failure)
        raise failure from exc
    _write_result(
        result_path,
        {**identity, "stickers": stickers, "diagnostics": diagnostics},
    )


def read_request(request_path: Path, identity: dict[str, object]) -> JobRequest:
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
    # Echo identity as soon as it is readable so Go can reject stale result files.
    for key in ("job_id", "pack_id"):
        value = payload.get(key)
        if isinstance(value, str):
            identity[key] = value
    version = payload.get("version")
    if isinstance(version, int) and not isinstance(version, bool):
        identity["version"] = version
    if payload.get("schema") != REQUEST_SCHEMA:
        raise ProcessingError("invalid_schema", f"schema must be {REQUEST_SCHEMA}")
    rows = _integer(payload, "rows", 1, MAX_GRID)
    columns = _integer(payload, "columns", 1, MAX_GRID)
    outline_color = _text(payload, "outline_color")
    if not COLOR_PATTERN.match(outline_color):
        raise ProcessingError("invalid_field", "outline_color must be #RRGGBB")
    sheet_path = _absolute_path(payload, "sheet_path")
    if not sheet_path.is_file():
        raise ProcessingError("invalid_path", "sheet_path must be an existing file")
    output_dir = _absolute_path(payload, "output_dir")
    if not output_dir.is_dir():
        raise ProcessingError("invalid_path", "output_dir must be an existing directory")
    return JobRequest(
        job_id=_text(payload, "job_id"),
        pack_id=_text(payload, "pack_id"),
        version=_integer(payload, "version", 1, 2**31 - 1),
        sheet_path=sheet_path,
        output_dir=output_dir,
        rows=rows,
        columns=columns,
        outline_width=_integer(payload, "outline_width", 0, MAX_OUTLINE_WIDTH),
        outline_color=outline_color,
    )


def _render(request: JobRequest) -> tuple[list[dict[str, object]], dict[str, object]]:
    sheet = _open_sheet(request.sheet_path)
    engine: str = (os.getenv("IMG_TOOL_ENGINE", "simple").strip().lower() or "simple")
    cutout = execute_cutout(sheet)
    alpha = np.array(cutout.getchannel("A"))
    foreground: float = float(np.mean(alpha > 32))
    if not MIN_FOREGROUND_RATIO <= foreground <= MAX_FOREGROUND_RATIO:
        raise ProcessingError(
            "no_foreground",
            f"foreground covers {foreground:.4f} of the sheet; use a plain near-white background",
        )
    cells: int = request.rows * request.columns
    min_area: int = max(64, int(MIN_CELL_AREA_FRACTION * alpha.size / cells))
    try:
        boxes = assign_boxes(
            find_sticker_boxes(cutout, min_area, 0),
            request.rows,
            request.columns,
            cutout.size,
        )
    except ValueError as exc:
        raise ProcessingError("layout_invalid", str(exc)) from exc
    stickers: list[dict[str, object]] = []
    written: list[Path] = []
    try:
        for index, box in enumerate(boxes):
            canvas = compose_sticker(
                cutout.crop(box), request.outline_width, request.outline_color
            )
            filename = f"{index:03d}.webp"
            thumb_name = f"thumb_{index:03d}.webp"
            (request.output_dir / filename).write_bytes(encode_webp(canvas, True))
            written.append(request.output_dir / filename)
            (request.output_dir / thumb_name).write_bytes(encode_webp(make_thumb(canvas), True))
            written.append(request.output_dir / thumb_name)
            stickers.append(
                {
                    "filename": filename,
                    "thumb": thumb_name,
                    "width": canvas.width,
                    "height": canvas.height,
                }
            )
    except OSError as exc:
        for path in written:
            path.unlink(missing_ok=True)
        raise ProcessingError("output_failed", f"cannot write artifacts: {exc.strerror}") from exc
    return stickers, {"engine": engine, "foreground_ratio": round(foreground, 4)}


def _open_sheet(path: Path) -> Image.Image:
    if path.stat().st_size > MAX_SHEET_BYTES:
        raise ProcessingError("image_too_large", "sheet exceeds 8 MiB")
    try:
        image = Image.open(path)
        if image.format not in ALLOWED_FORMATS:
            raise ProcessingError("unsupported_image", "sheet must be PNG, JPEG or WebP")
        if max(image.size) > MAX_SHEET_SIDE:
            raise ProcessingError("image_too_large", f"sheet side must not exceed {MAX_SHEET_SIDE} px")
        if getattr(image, "is_animated", False) or getattr(image, "n_frames", 1) > 1:
            raise ProcessingError("animation_not_supported", "animated sheets are not supported")
        image.load()
    except ProcessingError:
        raise
    except Exception as exc:  # noqa: BLE001 - any decoder failure is the same protocol error
        message = f"sheet decode failed: {type(exc).__name__}"
        raise ProcessingError("decode_failed", message) from exc
    return image.convert("RGBA")


def _text(payload: dict[str, object], key: str, limit: int = 200) -> str:
    value = payload.get(key)
    if not isinstance(value, str) or value.strip() == "":
        raise ProcessingError("invalid_field", f"{key} must be a non-empty string")
    if len(value) > limit:
        raise ProcessingError("invalid_field", f"{key} must not exceed {limit} characters")
    return value


def _integer(payload: dict[str, object], key: str, low: int, high: int) -> int:
    value = payload.get(key)
    if not isinstance(value, int) or isinstance(value, bool) or not low <= value <= high:
        raise ProcessingError("invalid_field", f"{key} must be an integer {low}-{high}")
    return value


def _absolute_path(payload: dict[str, object], key: str) -> Path:
    path = Path(_text(payload, key, limit=4096))
    if not path.is_absolute():
        raise ProcessingError("invalid_path", f"{key} must be absolute")
    if ".." in path.parts:
        raise ProcessingError("invalid_path", f"{key} must not contain '..'")
    if path.is_symlink():
        raise ProcessingError("invalid_path", f"{key} must not be a symlink")
    return path


def _write_error_result(
    result_path: Path,
    identity: dict[str, object],
    error: ProcessingError,
) -> None:
    try:
        _write_result(
            result_path,
            {**identity, "error": {"code": error.code, "message": error.message}},
        )
    except ProcessingError:
        pass  # the exit code still reports failure to the caller


def _write_result(result_path: Path, payload: dict[str, object]) -> None:
    text = json.dumps(payload, sort_keys=True) + "\n"
    if len(text.encode("utf-8")) > MAX_RESULT_BYTES:
        raise ProcessingError("output_failed", "result exceeds 1 MiB")
    temporary = result_path.with_name(result_path.name + ".tmp")
    try:
        temporary.write_text(text, encoding="utf-8")
        os.replace(temporary, result_path)
    except OSError as exc:
        temporary.unlink(missing_ok=True)
        raise ProcessingError("output_failed", f"cannot write result: {exc.strerror}") from exc
