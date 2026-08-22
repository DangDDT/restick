# Plan: Sticker Pack Ecosystem

Date: 2026-08-20
Status: draft — wait for đồng chí approval before code
Priority: P0
Updated: 2026-08-20 — DDD, lint, manual checklist (no unit tests)

## Goal

Process uploaded images into sticker packs, moderate them, serve only approved packs. Modules: `img_tool` (Python) → `ftp_svc` (Go) + MinIO → `api_svc` (Go) + **Postgres** → `clients/web` (Flutter Web, contributor+admin) and `clients/flutter` (SDK).

## Locked assumptions

- MVP, Docker Compose, &lt;100 CCU, original-IP policy, static WebP 512×512.
- **No in-product generation** (no prompt box, no keyword-to-image). Images arrive already made.
- Inter-service calls use **gRPC** on one host or across machines (`*_GRPC_ADDR`). Flutter/web stay HTTP later.
- Each backend has its own Compose file; root `docker-compose.yml` includes them all.
- Postgres is the only system of record for users, packs, reviews, tombstones.
- One Web UI, two roles: `contributor`, `admin`. Admin ⊇ contributor on **one login**; own packs need Confirm after process, not a second-account review.
- Serve list = status `published` only. Revoke ⇒ tombstone ⇒ SDK purge.
- **DDD** in every module (domain → application → infrastructure → interfaces). Domain has no SQL/HTTP/MinIO/Flutter imports.
- **Lint is mandatory** for the language of that module (see phase 00). Code that fails lint is not done.
- **No automated tests** (no unit/widget/pytest/httptest). QA = [manual-test-checklist.md](./manual-test-checklist.md).

## Phases

| Phase | File | Status |
| --- | --- | --- |
| 0 | [phase-00-decisions.md](./phase-00-decisions.md) | pending |
| 1 | [phase-01-repo-bootstrap.md](./phase-01-repo-bootstrap.md) | pending |
| 2 | [phase-02-img-tool.md](./phase-02-img-tool.md) | pending |
| 3 | [phase-03-ftp-svc.md](./phase-03-ftp-svc.md) | pending |
| 4 | [phase-04-api-svc.md](./phase-04-api-svc.md) | pending |
| 5 | [phase-05-pipeline-integration.md](./phase-05-pipeline-integration.md) | pending |
| 6 | [phase-06-webui.md](./phase-06-webui.md) | pending |
| 7 | [phase-07-flutter-sdk.md](./phase-07-flutter-sdk.md) | pending |
| 8 | [phase-08-hardening.md](./phase-08-hardening.md) | pending |
| QA | [manual-test-checklist.md](./manual-test-checklist.md) | per phase |

## Timeline

1–2 pipeline locally → 3–5 storage + catalog + jobs → 6 Web UI submit/review → 7 SDK consume **and purge** → 8 signed URL/CDN. Do not start 8 until approve→download and revoke→purge both work.

## Dependencies

`img_tool` CLI can run without Go. Worker mode needs ftp + api. Web UI needs JWT + processing jobs. SDK needs published packs **and** tombstone API.
