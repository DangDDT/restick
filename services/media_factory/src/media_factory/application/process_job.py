from __future__ import annotations

import json
import os
import re
from dataclasses import dataclass
from pathlib import Path

import numpy as np
from PIL import Image

from media_factory.infrastructure.cutout import execute_cutout, find_sticker_boxes
from media_factory.infrastructure.layout import MAX_SHEET_SIDE, assign_boxes, make_thumb
from media_factory.infrastructure.outline import MAX_OUTLINE_WIDTH, compose_sticker
from media_factory.infrastructure.theme_color import suggest_theme_color
from media_factory.infrastructure.webp_export import encode_webp

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
        """Khởi tạo lỗi có mã ổn định cho giao thức xử lý media.

        Algorithm:
            Khởi tạo Exception bằng message, rồi lưu code và message cho bên gọi.

        Techniques:
            - Custom exception với structured error code.

        Keywords: custom exception, structured error, error code

        References:
            - https://docs.python.org/3/tutorial/errors.html#user-defined-exceptions
        """
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
    """Chạy một job sheet theo giao thức restick.job.v1 và ghi restick.job-result.v1.

    Algorithm:
        1. Khởi tạo identity rỗng để luôn có thứ gì đó trả về.
        2. read_request rồi _render.
        3. ProcessingError: ghi kết quả lỗi rồi raise lại.
        4. Lỗi bất ngờ: bọc thành internal_error, ghi kết quả lỗi.
        5. Thành công: ghi kết quả qua _write_result (ghi nguyên tử).

    Techniques:
        - Request/result file protocol (JSON)
        - Atomic write (temp file + rename)
        - Coded error handling

    Keywords: machine job, protocol, restick.job.v1, atomic write, error code

    References:
        - https://en.wikipedia.org/wiki/JSON
        - https://en.wikipedia.org/wiki/Atomicity_(database_systems)
        - https://en.wikipedia.org/wiki/Error_code
    """
    identity: dict[str, object] = {
        "schema": RESULT_SCHEMA,
        "job_id": "",
        "pack_id": "",
        "version": 0,
    }
    try:
        request = read_request(request_path, identity)
        stickers, diagnostics, theme_color = _render(request)
    except ProcessingError as exc:
        _write_error_result(result_path, identity, exc)
        raise
    except Exception as exc:  # noqa: BLE001 - the protocol must always report a code
        failure = ProcessingError("internal_error", f"unexpected {type(exc).__name__}")
        _write_error_result(result_path, identity, failure)
        raise failure from exc
    _write_result(
        result_path,
        {
            **identity,
            "stickers": stickers,
            "diagnostics": diagnostics,
            "suggested_theme_color": theme_color,
        },
    )


