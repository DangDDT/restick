# Phase 04 — api_svc (Go catalog + auth + moderation)

Date: 2026-08-20
Priority: P0
Status: pending
Depends on: phase 01, contracts from phase 00
Updated: review/revoke/tombstones, JWT users, no auto-publish

## Overview

Postgres-backed system of record. Pagination, tags, pack detail, **users/roles**, **submit/review/revoke**, **tombstones**. Points clients at `ftp_svc` URLs. No binary I/O. No generation endpoints.

## Auth

| Caller | Credential | Use |
| --- | --- | --- |
| Web UI | JWT (email/password) | submit, my packs, admin review |
| Host SDK | `X-Api-Key` | list/get **published**, tombstones, bundle meta |
| img_tool | `X-Internal-Token` | claim jobs, PUT pack after process |

Bootstrap: if `users` empty, insert admin from `BOOTSTRAP_ADMIN_EMAIL` / `BOOTSTRAP_ADMIN_PASSWORD`. If `api_keys` empty, insert hash of `BOOTSTRAP_API_KEY`.

## SDK / public catalog (`X-Api-Key`)

- `GET /api/v1/packs?page=&page_size=&tag=` — **`published` only**
- `GET /api/v1/packs/{id}` — 200 if published; **410** if revoked (body includes `reason_code`); 404 otherwise
- `HEAD /api/v1/packs/{id}` — ETag if published; 410 if revoked
- `GET /api/v1/packs/{id}/bundle` — JSON `{ url, checksum_sha256, version }` if published; **410** if revoked; never stream zip
- `GET /api/v1/tombstones?since=` — `{ items: [{ pack_id, revoked_at, reason_code, checksum_sha256 }] }` monotonic by `revoked_at`. SDK polls this. **No pagination required under 10k tombstones in MVP;** cap 1000 per response + `since` cursor.

## Web auth

- `POST /api/v1/auth/register` `{ email, password, display_name }` → role `contributor`
- `POST /api/v1/auth/login` → `{ access_token, expires_in, user }`
- `GET /api/v1/me`

## Contributor APIs (`Authorization: Bearer`) — **admin uses these too**

Admin has every contributor route on the same JWT. No second user.

- `POST /api/v1/packs` — create draft `{ name, slug, author, tags, source_kind }` → id, upload prefix
- `POST /api/v1/packs/{id}/submit` — enqueue job, status `processing` (must own pack, status draft/rejected)
- `GET /api/v1/me/packs` — own packs all statuses
- `GET /api/v1/packs/{id}` with JWT — owner or admin may see non-published
- `POST /api/v1/packs/{id}/confirm` — **admin + owner only**, status `awaiting_confirm` → `published`. Body empty. Writes `pack_reviews` action `confirm`. 403 if contributor, 409 if not owner or wrong status.

No `POST /generate`. No `prompt` field anywhere.

## Admin (`role=admin`) — extra moderation

- `GET /api/v1/admin/packs?status=pending_review` — **community queue only** (exclude rows where `submitter_id = current user`)
- `GET /api/v1/admin/packs/{id}` — full stickers + source keys for preview
- `POST /api/v1/admin/packs/{id}/approve` — `pending_review` → `published`; `pack_reviews` action `approve`
- `POST /api/v1/admin/packs/{id}/reject` `{ reason_code, reason_text }` required
- `POST /api/v1/admin/packs/{id}/revoke` `{ reason_code, reason_text }` required — `published` → `revoked` + tombstone. Idempotent.

Do not send the admin’s own packs through `/approve` as if they were UGC. Own packs use `/confirm` after preview.

Revoke is the only way to pull a live pack from the serve list. Do not `DELETE` the pack row in v1 (breaks tombstone history). Soft-delete = revoke.

## Internal (`X-Internal-Token`)

