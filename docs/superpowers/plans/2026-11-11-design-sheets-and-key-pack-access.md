# Design Sheets and Per-Key Pack Access Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A design owns many provider-tagged sheets, a pack is created from one chosen sheet, and each API key sees only the packs granted to it.

**Architecture:** New tables `design_sheets`, `api_key_packs`, `api_key_pack_removals` (migration `0002`). Sheet logic lives in `internal/studio` (designs own sheets, same pattern as reference assets). Pack ↔ sheet linkage lives in `internal/packs` and `internal/jobs`. Key grants and removal tombstones live in `internal/auth`; the public handlers resolve the caller to a principal and filter. Web gets a Sheets panel on the design page, a design+sheet picker on the new-pack form, and a pack checklist per key.

**Tech Stack:** Go (modernc sqlite, `net/http` mux), Next.js 16 + React 19 + TanStack Query, Flutter client (unchanged).

**Spec:** `docs/superpowers/specs/2026-11-11-design-sheets-and-key-pack-access-design.md`

## Global Constraints

- Not live: no backfill; `packs.source_*` data is discarded; dev DBs are recreated.
- Migration file `services/api/internal/storage/migrations/0002_design_sheets.sql` (numbering has no gaps).
- Timestamps are UTC text `YYYY-MM-DDTHH:MM:SSZ` (`strftime('%Y-%m-%dT%H:%M:%SZ','now')`).
- Provider regex `^[a-z0-9_-]{1,32}$`; sheet limit 20 per design; source limits unchanged (8 MiB, 4096 px).
- `design_id` and `sheet_id` stay nullable in SQL; the pack service requires both on create.
- Non-granted pack on any public route returns **404 `not_found`** (never 410, so revoked packs do not leak).
- Removal tombstone `reason_code` is `"access_removed"`; the Flutter client treats `reason_code` as an opaque string and purges.
- Go module path `github.com/eykh/restick/api_svc`; HTTP package is `httpx`.
- Run Go tests from `services/api` with `go test ./...`; web checks from `clients/web-next` with `npm run lint && npx tsc --noEmit`.
- Commits follow repo style (`feat(api): ...`), one per task.

## Review Focus

1. Key with zero grants → `/api/v1/packs` returns `items: []`, `total: 0` (not 401/500). Pinned in Task 7.
2. Same image uploaded twice to one design (including two concurrent uploads) → one sheet row, no orphan file on disk. Pinned in Task 2.
3. Key rotation must carry the grants to the replacement key, otherwise a rotate silently blanks the client. Pinned in Task 8.
4. Re-granting a removed pack clears its removal tombstone; removing then re-granting must not purge the client. Pinned in Task 6.
5. Pack create/update with a sheet from another design, or an unknown sheet id → 422, not 500. Pinned in Task 3.

---

### Task 1: Migration 0002

**Files:**
- Create: `services/api/internal/storage/migrations/0002_design_sheets.sql`
- Modify: `services/api/internal/storage/db_test.go` (extend schema assertions)

**Interfaces:**
- Produces: tables `design_sheets(id, design_id, provider, note, prompt_snapshot, filename, media_type, byte_size, width, height, checksum_sha256, created_at)` with `UNIQUE(design_id, checksum_sha256)` and index on `design_id`; `packs.sheet_id`; `api_key_packs(key_id, pack_id)`; `api_key_pack_removals(key_id, pack_id, removed_at)`; `packs.source_*` columns dropped. `design_sheets.id` is also the directory name under `sources/` (it equals the `assets.Source.ID` returned by `SaveSource`).

