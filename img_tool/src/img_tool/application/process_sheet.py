from __future__ import annotations

import re
from pathlib import Path

from PIL import Image

from img_tool.domain.pack import Pack
from img_tool.domain.sticker import Sticker
from img_tool.infrastructure.cutout import execute_cutout, find_sticker_boxes
from img_tool.infrastructure.layout import (
    filter_outlier_boxes,
    fit_on_canvas,
    load_image,
    make_thumb,
    sort_boxes,
    upscale_source,
)
from img_tool.infrastructure.webp_export import encode_webp

SLUG_PATTERN = re.compile(r"^[a-z0-9]+(?:-[a-z0-9]+)*$")


def execute_process(
    *,
    pack_id: str,
    name: str,
    slug: str,
    author: str,
    tags: list[str],
    out_dir: Path,
    sheet: Path | None,
    files: list[Path],
    min_area: int,
    padding: int,
    lossless: bool,
) -> Pack:
    if not SLUG_PATTERN.match(slug):
        raise ValueError("invalid slug")
    if padding < 8 or padding > 16:
        raise ValueError("padding must be 8-16")
    out_dir.mkdir(parents=True, exist_ok=True)
    canvases = _load_canvases(sheet, files, min_area, padding)
    if len(canvases) == 0:
        raise ValueError("no contours; check background")
    stickers: list[Sticker] = []
    sticker_bytes: list[bytes] = []
    for index, canvas in enumerate(canvases):
        filename = f"{index:03d}.webp"
        thumb_name = f"thumb_{index:03d}.webp"
        webp = encode_webp(canvas, lossless)
        thumb = encode_webp(make_thumb(canvas), lossless)
        (out_dir / filename).write_bytes(webp)
        (out_dir / thumb_name).write_bytes(thumb)
        sticker_bytes.append(webp)
        stickers.append(
            Sticker(
                filename=filename,
                thumb=thumb_name,
                emoji_hint="",
                width=canvas.width,
                height=canvas.height,
                rgba=webp,
            )
        )
    pack = Pack(
        pack_id=pack_id,
        slug=slug,
        name=name,
        author=author,
        description="",
        tags=tags,
        version=1,
        stickers=stickers,
    )
    checksum = pack.checksum_sha256(sticker_bytes)
    pack_json = pack.to_json(checksum)
    (out_dir / "pack.json").write_text(pack_json, encoding="utf-8")
    (out_dir / "thumb.webp").write_bytes((out_dir / stickers[0].thumb).read_bytes())
    return pack


def _load_canvases(
    sheet: Path | None,
    files: list[Path],
    min_area: int,
    padding: int,
) -> list[Image.Image]:
    if len(files) > 0:
        return [fit_on_canvas(execute_cutout(upscale_source(load_image(path)))) for path in files]
    if sheet is None:
        raise ValueError("sheet or files required")
    rgba = execute_cutout(upscale_source(load_image(sheet)))
    boxes = sort_boxes(filter_outlier_boxes(find_sticker_boxes(rgba, min_area, padding)))
    return [fit_on_canvas(rgba.crop(box)) for box in boxes]
