from __future__ import annotations

import json
import logging
import tempfile
import time
from pathlib import Path

from img_tool.application.process_sheet import execute_process
from img_tool.infrastructure.config.config import Config
from img_tool.infrastructure.grpc_clients import JobClient, ObjectClient, list_prefix

IMAGE_SUFFIXES: tuple[str, ...] = (".png", ".jpg", ".jpeg", ".webp")


def run_worker(cfg: Config) -> None:
    jobs = JobClient(cfg.api_svc_grpc_addr, cfg.internal_token)
    objects = ObjectClient(cfg.ftp_svc_grpc_addr, cfg.internal_token)
    logging.info("img_tool worker polling jobs")
    while True:
        try:
            claimed = jobs.claim()
        except Exception as exc:  # noqa: BLE001
            logging.warning("claim failed err=%s", exc)
            time.sleep(2)
            continue
        if claimed.empty:
            time.sleep(2)
            continue
        try:
            _process_job(cfg, jobs, objects, claimed)
        except Exception as exc:  # noqa: BLE001
            logging.exception("job failed job_id=%s", claimed.job_id)
            try:
                jobs.fail(claimed.job_id, str(exc))
            except Exception as fail_exc:  # noqa: BLE001
                logging.warning("fail job err=%s", fail_exc)
        time.sleep(0.2)


def _process_job(cfg: Config, jobs: JobClient, objects: ObjectClient, claimed) -> None:
    keys = list_prefix(cfg.ftp_http_url, cfg.internal_token, claimed.source_prefix)
    image_keys = [key for key in keys if key.lower().endswith(IMAGE_SUFFIXES)]
    if len(image_keys) == 0:
        raise ValueError("no source images under prefix")
    with tempfile.TemporaryDirectory() as raw_dir_name:
        raw_dir = Path(raw_dir_name)
        local_files: list[Path] = []
        for key in image_keys:
            payload = objects.get(key)
            dest = raw_dir / Path(key).name
            dest.write_bytes(payload)
            local_files.append(dest)
        out_dir = raw_dir / "out"
        sheet: Path | None = None
        files: list[Path] = []
        if claimed.source_kind == "files":
            files = sorted(local_files)
        else:
            sheet = _pick_sheet(local_files)
        pack = execute_process(
            pack_id=claimed.pack_id,
            name=claimed.name,
            slug=claimed.slug,
            author=claimed.author,
            tags=list(claimed.tags),
            out_dir=out_dir,
            sheet=sheet,
            files=files,
            min_area=800,
            padding=12,
            lossless=True,
        )
        meta = json.loads((out_dir / "pack.json").read_text(encoding="utf-8"))
        prefix = f"packs/{pack.pack_id}/v{pack.version}/"
        object_keys: list[str] = []
        stickers_meta: list[dict[str, object]] = []
        for item in meta["stickers"]:
            filename = str(item["file"])
            thumb = str(item["thumb"])
            object_key = prefix + filename
            thumb_key = prefix + thumb
            objects.put(object_key, (out_dir / filename).read_bytes(), "image/webp")
            objects.put(thumb_key, (out_dir / thumb).read_bytes(), "image/webp")
            object_keys.extend([object_key, thumb_key])
            stickers_meta.append(
                {
                    "file": filename,
                    "thumb": thumb,
                    "emoji_hint": str(item.get("emoji_hint", "")),
                    "width": int(item["width"]),
                    "height": int(item["height"]),
                    "object_key": object_key,
                    "thumb_key": thumb_key,
                }
            )
        thumb_key = prefix + "thumb.webp"
        pack_key = prefix + "pack.json"
        objects.put(thumb_key, (out_dir / "thumb.webp").read_bytes(), "image/webp")
        objects.put(pack_key, (out_dir / "pack.json").read_bytes(), "application/json")
        object_keys.extend([thumb_key, pack_key])
        bundle_key = objects.build_bundle(pack.pack_id, pack.version, object_keys)
        jobs.complete(
            claimed.job_id,
            str(meta["checksum_sha256"]),
            thumb_key,
            bundle_key,
            pack.version,
            stickers_meta,
        )
        logging.info("job complete job_id=%s pack_id=%s", claimed.job_id, pack.pack_id)


def _pick_sheet(files: list[Path]) -> Path:
    for path in files:
        if path.stem.lower() == "sheet":
            return path
    return files[0]
