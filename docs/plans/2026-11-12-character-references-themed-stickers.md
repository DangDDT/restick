# Character References and Themed Stickers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Chuẩn bị nhân vật có nhiều ảnh tham chiếu, chọn/copy đúng ảnh cùng prompt bộ sticker, và biên dịch bộ nhiều chủ thể kèm chữ tiếng Việt cho agent bên ngoài.

**Architecture:** Giữ Go + SQLite + filesystem và Next.js static studio. Bảng liên kết thay reference đơn của character; một reference của thiết kế được kiểm tra membership và đóng băng trong snapshot. Label là dữ liệu riêng từng ô; compiler phân biệt nhận dạng nhân vật, chủ đề nhiều chủ thể và exact visible text. Web chỉ dùng native clipboard/canvas, không thêm dịch vụ hoặc dependency.

**Tech Stack:** Go 1.26, SQLite hiện có, TypeScript, Next.js 16/React 19, TanStack Query, native ClipboardItem/canvas; Python processor hiện hữu không đổi.

**Spec:** `docs/specs/2026-11-12-character-references-themed-stickers-design.md`

## Global Constraints

- Không có trạng thái duyệt/khóa nhân vật; có/không có `character_id` quyết định quy tắc nhận dạng.
- Đã chọn nhân vật thì bắt buộc chọn ảnh tham chiếu trước khi biên dịch; Save cũng tự compile nên áp cùng quy tắc.
- Character response dùng `reference_asset_ids: string[]`, luôn có array kể cả `[]`; bỏ singular character field, không alias fallback.
- Draft dùng `reference_asset_id?: string`; SnapshotCharacter dùng `reference_asset_id?: string`; Cell/SnapshotCell dùng `label?: string`.
- Giữ snapshot schema version `1`: chỉ thêm trường optional, không đổi nghĩa trường cũ; không đổi job request schema hay public Flutter/read API.
- Prompt hướng dẫn tiếp tục bằng tiếng Anh như compiler hiện tại; nội dung người dùng, nhất là label tiếng Việt, giữ nguyên, không tự dịch.
- Agent bên ngoài vẽ cả hình lẫn chữ; không thêm LLM/provider integration, font/layout engine, dependency, processor rewrite hoặc reference từng ô.
- Giữ nền trắng phẳng `#FFFFFF`, đúng rows × columns/ratio/số lượng, khoảng trống biên, không đường lưới/watermark/frame. Giới hạn hiện có: 1–8 hàng/cột, tối đa 64 ô; outline 0–16px, default 4px `#FFFFFF` không đổi.
- Không viết lại `designs.draft_json`, `snapshot_json`, `compiled_prompt` hoặc ảnh sheet/version/job cũ để đoán reference lịch sử.
- Bỏ association không xóa asset/file; ảnh vẫn private và bất biến.
- Không stash/commit thay đổi có sẵn của người dùng. Không chạy migration trên `.localdata*`, không dùng secret thật, không merge/push/deploy.
- Không thêm web test runner. Kiểm thử permanent chỉ cho hành vi/invariant; không test source text, mock echoes hay pin bố cục/câu chữ không thuộc contract.

## Review Focus

1. Backup schema 2 với ảnh nhân vật phải restore và upgrade nguyên vẹn; schema 3 thiếu bảng association phải fail trước khi thay dữ liệu — Task 1, `TestRestoreLegacyCharacterReferences` và `TestRestoreRejectsMissingCharacterReferences`.
2. Update character có asset lạ không được ghi nửa chừng; list nhiều character không treo do query nested trên một connection — Task 1, `TestCharacterReferenceUpdateIsAtomic` và `TestCharacterReferenceListsStayIndependent`.
3. Label Unicode có dấu/quote/xuống dòng, bộ trộn ô có/không chữ và label-only không được làm mất mô tả hoặc nới lỏng reference validation — Task 2, `TestThemedLabelsRoundTrip`, `TestMixedLabelsRemainExplicit`, `TestLabelOnlyCellRejected`, `TestCompileReferenceMembership`.
4. Compile A rồi chỉnh reference B, lỗi request hoặc đổi thiết kế khi request đang chạy không được tạo prompt A + ảnh B; cả mobile/desktop phải đúng — Task 4, browser scenario “frozen result and busy editor” với response thật được delay/fail có kiểm soát.
5. PNG/JPEG/WebP phải copy ra `image/png` thật; thiếu permission/session/decode không được báo thành công hoặc copy URL thay thế — Task 4, browser scenario “binary clipboard and failure states”, đọc clipboard/paste thực tế và kiểm tra dimensions/pixels.

