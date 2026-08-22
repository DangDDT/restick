from __future__ import annotations

import json
from pathlib import Path

from img_tool.infrastructure.grpc_clients import JobClient, ObjectClient


def execute_submit(out_dir: Path, objects: ObjectClient, jobs: JobClient) -> None:
    pack = json.loads((out_dir / "pack.json").read_text(encoding="utf-8"))
    pack_id: str = str(pack["id"])
    version: int = int(pack["version"])
    prefix = f"packs/{pack_id}/v{version}/"
    keys: list[str] = []
    stickers_meta: list[dict[str, object]] = []
    for item in pack["stickers"]:
        filename = str(item["file"])
        thumb = str(item["thumb"])
        object_key = prefix + filename
        thumb_key = prefix + thumb
        objects.put(object_key, (out_dir / filename).read_bytes(), "image/webp")
        objects.put(thumb_key, (out_dir / thumb).read_bytes(), "image/webp")
        keys.extend([object_key, thumb_key])
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
    keys.extend([thumb_key, pack_key])
    bundle_key = objects.build_bundle(pack_id, version, keys)
    jobs.upsert_cli(
        pack_id=pack_id,
        slug=str(pack["slug"]),
        name=str(pack["name"]),
        author=str(pack["author"]),
        tags=[str(tag) for tag in pack.get("tags", [])],
        checksum=str(pack["checksum_sha256"]),
        thumbnail_key=thumb_key,
        bundle_key=bundle_key,
        version=version,
        stickers=stickers_meta,
    )
