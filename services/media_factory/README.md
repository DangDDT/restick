# media_factory

Local image/video pipeline for Restick. Static sheets (or already-cut files)
produce 512×512 WebP stickers; individual video clips produce animated variants.
The Go server serializes both jobs through one Python-child lock. Processing
jobs do not submit content to an external AI provider.

## Install for local use

```bash
cd services/media_factory
python3 -m venv venv
source venv/bin/activate
pip install -e .
media_factory --help
```

Dependencies are Pillow, NumPy, OpenCV headless and Typer. The production image
already installs this package; there is no separate media_factory container.

The cutout engine is still selected with the legacy env var `IMG_TOOL_ENGINE`
(name kept unchanged for compatibility).

## Local CLI

```bash
# Slice a sheet and/or already-cut images into a pack folder.
media_factory process ./testdata/sample_sheet.png \
  --name "My Pack" --slug my-pack --out ./output

# Also accepted: --files a.png,b.png --min-area 800 --padding 12 --lossy
media_factory version
```

`media_factory process` is the standalone artist utility. It does not upload or
publish anything; the single-operator studio does that through the Go server.

## Machine job protocol

The Go runner writes `request.json` and execs, without a shell:

```bash
python -m media_factory.interfaces.machine_job --request request.json --result result.json
```

The request/result documents are `restick.job.v1` / `restick.job-result.v1`
(spec §7.1). A non-zero exit is always a failure; the Go side validates every
output file, dimension and checksum before anything is published.

## Clip job protocol

The separate animated path leaves `machine_job` and its static encoding unchanged:

```bash
python -m media_factory.interfaces.clip_job --request request.json --result result.json
```

The request schema is `restick.clip-request.v1`, with `job_id`, `pack_id`,
`sticker_index`, absolute `clip_path` and existing absolute `output_dir`,
`outline_width`, and `outline_color` (`#RRGGBB`). Optional `fps` defaults to 12
and `quality` to 75. The API upload cap is 20 MiB; the Python worker independently
rejects clips over 32 MiB, four seconds, or 1920 px per side. Files must decode
with monotonic timestamps; unreadable input fails instead of silently losing frames.

Frames share one union bounding box before aspect-preserving composition, so
movement is not normalized away by a separate crop/scale on each frame. Every
frame must have usable foreground. Animated WebP uses `method=4`, lossless alpha,
and `loop=0`; encoding tries lower quality and frame rate if needed to stay under
512,000 bytes. Static files and thumbs remain lossless with `method=6`.

Success writes `output_dir/animated/NNN.webp` plus `restick.clip-result.v1`
JSON with echoed identity, `animation` metadata (`filename`, `frames`, `fps`,
`duration_ms`, `byte_size`, `quality`, `width`, `height`), and mask diagnostics.
`frames` counts encoder-input frames; libwebp may merge consecutive identical
frames while retaining their total duration. Timing uses integer per-frame
milliseconds, so 12 fps uses 83 ms, not an exact 1/12 second. Both media output
and result are placed atomically; failures have a nonzero exit and an error result.
The Go worker independently checks size, dimensions, identity and checksum before
marking a draft ready. The operator must publish a batch before it enters a pack.

## Algorithm documentation

Function docstrings explain algorithms in Vietnamese with the sections
`Algorithm`, `Techniques`, `Keywords` and `References`. Technical search terms
remain English. For the manual coverage/link check:

```bash
python scripts/check_docstrings.py src/media_factory
```

This performs network requests only for documentation links; it is not part of
media processing or CI. The checker may fall back to `curl` for blocked HTTP
clients, so install `curl` for that manual check.

## Tests

```bash
cd services/media_factory
python -m unittest discover -s tests -v
```