## Workspace, ownership và execution

Trước implementation, đọc spec + plan, kiểm tra git working tree theo SDD. Baseline khảo sát nằm trên `feat/design-sheets-key-access`, có 36 tracked modifications và 5 untracked paths trước khi thêm docs này. README, character/factory pages, i18n, OpenAPI, compiler, studio/HTTP tests và Vietnam seed đều có thay đổi của người dùng. Không tạo worktree từ HEAD rồi xem đó là baseline đủ, vì sẽ bỏ các thay đổi đang dùng làm nền thiết kế.

Kế hoạch dựa trên nội dung working tree đã khảo sát. Ưu tiên người dùng commit baseline đó; nếu chọn stash/baseline khác, khảo sát lại các file liên quan và cập nhật cả spec/plan trước implementation. Khi baseline sạch và hai tài liệu được duyệt, tạo branch `feature/character-references-themed-stickers` trong checkout hiện tại; không tự tạo session worktree hay cần `/move`. Task worktrees của một wave vẫn do SDD quản lý theo skill, luôn truyền absolute path cho agent. Chỉ commit hai doc mới trước khi bắt đầu, không gom các file dirty có sẵn.

| Wave | Task | Lý do |
| --- | --- | --- |
| 1 | 1 | Contract và storage prerequisite |
| 2 | 2 và 3 | Backend compiler/seed và character web UI sửa các file khác nhau; cả hai chỉ cần contract character của Task 1 |
| 3 | 4 | Bàn ghép/clipboard tích hợp kết quả Task 2 và component/prompt workflow Task 3 |

Task 1 và 2 có per-task review vì migration/persisted format. Task 3 review ở final. Task 4 có per-task review vì request-state concurrency và cặp kết quả; final review vẫn kiểm tra seams toàn nhánh.

**Verification ownership:** agent không chạy build/lint/format/full suites trong lúc sửa. TDD có hai phase boundary: viết test rồi báo controller chạy focused RED một lần; nhận RED mới viết implementation, sau đó controller chạy focused GREEN khi task/wave đã xong. Báo cáo chỉ ghi kết quả thực sự chạy, không suy đoán. Controller không viết thay product code. Nếu harness cho worker thực thi focused checks, phải ghi rõ lệnh và hai phase trong dispatch; không nhân đôi cùng check giữa worker/controller. Không áp mặc định full-suite của skill khi trái repo rule: chỉ các lệnh hẹp dưới đây, không `go test ./...`.

Duyệt plan bao gồm các focused checks nêu dưới đây và **một lần web static export sau khi tích hợp Task 4** để kiểm chứng surface thật. Nếu export sẵn chưa phản ánh Task 3, dùng Next dev + API routing của browser cho smoke trung gian, không build export lần thứ hai. Không full suite, deployment build/Docker, browser sweep toàn bộ hay migration dữ liệu thật.

---

### Task 1: Quan hệ nhiều ảnh của nhân vật, upgrade và restore

**Review:** per-task — data migration và persisted format

