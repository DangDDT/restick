from __future__ import annotations

import json
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

from PIL import Image, ImageDraw

from media_factory.application.process_job import ProcessingError, process_job

QUADRANT_COLORS: list[tuple[int, int, int]] = [
    (220, 40, 40),
    (40, 180, 60),
    (40, 80, 220),
    (230, 200, 40),
]
STICKER_KEYS: set[str] = {"filename", "thumb", "width", "height"}


def quadrant_centers(size: int) -> list[tuple[float, float]]:
    return [
        ((column + 0.5) * size / 2, (row + 0.5) * size / 2)
        for row in range(2)
        for column in range(2)
    ]


def white_sheet(size: int = 1024, radius: int = 150) -> Image.Image:
    image = Image.new("RGB", (size, size), (255, 255, 255))
    draw = ImageDraw.Draw(image)
    for (cx, cy), color in zip(quadrant_centers(size), QUADRANT_COLORS, strict=True):
        draw.ellipse((cx - radius, cy - radius, cx + radius, cy + radius), fill=color)
    return image


def alpha_sheet(size: int = 800, half: int = 120) -> Image.Image:
    image = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(image)
    for (cx, cy), color in zip(quadrant_centers(size), QUADRANT_COLORS, strict=True):
        draw.rounded_rectangle(
            (cx - half, cy - half, cx + half, cy + half), radius=24, fill=(*color, 255)
        )
    return image


class ProcessJobTestCase(unittest.TestCase):
    def setUp(self) -> None:
        self.root = Path(tempfile.mkdtemp(prefix="machine-job-"))
        self.addCleanup(shutil.rmtree, self.root, ignore_errors=True)
        self.sheet = self.root / "sheet.png"
        self.output = self.root / "output"
        self.output.mkdir()
        self.request = self.root / "request.json"
        self.result = self.root / "result.json"

    def write_request(self, **overrides: object) -> None:
        payload: dict[str, object] = {
            "schema": "restick.job.v1",
            "job_id": "job-1",
            "pack_id": "pack-1",
            "version": 7,
            "sheet_path": str(self.sheet),
            "output_dir": str(self.output),
            "rows": 2,
            "columns": 2,
            "outline_width": 4,
            "outline_color": "#FFFFFF",
        }
        payload.update(overrides)
        self.request.write_text(json.dumps(payload), encoding="utf-8")

    def read_result(self) -> dict[str, object]:
        return json.loads(self.result.read_text(encoding="utf-8"))

    def assert_processing_error(self, code: str) -> dict[str, object]:
        with self.assertRaises(ProcessingError) as caught:
            process_job(self.request, self.result)
        self.assertEqual(caught.exception.code, code)
        result = self.read_result()
        error = result["error"]
        assert isinstance(error, dict)
        self.assertEqual(error["code"], code)
        self.assertEqual(result["schema"], "restick.job-result.v1")
        identity = (result["job_id"], result["pack_id"], result["version"])
        self.assertEqual(identity, ("job-1", "pack-1", 7))
        return result


