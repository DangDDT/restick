# Phase 08 — Hardening and ops

Date: 2026-08-20
Priority: P1
Status: pending
Depends on: phase 07 demo green **including revoke→purge**

## Overview

Sync polish, abuse controls, production storage. Do not start here. Generation remains forbidden.

## Work items (order)

1. **ETag / version sync** (non-revoked updates)
   - `HEAD /api/v1/packs/{id}` vs local checksum.
   - Mismatch → re-download zip. Revoke still uses tombstones (already in phase 07).

2. **Presigned GET**
   - Short-lived MinIO URLs for bundle/objects.
   - Unpublished/revoked keys must not presign for API-key callers.

3. **CDN**
   - Cloudflare/CloudFront in front of versioned keys. Compose stays MinIO-only.

4. **Hotlink**
   - Signed URL is enough. No watermark unless leaks become a problem.

5. **Observability**
   - JSON request logs without tokens/passwords.
   - `/readyz`: Postgres (api), MinIO (ftp), worker heartbeat optional.

6. **Rate limit**
   - Register/login, catalog 60/min per API key, bundle 10/min, upload 20 files/hour per user.

7. **Job timeout**
   - `running` jobs older than 10 min → `queued` retry, max 3, then `rejected`.

8. **SAM escape hatch**
   - Only if rembg fails in production QA. Not a generate model.

9. **Legal copy**
   - Web + SDK example: upload original work; admin may revoke; devices will delete revoked packs.

## Success criteria

[manual-test-checklist.md](./manual-test-checklist.md) phase 08. All linters still clean.

## Explicit non-goals

- Keyword / text-to-image generation
- gRPC
- Social OAuth
- Animated stickers
- WhatsApp Business upload API
- Creator payouts

## After phase 08

New plan folder if đồng chí wants Telegram `.tgs`, animated WebP, or a storefront.