**Files:**
- Create: `services/api/internal/storage/migrations/0003_character_references.sql`
- Create: `services/api/internal/storage/character_references_test.go`
- Create: `services/api/internal/studio/character_references_test.go`
- Create: `services/api/internal/maintenance/character_references_test.go`
- Create: `services/api/internal/interfaces/http/character_references_test.go`
- Modify: `services/api/internal/studio/types.go:85-105`
- Modify: `services/api/internal/studio/library.go:328-410,451-463,507-528`
- Modify: `services/api/internal/storage/db_test.go:34-50`
- Modify: `services/api/internal/maintenance/restore.go:228-279`
- Modify: `services/api/internal/maintenance/maintenance_test.go` (fixture dùng singular column)
- Modify: `services/api/internal/studio/studio_test.go` (CharacterDraft field và reference assertion hiện hữu)
- Modify: `services/api/internal/interfaces/http/admin_test.go:1004-1026`
- Modify: `clients/web-next/lib/types.ts:73-91`
- Modify: `contracts/openapi/v1.yaml:235-260`
- Modify: `clients/web-next/qa/mock-server.mjs` (character fixtures và CRUD contract array, không compiler semantics)

**Interfaces:**
- Consumes: `Service.SaveReferenceAsset`, private GET reference asset route, current character CRUD route/method signatures, embedded sequential storage migrations.
- Produces: Go `Character.ReferenceAssetIDs []string` (`json:"reference_asset_ids"`, no omitempty), `CharacterDraft.ReferenceAssetIDs []string`; TS `Character.reference_asset_ids: string[]`, `CharacterDraft.reference_asset_ids: string[]`; request may omit list, resulting empty list. `CreateCharacter/UpdateCharacter/GetCharacter/ListCharacters` signatures unchanged.
- Produces: `character_reference_assets(character_id TEXT, asset_id TEXT, position INTEGER)` and conditional restore validation at schema >=3. No singular character compatibility field remains in active code/contracts/fixtures.

- [ ] **Step 1: Write focused failing behavior tests**

Use existing real temp DB/files helpers, not an in-memory mock repository. Define these cases:

```text
TestCharacterReferencesMigrationPreservesData:
  Create schema 2 using its actual two migration SQL files.
  Existing char owns refA; another has NULL. Existing design prompt/snapshot/draft contain sentinel bytes.
  Open with current storage -> first list [refA], second [] through SQL relation; all sentinel JSON/prompt unchanged.
  Reopen -> no duplicate association, FK valid, old singular column absent.
TestCharacterReferenceUpdateIsAtomic:
  Create char [refB, refA]; update name plus [refA, missing] -> ErrInvalid.
  Get -> old name and [refB, refA], assets still readable.
  Valid update [refA, refA, " "] -> [refA]; detached refB bytes still readable.
TestCharacterReferenceListsStayIndependent:
  Two characters with distinct ordered lists and one empty; List/Get match correct IDs/order and finish under test deadline.
TestRestoreLegacyCharacterReferences:
  Valid schema-2 backup containing a real reference file -> Restore to fresh temp destination -> schema3 association and bytes preserved.
TestRestoreRejectsMissingCharacterReferences:
  Valid schema3 backup whose DB lacks the new table, with manifest hashes updated honestly -> ErrBackupInvalid; destination remains uninstalled.
TestAdminCharacterReferenceCollection:
  Authenticated POST/PUT/GET/list round-trip arrays; [] encoded not null; unknown ID ->422 and old state preserved.
  Unauthenticated/client-key GET asset still denied as before.
```

Upgrade fixtures must actually use old SQL, not write old columns against latest schema. Maintenance fixtures may use existing backup helpers and production checksum/manifest format; no production-only test hook. Update only existing char singular fixtures so packages compile; Task 2 owns new compile requirement fixtures.

- [ ] **Step 2: Verify RED at the test boundary**

Controller runs, from repo root:
`go -C services/api test -count=1 -run 'Test(CharacterReferencesMigrationPreservesData|CharacterReferenceUpdateIsAtomic|CharacterReferenceListsStayIndependent|RestoreLegacyCharacterReferences|RestoreRejectsMissingCharacterReferences|AdminCharacterReferenceCollection)$' ./internal/storage ./internal/studio ./internal/maintenance ./internal/interfaces/http`

