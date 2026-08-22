# Phase 06 — clients/web (contributor + admin)

Date: 2026-08-20
Priority: P0
Status: pending
Depends on: phase 05 worker green

## Overview

One Flutter Web app `clients/web` (`restick_web`). **Admin uses the exact contributor flows on the same login** plus moderation nav. No second account. **No generate/prompt screen.**

DDD + Flutter: domain / application / infrastructure / presentation. Separate widget classes. `ColorScheme` only. No custom `TextStyle` / `fontSize`. English. Typed. No `dynamic`. **No widget tests.**

## Roles

| Role | Screens |
| --- | --- |
| `contributor` | Login, Register, My packs, New pack (upload), Pack detail (status, reject reason) |
| `admin` | **Same contributor screens** + Review queue (UGC only), Review detail, Published list, Revoke |

Shared `AppShell`. Admin default home can be My packs (they upload here). Review queue is extra nav, not a different user. `/admin/*` requires `admin`.

## Contributor flow

1. Register / login.
2. New pack: name, slug, tags, `source_kind` sheet vs files.
3. File picker: one sheet **or** multiple images. PUT to ftp_svc `uploads/{user}/{packId}/`.
4. Submit. Poll until `pending_review` | `awaiting_confirm` | `rejected`.
5. Copy: “Restick does not generate stickers. Upload a sheet or files you created elsewhere.”

**If the logged-in user is admin:** after processing, pack detail shows the sticker grid and a primary **Confirm publish** (calls `POST /packs/{id}/confirm`). Optional second-line copy: “This skips the community review queue because you are an admin.” Dialog: “Publish to the serve list?” — required, not a silent go-live.

## Admin moderation flow (other people’s packs)

1. Review queue: `pending_review` only (not the current user’s `awaiting_confirm`).
2. Detail: Approve / Reject (reason required).
3. Published list: Revoke (reason + confirm: devices must purge).
4. No “generate” button. No “log in as contributor to upload.”

## Files

```
clients/web/lib/
  main.dart
  src/domain/user.dart
  src/domain/pack.dart
  src/application/session_controller.dart
  src/application/pack_controller.dart
  src/infrastructure/auth_api.dart
  src/infrastructure/pack_api.dart
  src/infrastructure/upload_client.dart
  src/presentation/app_shell.dart
  src/presentation/login_screen.dart
  src/presentation/register_screen.dart
  src/presentation/my_packs_screen.dart
  src/presentation/new_pack_screen.dart
  src/presentation/pack_detail_screen.dart
  src/presentation/review_queue_screen.dart
  src/presentation/review_detail_screen.dart
  src/presentation/published_packs_screen.dart
  src/presentation/revoke_dialog.dart
```

One export is not required for an app (only for the SDK package). Keep presentation as widget classes.

## Implementation notes

- `go_router` + redirect: unauthenticated → login; `/admin/*` requires `admin`.
- API base and ftp base via `--dart-define`.
- Polling: 2s while `processing`, stop on terminal status.
- Revoke/reject: disable submit until `reason_text` non-empty.

Agents do not run `flutter` commands; đồng chí does (`dart analyze` + this UI checklist).

## Success criteria

[manual-test-checklist.md](./manual-test-checklist.md) phase 06. Analyze clean.

## Risks

Flutter Web file picker + PUT CORS: ftp_svc and api_svc must allow `WEB_ORIGIN` including `PUT`.

Cleartext localhost: document running Chrome against Compose.

## Next

Phase 07 SDK: consume published + **purge revoked**.
