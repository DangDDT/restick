from __future__ import annotations

import tempfile
import unittest
from io import BytesIO
from pathlib import Path
from unittest import mock

import cv2
import numpy as np
from PIL import Image, ImageDraw

from media_factory.application.process_job import ProcessingError
from media_factory.infrastructure.video import read_frames
from media_factory.infrastructure.webp_export import encode_animated_webp


def write_clip(directory: Path, frames: int, fps: int = 24) -> Path:
    """Write a synthetic mp4v clip: a dark square moving over white."""
    path = directory / "clip.mp4"
    writer = cv2.VideoWriter(path.as_posix(), cv2.VideoWriter_fourcc(*"mp4v"), fps, (64, 48))
    if not writer.isOpened():
        raise unittest.SkipTest("mp4v VideoWriter unavailable")
    for i in range(frames):
        frame = np.full((48, 64, 3), 255, np.uint8)
        frame[10:30, i % 40 : i % 40 + 20] = 20
        writer.write(frame)
    writer.release()
    return path


class FakeCapture:
    """Capture with irregular timestamps (ms), as a variable-frame-rate phone clip."""

    TIMES = [0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 500, 900, 1000, 1010, 1020]

    def __init__(self, _path: str) -> None:
        self.i = -1

    def isOpened(self) -> bool:  # noqa: N802
        return True

    def grab(self) -> bool:
        self.i += 1
        return self.i < len(self.TIMES)

    def retrieve(self) -> tuple[bool, np.ndarray]:
        return True, np.full((8, 8, 3), 255, np.uint8)

    def get(self, prop: int) -> float:
        if prop == cv2.CAP_PROP_POS_MSEC:
            return float(self.TIMES[self.i])
        return 8.0 if prop in (cv2.CAP_PROP_FRAME_WIDTH, cv2.CAP_PROP_FRAME_HEIGHT) else 0.0

    def release(self) -> None:
        pass


class ReadFramesTest(unittest.TestCase):
    def setUp(self) -> None:
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.dir = Path(tmp.name)

    def test_reads_expected_frame_count_at_12_fps(self) -> None:
        frames = read_frames(write_clip(self.dir, 24), fps=12, max_frames=48)
        self.assertEqual(len(frames), 12)
        self.assertEqual(frames[0].mode, "RGBA")
        self.assertGreater(min(frames[0].getpixel((0, 0))), 240)  # mp4v is lossy

    def test_rejects_clip_over_four_seconds(self) -> None:
        with self.assertRaises(ProcessingError) as ctx:
            read_frames(write_clip(self.dir, 120), fps=12, max_frames=48)
        self.assertEqual(ctx.exception.code, "clip_too_long")

    def test_rejects_unreadable_file(self) -> None:
        bad = self.dir / "bad.mp4"
        bad.write_bytes(b"not a video")
        with self.assertRaises(ProcessingError) as ctx:
            read_frames(bad, fps=12, max_frames=48)
        self.assertEqual(ctx.exception.code, "decode_failed")

    def test_rejects_oversized_side(self) -> None:
        with self.assertRaises(ProcessingError) as ctx:
            read_frames(write_clip(self.dir, 6), fps=12, max_frames=48, max_side=32)
        self.assertEqual(ctx.exception.code, "clip_too_large")

    def test_samples_by_timestamp_for_variable_rate(self) -> None:
        with mock.patch("media_factory.infrastructure.video.cv2.VideoCapture", FakeCapture):
            frames = read_frames(self.dir / "x.mp4", fps=10, max_frames=48)
        # 16 frames spanning ~1.03 s -> round(1.03 * 10) = 10 frames, not 16.
        self.assertEqual(len(frames), 10)


class EncodeAnimatedWebpTest(unittest.TestCase):
    def test_animated_webp_has_all_frames_and_alpha(self) -> None:
        frames = []
        for i in range(3):
            frame = Image.new("RGBA", (64, 64), (0, 0, 0, 0))
            ImageDraw.Draw(frame).ellipse((10 + i * 4, 10, 50, 50), fill=(200, 30, 30, 255))
            frames.append(frame)
        data = encode_animated_webp(frames, fps=12, quality=80)
        image = Image.open(BytesIO(data))
        self.assertEqual(image.n_frames, 3)
        self.assertEqual(image.convert("RGBA").getpixel((0, 0))[3], 0)


if __name__ == "__main__":
    unittest.main()