Expected: focused assertions fail because relation/API list/upgrade behavior is absent; do not accept fixture syntax/setup errors as RED. No product edits before this evidence.

- [ ] **Step 3: Implement the relation and migrate all active character callers**

Create migration exactly as spec §6: backfill real IDs before DROP index/column, one migration transaction. Change character scan/write/load paths to arrays, normalize via existing `uniqueStrings`, validate all assets and update association plus fields atomically. Batch load associations after closing character rows; never nested query with an open row set on the single connection, no per-character N+1.

Make restore table check schema-version-aware for **only** the new table; preserve existing integrity/FK/file validation. Migrate Go/TS/OpenAPI and QA character fixtures in the same task. Use symbol references via available LSP before changing exported fields, then bounded literal search for JSON fixtures; intentionally retain singular references only in old-migration fixtures and the distinct draft/snapshot contract added by Task 2. No unrequested service refactor.

- [ ] **Step 4: Verify GREEN and affected package compatibility**

Controller runs once after implementation:
`go -C services/api test -count=1 ./internal/storage ./internal/studio ./internal/maintenance ./internal/interfaces/http`

Expected: all four affected packages PASS. No web export needed; no real DB migrated. Per-task reviewer reads diff + actual test output, checks atomicity/backfill/restore compatibility, not only Go signatures.

- [ ] **Step 5: Commit only owned paths**

Stage paths listed above explicitly and commit `feat(studio): support multiple character reference images`. Record commit, RED/GREEN commands/output, migration and restore evidence in task report. No `git add -A`, no user hunks.

### Task 2: Compiler cho reference đã chọn và chữ từng sticker

**Review:** per-task — persisted draft/snapshot format

**Files:**
- Create: `services/api/internal/studio/composition_prompt_test.go`
- Create: `services/api/internal/interfaces/http/composition_prompt_test.go`
- Modify: `services/api/internal/studio/types.go` (Draft/Cell/SnapshotCharacter/SnapshotCell)
- Modify: `services/api/internal/studio/prompt.go`
- Modify: `services/api/internal/studio/studio_test.go` (character compile fixtures, suggest/seed invariants; bỏ incidental wording assertions)
- Modify: `services/api/internal/interfaces/http/admin_test.go` (all existing character compile/save payloads)
- Modify: `services/api/internal/studio/seed/vietnam.json` (starter designs only)
- Modify: `services/api/internal/jobs/runner_test.go` (extended snapshot handoff case)
- Modify: `clients/web-next/lib/types.ts` (Draft/Cell/Snapshot optional fields)
- Modify: `contracts/openapi/v1.yaml` (Draft/Cell/Snapshot contracts và conditional validation descriptions)

**Interfaces:**
- Consumes: Task 1 `Character.ReferenceAssetIDs`, existing Compile/Suggest/SaveDesign/HTTP envelopes and seed loader.
- Produces: Go `Draft.ReferenceAssetID string`, `SnapshotCharacter.ReferenceAssetID string`, `Cell.Label string`, `SnapshotCell.Label string` with optional JSON names from Global Constraints. TS adds corresponding optional properties.
- Produces: `Compile(ctx, Draft) (CompiledPrompt,error)` validates membership; SaveDesign inherits it; `Suggest(ctx,Draft,*rand.Rand)` preserves the fields without requiring a reference yet. No new endpoint or changed response envelope.

- [ ] **Step 1: Write failing consumer-visible scenarios**