- [ ] **Step 1: Write the failing test** in `db_test.go`: `TestMigration0002Schema` opens a fresh DB with `Open`, then asserts via `pragma_table_info`/`sqlite_master` that `design_sheets`, `api_key_packs`, `api_key_pack_removals` exist, `packs` has `sheet_id` and has none of `source_id`, `source_filename`, `source_checksum_sha256`, `source_width`, `source_height`, `source_byte_size`; and that inserting two `design_sheets` rows with the same `(design_id, checksum_sha256)` fails, and deleting a design that has a sheet fails (RESTRICT, with `PRAGMA foreign_keys` on as `Open` sets it).
- [ ] **Step 2: Run** `go test ./internal/storage -run TestMigration0002Schema -v`. Expected: FAIL.
- [ ] **Step 3: Write the migration** per the spec's Data model: `design_id` FK `ON DELETE RESTRICT`; `provider` and `checksum_sha256` CHECKs (`length(checksum_sha256) = 64`); `ALTER TABLE packs ADD COLUMN sheet_id TEXT REFERENCES design_sheets (id) ON DELETE RESTRICT`; six `ALTER TABLE packs DROP COLUMN source_*`; `api_key_packs` / `api_key_pack_removals` with composite PK and `ON DELETE CASCADE` FKs; index `api_key_packs_pack(pack_id)`.
- [ ] **Step 4: Run** `go test ./internal/storage -v`. Expected: PASS. (Other packages will not compile-test cleanly until Task 3/4; do not run `./...` yet.)
- [ ] **Step 5: Commit** `feat(api): add design_sheets and key grant tables`.

---

### Task 2: Sheet service in studio

**Files:**
- Modify: `services/api/internal/assets/store.go` (add `RemoveSource`), `services/api/internal/assets/store_test.go`
- Modify: `services/api/internal/studio/types.go` (types), `services/api/internal/studio/designs.go` (`DeleteDesign`)
- Create: `services/api/internal/studio/sheets.go`
- Test: `services/api/internal/studio/studio_test.go` (uses existing `newTestService`)

**Interfaces:**
- Produces in `assets`: `func (s *Store) RemoveSource(id string) error`: removes `sources/<id>` after `verifyDir`; missing dir is not an error.
- Produces in `studio`: `type Sheet struct { ID, DesignID, Provider, Note, PromptSnapshot, Filename, MediaType string; ByteSize int64; Width, Height int; ChecksumSHA256 string; CreatedAt time.Time }` (json tags snake_case matching spec); `var ErrSheetLimit, ErrSheetInUse, ErrDesignHasSheets error` (plus `ErrNotFound`, `ErrInvalid`, `ErrFilesUnavailable` existing); `const MaxSheetsPerDesign = 20`.
- Produces methods on `*Service`: `SaveSheet(ctx, designID string, r io.Reader, filename, provider, note string) (Sheet, error)`, `ListSheets(ctx, designID string) ([]Sheet, error)` (newest first), `GetSheet(ctx, designID, sheetID string) (Sheet, error)`, `OpenSheet(ctx, designID, sheetID string) (Sheet, *os.File, error)`, `DeleteSheet(ctx, designID, sheetID string) error`.
- `DeleteDesign` now returns `ErrDesignHasSheets` when sheets exist.

- [ ] **Step 1: Write failing tests** in `studio_test.go`, each saving a real PNG built in-test (reuse any existing PNG helper; otherwise `image/png` of 8×8):
  - `TestSaveSheetStoresProviderAndPromptSnapshot`: after saving a design, `SaveSheet(..., "gemini", "v2")` returns a sheet with `Provider=="gemini"`, `Note=="v2"`, `PromptSnapshot==design.Prompt`; `ListSheets` returns it.
  - `TestSaveSheetRejectsBadProvider`: `""`, `"Bad Provider"`, 33 chars → `ErrInvalid`; and no directory is left under `sources/`.
  - `TestSaveSheetDuplicateChecksumReturnsExisting`: same bytes twice → same `ID`, `ListSheets` length 1, `sources/` has exactly one subdirectory. Add a concurrent variant with two goroutines and `sync.WaitGroup`: still one row, one directory.
  - `TestSaveSheetLimit`: 20 distinct images succeed, the 21st returns `ErrSheetLimit` and leaves no directory.
  - `TestDeleteSheetBlockedWhilePackUsesIt`: insert a `packs` row with `sheet_id` via raw SQL, `DeleteSheet` → `ErrSheetInUse`; after clearing `sheet_id`, `DeleteSheet` succeeds and the `sources/<id>` directory is gone.
  - `TestDeleteDesignBlockedWithSheets`: `DeleteDesign` → `ErrDesignHasSheets`; after deleting the sheet it succeeds.
  - In `store_test.go`: `TestRemoveSource` removes a saved source dir, is idempotent, and rejects a non-uuid id with `ErrUnsafePath`/`ErrNotFound`.
