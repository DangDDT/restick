# Phase 02 — img_tool (Python pipeline)

Date: 2026-08-20
Priority: P0
Status: pending
Depends on: phase 01

## Overview

Turn uploaded images (sprite sheet **or** already-cut files) into per-sticker WebP + `pack.json`. This tool **never** calls a generation API and has **no prompt input**. Publish/submit is phase 05; human review is phase 04/06.

## Contributor prep note (docs only, not a product feature)

If someone generates art elsewhere, a useful sheet prompt is:

`sheet of multiple die-cut cute stickers, isolated on solid white background, flat 2d, even spacing between items, no text unless original, no existing copyrighted characters`

Restick does not run this prompt. IP gate is **admin review**, not the cutout script.

## Functional requirements

- Input: PNG/JPEG/WebP sheet.
- rembg → RGBA.
- Morphological close on alpha to join broken parts of one sticker.
- `findContours` on binary alpha; drop contours below `--min-area` (default 800 px²).
- Bounding box + padding 12 px (flag `--padding`, 8–16).
- Fit inside 512×512, keep aspect, center on transparent canvas.
- Export `{index:03d}.webp` (quality 90 or lossless flag) and `thumb_{index:03d}.webp` (128×128).
- `pack.json` + `thumb.webp` (first sticker or largest contour).
- CLI `process` writes to `--out`. No network in phase 02.
- Second input mode: `--files a.webp,b.webp` skips contour split; each file is one sticker (still rembg + 512 canvas).

## Non-functional

- CPU-only. Batch one sheet at a time.
- Deterministic sort: top-to-bottom, then left-to-right (`y` then `x` of bbox).
- Max sheet side 4096; refuse larger.

## Files to create

| Path | Action | Layer |
| --- | --- | --- |
| `img_tool/src/img_tool/domain/pack.py` | create | pack aggregate + checksum |
| `img_tool/src/img_tool/domain/sticker.py` | create | sticker VO |
| `img_tool/src/img_tool/application/process_sheet.py` | create | use case |
| `img_tool/src/img_tool/infrastructure/cutout.py` | create | rembg + OpenCV |
| `img_tool/src/img_tool/infrastructure/layout.py` | create | 512 canvas, sort |
| `img_tool/src/img_tool/infrastructure/webp_export.py` | create | Pillow WebP |
| `img_tool/src/img_tool/interfaces/cli.py` | modify | Typer `process` |
| `img_tool/pyproject.toml` | modify | rembg, opencv-python-headless, pillow, numpy, typer, ruff |
| `img_tool/testdata/sample_sheet.png` | create | white sheet, 2 blobs — for **manual** process |

## `pack.json` schema (v1)

```json
{
  "schema": "restick.pack.v1",
  "id": "uuid",
  "slug": "chibi-cats",
  "name": "Chibi Cats",
  "author": "Restick",
  "description": "",
  "tags": ["cats", "chibi"],
  "ip_policy": "original_only",
  "version": 1,
  "checksum_sha256": "<hash of sorted sticker bytes>",
  "stickers": [
    {
      "file": "000.webp",
      "thumb": "thumb_000.webp",
      "emoji_hint": "",
      "width": 512,
      "height": 512
    }
  ]
}
```

Checksum: SHA-256 of concatenation of sticker WebP files in `file` order. Same value stored by `api_svc`.

## Cutout algorithm (do this, not SAM)

1. `rembg.remove(sheet, alpha_matting=False)` — default u2net/bria as dependency allows.
2. Alpha `> 16` → binary uint8.
3. `cv2.morphologyEx(MORPH_CLOSE, 5×5, iterations=1)`.
4. `findContours(RETR_EXTERNAL, CHAIN_APPROX_SIMPLE)`.
5. Filter area; skip contours touching image border if area > 40% of sheet (full-frame false positive).
6. Crop with padding; if crop hits edge, clip.
7. If sticker count is 0, exit non-zero with "no contours; check background".

Do **not** use `findContours` on grayscale RGB. Shadows will be treated as stickers.

## Implementation steps

1. Fixture sheet under `testdata/` (1024×512 white, two filled circles).
2. Use cases in `application/`; rembg/OpenCV only in `infrastructure/`.
3. CLI flags: `--name --slug --author --tags --min-area --padding --lossless`.
4. Refuse `--slug` not matching `^[a-z0-9]+(?:-[a-z0-9]+)*$`.
5. No pytest. Manual process of testdata.

## Success criteria

[manual-test-checklist.md](./manual-test-checklist.md) phase 02. `ruff check` clean. No `findContours` on RGB.

## Risks

- rembg model download on first run — document in README; vendor or cache dir `IMG_TOOL_U2NET_HOME`.
- Over-merged contours if stickers touch — operator must leave spacing; document.

## Next

Phase 03–04 can proceed with sample `pack.json` + 2 tiny WebPs under `img_tool/testdata/sample_pack/`. Worker HTTP comes in phase 05.
