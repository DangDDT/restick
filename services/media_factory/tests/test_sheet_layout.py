from __future__ import annotations

import unittest

from PIL import Image

from media_factory.infrastructure.layout import (
    CANVAS_SIZE,
    assign_boxes,
    fit_in_square,
    fit_on_canvas,
    make_thumb,
)

SHEET: tuple[int, int] = (1024, 1024)


def box(center_x: int, center_y: int, half: int = 100) -> tuple[int, int, int, int]:
    return (center_x - half, center_y - half, center_x + half, center_y + half)


def center(box_: tuple[int, int, int, int]) -> tuple[int, int]:
    return ((box_[0] + box_[2]) // 2, (box_[1] + box_[3]) // 2)


class AssignBoxesTest(unittest.TestCase):
    def test_orders_boxes_row_major(self) -> None:
        boxes = [box(768, 768), box(256, 256), box(768, 256), box(256, 768)]
        ordered = assign_boxes(boxes, 2, 2, SHEET)
        self.assertEqual(
            [center(item) for item in ordered],
            [(256, 256), (768, 256), (256, 768), (768, 768)],
        )

    def test_rejects_two_boxes_in_one_cell(self) -> None:
        boxes = [box(200, 200), box(400, 200), box(768, 256), box(256, 768)]
        with self.assertRaisesRegex(ValueError, "same cell"):
            assign_boxes(boxes, 2, 2, SHEET)

    def test_rejects_missing_cell(self) -> None:
        boxes = [box(256, 256), box(768, 256), box(256, 768)]
        with self.assertRaisesRegex(ValueError, "missing"):
            assign_boxes(boxes, 2, 2, SHEET)

    def test_rejects_box_crossing_a_cell_boundary(self) -> None:
        crossing = (330, 156, 610, 356)
        boxes = [crossing, box(768, 256, 100), box(256, 768), box(768, 768)]
        with self.assertRaisesRegex(ValueError, "crosses"):
            assign_boxes(boxes, 2, 2, SHEET)

    def test_box_touching_a_boundary_exactly_is_allowed(self) -> None:
        boxes = [
            (0, 0, 512, 512),
            (512, 0, 1024, 512),
            (0, 512, 512, 1024),
            (512, 512, 1024, 1024),
        ]
        self.assertEqual(len(assign_boxes(boxes, 2, 2, SHEET)), 4)


class FitTest(unittest.TestCase):
    def test_fit_on_canvas_does_not_square_alpha(self) -> None:
        source = Image.new("RGBA", (256, 256), (10, 20, 30, 128))
        canvas = fit_on_canvas(source)
        self.assertEqual(canvas.size, (512, 512))
        self.assertAlmostEqual(canvas.getpixel((256, 256))[3], 128, delta=3)

    def test_make_thumb_scales_down_and_keeps_alpha(self) -> None:
        source = Image.new("RGBA", (1024, 1024), (1, 2, 3, 255))
        thumb = make_thumb(source)
        self.assertEqual(thumb.size, (128, 128))
        self.assertEqual(thumb.getpixel((64, 64))[3], 255)

    def test_fit_in_square_keeps_margin_clear(self) -> None:
        canvas = fit_in_square(Image.new("RGBA", (64, 64), (255, 255, 255, 255)), CANVAS_SIZE, 20)
        self.assertEqual(canvas.getpixel((0, 0))[3], 0)
        self.assertEqual(canvas.getchannel("A").getbbox(), (20, 20, 492, 492))


if __name__ == "__main__":
    unittest.main()
