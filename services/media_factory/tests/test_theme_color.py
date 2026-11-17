from __future__ import annotations

import colorsys
import re
import unittest

from PIL import Image, ImageDraw

from media_factory.infrastructure.theme_color import DEFAULT_THEME_COLOR, suggest_theme_color

HEX = re.compile(r"^#[0-9A-F]{6}$")


def disc(fill: tuple[int, int, int], outline: tuple[int, int, int] | None = None,
         ring: int = 0, size: int = 200) -> Image.Image:
    image = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(image)
    if outline is not None:
        draw.ellipse((5, 5, size - 6, size - 6), fill=(*outline, 255))
    inset = 5 + ring
    draw.ellipse((inset, inset, size - inset - 1, size - inset - 1), fill=(*fill, 255))
    return image


def hls(color: str) -> tuple[float, float, float]:
    red, green, blue = (int(color[i : i + 2], 16) / 255 for i in (1, 3, 5))
    hue, light, sat = colorsys.rgb_to_hls(red, green, blue)
    return hue * 360, light, sat


def hue_gap(a: float, b: float) -> float:
    diff = abs(a - b) % 360
    return min(diff, 360 - diff)


class SuggestThemeColorTest(unittest.TestCase):
    def test_dominant_purple_gives_light_purple(self) -> None:
        color = suggest_theme_color([disc((0xA9, 0x8B, 0xF0))])
        self.assertRegex(color, HEX)
        hue, light, _ = hls(color)
        self.assertLessEqual(hue_gap(hue, hls("#A98BF0")[0]), 15)
        self.assertGreaterEqual(light, 0.85)

    def test_orange_cat_gives_light_orange(self) -> None:
        color = suggest_theme_color([disc((240, 150, 40), outline=(255, 255, 255), ring=12)])
        hue, light, _ = hls(color)
        self.assertLessEqual(hue_gap(hue, 33), 15)
        self.assertGreaterEqual(light, 0.85)

    def test_vivid_colours_keep_their_hue(self) -> None:
        for fill in [(255, 0, 0), (255, 220, 0)]:
            color = suggest_theme_color([disc(fill)])
            hue, light, _ = hls(color)
            self.assertLessEqual(hue_gap(hue, hls("#%02X%02X%02X" % fill)[0]), 15, color)
            self.assertGreaterEqual(light, 0.85)
            self.assertLessEqual(light, 0.95)

    def test_all_gray_returns_default(self) -> None:
        self.assertEqual(suggest_theme_color([disc((128, 128, 128))]), DEFAULT_THEME_COLOR)

    def test_white_and_black_only_returns_default(self) -> None:
        image = disc((0, 0, 0), outline=(255, 255, 255), ring=20)
        self.assertEqual(suggest_theme_color([image]), DEFAULT_THEME_COLOR)

    def test_fully_transparent_returns_default(self) -> None:
        image = Image.new("RGBA", (64, 64), (200, 30, 30, 0))
        self.assertEqual(suggest_theme_color([image]), DEFAULT_THEME_COLOR)

    def test_empty_list_returns_default(self) -> None:
        self.assertEqual(suggest_theme_color([]), DEFAULT_THEME_COLOR)

    def test_white_outline_is_ignored(self) -> None:
        color = suggest_theme_color([disc((220, 30, 30), outline=(255, 255, 255), ring=30)])
        self.assertNotEqual(color, "#FFFFFF")
        hue, _, sat = hls(color)
        self.assertLessEqual(hue_gap(hue, 0), 15)
        self.assertGreater(sat, 0.5)

    def test_contrast_keeps_sticker_visible(self) -> None:
        for fill in [(220, 30, 30), (40, 180, 60), (40, 80, 220), (230, 200, 40)]:
            self.assertLessEqual(hls(suggest_theme_color([disc(fill)]))[1], 0.95)

    def test_heaviest_hue_wins_across_images(self) -> None:
        color = suggest_theme_color([disc((40, 80, 220)), disc((40, 80, 220)), disc((220, 30, 30))])
        self.assertLessEqual(hue_gap(hls(color)[0], hls("#2850DC")[0]), 15)

    def test_tiny_colored_speck_returns_default(self) -> None:
        image = disc((128, 128, 128))
        image.putpixel((100, 100), (255, 0, 0, 255))
        self.assertEqual(suggest_theme_color([image]), DEFAULT_THEME_COLOR)


if __name__ == "__main__":
    unittest.main()