- [ ] **Step 2: Run** `go test ./internal/studio ./internal/assets -v`. Expected: FAIL (undefined).
- [ ] **Step 3: Implement** `RemoveSource` and `sheets.go`. Approach: validate provider and note length (≤ 500) first; load the design (`GetDesign`) for `PromptSnapshot`; check the count against `MaxSheetsPerDesign` before saving; call `s.files.SaveSource`; `INSERT` with the source id as `id`; on a UNIQUE conflict or a failed insert, `RemoveSource(source.ID)` then (conflict only) return the existing row looked up by `(design_id, checksum)`. `DeleteSheet` maps the FK-restrict SQLite error to `ErrSheetInUse`, and calls `RemoveSource` only after the row is deleted.
- [ ] **Step 4: Run** the same command. Expected: PASS.
- [ ] **Step 5: Commit** `feat(studio): design sheets with provider and prompt snapshot`.

---

### Task 3: Packs reference a sheet

**Files:**
- Modify: `services/api/internal/packs/types.go`, `services/api/internal/packs/catalog.go`, `services/api/internal/packs/admin.go` (column list reuse only)
- Test: `services/api/internal/packs/catalog_test.go` (replace the 7 `SetSource` usages)

**Interfaces:**
- Consumes: table `design_sheets` (Task 1).
- Produces: `PackDraft` and `Metadata` gain `SheetID string`. `Pack.Source *PackSource` is replaced by `Pack.SheetID string` and `Pack.Sheet *PackSheet` where `type PackSheet struct { ID, Provider, Filename, ChecksumSHA256 string; Width, Height int; ByteSize int64 }`. `SetSource` and `PackSource` are removed. `Create` and `UpdateCatalog` require a non-empty `SheetID` that belongs to `DesignID` (else `ErrInvalid`); `UpdateCatalog` additionally returns `ErrJobActive` when it changes the sheet while a job is queued/running (same check as old `SetSource`, `hasActiveJob`).
- `packColumns` / `packFrom` join `design_sheets s ON s.id = p.sheet_id` and select `p.sheet_id`, `s.provider`, `s.filename`, `s.checksum_sha256`, `s.width`, `s.height`, `s.byte_size`.

- [ ] **Step 1: Write failing tests** in `catalog_test.go` (helper inserts a design row and a sheet row via raw SQL):
  - `TestCreateRequiresSheetOfSameDesign`: create with no sheet → `ErrInvalid`; with an unknown sheet id → `ErrInvalid`; with a sheet of design B and `DesignID` A → `ErrInvalid`; with a matching sheet → `Pack.SheetID == sheet.ID` and `Pack.Sheet.Provider == "gemini"`.
  - `TestUpdateCatalogChangesSheet`: switches to a second sheet of the same design; with an active job inserted in `processing_jobs` → `ErrJobActive`; on a revoked pack → `ErrRevoked`.
  - Convert existing `SetSource` tests to use `SheetID` on create.
- [ ] **Step 2: Run** `go test ./internal/packs -v`. Expected: FAIL (compile errors).
- [ ] **Step 3: Implement.** Validation is one query: `SELECT 1 FROM design_sheets WHERE id = ? AND design_id = ?` inside the transaction. Keep `scanPack` null-safe (`sql.NullString/NullInt64`) because the join is LEFT.
- [ ] **Step 4: Run** `go test ./internal/packs -v`. Expected: PASS.
- [ ] **Step 5: Commit** `feat(packs): bind packs to a design sheet`.