```text
TestCompileReferenceMembership:
  charA=[refA,refB], charB=[refC]; valid draft charA/refB -> snapshot reference refB.
  table: charA/no ref, charA/refC, charA/unknown, no char/refA -> ErrInvalid.
  no char/no ref -> valid. Repeat validation through SaveDesign and authenticated compile/create/update endpoints ->422 for invalid combinations.
TestAdminCompositionReferencesAndLabels:
  Authenticated compile/create/update exercise the same invalid-reference table ->422;
  valid three-food design returns/persists exact per-cell Unicode labels and selected image where applicable.
TestFrozenReferenceSurvivesCharacterEdit:
  Save charA/refA, change charA association to [refB].
  Load saved -> still refA and original prompt; refA bytes readable. Recompile old draft ->ErrInvalid until explicit refB.
TestThemedLabelsRoundTrip:
  1x3 no-character design, free_text art descriptions and labels exactly:
  "Bún bò Huế", "Phở Hà Nội", "Bánh xèo miền Tây".
  Compile/save/get/suggest retain names, accents, per-cell association and no automatic character/reference.
  Generated instructions allow different subjects and make labels distinct from art instructions; no global ban contradicts labels.
TestMixedLabelsRemainExplicit:
  First cell label includes quote/newline and Vietnamese accents; second empty.
  Snapshot retains trimmed-edge label and interior content, rendering quotes/escapes the payload.
  First cell has exact-visible-text instruction; second explicitly no text. No extra caption/number/text from description is requested.
TestLabelOnlyCellRejected:
  Cell{Label:"Phở Hà Nội"} with no art content ->ErrInvalid.
TestSuggestPreservesReferenceAndLabels:
  Valid char/ref plus labels and pins survives suggestion; char with no chosen ref still may suggest.
TestSeedDesignsKeepExplicitSubjects:
  All ten converted starter themes retain the matching seed character description and identity notes;
  names/grid/cell content stay unchanged, no fake reference/new labels, and restart does not overwrite saved edits.
TestEnqueueKeepsExtendedDesignSnapshot:
  Existing real jobs harness enqueues design with label and frozen ref, request snapshot still schema1, row-major, fields intact.
```

Keep testing prompt semantics, not whole-string snapshots or source files. Delete existing formatting/order/wording assertions that only pin numbered sentence layout; retain meaningful grid/white-background/outline contract tests without repinning incidental wording. Existing identity-conflict tests need **valid chosen images** so missing-reference errors cannot mask the identity invariant.

- [ ] **Step 2: Verify focused RED**

Run once after tests are written:
`go -C services/api test -count=1 -run 'Test(CompileReferenceMembership|FrozenReferenceSurvivesCharacterEdit|ThemedLabelsRoundTrip|MixedLabelsRemainExplicit|LabelOnlyCellRejected|SuggestPreservesReferenceAndLabels|SeedDesignsKeepExplicitSubjects|AdminCompositionReferencesAndLabels|EnqueueKeepsExtendedDesignSnapshot)$' ./internal/studio ./internal/interfaces/http ./internal/jobs`

Expected: new-field behavior/validation/Unicode persistence fails for missing functionality. If needed introduce type declarations with tests as contract-only RED preparation; do not implement compiler behavior before RED.

- [ ] **Step 3: Implement conditional compilation and preserved data**

Extend normalize/validation/snapshot construction to carry selected reference and label. Validate reference against selected character for Compile and Save; return `ErrInvalid` for membership mismatch/orphan selected ref. Do not change existing archival policy or require ref in Suggest. Render character/no-character openings, exact label versus drawing instructions, per-cell no-text, shared typography, connected colored ribbon supporting diacritics, and palette precedence per spec §7. Preserve technical sheet instructions and never embed private URLs.

For all ten starter designs remove character_id and append the matching existing seed profile description + identity notes to their existing free_text. Keep design names/count/grid/cell content/fragments unchanged; do not rewrite them as different object/food packs or add labels. Keep seed markers/profile records and restart semantics. No runtime seed bypass or fake references; installed saved designs remain untouched. Update all affected compile fixtures and Go/TS/OpenAPI fields.

- [ ] **Step 4: Verify GREEN at wave completion**

Controller runs:
`go -C services/api test -count=1 ./internal/studio ./internal/interfaces/http ./internal/jobs`

Expected: PASS including seed startup/restart, identity-conflict validation, optional schema1 fields and jobs handoff. Task 2 gets a fresh per-task reviewer before dependent Task 4. It shares no edited files with Task 3.

