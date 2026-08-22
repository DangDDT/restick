# Phase 05 — Pipeline integration (img_tool worker + CLI)

Date: 2026-08-20
Priority: P0
Status: pending
Depends on: phases 02, 03, 04
Updated: job complete → pending_review or awaiting_confirm; never published

## Overview

Two entry points. Processed files in MinIO. api_svc chooses `pending_review` (contributor / CLI) or `awaiting_confirm` (admin submitter). **Worker never publishes.** No generation.

## A — CLI (operators / debug)

```
img_tool process SHEET --name NAME --slug SLUG --out ./out
img_tool submit ./out/<pack-id> --ftp-url http://localhost:8081 --api-url http://localhost:8080
```

`submit` (rename from old `publish`): PUT objects, POST bundle, `PUT /internal/v1/packs` → **pending_review**. Env `RESTICK_INTERNAL_TOKEN`. Token never on argv.

## B — Worker (Web UI path)

Compose service `img_tool` polls:

1. `GET /internal/v1/jobs/claim`
2. `GET` source objects from ftp (`uploads/...`)
3. Run the same cutout as CLI (`sheet` or `files`)
4. PUT processed keys `packs/{id}/v1/...`, POST bundle
5. `PUT /internal/v1/jobs/{id}/complete` or `fail`

Idle poll 2s. One job at a time per worker replica (MVP: 1 replica).

## Submit/complete payload

Same `pack.json` fields as phase 02. img_tool does **not** set status. api_svc: contributor/CLI → `pending_review`; admin submitter → `awaiting_confirm`.

## Files to create

| Path | Layer |
| --- | --- |
| `img_tool/src/img_tool/application/submit_pack.py` | use case |
| `img_tool/src/img_tool/infrastructure/ftp_client.py` | HTTP to ftp_svc |
| `img_tool/src/img_tool/infrastructure/api_client.py` | HTTP to api_svc |
| `img_tool/src/img_tool/interfaces/cli.py` | `process`, `submit`, `worker` |
| `img_tool/src/img_tool/interfaces/worker.py` | poll loop |

Do not add `tests/` or respx suites.

## Failure modes

| Failure | Behavior |
| --- | --- |
| PUT object mid-way | fail job; orphans OK in v1 |
| Bundle fails | fail job; do not complete |
| Zero contours | fail job; pack `rejected`, reason quality |
| Claim empty | sleep, retry |

Idempotency: same pack id + same checksum → no version bump. Re-submit after reject increments version when checksum changes.

## Manual QA

1. Compose up (including img_tool worker).
2. CLI submit testdata → `pending_review` (internal token).
3. `GET /api/v1/packs` with API key → empty.
4. Admin JWT `POST /confirm` only works for **that admin’s** `awaiting_confirm` packs; CLI packs need `/admin/.../approve`.
5. Web admin upload (phase 06) → confirm on the same account.

## Success criteria

[manual-test-checklist.md](./manual-test-checklist.md) phase 05. `ruff check` clean. Worker never sets `published`.

## Next

Phase 06 Web UI: same-account admin upload+confirm; UGC review queue; revoke.
