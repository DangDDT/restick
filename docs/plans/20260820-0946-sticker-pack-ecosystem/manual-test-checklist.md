# Manual test checklist (no automated tests)

Tick in order. Skip a phase’s rows until that phase is implemented. Do not add unit/widget/pytest suites.

## Phase 01 — Bootstrap

- [ ] `docker compose up` — Postgres, MinIO, `api_svc`, `ftp_svc` start
- [ ] `curl localhost:8080/healthz`, `:8081/healthz`, `:8090/healthz` → ok
- [ ] Logs contain `grpc peer ok` for api_svc, ftp_svc, img_tool
- [ ] MinIO console + `psql` reachable
- [ ] `img_tool --help` prints
- [ ] `golangci-lint run` in `ftp_svc` and `api_svc` — clean
- [ ] `ruff check` in `img_tool` — clean
- [ ] `dart analyze` in `clients/flutter/restick` and `clients/web` — clean (đồng chí runs)

## Phase 02 — img_tool process

- [ ] `img_tool testdata/` sheet (white + 2 shapes) → 2× WebP 512×512, alpha interior non-zero
- [ ] `pack.json` matches schema v1; slug invalid → non-zero exit
- [ ] `--files` mode: two images → two stickers, no contour split
- [ ] Empty/noisy sheet → non-zero exit, message about no contours
- [ ] `ruff check` still clean

## Phase 03 — ftp_svc

- [ ] Internal PUT two WebPs + `pack.json` with `X-Internal-Token`
- [ ] POST bundle → GET zip → unzip matches files
- [ ] Missing token → 401; key with `..` → 400; HTML body → 400
- [ ] JWT PUT under `uploads/{sub}/{packId}/file.png` succeeds; wrong prefix → 403
- [ ] `golangci-lint run` in `ftp_svc` — clean

## Phase 04 — api_svc

- [ ] Register contributor, login, `GET /me`
- [ ] Bootstrap admin login from env
- [ ] Contributor `POST /confirm` → 403
- [ ] Contributor cannot `GET /admin/packs` → 403
- [ ] Internal complete as contributor submitter → status `pending_review`; SDK `GET /packs` empty
- [ ] Admin approve → pack in SDK list; bundle JSON has url + checksum
- [ ] Admin submitter complete → `awaiting_confirm`; SDK list still empty
- [ ] Same admin `POST /packs/{id}/confirm` → pack in SDK list
- [ ] Review queue omits current admin’s own `awaiting_confirm`
- [ ] Revoke → SDK list drops id; `GET /packs/{id}` → 410 `pack_revoked`; `GET /tombstones` contains id
- [ ] Grep repo: no generate/prompt/imagine routes
- [ ] `golangci-lint run` in `api_svc` — clean

## Phase 05 — Worker + CLI

- [ ] CLI `submit` testdata → `pending_review`; SDK list empty
- [ ] CLI pack: admin `/approve` → listed
- [ ] Web/worker path: contributor job → `pending_review`; admin job → `awaiting_confirm`; never `published` from worker
- [ ] Zero-contour job → pack `rejected`
- [ ] `ruff check` clean

## Phase 06 — Web UI

- [ ] Contributor: register, upload sheet, status goes processing → pending_review; cannot open `/admin`
- [ ] Admin: same account, New pack (contributor screens), after process **Confirm publish** → curl SDK list has pack
- [ ] Admin: review queue shows contributor pack, not own `awaiting_confirm`
- [ ] Approve / reject (reason required) / revoke (reason + dialog) behave as API phase 04
- [ ] UI has no generate/prompt feature (help may say generate elsewhere)
- [ ] `dart analyze` in `clients/web` — clean

## Phase 07 — Flutter SDK

- [ ] Example lists published pack, download, picker shows local files, tap → `Image.file`
- [ ] Re-open app: no re-download if checksum same
- [ ] Admin revokes pack → `syncRevocations` (startup + before picker): files and SQLite rows gone; picker hides pack
- [ ] Re-open: still gone
- [ ] Host can use `StickerService` without importing presentation
- [ ] `dart analyze` in `clients/flutter/restick` — clean

## Phase 08 — Hardening

- [ ] Publish v2 of a pack → client ETag/HEAD replaces local files
- [ ] Revoke still 410 + tombstone + disk purge
- [ ] Presigned bundle URL expires; replay fails
- [ ] Compose `healthcheck` / `/readyz` green