- [ ] **Step 5: Commit owned compiler/contract paths**

Commit `feat(studio): compile selected references and explicit sticker labels`. Report source seed changes separately from installed-data behavior and include exact RED/GREEN evidence.

### Task 3: Quy trình tạo nhân vật và gallery ảnh tham chiếu

**Files:**
- Create: `clients/web-next/lib/character-prompt.ts`
- Create: `clients/web-next/components/reference-preview.tsx`
- Modify: `clients/web-next/app/characters/page.tsx`
- Modify: `clients/web-next/lib/i18n-messages.ts` (character prompt/gallery strings only)

**Interfaces:**
- Consumes: Task 1 Character/CharacterDraft `reference_asset_ids`, upload endpoint returning `ReferenceAsset`, private asset GET, existing `CopyButton`, `ArtFrame`, identity-fragment import and query keys.
- Produces: `buildCharacterPrompt(character: Pick<CharacterDraft,"name"|"description"|"identity_notes">): string` in `lib/character-prompt.ts` (pure deterministic web helper, no API or persisted field).
- Produces: `ReferencePreview({assetId,className,alt}: {assetId:string;className?:string;alt?:string})` in `components/reference-preview.tsx`, rendering private same-origin asset in the existing frame. Replace old local duplicate, no URL fallback layer.
- Does not edit `lib/types.ts`, OpenAPI, backend, QA mock compiler or factory; this keeps Wave 2 file-disjoint.

- [ ] **Step 1: Capture the missing workflow before edits**

Use the browser companion only as verification, not a new mockup process: open existing Characters page against disposable API/Task 1 fixture. Observe that two reference uploads replace one image and that there is no generation prompt to copy. This is UI RED; do not add source-inspection/unit-runner tests. Record observable behavior in task report.

- [ ] **Step 2: Implement the source-based prompt and multi-image form**

Generate readonly prompt from live form source fields; request one full-body neutral character study on flat white with identity details, no text/grid/collage/sticker-sheet instructions. Existing identity-fragment import keeps changing notes. Reuse CopyButton without changing its behavior; do not add provider-link UI or another stored prompt.

Initialize gallery from reference_asset_ids. Upload selected files sequentially through existing endpoint, append successful assets in stable order, retain successes on a later failure and show which file failed; use existing upload constraints. Remove association locally, save full list, cancel leaves server character relations unchanged; block save until uploads finish. Show first thumbnail plus count in character list. Expose shared ReferencePreview and add Vietnamese messages through existing catalog/fallback convention. Use existing UI components/responsive style rather than restyle the whole page.

- [ ] **Step 3: Verify actual character surface once after edits**

Controller uses real browser at 390px and 1440px against Next dev + disposable API. Route `/api/*` to the real disposable backend with correct Origin/session using browser's native request routing; route UI to Next dev. If native routing is unavailable, use a temporary local forwarding server, not a committed proxy abstraction. No export build in this task.

Exercise: copy generated prompt before uploading; edit identity notes and copy again; upload two images, remove one/cancel/reopen; save/reload two-image list; simulated second-upload failure preserves first and existing images. Verify focus, labels, no horizontal overflow, and that images remain private. Clipboard text assertions are from actual clipboard, not button labels alone. No ability to use external agent is required or claimed.

- [ ] **Step 4: Commit only character surface paths**

Commit `feat(web): prepare character prompts and reference galleries`. Report exact browser scenarios and visual limits. No independent review dispatch from the implementer; final reviewer owns this ordinary UI task.

### Task 4: Bàn ghép, palette trực quan và sao chép ảnh đúng snapshot

**Review:** per-task — request-state concurrency