---

### Task 4: Jobs read the source from the sheet

**Files:**
- Modify: `services/api/internal/jobs/runner.go` (`loadEnqueueSnapshot`, `sourceRef`), `services/api/internal/jobs/types.go` only if `ErrNoSource` doc changes
- Test: `services/api/internal/jobs/runner_test.go` (the harness's source setup, 1 reference)

**Interfaces:**
- Consumes: `design_sheets` and `packs.sheet_id`.
- Produces: unchanged `Enqueue(ctx, packID, sourceID, designID string) (Job, error)` where `sourceID`, if non-empty, must equal the pack's `sheet_id` (it is the sheet id = source dir id). `JobRequest.Source` still carries `ID` (= sheet id), `Filename`, `ChecksumSHA256`, `Width`, `Height`, `ByteSize`.

- [ ] **Step 1: Update the harness** to create a design, sheet and pack with `sheet_id` instead of `source_*`, then write `TestEnqueueUsesSheetSource` asserting the stored `request_json`'s `source.id` equals the sheet id and `source.checksum_sha256` equals the sheet checksum, and `TestEnqueueWithoutSheetReturnsErrNoSource`.
- [ ] **Step 2: Run** `go test ./internal/jobs -run Enqueue -v`. Expected: FAIL.
- [ ] **Step 3: Change the SELECT** in `loadEnqueueSnapshot` to `LEFT JOIN design_sheets s ON s.id = p.sheet_id` reading `p.sheet_id, s.filename, s.checksum_sha256, s.width, s.height, s.byte_size`; `sourceRef` treats a NULL sheet as `ErrNoSource`.
- [ ] **Step 4: Run** `go test ./internal/jobs ./internal/packs ./internal/studio ./internal/storage -v`. Expected: PASS.
- [ ] **Step 5: Commit** `feat(jobs): process packs from their design sheet`.

---

### Task 5: Admin HTTP: sheets and pack handlers

**Files:**
- Create: `services/api/internal/interfaces/http/admin_sheets.go`
- Modify: `services/api/internal/interfaces/http/admin_auth.go` (route table, `writePackError` source_missing text), `admin_packs.go` (create/update bodies, `adminPack`, detail), `admin_studio.go` (`writeStudioError` mapping)
- Modify: `services/api/internal/interfaces/http/docs/docs.html` (replace the pack `/source` route with design sheets)
- Test: `services/api/internal/interfaces/http/admin_test.go` (uses `adminFixture.upload`, `mutate`, `get`)

**Interfaces:**
- Consumes: Task 2 studio methods, Task 3 pack fields.
- Produces routes (all under `/api/v1/admin`): `POST /designs/{id}/sheets` (multipart `file`, `provider`, `note`; 201 new, 200 duplicate), `GET /designs/{id}/sheets` → `{"items":[Sheet]}`, `GET /designs/{id}/sheets/{sheetId}/image` (owner session, `no-store`), `DELETE /designs/{id}/sheets/{sheetId}` (204). `POST/PUT /packs` bodies gain `sheet_id`. `adminPack` JSON: `source` removed; adds `sheet_id` and `sheet` (`{id, provider, filename, width, height, byte_size, checksum_sha256, image_url}` where `image_url` is the admin image route). `adminPackDetail` adds `design` (`{id, name, rows, columns, outline_width, outline_color, character_id, compiled_prompt}`) when `design_id` is set. `POST /packs/{id}/source` is deleted. Error codes: `sheet_limit` 409, `sheet_in_use` 409, `design_has_sheets` 409, `unprocessable` 422.
- `a.upload` currently hands the callback only `(reader, filename)`; read `provider` and `note` form fields by extending the callback signature to receive a `fields map[string]string` (update its 3 existing call sites).

- [ ] **Step 1: Write failing tests:** upload a PNG with `provider=chatgpt` → 201 with `provider`, list shows it, image route returns the bytes with `Content-Type: image/png`; duplicate upload → 200 same id; bad provider → 422; delete while used by a pack → 409 `sheet_in_use`; delete design with sheets → 409; `POST /packs` without `sheet_id` → 422, with a matching `sheet_id` → 201 and `sheet.provider` present; `GET /packs/{id}` includes `design.compiled_prompt`; `POST /packs/{id}/source` → 404/405; a client API key gets 401 on every new route.
- [ ] **Step 2: Run** `go test ./internal/interfaces/http -run 'Sheet|Pack' -v`. Expected: FAIL.
- [ ] **Step 3: Implement.** The admin pack detail loads the design through `a.studio.GetDesign`. Remove `uploadPackSource`, `adminPackSource` and `writeUploadDomainError` only if no other caller remains (check with grep).
- [ ] **Step 4: Run** `go test ./internal/interfaces/http -v`. Expected: PASS (public tests may still compile against the unchanged public API).
- [ ] **Step 5: Commit** `feat(api): admin routes for design sheets, packs use sheet_id`.

---

### Task 6: Key grants service in auth

**Files:**
- Create: `services/api/internal/auth/grants.go`
- Modify: `services/api/internal/auth/keys.go` (`RotateKey`, new `AuthenticateKey`)
- Test: `services/api/internal/auth/auth_test.go`

**Interfaces:**
- Produces on `*auth.Service`:
  - `AuthenticateKey(ctx, raw string, now time.Time) (keyID string, err error)`: same checks as `ValidateKey`, which now delegates to it.
  - `KeyPacks(ctx, keyID string) ([]string, error)` (sorted pack ids; `ErrKeyNotFound` if the key does not exist).
  - `SetKeyPacks(ctx, keyID string, packIDs []string) error`: replaces the set in one transaction; dedupes; unknown pack id → `ErrUnknownPack` (new error); removed ids get an `api_key_pack_removals` row (`removed_at` = now UTC seconds, replaced if present); newly granted ids delete any removal row.
  - `KeyHasPack(ctx, keyID, packID string) (bool, error)`.
  - `KeyRemovals(ctx, keyID string, since time.Time) ([]Removal, error)` with `type Removal struct { PackID string; RemovedAt time.Time }`, inclusive `>= since`, ordered by `(removed_at, pack_id)`.
- `RotateKey` copies `api_key_packs` rows to the replacement key and its removals, inside its existing transaction.

- [ ] **Step 1: Write failing tests:**
  - `TestSetKeyPacksGrantsAndRemoves`: grant {A,B}, then set {B,C}: `KeyPacks` = {B,C}; `KeyRemovals` contains A.
  - `TestRegrantClearsRemoval`: remove A then grant A again → `KeyRemovals` empty.
  - `TestSetKeyPacksUnknownPack` → `ErrUnknownPack`, and the previous set is unchanged.
  - `TestRotateKeyCarriesGrants`: grants on the old key appear on the replacement key.
  - `TestAuthenticateKeyReturnsID` and that disabled/expired keys give `ErrKeyNotFound`.
- [ ] **Step 2: Run** `go test ./internal/auth -v`. Expected: FAIL.
- [ ] **Step 3: Implement.** Pack existence is checked with a `SELECT COUNT(*) FROM packs WHERE id IN (...)` inside the transaction.
- [ ] **Step 4: Run** `go test ./internal/auth -v`. Expected: PASS.
- [ ] **Step 5: Commit** `feat(auth): per-key pack grants with removal markers`.

---

### Task 7: Public API enforces grants

**Files:**
- Modify: `services/api/internal/interfaces/http/public.go` (`authorize`, `listPacks`, `visiblePack`, tombstones), `services/api/internal/interfaces/http/assets.go` (`serveObject`, `serveBundle`), `services/api/internal/packs/types.go` + `catalog.go` (`Filter.KeyID`)
- Test: `services/api/internal/interfaces/http/public_test.go` (uses `publicFixture`; add helper `grant(keyID string, packIDs ...string)` that calls `auth.SetKeyPacks`)

**Interfaces:**
- Produces: `type principal struct { keyID string; owner bool }` and `func (a *publicAPI) authenticate(r *http.Request) (principal, bool)` replacing `authorize` (owner cookie → `owner: true`; key → `keyID`). `packs.Filter` gains `KeyID string`; when set, `ListReady` adds `AND p.id IN (SELECT pack_id FROM api_key_packs WHERE key_id = ?)` to both the count and the page query. A helper `a.granted(ctx, p principal, packID string) bool` is used by `visiblePack`, `serveObject`, `serveBundle` **before** any status check; a non-granted pack returns 404 `not_found`.
- Tombstones for a key: `packs.Tombstones(since)` filtered to granted pack ids (via `KeyPacks`), merged with `KeyRemovals(since)` as `{pack_id, revoked_at: removed_at, reason_code: "access_removed", checksum_sha256: ""}`, sorted by `(revoked_at, pack_id)`. Owner sees the unfiltered list.

- [ ] **Step 1: Write failing tests** (adapt the existing key-based tests by granting first):
  - `TestKeyWithoutGrantsSeesNothing`: list → 200 `items: []`, `total: 0`; detail, bundle meta, object, bundle download all 404.
  - `TestGrantedPackVisibleOnAllRoutes`: grant one of two ready packs; list shows exactly it; its detail/bundle/object/bundle download return 200; the other returns 404 on all four.
  - `TestNonGrantedRevokedPackIs404NotGone`: revoked non-granted → 404; revoked granted → 410 as before.
  - `TestOwnerSessionIgnoresGrants`.
  - `TestTombstonesRespectGrants`: revoke packs A (granted) and B (not granted): key sees only A; remove a grant → the key sees `access_removed` for it; re-grant → it disappears; `since` filtering is inclusive on both sources.
  - HEAD on `/api/v1/packs/{id}` for a non-granted pack → 404.
- [ ] **Step 2: Run** `go test ./internal/interfaces/http -run 'Grant|Public|Tombstone' -v`. Expected: FAIL.
- [ ] **Step 3: Implement.** `publicAPI` already holds `auth *auth.Service`; no new dependency.
- [ ] **Step 4: Run** `go test ./internal/interfaces/http -v`. Expected: PASS. Then verify the Flutter side needs no change: `rg -n "reasonCode" clients/flutter/restick/lib` shows it is only copied into the entity, not switched on.
- [ ] **Step 5: Commit** `feat(api): scope public pack access to granted packs`.

---

### Task 8: Admin routes for key grants

**Files:**
- Modify: `services/api/internal/interfaces/http/admin_keys.go`, `admin_auth.go` (route table + `writeKeyError` mapping for `ErrUnknownPack` → 422)
- Modify: `services/api/internal/interfaces/http/docs/docs.html`
- Test: `services/api/internal/interfaces/http/admin_test.go`

**Interfaces:**
- Consumes: Task 6 methods.
- Produces: `GET /api/v1/admin/api-keys/{id}/packs` → `{"pack_ids":[...]}`; `PUT /api/v1/admin/api-keys/{id}/packs` body `{"pack_ids":[...]}` → same shape. Key list items gain `pack_count` (int). 404 for unknown key, 422 for unknown pack id.

- [ ] **Step 1: Write failing tests:** PUT then GET round-trips; PUT with an unknown pack → 422 and unchanged set; rotating a key via the existing route keeps `GET .../packs` on the new key id equal to the old set; list shows `pack_count`; routes require an owner session and Origin check (PUT without the Origin header → 403).
- [ ] **Step 2: Run** `go test ./internal/interfaces/http -run Key -v`. Expected: FAIL.
- [ ] **Step 3: Implement**; extend `auth.Service.ListKeys` (and the `Key` struct) with `PackCount` from a `LEFT JOIN`/subquery.
- [ ] **Step 4: Run** `go test ./... ` from `services/api`. Expected: PASS except `maintenance` (next task).
- [ ] **Step 5: Commit** `feat(api): manage pack grants per API key`.

---

### Task 9: Backup and restore cover the new tables

**Files:**
- Modify: `services/api/internal/maintenance/restore.go` (`requiredTables`, replace `checkPackSources` with `checkDesignSheets`), `services/api/internal/maintenance/backup.go` if it lists tables
- Test: `services/api/internal/maintenance/maintenance_test.go` (`buildFixture`, 2 `source_*` references)

**Interfaces:**
- `requiredTables` gains `design_sheets`, `api_key_packs`, `api_key_pack_removals`. `checkDesignSheets(ctx, db, sources)` verifies every `design_sheets` row has `sources/<id>/original.*` in the manifest with matching checksum and size (same logic and error wording as the old `checkPackSources`).

- [ ] **Step 1: Update `buildFixture`** to create a design, a sheet (real file under `sources/`), a pack with `sheet_id`, and a key with a grant. Add `TestRestoreRejectsMissingSheetFile` (delete the file from the archive → `ErrBackupInvalid`) and `TestRoundTripKeepsGrants` (after restore, `api_key_packs` rows exist; keys are disabled as before).
- [ ] **Step 2: Run** `go test ./internal/maintenance -v`. Expected: FAIL.
- [ ] **Step 3: Implement** the renames above.
- [ ] **Step 4: Run** `go test ./...` and `go vet ./...` in `services/api`. Expected: PASS.
- [ ] **Step 5: Commit** `feat(maintenance): back up and verify design sheets and grants`.

---

### Task 10: Web foundation, mock server and design Sheets panel

**Files:**
- Modify: `clients/web-next/lib/types.ts` (add `Sheet`, `AdminPackSheet`, `DesignSummary`; change `AdminPack.source` → `sheet`, add `sheet_id`; `PackCreate.sheet_id`; key `pack_count`), `lib/api.ts` (error texts for `sheet_limit`, `sheet_in_use`, `design_has_sheets`; drop `source_missing` wording → "Pack chưa có sheet"), `lib/i18n-messages.ts` (new keys), `lib/ai-providers.ts` (export `providerName(id: string): string` returning the name, or "Khác"/the raw id)
- Create: `clients/web-next/components/design-sheets.tsx`
- Modify: `clients/web-next/app/factory/page.tsx` (render `<DesignSheets designId>` for a saved design), `clients/web-next/qa/mock-server.mjs` (sheet routes, `sheet_id` on packs, key packs routes, remove pack `/source`)
- Modify: `clients/web-next/qa/interact.mjs`

**Interfaces:**
- Produces: `DesignSheets({ designId }: { designId: string })`: upload form (file, provider `<select>` from `AI_PROVIDERS` plus `other`, note), thumbnail grid (image from `/api/v1/admin/designs/{id}/sheets/{sheetId}/image`, provider badge, date, delete with confirm, prompt snapshot in an expandable block). Also export `SheetPicker({ designId, value, onChange })` from the same file: the selectable grid reused in Task 11.
- Mock routes mirror Task 5 and Task 8 exactly (status codes and error codes included), serving an SVG placeholder for the image route.

- [ ] **Step 1: Update the mock server and types first** so the page can be built against them; update `interact.mjs` with a scenario: open a saved design, upload a file with provider "gemini", assert the badge text "Gemini", delete it.
- [ ] **Step 2: Implement `design-sheets.tsx`** following existing components in `components/ui.tsx` (`Section`, `Field`, `Button`, `ErrorBanner`, `selectClass`, `inputClass`).
- [ ] **Step 3: Run** `npx tsc --noEmit && npm run lint` in `clients/web-next`. Expected: no errors.
- [ ] **Step 4: Run the QA flow** per `qa/README.md` (start the mock on :4500, run `node qa/interact.mjs`). Expected: the new scenario passes.
- [ ] **Step 5: Commit** `feat(web): design sheets panel with provider badges`.

---

### Task 11: New-pack form shows design info and sheet images

**Files:**
- Modify: `clients/web-next/app/packs/new/page.tsx`, `clients/web-next/app/packs/detail/page.tsx` (replace the upload section and `data.source` usages with the chosen sheet, design panel, and a sheet change control), `lib/i18n-messages.ts`, `qa/interact.mjs`
- Create: `clients/web-next/components/design-panel.tsx`

**Interfaces:**
- Consumes: `SheetPicker` (Task 10), `Design` type.
- Produces: `DesignPanel({ design }: { design: Design | DesignSummary })` rendering name, rows×columns, outline width/colour, character id/name when present, and the compiled prompt in a scrollable `<pre>` with a copy button.

- [ ] **Step 1: Extend `interact.mjs`:** new-pack flow: choose a design → panel shows its name and prompt text; grid shows the design's sheets with provider badges; submit is blocked with a visible error until a sheet is selected; a design with no sheets shows the "upload a sheet first" hint with a link to the design page; after creating, the pack detail shows the same design panel and the chosen sheet image.
- [ ] **Step 2: Implement.** Submit sends `sheet_id`; `designRequired` style validation for `sheetRequired`. On the detail page, "Process" is disabled only when `sheet` is null or the pack is revoked. Changing the sheet calls `PUT /api/v1/admin/packs/{id}` and surfaces the 409 `job_active` message.
- [ ] **Step 3: Run** `npx tsc --noEmit && npm run lint`, then the QA flow. Expected: PASS.
- [ ] **Step 4: Commit** `feat(web): pick a design sheet when creating a pack`.

---

### Task 12: Keys page pack checklist

**Files:**
- Modify: `clients/web-next/app/keys/page.tsx`, `lib/i18n-messages.ts`, `qa/mock-server.mjs` (already has routes from Task 10), `qa/interact.mjs`
- Create: `clients/web-next/components/key-pack-access.tsx`

**Interfaces:**
- Produces: `KeyPackAccess({ keyId }: { keyId: string })`: loads all packs (paged `GET /api/v1/admin/packs?page_size=100`, same loop as `fetchAllDesigns`) and `GET .../api-keys/{id}/packs`; search box; one checkbox per pack (name, slug, status pill); each toggle immediately `PUT`s the full new set and shows a saved/error state (rollback on error); "select all visible" and "clear" actions apply to the filtered list. The same component is shown in the one-time "key created" dialog so a new key can be configured at once (a new key starts with none).

- [ ] **Step 1: Extend `interact.mjs`:** create a key → open access → tick two packs → reload → still ticked; untick one → count in the list shows 1; rotate the key → the new key shows the same ticked pack.
- [ ] **Step 2: Implement** the component and mount it per key row (expandable) and in the created dialog.
- [ ] **Step 3: Run** `npx tsc --noEmit && npm run lint`, then the QA flow. Expected: PASS.
- [ ] **Step 4: Commit** `feat(web): grant packs per API key`.

---

### Task 13: End-to-end check and local rerun

**Files:** none new.

- [ ] **Step 1: Run the full Go suite** `go test ./... && go vet ./...` in `services/api`. Expected: all PASS.
- [ ] **Step 2: Run web checks** `npx tsc --noEmit && npm run lint && npm run build` in `clients/web-next`, then the interaction QA. Expected: all PASS.
- [ ] **Step 3: Real-API smoke** with `qa/real-proxy.mjs` against a fresh Go server: create design → upload sheet (provider) → create pack → process → create key → grant → `curl -H "X-Api-Key: …" /api/v1/packs` returns only the granted pack and a non-granted pack id returns 404. Record the actual output in the final report.
- [ ] **Step 4: Rebuild and start the mock on :4500** so the result can be inspected (user's standing request).
- [ ] **Step 5: Inspect `git status`/`git diff --stat`** for accidental changes; do not touch the unrelated modified files listed at session start.
