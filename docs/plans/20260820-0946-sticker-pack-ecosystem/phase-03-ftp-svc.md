# Phase 03 — ftp_svc (Go object gateway)

Date: 2026-08-20
Priority: P0
Status: pending
Depends on: phase 01 (sample objects from phase 02 testdata)

## Overview

HTTP service in front of MinIO. Stores sticker bytes and builds pack zip bundles. Does not know tags or search.

## Functional requirements

Internal (header `X-Internal-Token`):

- `PUT /internal/v1/objects/{key}` — body = file, `Content-Type` required. Key must match `packs/[0-9a-f-]{36}/v[0-9]+/[a-z0-9._-]+` **or** worker writing processed output.
- `POST /internal/v1/bundles` — JSON `{ "pack_id", "version", "keys": ["...webp"] }` → writes `packs/{id}/v{n}/bundle.zip`, returns `bundle_key`.
- `GET /internal/v1/objects/{key}` — head metadata (size, etag) for api_svc.

Host-app (header `X-Api-Key`):

- `GET /v1/objects/{key}` — stream from MinIO (presign in phase 08). Only keys of **published** packs should be handed out by `api_svc`; ftp_svc may still 200 if the key is guessed — do not list the bucket.
- `GET /v1/bundles/{pack_id}/v/{version}` — stream `bundle.zip`. `api_svc` must not give this URL for revoked packs.

Web contributor (header `Authorization: Bearer JWT`, same `JWT_SECRET` as `api_svc`):

- `PUT /v1/objects/{key}` where key matches `uploads/{jwt.sub}/{pack_uuid}/{filename}`.
- Allowed types: PNG/JPEG/WebP. Max 8 MiB. This is **source** art only.

`GET /healthz` stays open.

## Non-functional

- Max PUT 8 MiB per object; max zip 32 MiB.
- Sniff first bytes: WebP `RIFF....WEBP` or PNG/JPEG for thumbs only. Reject HTML/XML.
- Timeouts: 30s PUT, 60s bundle.

## Files to create

| Path | Action | Layer |
| --- | --- | --- |
| `ftp_svc/internal/domain/objectstore/port.go` | `ObjectStore` port | domain |
| `ftp_svc/internal/domain/objectkey/key.go` | key validation | domain |
| `ftp_svc/internal/application/put_object.go` | use case | application |
| `ftp_svc/internal/application/build_bundle.go` | use case | application |
| `ftp_svc/internal/infrastructure/minio/store.go` | MinIO adapter | infrastructure |
| `ftp_svc/internal/interfaces/http/router.go` | chi or stdlib | interfaces |
| `ftp_svc/internal/interfaces/http/auth.go` | tokens + JWT | interfaces |
| `ftp_svc/internal/interfaces/http/handlers.go` | PUT/GET/bundle | interfaces |

Domain port (implemented only in infrastructure):

```go
type ObjectStore interface {
    Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
    Get(ctx context.Context, key string) (io.ReadCloser, int64, string, error)
    Stat(ctx context.Context, key string) (size int64, etag string, err error)
}
```

Do not add `*_test.go`.

## Bundle zip layout

```
pack.json
000.webp
001.webp
...
thumbs/thumb_000.webp
thumb.webp
```

`pack.json` is uploaded by img_tool as an object, then included in zip by listing keys from the POST body — ftp_svc does not regenerate JSON.

## Implementation steps

1. MinIO client from env `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET`, `MINIO_USE_SSL`.
2. Auth middleware: constant-time compare for tokens.
3. PUT path: validate key, size, magic, `PutObject`.
4. Bundle: stream objects into `archive/zip`, `Put` zip key.
5. GET: `Content-Type` from stored metadata; `ETag` from MinIO; `Cache-Control: public, max-age=31536000, immutable` for versioned keys.

## Success criteria

[manual-test-checklist.md](./manual-test-checklist.md) phase 03. `golangci-lint run` clean.

## Security

- Do not list bucket publicly.
- Do not accept `../` or absolute keys.
- Internal routes not registered on a second public mux by mistake — same port is OK if auth is mandatory on `/internal/*`.

## Risks

Zip-bomb: cap uncompressed count (max 200 files) and size.

## Next

`api_svc` stores `bundle_key` and thumbnail keys only; never file bytes.