**Files:**
- Create: `clients/web-next/lib/copy-reference-image.ts`
- Create: `clients/web-next/components/compiled-reference.tsx`
- Create: `clients/web-next/components/palette-picker.tsx`
- Modify: `clients/web-next/app/factory/page.tsx`
- Modify: `clients/web-next/lib/i18n-messages.ts` (composition/copy/palette/label strings)
- Modify: `clients/web-next/qa/mock-server.mjs` (compile/save/suggest/reference validation and snapshot fixtures)
- Modify: `clients/web-next/qa/README.md` (targeted manual scenarios, no all-route sweep required)
- Modify: `README.md` (character-before-composition workflow, no-character labeled packs, migration/API cutover and browser/AI lettering limitations)

**Interfaces:**
- Consumes: Task 2 optional Draft/Cell/Snapshot fields and compile/save envelope; Task 3 ReferencePreview and existing form/UI patterns.
- Produces: `copyReferenceImage(assetId:string): Promise<void>`; it starts native clipboard write inside the caller's click gesture, resolves only on successful PNG binary copy, rejects on all failures, no textual fallback.
- Produces: `CompiledReference({character}: {character: Snapshot["character"]})`, showing nothing if no frozen reference; otherwise preview/copy/manual private-image link and error/success status. Reuse in desktop and mobile prompt surfaces.
- Produces: `PalettePicker({palettes,value,onChange,disabled,characterSelected}: {palettes: readonly Fragment[];value:string;onChange:(id:string)=>void;disabled?:boolean;characterSelected:boolean})`; retain selected conflicting identity palette as visible/disabled according to existing filter rule.
- Factory result state becomes `CompiledDesign | null`, not independent text/image state. BuilderState adds `referenceAssetId:string`; CellState adds `label:string` normalized from optional wire fields.

- [ ] **Step 1: Capture missing behavior and prepare focused browser assertions**

Against real API after Task 2, record missing reference picker, palette swatches-before-selection, explicit per-cell labels, compiled-image preview and binary-copy action. Prepare throwaway browser scenarios below; assertions fail on current UI. No permanent UI test runner or source-text assertions.

- [ ] **Step 2: Implement composition state and surface**

Carry fields through empty/resize/stateFromDesign/draftFromState, load/clear/save/compile and all cells. Reference change is explicit; changing character clears reference. List active characters as before; fetch selected saved archived character by existing GET when absent from list. Show current character's image choices and no-images hint. For an archived image-less profile, explain that the user can retain/copy the old result or explicitly choose an active prepared profile/no-character composition before recompiling; do not link to an inaccessible archived editor or add unarchive. Compile/Save have local missing/invalid-reference feedback as well as backend validation; Suggest keeps its less restrictive behavior.

Use one busy guard for compile/save/suggest/delete and lock editor/new/load transitions during requests; no generic request coordinator. Successful compile/save/load replaces one result object from the same response; editing or failed request leaves old result alone, new/delete clears it. Render CompiledReference under prompt in desktop panel and mobile dialog; all existing text copy/provider links use result text, never a separately stale variable. Do not change sheet-upload prompt snapshots.

Add label field per cell and clarify drawing-description wording. Implement PalettePicker radio/cards with four swatches + role/hex, no-palette entry, keyboard focus and current identity conflict behavior. Do not force a new country/food mode or auto-select reference.

Implement image helper using private same-origin fetch and PNG ClipboardItem promise initialized synchronously in click; native decode/canvas only for non-PNG. Release object URLs/temporary image resources; check HTTP/decode/canvas/clipboard failures, never success on fallback URL. UI error and manual open link meet spec; keep image access private. Update QA fixtures to carry label/reference and meaningful validation, but use real API for compiler proof.

- [ ] **Step 3: Execute narrow integration verification once**

Controller runs in `clients/web-next`: `npx tsc --noEmit`; scoped lint via `npx eslint app/characters/page.tsx app/factory/page.tsx components/reference-preview.tsx components/compiled-reference.tsx components/palette-picker.tsx lib/character-prompt.ts lib/copy-reference-image.ts lib/types.ts lib/i18n-messages.ts`; then **one approved** `npm run build` to produce the actual static surface. Expected: exit 0. This is the only planned export build, not a Docker/deployment build.

