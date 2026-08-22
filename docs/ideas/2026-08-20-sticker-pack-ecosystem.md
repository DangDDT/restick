# Idea: Sticker Pack Ecosystem (process → moderate → serve)

Date: 2026-08-20
Status: captured; execution plan in `docs/plans/`
Updated: 2026-08-20 — community submit, admin review/revoke, no in-product generation

## Problem

Host apps need original sticker packs. The platform must **process images into packs and serve them** — not generate images from keywords. Community contributors upload already-generated (or drawn) sheets. Admins review for copyright and community standards before anything is served. Revoked packs must disappear from the catalog **and** from devices that already downloaded them.

## Product thesis

1. Humans create images **outside** Restick (Midjourney, Flux, drawing tools). Restick never exposes a prompt/generate API.
2. Contributors (or admins) upload a sheet or sticker files via **one Web UI**.
3. `img_tool` cutouts, normalizes, writes `pack.json`.
4. Pack enters **review**. Only `published` is served to SDKs.
5. Admin may **reject**, **revoke**, or remove from the serve list. Revoke writes a **tombstone**; the Flutter SDK **must purge** local files.
6. `ftp_svc` stores bytes (MinIO). `api_svc` + **Postgres** own users, roles, catalog, reviews, tombstones.

## Legal / IP (non-negotiable)

- Generator ToS and LoRA licenses are the uploader’s duty; Restick only stores and serves.
- Character / trademark likeness is still infringement. Admin review is the gate; the pipeline cannot detect Disney.
- AI pixels are weakly copyrightable. Enforcement is Terms of Use, RBAC, revoke+purge — not “these PNGs are copyrighted.”

## Technical thesis

| Stage | Tool | Why |
| --- | --- | --- |
| Create pixels | **Out of band** | No keyword gen in product |
| Upload / review | `clients/web` (Flutter Web, one app, RBAC) | Contributor + admin in one UI |
| Cutout | rembg then OpenCV | Alpha-first; RGB contours fail |
| Pack | WebP 512×512 | WA/TG-sized |
| Store | MinIO + `ftp_svc` | Do not serve static files from Go disk in prod |
| Catalog + auth | `api_svc` + Postgres | Users, roles, status, tombstones |
| Consume | `clients/flutter` SDK | Cache + **mandatory purge on revoke** |

`ftp_svc` means **HTTP object/file service**, not RFC 959 FTP.

## Out of scope (v1)

- Any text-to-image / “generate from keyword” endpoint or UI
- Animated stickers, Lottie, video
- Creator payouts / public marketplace storefront
- gRPC, social OAuth
- SAM (only if rembg QA fails later)

## Success picture

A contributor uploads a sheet; an admin reviews then approves. An admin uploading on the **same** account confirms after preview (no second user). The Flutter demo can download published packs. Revoke removes the pack from `GET /packs` and the SDK purges device storage.
