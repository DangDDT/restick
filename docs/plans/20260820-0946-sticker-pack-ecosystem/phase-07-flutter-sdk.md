# Phase 07 — clients/flutter (SDK + example)

Date: 2026-08-20
Priority: P0
Status: pending
Depends on: phase 04–05 APIs running locally (manual against Compose)
Updated: mandatory revoke purge

## Overview

Package `restick`: DDD (`domain` / `application` / `infrastructure` / `presentation`). Separate widget classes. `ColorScheme` only. No custom `TextStyle` / `fontSize`. English. Typed. No `dynamic`. Download/cache **published** packs. **On tombstone, delete local files.** Example is the host. **No unit/widget tests.**

## Public API

```dart
class RestickClient {
  RestickClient({required Uri apiBase, required String apiKey});
}

class StickerService {
  Future<List<StickerPackSummary>> listPacks({String? tag});
  Future<StickerPackDetail> getPack(String packId);
  Future<void> downloadPack(String packId);
  Future<void> deletePack(String packId);
  Future<List<StickerAsset>> getStickers(String packId);
  Stream<PackDownloadState> watchPack(String packId);
  /// Fetch tombstones and delete local bytes. Call on startup and before picker.
  Future<List<String>> syncRevocations();
}

class StickerPickerWidget extends StatelessWidget { /* ... */ }
class StickerPackManagerScreen extends StatelessWidget { /* ... */ }
```

`getPack` / `downloadPack`: if API returns 410, treat as revoked — purge and throw/return `PackDownloadState.revoked`. Do not keep the zip.

`StickerAsset.file` is local `File` after download. Host must not cache that path beyond the SDK.

## Storage

- Drift: `cached_packs` (id, version, checksum, status, localDir), `cached_stickers`.
- Status: `remote | downloading | downloaded | failed | revoked`.
- Files: `getApplicationDocumentsDirectory()/restick/packs/{id}/v{n}/`.
- Checksum verify after unzip; mismatch → delete dir, `failed`.

## Revocation (P0, not optional)

1. Persist `last_tombstone_at` (ISO timestamp).
2. `GET /api/v1/tombstones?since={last}` on `StickerService` init, app resume, and immediately before opening `StickerPickerWidget`.
3. For each `pack_id`: delete directory, delete SQLite rows, emit `revoked`.
4. Advance cursor even if some deletes fail; retry those ids next time (keep a `pending_purge` table if needed).
5. `listPacks` + local union must **hide** revoked packs from the picker.
6. If user had the pack downloaded, manager shows a one-line “Removed by server” then drops the row after purge.

Do not wait for phase 08. A revoked pack remaining on disk is a product bug.

## Download manager

1. `syncRevocations` first.
2. `getPack` → bundle URL + checksum + version. 410 → purge.
3. Local version+checksum match → no-op.
4. Download zip, unzip, swap index, delete old version dir.
5. One download per pack id at a time.

`http` + `archive`. No Dio unless required. Grid decode with `cacheWidth` / `cacheHeight`.

## Widget split

| Widget | Role |
| --- | --- |
| `StickerPickerWidget` | Grid, pack chips, onStickerSelected; calls syncRevocations |
| `StickerPackChip` | One pack |
| `StickerGrid` | Stickers of selected pack |
| `StickerGridTile` | One thumb |
| `StickerPackManagerScreen` | List, download/delete |
| `PackDownloadBanner` | Progress / error / revoked |

Spacing: `Row`/`Column` `spacing`. Const constructors where possible.

## Package layout

```
clients/flutter/restick/lib/
  restick.dart
  src/domain/sticker_pack.dart
  src/domain/sticker_asset.dart
  src/domain/pack_download_state.dart
  src/application/sticker_service.dart
  src/infrastructure/restick_api.dart
  src/infrastructure/pack_local_store.dart
  src/infrastructure/pack_downloader.dart
  src/infrastructure/tombstone_sync.dart
  src/presentation/sticker_picker_widget.dart
  src/presentation/sticker_pack_manager_screen.dart
  src/presentation/sticker_grid.dart
  src/presentation/sticker_grid_tile.dart
  src/presentation/sticker_pack_chip.dart
  src/presentation/pack_download_banner.dart
```

One export: `lib/restick.dart` only.

## Example host app

- `clients/flutter/restick/example`
- `--dart-define=RESTICK_API_BASE=http://10.0.2.2:8080`
- Manager + fake composer `Image.file`.
- Button “Sync removals” calling `syncRevocations` for QA.

## Manual QA

[manual-test-checklist.md](./manual-test-checklist.md) phase 07. `dart analyze` (đồng chí). No `test/` in the package.

## Success criteria

Checklist phase 07 ticked. Analyze clean.

## Risks

Android cleartext to Compose: example only. iOS ATS localhost exception in example.

## Next

Phase 08 hardening (ETag, presign, CDN, rate limit).
