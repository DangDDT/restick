# media_factory

Local image pipeline for Restick. It slices a composite sticker sheet (or
already-cut files) into 512×512 WebP stickers, and runs as the one Python child
the Go server execs for a processing job. It never talks to the network.

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

## Tests

```bash
cd services/media_factory
python -m unittest discover -s tests -v
```