class ProcessJobSuccessTest(ProcessJobTestCase):
    def test_processes_white_sheet_in_row_major_order(self) -> None:
        white_sheet().save(self.sheet)
        self.write_request()
        process_job(self.request, self.result)
        result = self.read_result()

        self.assertEqual(result["schema"], "restick.job-result.v1")
        identity = (result["job_id"], result["pack_id"], result["version"])
        self.assertEqual(identity, ("job-1", "pack-1", 7))
        self.assertNotIn("error", result)
        diagnostics = result["diagnostics"]
        assert isinstance(diagnostics, dict)
        self.assertEqual(diagnostics["engine"], "simple")

        stickers = result["stickers"]
        assert isinstance(stickers, list)
        self.assertEqual(len(stickers), 4)
        self.assertEqual(
            [item["filename"] for item in stickers],
            ["000.webp", "001.webp", "002.webp", "003.webp"],
        )
        self.assertEqual(
            [item["thumb"] for item in stickers],
            ["thumb_000.webp", "thumb_001.webp", "thumb_002.webp", "thumb_003.webp"],
        )
        for item in stickers:
            self.assertEqual(set(item), STICKER_KEYS)
            self.assertEqual((item["width"], item["height"]), (512, 512))
            with Image.open(self.output / str(item["filename"])) as art:
                self.assertEqual(art.format, "WEBP")
                self.assertEqual(art.size, (512, 512))
            with Image.open(self.output / str(item["thumb"])) as thumb:
                self.assertEqual(thumb.size, (128, 128))

        for index, color in enumerate(QUADRANT_COLORS):
            with Image.open(self.output / f"{index:03d}.webp") as art:
                pixel = art.convert("RGB").getpixel((256, 256))
            for observed, expected in zip(pixel, color, strict=True):
                self.assertLess(abs(observed - expected), 40, f"cell {index} color {pixel}")
        created = sorted(path.name for path in self.root.iterdir())
        self.assertEqual(created, ["output", "request.json", "result.json", "sheet.png"])

    def test_result_contains_suggested_theme_color(self) -> None:
        white_sheet().save(self.sheet)
        self.write_request()
        process_job(self.request, self.result)
        color = self.read_result()["suggested_theme_color"]
        self.assertRegex(str(color), r"^#[0-9A-F]{6}$")

    def test_processes_true_alpha_sheet(self) -> None:
        alpha_sheet().save(self.sheet)
        self.write_request(outline_width=0, outline_color="#FFFFFF")
        process_job(self.request, self.result)
        result = self.read_result()
        stickers = result["stickers"]
        assert isinstance(stickers, list)
        self.assertEqual(len(stickers), 4)
        with Image.open(self.output / "003.webp") as art:
            self.assertEqual(art.size, (512, 512))
            self.assertGreater(art.convert("RGB").getpixel((256, 256))[0], 180)

    def test_serializes_json_result_atomically(self) -> None:
        white_sheet(size=512, radius=80).save(self.sheet)
        self.write_request()
        process_job(self.request, self.result)
        self.assertTrue(self.result.is_file())
        self.assertFalse(any(item.name.endswith(".tmp") for item in self.root.iterdir()))


class ProcessJobLayoutFailureTest(ProcessJobTestCase):
    def test_wrong_sticker_count_fails_without_artifacts(self) -> None:
        image = Image.new("RGB", (1024, 1024), (255, 255, 255))
        draw = ImageDraw.Draw(image)
        for (cx, cy), color in list(
            zip(quadrant_centers(1024), QUADRANT_COLORS, strict=True)
        )[:3]:
            draw.ellipse((cx - 150, cy - 150, cx + 150, cy + 150), fill=color)
        image.save(self.sheet)
        self.write_request()
        result = self.assert_processing_error("layout_invalid")
        self.assertIn("missing", str(result["error"]))
        self.assertEqual(list(self.output.iterdir()), [])

    def test_two_boxes_in_one_cell_fail(self) -> None:
        image = white_sheet()
        draw = ImageDraw.Draw(image)
        draw.ellipse((10, 10, 70, 70), fill=(0, 0, 0))
        image.save(self.sheet)
        self.write_request()
        result = self.assert_processing_error("layout_invalid")
        self.assertIn("same cell", str(result["error"]))

    def test_box_crossing_cell_boundary_fails(self) -> None:
        image = Image.new("RGB", (1024, 1024), (255, 255, 255))
        draw = ImageDraw.Draw(image)
        draw.ellipse((330, 156, 610, 356), fill=QUADRANT_COLORS[0])
        draw.ellipse((668, 156, 868, 356), fill=QUADRANT_COLORS[1])
        draw.ellipse((106, 618, 406, 918), fill=QUADRANT_COLORS[2])
        draw.ellipse((618, 618, 918, 918), fill=QUADRANT_COLORS[3])
        image.save(self.sheet)
        self.write_request()
        result = self.assert_processing_error("layout_invalid")
        self.assertIn("crosses", str(result["error"]))


