# img_tool

Local image pipeline for Restick. It slices a composite sticker sheet (or
already-cut files) into 512×512 WebP stickers, and runs as the one Python child
the Go server execs for a processing job. It never talks to the network.

## Install for local use

```bash
cd services/img_tool
python3 -m venv venv
source venv/bin/activate
pip install -e .
img_tool --help
```

Dependencies are Pillow, NumPy, OpenCV headless and Typer. The production image
already installs this package; there is no separate img_tool container.

## Local CLI

```bash
# Slice a sheet and/or already-cut images into a pack folder.
img_tool process ./testdata/sample_sheet.png \
  --name "My Pack" --slug my-pack --out ./output

# Also accepted: --files a.png,b.png --min-area 800 --padding 12 --lossy
img_tool version
```

`img_tool process` is the standalone artist utility. It does not upload or
publish anything; the single-operator studio does that through the Go server.

## Machine job protocol

The Go runner writes `request.json` and execs, without a shell:

```bash
python -m img_tool.interfaces.machine_job --request request.json --result result.json
```

The request/result documents are `restick.job.v1` / `restick.job-result.v1`
(spec §7.1). A non-zero exit is always a failure; the Go side validates every
output file, dimension and checksum before anything is published.

## Tests

```bash
cd services/img_tool
python -m unittest discover -s tests -v
```