def read_request(request_path: Path, identity: dict[str, object]) -> JobRequest:
    """Đọc và kiểm tra request JSON, trả JobRequest.

    Algorithm:
        1. Kiểm tra file tồn tại và <= 1 MiB.
        2. Parse JSON, yêu cầu là object.
        3. Ghi sớm job_id/pack_id/version vào identity để Go loại kết quả cũ.
        4. Kiểm tra schema, rows/columns (1-8), outline_color, outline_width.
        5. Kiểm tra sheet_path và output_dir (tuyệt đối, tồn tại, không '..', không symlink).

    Techniques:
        - Input validation tại trust boundary
        - JSON parsing
        - Regular expression

    Keywords: request validation, JSON, schema, path validation

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


def _render(request: JobRequest) -> tuple[list[dict[str, object]], dict[str, object], str]:
    """Cắt nền tờ sheet, tách từng ô, dựng sticker, ghi file và gợi ý màu nền pack.

    Algorithm:
        1. Mở sheet an toàn (_open_sheet), đọc engine từ biến môi trường.
        2. execute_cutout; tính tỉ lệ foreground, ngoài [0.005, 0.995] thì lỗi no_foreground.
        3. Tính min_area theo ô; find_sticker_boxes rồi assign_boxes (lỗi: layout_invalid).
        4. Mỗi box: crop, compose_sticker (viền), encode WebP lossless và thumbnail.
        5. Lỗi OSError: xóa các file đã ghi rồi báo output_failed.
        6. Giữ bản thu nhỏ (<= 64px) của mỗi canvas, rồi suggest_theme_color chọn màu pastel.

    Techniques:
        - Foreground-ratio sanity check
        - Grid layout assignment
        - Cleanup on failure (rollback)
        - Downsampled copies (không giữ canvas 512px trong bộ nhớ)

    Keywords: sheet job, cutout, foreground ratio, grid, outline, WebP

    References:
        - https://en.wikipedia.org/wiki/Connected-component_labeling
        - https://en.wikipedia.org/wiki/Row-_and_column-major_order
        - https://en.wikipedia.org/wiki/Exception_handling
    """
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
    samples: list[Image.Image] = []
    try:
        for index, box in enumerate(boxes):
            canvas = compose_sticker(
                cutout.crop(box), request.outline_width, request.outline_color
            )
            sample = canvas.copy()
            sample.thumbnail((64, 64))
            samples.append(sample)
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
    diagnostics = {"engine": engine, "foreground_ratio": round(foreground, 4)}
    return stickers, diagnostics, suggest_theme_color(samples)


def _open_sheet(path: Path) -> Image.Image:
    """Mở và kiểm tra ảnh sheet, trả RGBA.

    Algorithm:
        1. Từ chối file > 8 MiB.
        2. Mở bằng context manager (đóng file ở mọi nhánh lỗi).
        3. Chỉ nhận PNG/JPEG/WEBP, cạnh <= MAX_SHEET_SIDE, không animated.
        4. load() và convert RGBA; mọi lỗi decoder thành decode_failed.

    Techniques:
        - Input validation
        - Whitelist định dạng

    Keywords: image validation, format whitelist, size limit, decode

    References:
        - https://en.wikipedia.org/wiki/Input_validation
        - https://pillow.readthedocs.io/en/stable/handbook/image-file-formats.html
    """
    if path.stat().st_size > MAX_SHEET_BYTES:
        raise ProcessingError("image_too_large", "sheet exceeds 8 MiB")
    try:
        # The context manager closes the underlying file on every rejection path too.
        with Image.open(path) as image:
            if image.format not in ALLOWED_FORMATS:
                raise ProcessingError("unsupported_image", "sheet must be PNG, JPEG or WebP")
            if max(image.size) > MAX_SHEET_SIDE:
                raise ProcessingError(
                    "image_too_large", f"sheet side must not exceed {MAX_SHEET_SIDE} px"
                )
            if getattr(image, "is_animated", False) or getattr(image, "n_frames", 1) > 1:
                raise ProcessingError(
                    "animation_not_supported", "animated sheets are not supported"
                )
            image.load()
            return image.convert("RGBA")
    except ProcessingError:
        raise
    except Exception as exc:  # noqa: BLE001 - any decoder failure is the same protocol error
        message = f"sheet decode failed: {type(exc).__name__}"
        raise ProcessingError("decode_failed", message) from exc


def _text(payload: dict[str, object], key: str, limit: int = 200) -> str:
    """Lấy trường chuỗi không rỗng từ payload.

    Algorithm:
        1. Kiểm tra kiểu str và không rỗng sau strip.
        2. Kiểm tra độ dài <= limit.

    Techniques:
        - Input validation

    Keywords: string field, validation, limit

    References:
        - https://en.wikipedia.org/wiki/Input_validation
    """
    value = payload.get(key)
    if not isinstance(value, str) or value.strip() == "":
        raise ProcessingError("invalid_field", f"{key} must be a non-empty string")
    if len(value) > limit:
        raise ProcessingError("invalid_field", f"{key} must not exceed {limit} characters")
    return value


def _integer(payload: dict[str, object], key: str, low: int, high: int) -> int:
    """Lấy trường số nguyên trong khoảng [low, high].

    Algorithm:
        1. Kiểm tra là int và không phải bool (bool là subclass của int).
        2. Kiểm tra low <= value <= high.

    Techniques:
        - Input validation

    Keywords: integer field, range check, bool pitfall

    References:
        - https://en.wikipedia.org/wiki/Input_validation
    """
    value = payload.get(key)
    if not isinstance(value, int) or isinstance(value, bool) or not low <= value <= high:
        raise ProcessingError("invalid_field", f"{key} must be an integer {low}-{high}")
    return value


def _absolute_path(payload: dict[str, object], key: str) -> Path:
    """Lấy đường dẫn an toàn từ payload.

    Algorithm:
        1. Lấy chuỗi bằng _text (tối đa 4096).
        2. Từ chối đường dẫn tương đối, có '..', hoặc là symlink.

    Techniques:
        - Path validation
        - Symlink rejection

    Keywords: absolute path, path traversal, symlink, validation

    References:
        - https://en.wikipedia.org/wiki/Path_(computing)
        - https://en.wikipedia.org/wiki/Symbolic_link
        - https://en.wikipedia.org/wiki/Input_validation
    """
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
    """Ghi kết quả lỗi, nuốt lỗi ghi phụ.

    Algorithm:
        1. Gộp identity với {error: code, message}.
        2. Nếu ghi thất bại thì bỏ qua; exit code vẫn báo lỗi cho caller.

    Techniques:
        - Best-effort error reporting
        - Atomic write

    Keywords: error result, best effort, protocol

    References:
        - https://en.wikipedia.org/wiki/Exception_handling
        - https://en.wikipedia.org/wiki/Error_code
        - https://en.wikipedia.org/wiki/Atomicity_(database_systems)
    """
    try:
        _write_result(
            result_path,
            {**identity, "error": {"code": error.code, "message": error.message}},
        )
    except ProcessingError:
        pass  # the exit code still reports failure to the caller


def _write_result(result_path: Path, payload: dict[str, object]) -> None:
    """Ghi payload JSON nguyên tử ra result_path.

    Algorithm:
        1. Serialize JSON (sort_keys) kèm xuống dòng.
        2. Từ chối nếu > 1 MiB.
        3. Ghi file .tmp rồi os.replace sang đích (nguyên tử trên cùng filesystem).
        4. Lỗi OSError: xóa file tạm và báo output_failed.

    Techniques:
        - Atomic write (temp file + rename)
        - JSON serialization

    Keywords: atomic write, os.replace, temp file, JSON

    References:
        - https://en.wikipedia.org/wiki/Atomicity_(database_systems)
        - https://en.wikipedia.org/wiki/JSON
    """
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