- `GET /internal/v1/jobs/claim` — atomically `queued` → `running`, return job + source prefix
- `PUT /internal/v1/jobs/{id}/complete` — body pack.json + keys. api_svc sets status from submitter role: `contributor` → `pending_review`; `admin` → `awaiting_confirm`. **Never** `published` here.
- `PUT /internal/v1/jobs/{id}/fail` — pack → `rejected`, reason `quality` / processing error text
- `PUT /internal/v1/packs` — CLI; always `pending_review` (internal token is not a user). Publish still needs Web confirm/approve.

## URL composition

Env `PUBLIC_FTP_BASE`. Persist **keys** only. Build URLs at read time.

Admin preview URLs for unpublished objects: short-lived path via `GET /api/v1/admin/packs/{id}` returning ftp URLs that require JWT on ftp_svc **or** api_svc-issued download token (MVP: ftp_svc GET with JWT for keys under that pack + `uploads/` of that pack).

## Schema (`api_svc/migrations/0001_init.sql`)

Tables as in phase 00: `users`, `packs`, `stickers`, `pack_versions`, `pack_reviews`, `pack_tombstones`, `processing_jobs`, `api_keys`.

Indexes: `packs(status, updated_at desc)`, GIN `packs.tags`, unique `packs.slug`, `pack_tombstones(revoked_at)`, `processing_jobs(status)`.

`packs.status` check: `processing | pending_review | awaiting_confirm | rejected | published | revoked`.

## Files to create

| Path | Layer |
| --- | --- |
| `api_svc/internal/domain/user/` | user aggregate + role |
| `api_svc/internal/domain/pack/` | pack aggregate, status transitions, `PackRepository` port |
| `api_svc/internal/domain/tombstone/` | tombstone VO |
| `api_svc/internal/application/auth/` | register/login |
| `api_svc/internal/application/catalog/` | list/get published |
| `api_svc/internal/application/submit/` | create, submit, confirm |
| `api_svc/internal/application/moderate/` | approve, reject, revoke |
| `api_svc/internal/application/jobs/` | claim, complete, fail |
| `api_svc/internal/infrastructure/postgres/` | pgx adapters |
| `api_svc/internal/interfaces/http/` | router, JWT middleware, handlers |
| `api_svc/migrations/0001_init.sql` | schema |

```go
type PackRepository interface {
    ListPublished(ctx context.Context, q ListQuery) (items []PackSummary, total int, err error)
    GetPublished(ctx context.Context, id uuid.UUID) (PackDetail, error)
    ListTombstones(ctx context.Context, since time.Time) ([]Tombstone, error)
    Revoke(ctx context.Context, id uuid.UUID, by uuid.UUID, reason Reason) error
    Approve(ctx context.Context, id uuid.UUID, by uuid.UUID) error
    ConfirmOwn(ctx context.Context, id uuid.UUID, by uuid.UUID) error
}
```

Port lives in `domain`. Status rules (`pending_review` vs `awaiting_confirm`) live in domain, not in handlers. No `*_test.go`.

Revoke, approve, and confirm are **transactions**: status update + review row + (revoke) tombstone insert.

## Implementation steps

1. Migrate on boot. Seed admin + API key.
2. JWT middleware; role guard.
3. SDK list filters `published` only.
4. Revoke: 410 on get/bundle; tombstone visible via `since=epoch` for full catch-up.
5. Structured errors `{ "error": { "code", "message" } }`. `code=pack_revoked` for 410.
6. CORS for `WEB_ORIGIN`.

## Success criteria

[manual-test-checklist.md](./manual-test-checklist.md) phase 04. `golangci-lint run` clean. Domain owns status transitions.

## Security

- UUID in path. pgx args only.
- Contributor cannot approve or confirm. Admin confirms **own** `awaiting_confirm` only; approves **others’** `pending_review`.
- Rate-limit register/login (phase 08 can tighten; add a simple per-IP cap now if cheap).

## Next

Phase 05: worker completes into `pending_review` or `awaiting_confirm`.
