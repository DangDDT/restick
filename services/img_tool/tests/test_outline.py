from __future__ import annotations

import unittest

from PIL import Image

from img_tool.infrastructure.outline import compose_sticker

RED: tuple[int, int, int, int] = (220, 30, 30, 255)


def solid(size: int = 64, color: tuple[int, int, int, int] = RED) -> Image.Image:
    return Image.new("RGBA", (size, size), color)


class ComposeStickerTest(unittest.TestCase):
    def test_canvas_is_512_rgba_with_transparent_corner(self) -> None:
        canvas = compose_sticker(solid(), 0, "#FFFFFF")
        self.assertEqual(canvas.size, (512, 512))
        self.assertEqual(canvas.mode, "RGBA")
        self.assertEqual(canvas.getpixel((0, 0))[3], 0)
        self.assertEqual(canvas.getpixel((256, 256))[3], 255)

    def test_zero_outline_keeps_foreground_color(self) -> None:
        canvas = compose_sticker(solid(), 0, "#FFFFFF")
        red, green, blue, alpha = canvas.getpixel((256, 256))
        self.assertEqual(alpha, 255)
        self.assertGreater(red, 180)
        self.assertLess(green, 80)
        self.assertLess(blue, 80)

    def test_white_outline_is_drawn_outside_foreground(self) -> None:
        canvas = compose_sticker(solid(), 4, "#FFFFFF")
        self.assertEqual(canvas.getpixel((5, 256)), (255, 255, 255, 255))
        self.assertEqual(canvas.getpixel((2, 256))[3], 0)
        red, _, _, alpha = canvas.getpixel((256, 256))
        self.assertEqual(alpha, 255)
        self.assertGreater(red, 180, "outline must not recolor the foreground")

    def test_colored_outline_uses_requested_color(self) -> None:
        canvas = compose_sticker(solid(), 8, "#00FF00")
        self.assertEqual(canvas.getpixel((6, 256)), (0, 255, 0, 255))

    def test_wide_outline_is_not_clipped_by_canvas(self) -> None:
        canvas = compose_sticker(solid(), 16, "#FFFFFF")
        for x in range(0, 4):
            self.assertEqual(canvas.getpixel((x, 256))[3], 0, f"x={x}")
        self.assertEqual(canvas.getpixel((5, 256)), (255, 255, 255, 255))
        self.assertEqual(canvas.getchannel("A").getbbox(), (4, 4, 508, 508))

    def test_alpha_channel_is_not_squared(self) -> None:
        canvas = compose_sticker(Image.new("RGBA", (64, 64), (0, 0, 255, 128)), 0, "#FFFFFF")
        self.assertAlmostEqual(canvas.getpixel((256, 256))[3], 128, delta=3)

    def test_outline_keeps_translucent_interior_alpha(self) -> None:
        canvas = compose_sticker(Image.new("RGBA", (64, 64), (0, 0, 255, 128)), 4, "#FFFFFF")
        self.assertAlmostEqual(canvas.getpixel((256, 256))[3], 128, delta=3)

    def test_rejects_invalid_color(self) -> None:
        with self.assertRaises(ValueError):
            compose_sticker(solid(), 4, "white")

    def test_rejects_outline_out_of_range(self) -> None:
        with self.assertRaises(ValueError):
            compose_sticker(solid(), 17, "#FFFFFF")
        with self.assertRaises(ValueError):
            compose_sticker(solid(), -1, "#FFFFFF")


if __name__ == "__main__":
    unittest.main()
