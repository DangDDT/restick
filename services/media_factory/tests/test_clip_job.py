from __future__ import annotations

import json
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

import cv2
import numpy as np
from PIL import Image

from media_factory.application.process_clip import fit_under_ceiling, process_clip, shared_box
from media_factory.application.process_job import ProcessingError
from media_factory.infrastructure.webp_export import encode_animated_webp


def write_clip(
    path: Path, size: tuple[int, int] = (512, 512), frames: int = 24, blank: int | None = None
) -> Path:
    """Synthetic clip: a red disc drifting over white; frame `blank` is a near-empty one."""
    writer = cv2.VideoWriter(path.as_posix(), cv2.VideoWriter_fourcc(*"mp4v"), 24, size)
    if not writer.isOpened():
        raise unittest.SkipTest("mp4v VideoWriter unavailable")
    width, height = size
    radius = min(width, height) // 4
    for i in range(frames):
        frame = np.full((height, width, 3), 255, np.uint8)
        if i == blank:
            frame[5, 5] = 0  # one opaque pixel: libwebp would drop alpha for this frame
        else:
            cv2.circle(frame, (width // 2 + i, height // 2), radius, (40, 40, 220), -1)
        writer.write(frame)
    writer.release()
    return path


class ClipJobTest(unittest.TestCase):
    def setUp(self) -> None:
        self.root = Path(tempfile.mkdtemp(prefix="clip-job-"))
        self.addCleanup(shutil.rmtree, self.root, ignore_errors=True)
        self.output = self.root / "output"
        self.output.mkdir()
        self.request = self.root / "request.json"
        self.result = self.root / "result.json"

    def run_job(self, clip: Path | str, **overrides: object) -> None:
        payload: dict[str, object] = {
            "schema": "restick.clip-request.v1",
            "job_id": "anim-1",
            "pack_id": "pack-1",
            "sticker_index": 3,
            "clip_path": str(clip),
            "output_dir": str(self.output),
            "outline_width": 4,
            "outline_color": "#FFFFFF",
        }
        payload.update(overrides)
        self.request.write_text(json.dumps(payload), encoding="utf-8")
        process_clip(self.request, self.result)

    def read_result(self) -> dict[str, object]:
        return json.loads(self.result.read_text(encoding="utf-8"))

    def test_clip_job_writes_animated_webp_and_result(self) -> None:
        self.run_job(write_clip(self.root / "c.mp4"))
        result = self.read_result()
        animation = result["animation"]
        self.assertEqual("restick.clip-result.v1", result["schema"])
        self.assertEqual(("anim-1", "pack-1"), (result["job_id"], result["pack_id"]))
        self.assertEqual("animated/003.webp", animation["filename"])
        self.assertGreater(animation["frames"], 1)
        self.assertEqual((512, 512), (animation["width"], animation["height"]))
        self.assertLessEqual(animation["byte_size"], 512_000)
        data = (self.output / "animated" / "003.webp").read_bytes()
        self.assertEqual(animation["byte_size"], len(data))
        self.assertGreaterEqual(result["diagnostics"]["mask_iou_min"], 0.9)
        with Image.open(self.output / "animated" / "003.webp") as webp:
            self.assertEqual(animation["frames"], webp.n_frames)

    def test_shared_box_is_union(self) -> None:
        a = np.zeros((20, 20), np.uint8)
        b = np.zeros((20, 20), np.uint8)
        a[2:5, 2:5] = 255
        b[10:15, 8:12] = 255
        self.assertEqual((2, 2, 12, 15), shared_box([a, b]))

    def test_blank_middle_frame_fails_no_foreground(self) -> None:
        clip = write_clip(self.root / "c.mp4", blank=12)
        with self.assertRaises(ProcessingError) as caught:
            self.run_job(clip)
        self.assertEqual("no_foreground", caught.exception.code)
        self.assertEqual("no_foreground", self.read_result()["error"]["code"])

    def test_portrait_clip_is_composed_on_512_canvas(self) -> None:
        self.run_job(write_clip(self.root / "p.mp4", size=(384, 848), frames=12))
        animation = self.read_result()["animation"]
        self.assertEqual((512, 512), (animation["width"], animation["height"]))

    def test_ladder_reduces_to_fit_ceiling(self) -> None:
        rng = np.random.default_rng(1)
        frames = [
            Image.fromarray(rng.integers(0, 255, (128, 128, 4), dtype=np.uint8), "RGBA")
            for _ in range(12)
        ]
        high = len(encode_animated_webp(frames, 12, 75))
        low = len(encode_animated_webp(frames, 12, 60))
        self.assertLess(low, high)
        data, fps_used, quality_used = fit_under_ceiling(frames, 12, 75, ceiling=(low + high) // 2)
        self.assertEqual((12, 60), (fps_used, quality_used))
        self.assertLessEqual(len(data), (low + high) // 2)
        data, fps_used, _ = fit_under_ceiling(frames, 12, 75, ceiling=low - 1)
        self.assertEqual(10, fps_used)

    def test_output_too_large_when_ladder_exhausted(self) -> None:
        frames = [Image.new("RGBA", (32, 32), (9, 9, 9, 255))] * 4
        with self.assertRaises(ProcessingError) as caught:
            fit_under_ceiling(frames, 12, 75, ceiling=1)
        self.assertEqual("output_too_large", caught.exception.code)

    def test_relative_clip_path_rejected(self) -> None:
        with self.assertRaises(ProcessingError) as caught:
            self.run_job("clip.mp4")
        self.assertEqual("invalid_path", caught.exception.code)

    def test_cli_exit_code(self) -> None:
        self.request.write_text("{}", encoding="utf-8")
        done = subprocess.run(
            [sys.executable, "-m", "media_factory.interfaces.clip_job",
             "--request", str(self.request), "--result", str(self.result)],
            capture_output=True, text=True, check=False,
        )
        self.assertEqual(1, done.returncode)
        self.assertIn("invalid_schema:", done.stderr)


if __name__ == "__main__":
    unittest.main()
