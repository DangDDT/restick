# Design sheets and per-key pack access

Date: 2026-11-11. Status: draft for review. The system is not live; no data migration is required.

## Goals
1. A design owns many generated sheets (images from external image providers). A pack is created from one chosen sheet of its design. One design → N sheets, N packs; each pack → exactly 1 sheet.
2. Each sheet records which provider generated it and the prompt that was used.
3. While creating a pack, the operator sees the design info and the sheet images.
4. An API key sees only the packs the operator granted to it, enforced on list, detail, bundle, object and tombstone endpoints.

## Non-goals (deferred)
Key grants by tag/design, sheet comparison, provider statistics, key usage limits, a "primary sheet" flag.

## Data model (migration `0002_design_sheets.sql`)

Not live: no backfill. Existing `packs.source_*` data is discarded.

**`design_sheets`**
- `id` TEXT PK (uuid), `design_id` TEXT NOT NULL → `designs(id)` ON DELETE RESTRICT
- `provider` TEXT NOT NULL (`chatgpt`, `gemini`, ..., `other`), `note` TEXT NOT NULL DEFAULT ''
- `prompt_snapshot` TEXT NOT NULL: the design's `compiled_prompt` at upload time
- file metadata: `filename`, `media_type`, `byte_size`, `width`, `height`, `checksum_sha256`, `relative_path` (under `sources/`)
- `created_at`
- `UNIQUE (design_id, checksum_sha256)`; index on `design_id`
- Immutable: only add or delete, never edit the image.

**`packs`**: add `sheet_id` TEXT → `design_sheets(id)` ON DELETE RESTRICT; drop `source_id`, `source_filename`, `source_checksum_sha256`, `source_width`, `source_height`, `source_byte_size`. Both `design_id` and `sheet_id` stay nullable at the DB level (SQLite cannot add a NOT NULL column without rebuilding `packs`); the service requires both on create and enforces `sheet.design_id == pack.design_id`.

**`api_key_packs`** `(key_id → api_keys ON DELETE CASCADE, pack_id → packs ON DELETE CASCADE, PRIMARY KEY (key_id, pack_id))`.

**`api_key_pack_removals`** `(key_id, pack_id, removed_at)`: written when a grant is removed, deleted on re-grant. Feeds delta sync (below).

**Provider validation:** server checks only `^[a-z0-9_-]{1,32}$`. The canonical list lives in the web client (`AI_PROVIDERS` + `other`). Adding a provider needs no migration.

**Limits:** 20 sheets per design (soft, returns 409 `sheet_limit`). Existing 8 MB / 4096 px source limits apply. Deleting a design is blocked while it has sheets (409). Deleting a sheet is blocked while a pack uses it (409), and the file is removed from disk on success.

## API

Admin (owner session):
- `POST /api/v1/admin/designs/{id}/sheets` multipart: `file`, `provider`, `note?`. Reuses `assets.SaveSource`; stores `prompt_snapshot` from the design. A duplicate checksum returns the existing sheet with 200.
- `GET /api/v1/admin/designs/{id}/sheets`; `DELETE /api/v1/admin/designs/{id}/sheets/{sheetId}`.
- Sheet image served through the existing admin asset path pattern, owner-only.
- Create/update pack takes `design_id` and `sheet_id` (both required on create). The sheet can change while the pack has no active job and is not revoked (same rules as the old `SetSource`).
- `POST /admin/packs/{id}/source` is removed.
- Pack detail returns `sheet` (id, provider, thumb URL, size) and `design` summary.
- `GET/PUT /api/v1/admin/keys/{id}/packs`: `PUT` replaces the full set of pack ids; the diff writes `api_key_packs` and `api_key_pack_removals` in one transaction. A new key starts with no packs. The per-pack add/remove UI uses the same endpoint.

Public (X-Api-Key):
- `/api/v1/packs`, `/api/v1/packs/{id}`, bundle and `/v1/objects` / `/v1/bundles` resolve the pack id and return 404 unless the key holds a grant. Owner session is unfiltered.
- Tombstones for a key = revoke tombstones of granted packs ∪ `api_key_pack_removals` entries, using the same `revoked_at` cursor semantics (`reason_code = "access_removed"`). The Flutter client already purges on a tombstone, so it needs no change beyond handling the new reason code as a purge.

## Jobs
`Enqueue` reads the source from the pack's sheet (join `design_sheets`) instead of `packs.source_*`. The job request schema is unchanged: the `source` ref still carries id, path and checksum. Committed versions keep their own snapshot as before.

## Web UI (`clients/web-next`)
- **Design page, new "Sheets" tab:** upload form (file, provider dropdown from `AI_PROVIDERS` + "Khác", note), thumbnail grid with provider badge and date, delete, preview of the stored prompt snapshot.
- **New pack form:** pick design, then show a design panel (name, character, rows×columns, outline, compiled prompt) and a grid of that design's sheet images with provider badges. The operator selects one sheet; the selection is required to submit. A design with no sheets shows a prompt to upload one.
- **Pack detail:** shows the design panel and the chosen sheet.
- **Keys page:** per key, a checklist of all packs with search. Tick/untick saves per change (or one save button); the same control is available when creating the key.

## Backup / restore
`maintenance` backup and restore must include the new tables; sheet files already live under `sources/`. Verify the restore fixture covers `design_sheets`, `api_key_packs` and `api_key_pack_removals`.

## Testing
- Go: sheet add/dedupe/limit/delete (RESTRICT paths), design delete blocked, pack create rejects a mismatched design/sheet, key filtering on every public route including bundle/object, tombstone merge for removals and re-grant, `Enqueue` from a sheet, maintenance round-trip.
- Web: update `qa/interact.mjs` for the Sheets tab, pack form selection and key checklist.

## Risks
- Disk growth (up to 20 × 8 MB per design); the limit is a constant to tune.
- Removal tombstones share the cursor with revoke tombstones; a client that has never synced sees both on first run, which is harmless.
- Dropping `source_*` and `design_id` nullability breaks any dev database and seed code that creates packs without a design; the seed and tests must be updated.
