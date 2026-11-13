from __future__ import annotations

import argparse
import sys
from pathlib import Path

from media_factory.application.process_job import ProcessingError, process_job

MAX_DIAGNOSTIC_CHARS: int = 400


def main(argv: list[str] | None = None) -> int:
    """Điểm vào của machine job (Go gọi qua tiến trình con).

    Algorithm:
        1. Parse --request và --result.
        2. Gọi process_job.
        3. Lỗi: ghi mã và thông điệp ngắn ra stderr, trả exit code 1; thành công trả 0.

    Techniques:
        - Exit status protocol
        - Standard error diagnostics

    Keywords: machine job, exit code, stderr, subprocess

    References:
        - https://en.wikipedia.org/wiki/Exit_status
        - https://en.wikipedia.org/wiki/Standard_streams
        - https://en.wikipedia.org/wiki/Command-line_interface
    """
    parser = argparse.ArgumentParser(prog="media_factory.interfaces.machine_job")
    parser.add_argument("--request", required=True, type=Path)
    parser.add_argument("--result", required=True, type=Path)
    args = parser.parse_args(argv)
    try:
        process_job(args.request, args.result)
    except ProcessingError as exc:
        _diagnose(f"{exc.code}: {exc.message}")
        return 1
    except Exception as exc:  # noqa: BLE001 - nothing may escape with exit code 0
        _diagnose(f"internal_error: {type(exc).__name__}")
        return 1
    return 0


def _diagnose(message: str) -> None:
    """Ghi một dòng chẩn đoán ra stderr, cắt ngắn.

    Algorithm:
        1. Cắt message tới MAX_DIAGNOSTIC_CHARS.
        2. Ghi kèm xuống dòng.

    Techniques:
        - Standard error stream

    Keywords: stderr, diagnostics, truncation

    References:
        - https://en.wikipedia.org/wiki/Standard_streams
    """
    sys.stderr.write(message[:MAX_DIAGNOSTIC_CHARS] + "\n")


if __name__ == "__main__":
    raise SystemExit(main())
