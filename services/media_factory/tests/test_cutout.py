from __future__ import annotations

import os
import unittest
from unittest.mock import patch

import numpy as np
from PIL import Image, ImageDraw

from media_factory.infrastructure.cutout import execute_cutout


def noisy_white_stickers() -> Image.Image:
    image = Image.new("RGB", (512, 256), "white")
    draw = ImageDraw.Draw(image)
    # Near-white specks trigger the fallback without becoming foreground ink.
    for y in range(8, 256, 16):
        for x in range(8, 512, 16):
            draw.rectangle((x, y, x + 1, y + 1), fill=(240, 240, 240))
    for x in (128, 384):
        draw.ellipse((x - 70, 58, x + 70, 198), fill="white", outline="black", width=8)
    return image


@patch.dict(os.environ, {"IMG_TOOL_ENGINE": "simple"})
class CutoutTest(unittest.TestCase):
    def test_fallback_preserves_white_interiors_without_joining_stickers(self) -> None:
        cutout = execute_cutout(noisy_white_stickers())
        for x in (128, 384):
            self.assertEqual(cutout.getpixel((x, 128)), (255, 255, 255, 255))
        self.assertEqual(cutout.getpixel((256, 128))[3], 0)
        self.assertEqual(cutout.getpixel((0, 0))[3], 0)

    def test_generated_cutout_has_coverage_at_diagonal_edges(self) -> None:
        image = Image.new("RGB", (128, 128), "white")
        ImageDraw.Draw(image).polygon(((20, 100), (64, 20), (108, 100)), fill="black")
        alpha = np.array(execute_cutout(image).getchannel("A"))
        self.assertTrue(np.any((alpha[25:90, 25:60] > 0) & (alpha[25:90, 25:60] < 255)))
        self.assertEqual(alpha[60, 64], 255)
        self.assertEqual(alpha[0, 0], 0)

    def test_existing_transparency_is_preserved_exactly(self) -> None:
        image = Image.new("RGBA", (128, 128), (0, 0, 0, 0))
        draw = ImageDraw.Draw(image)
        draw.ellipse((10, 10, 118, 118), fill=(20, 40, 200, 128))
        draw.ellipse((40, 40, 88, 88), fill=(0, 0, 0, 0))
        np.testing.assert_array_equal(np.array(execute_cutout(image)), np.array(image))


if __name__ == "__main__":
    unittest.main()
