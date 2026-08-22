# Phase 01 — Repo bootstrap

Date: 2026-08-20
Priority: P0
Status: pending
Depends on: phase 00 approved

## Overview

Create monorepo skeleton, Compose, and contracts. No business logic yet.

## Requirements

- Five module roots exist: `ftp_svc`, `api_svc`, `img_tool`, `clients/flutter`, `clients/web`.
- Compose starts **Postgres 16** + MinIO + both Go stubs + img_tool worker stub.
- OpenAPI describes public catalog, auth, submit/review/revoke, tombstones, and internal job APIs.
- README first paragraph: **Restick does not generate images from keywords.**
- Lint configs exist per module (phase 00). DDD folders exist even if empty besides `healthz`.
- No `*_test.go` / `tests/` directories.

## Target tree

```
restick/
  README.md
  docs/
  contracts/openapi/v1.yaml
  deploy/docker-compose.yml
  deploy/minio-init.sh
  ftp_svc/cmd/ftp_svc/main.go
  ftp_svc/internal/domain/
  ftp_svc/internal/application/
  ftp_svc/internal/infrastructure/
  ftp_svc/internal/interfaces/http/
  ftp_svc/.golangci.yml
  ftp_svc/go.mod
  ftp_svc/Dockerfile
  api_svc/cmd/api_svc/main.go
  api_svc/internal/domain/
  api_svc/internal/application/
  api_svc/internal/infrastructure/
  api_svc/internal/interfaces/http/
  api_svc/.golangci.yml
  api_svc/migrations/0001_init.sql
  api_svc/go.mod
  api_svc/Dockerfile
  img_tool/pyproject.toml
  img_tool/src/img_tool/domain/
  img_tool/src/img_tool/application/
  img_tool/src/img_tool/infrastructure/
  img_tool/src/img_tool/interfaces/cli.py
  img_tool/Dockerfile
  clients/flutter/restick/analysis_options.yaml
  clients/flutter/restick/pubspec.yaml
  clients/flutter/restick/lib/restick.dart
  clients/flutter/restick/example/lib/main.dart
  clients/web/analysis_options.yaml
  clients/web/pubspec.yaml
  clients/web/lib/main.dart
```

## Implementation steps

1. Add root `README.md`: purpose, module map, `docker compose up`, `img_tool` CLI one-liner. No marketing.
2. Go: module paths `github.com/eykh/restick/ftp_svc` and `.../api_svc` unless đồng chí has a real module prefix — then use that. Each `main` serves `GET /healthz` → `{"status":"ok"}`.
3. Config via env: `HTTP_ADDR`, `INTERNAL_TOKEN`, `JWT_SECRET`, `BOOTSTRAP_ADMIN_EMAIL`, `BOOTSTRAP_ADMIN_PASSWORD`, `BOOTSTRAP_API_KEY`, MinIO/Postgres DSN. No `.env` committed; `.env.example` only.
4. Postgres: one database `restick`. User/password in Compose only. This is the system of record (users, packs, jobs, tombstones).
5. MinIO: bucket `restick-assets`. Path style. Root user in Compose.
6. Python: `pyproject.toml` with `img_tool` script entry. CLI `img_tool --help` works with Typer. No rembg install required to print help.
7. Flutter SDK package `restick` as before. Flutter Web app `clients/web` named `restick_web`: placeholder login screen, no generate UI.
8. `contracts/openapi/v1.yaml`: paths in phases 03–04. Source of truth; do not generate Go yet (YAGNI).
9. Commit lint configs: identical `.golangci.yml` for both Go modules; Ruff in `pyproject.toml`; `analysis_options.yaml` includes `package:flutter_lints/flutter.yaml` plus strict-casts / strict-raw-types.

## Compose services (phase 1)

| Service | Image / build | Ports |
| --- | --- | --- |
| postgres | postgres:16 | 5432 |
| minio | minio/minio | 9000, 9001 |
| ftp_svc | ./ftp_svc | HTTP 8081, gRPC 50052 |
| api_svc | ./api_svc | HTTP 8080, gRPC 50051 |
| img_tool | ./img_tool | HTTP 8090, gRPC 50053 |

`clients/web` and the SDK example run on the host (`flutter run -d chrome` / device). Do not Compose-build Flutter in v1.

## Success criteria

Manual: [manual-test-checklist.md](./manual-test-checklist.md) phase 01. Lint configs present. DDD dirs exist.

## Risks

Empty DDD folders invite fake layers. Until phase 3, only `cmd` + `interfaces/http` healthz + `infrastructure/config` is enough — do not invent unused aggregates.

## Next

Phase 02 first so sample assets exist for **manual** ftp/api checks.