class ProcessJobRequestFailureTest(ProcessJobTestCase):
    def test_unknown_schema_fails(self) -> None:
        white_sheet().save(self.sheet)
        self.write_request(schema="restick.job.v0")
        self.assert_processing_error("invalid_schema")

    def test_missing_field_fails(self) -> None:
        white_sheet().save(self.sheet)
        self.write_request()
        payload = json.loads(self.request.read_text(encoding="utf-8"))
        del payload["outline_width"]
        self.request.write_text(json.dumps(payload), encoding="utf-8")
        result = self.assert_processing_error("invalid_field")
        self.assertIn("outline_width", str(result["error"]["message"]))

    def test_outline_width_out_of_range_fails(self) -> None:
        white_sheet().save(self.sheet)
        self.write_request(outline_width=20)
        self.assert_processing_error("invalid_field")

    def test_invalid_outline_color_fails(self) -> None:
        white_sheet().save(self.sheet)
        self.write_request(outline_color="white")
        self.assert_processing_error("invalid_field")

    def test_relative_sheet_path_fails(self) -> None:
        white_sheet().save(self.sheet)
        self.write_request(sheet_path="sheet.png")
        self.assert_processing_error("invalid_path")

    def test_missing_output_dir_fails(self) -> None:
        white_sheet().save(self.sheet)
        self.write_request(output_dir=str(self.root / "absent"))
        self.assert_processing_error("invalid_path")

    def test_animated_webp_fails(self) -> None:
        self.sheet = self.root / "sheet.webp"
        frames = [
            Image.new("RGBA", (64, 64), (255, 0, 0, 255)),
            Image.new("RGBA", (64, 64), (0, 0, 255, 255)),
        ]
        frames[0].save(
            self.sheet, format="WEBP", save_all=True, append_images=frames[1:], duration=100
        )
        self.write_request()
        self.assert_processing_error("animation_not_supported")

    def test_oversized_sheet_fails_before_decode(self) -> None:
        Image.new("RGB", (4097, 16), (255, 255, 255)).save(self.sheet)
        self.write_request()
        result = self.assert_processing_error("image_too_large")
        self.assertIn("4096", str(result["error"]["message"]))

    def test_undecodable_sheet_fails(self) -> None:
        self.sheet.write_bytes(b"not an image")
        self.write_request()
        self.assert_processing_error("decode_failed")

    def test_malformed_request_json_fails(self) -> None:
        self.request.write_text("{", encoding="utf-8")
        with self.assertRaises(ProcessingError) as caught:
            process_job(self.request, self.result)
        self.assertEqual(caught.exception.code, "invalid_request")


class MachineJobCliTest(ProcessJobTestCase):
    def run_cli(self) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [
                sys.executable,
                "-m",
                "media_factory.interfaces.machine_job",
                "--request",
                str(self.request),
                "--result",
                str(self.result),
            ],
            cwd=self.root,
            capture_output=True,
            text=True,
            timeout=120,
            check=False,
        )

    def test_cli_roundtrip_writes_valid_artifacts(self) -> None:
        white_sheet().save(self.sheet)
        self.write_request(outline_width=8, outline_color="#FFFFFF")
        completed = self.run_cli()
        self.assertEqual(completed.returncode, 0, completed.stderr)
        self.assertEqual(completed.stderr, "")
        result = self.read_result()
        stickers = result["stickers"]
        assert isinstance(stickers, list)
        self.assertEqual(len(stickers), 4)
        for index, item in enumerate(stickers):
            self.assertEqual(set(item), STICKER_KEYS)
            self.assertEqual(item["filename"], f"{index:03d}.webp")
            self.assertEqual((item["width"], item["height"]), (512, 512))
            self.assertTrue((self.output / str(item["filename"])).is_file())
            self.assertTrue((self.output / str(item["thumb"])).is_file())
        self.assertFalse(any(item.name.endswith(".tmp") for item in self.root.iterdir()))

    def test_cli_exits_nonzero_on_invalid_request(self) -> None:
        self.write_request(schema="restick.job.v0")
        completed = self.run_cli()
        self.assertNotEqual(completed.returncode, 0)
        self.assertIn("invalid_schema", completed.stderr)
        self.assertEqual(self.read_result()["error"]["code"], "invalid_schema")


if __name__ == "__main__":
    unittest.main()
