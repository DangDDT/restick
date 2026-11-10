from __future__ import annotations

import argparse
import sys
from pathlib import Path

from img_tool.application.process_job import ProcessingError, process_job

MAX_DIAGNOSTIC_CHARS: int = 400


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="img_tool.interfaces.machine_job")
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
    sys.stderr.write(message[:MAX_DIAGNOSTIC_CHARS] + "\n")


if __name__ == "__main__":
    raise SystemExit(main())