Launch actual Go server with fresh temporary `RESTICK_DATA_DIR`, temporary bootstrap password file, unique loopback `RESTICK_HTTP_ADDR`, `RESTICK_PUBLIC_BASE_URL=https://localhost`, the newly built `RESTICK_WEB_DIR` and current contract path. Use existing `qa/real-proxy.mjs` with explicit free loopback ports for Chromium/localhost cookie and Origin support; do not weaken production cookie/security code. Start/stop named services through tools, close browser tab and remove only owned temp data after proof.

Browser scenarios at 390px/1440px:

```text
Character composition:
  Create character [PNG_A, JPEG_B, WebP_C]; selection initially empty even with one available image.
  Compile/Save without reference -> clear required feedback; choose A -> success.
  Under prompt A is visible; switch editor selection to B without compiling -> preview/copy remain A.
  Compile B -> pair becomes B; save/load -> B retained. Detach B in character editor -> old compiled image B remains readable; new compile rejects stale membership.
  During delayed request, editor/new/load actions cannot switch context; failure preserves old result.
  Open saved archived character via GET -> refs shown without dropping character. Load legacy no-ref snapshot -> old prompt visible, new compile requires explicit choice.
  New/delete clears prompt and image. Mobile dialog and desktop agree.
Themed composition:
  No character, 1x3 free_text descriptions + three exact food labels from spec -> compile/save/reload/suggest retain all values.
  Mixed empty label and quoted/multiline Vietnamese label remain distinct; label-only cell invalid.
  Real returned prompt has per-cell drawing/exact text guidance and no same-subject/global-text contradiction.
Palette:
  Before choosing see all four swatches/name; keyboard select/clear; identity palette conflict stays visible/disabled.
Binary clipboard:
  A/B/C each copy -> clipboard image/png; decode clipboard image and compare dimensions/pixels to displayed source (JPEG tolerance after decode), not just MIME/nonempty.
  Use actual permission denial/unsupported capability simulation, private asset HTTP401, and invalid image response -> visible error, no success, manual link remains, no URL copied.
```

Do not click any real AI submission or claim a model generated correctly spelled text. As a throwaway extraction smoke, generate a 1x3 white sheet containing three connected dish-like shapes with colored caption ribbons and ink/diacritic-like marks. Invoke existing machine job in temp paths with Python from existing environment; decode three WebPs and assert ribbon/mark colors survive inside each correct output, not only output count. Do not modify processor code, add fonts/dependencies, or keep the synthetic script. If that existing connected-shape contract fails, report evidence to controller; do not silently weaken extraction validation.

- [ ] **Step 4: Review and commit integrated workflow**

Per-task reviewer checks result pairing, busy fence, archived/legacy paths, clipboard success/error semantics and read-private contract using evidence above. Commit `feat(web): compose labeled stickers with frozen reference previews` with owned product/QA/doc paths. Final whole-branch reviewer then covers Task 3 and seams across all tasks; no duplicate review subagents from implementers.

## Final integration and approval boundary

Self-review the plan/spec mapping before approval: spec §§5.1→Task3, §§5.2–5.4→Task4, §6→Tasks1/2, §7→Task2, §8→owning task checks; all five Review Focus lines have explicit scenarios. Schema/field/signature names above are the single shared contracts. If the approved baseline changes a referenced path/signature, rule on the smallest adjustment in the SDD ledger rather than overwrite user work.

Final reviewer receives full branch package, spec/plan and every ledger ruling/parked finding. Follow SDD bounded fix/re-review loops. Report only exercised evidence, commit range, residual risks, user changes not touched and all rulings. Do not run extra repo-wide suites/audits after scoped proof.

**Approval request:** duyệt spec + plan cùng nhau, SDD theo ba waves trên, branch mới trong checkout hiện tại sau khi xử lý baseline dirty, và các focused checks bao gồm một web export build. Chưa sửa product code trước câu trả lời này; nếu người dùng đổi design, cập nhật cả spec và plan trước khi hỏi duyệt lại.
