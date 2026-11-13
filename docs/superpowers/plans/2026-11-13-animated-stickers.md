# Animated Stickers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let an operator turn selected static stickers of a pack into animated WebP variants from externally generated video clips, with a per-sticker toggle and one batch "publish" that creates the next immutable pack version.

**Architecture:** A new `clip_job` in the renamed `media_factory` Python package turns a clip into an animated WebP. Go keeps a draft table `sticker_animations` with its own queue (sharing the single-child lock), and a Go-only `publish` job in `processing_jobs` copies static files plus enabled animations into version v+1 through the existing `StageBundle`/`CommitFiles`/`CommitProcessed`. Everything is additive; the static path keeps its behavior.

**Tech Stack:** Python 3.12 (Pillow 13, OpenCV headless, numpy, unittest, ruff), Go 1.26 (SQLite, `golang.org/x/image/webp`), Next.js 16 static export (read `clients/web-next/node_modules/next/dist/docs/` before coding), Flutter 3.47.

**Spec:** `docs/superpowers/specs/2026-11-13-animated-stickers-design.md`

## Global Constraints

- Animated WebP 512x512, 12 fps, `quality=75`, `method=4`, `alpha_quality=100`, `loop=0`; clip max 4 s (48 frames), max about 20 MB, max 1920 px per side; ceiling 500 KB with ladder q75, q60, fps 10, fps 8, then `output_too_large`.
- Static stickers and thumbs stay lossless `method=6`; bundle ZIP stays `zip.Store`.
- Cut every frame with the same union bounding box; never per-frame boxes.
- Go must call only `webp.DecodeConfig` on animated files, never `webp.Decode`.
- Static-only packs keep their exact aggregate checksum; animated bytes are appended per sticker only when present.
- `processing_jobs` constraints stay (`version > 0`, `UNIQUE (pack_id, version)`, one active job per pack, `schema_version = 1`); clips never use it.
- Every `media_factory` function has a docstring with Algorithm, Techniques, Keywords, References (Vietnamese prose, English terms; only links verified HTTP 200).
- Python tests are `unittest` (`python -m unittest discover -s tests -v` from `services/media_factory`), not pytest. Python via `/Users/doduongtamdang/Organized/Development/Projects/eykh/restick/.venv/bin/python`.
- Do not hand-edit generated files. Do not touch historical `docs/plans/*`. Commit messages end with `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>`.

## Review Focus

1. A middle frame with no foreground (blank or white) must fail the clip with `no_foreground`, never silently drop a frame. (Task 4)
2. Portrait or non-16:9 clips and variable-frame-rate phone recordings must sample by timestamp and keep the shared box correct. (Tasks 3, 4)
3. A clip uploaded for sticker i, then the pack republished with a different static sticker at i, must be flagged stale and excluded from publish. (Tasks 5, 8)
4. Toggling every animation off, or publishing with no change against the current version, returns 409 `no_changes` and creates no version. (Task 8)
5. Upload, delete or toggle during a running publish or sheet job, and publish while a clip is `queued`/`running`, return 409 (`animation_busy` / `job_active`). (Tasks 5, 8, 9)
6. Old `pack.json` without `animated` and a manifest whose animated file is missing or not 512x512 must behave: old parses fine, bad file rejects the stage. (Tasks 7, 11)

---

### Task 1: Rename `img_tool` to `media_factory`

**Files:**
- Move: `services/img_tool` -> `services/media_factory`; `src/img_tool` -> `src/media_factory`
- Modify: `services/media_factory/pyproject.toml` (name `media-factory`, script `media_factory`), every import of `img_tool` in `src/` and `tests/`, `Dockerfile:39-41`, `.github/workflows/ci-backend.yml` (paths `services/media_factory`, commands `media_factory --help`, `media_factory.interfaces.machine_job`), `README.md`, `services/media_factory/README.md`, `services/api/internal/jobs/processor.go` (`machineJobModule`), env var references to `IMG_TOOL_ENGINE` (keep the name; note in README)
- Do not modify: `docs/plans/**`, `docs/research/**`, `docs/specs/**` (historical)

**Interfaces:**
- Produces: importable package `media_factory` with the same public functions as `img_tool` had; module path `media_factory.interfaces.machine_job`.

- [ ] **Step 1: Capture the baseline.** Run from repo root: `.venv/bin/python -m unittest discover -s services/img_tool/tests -v` and `cd services/api && go test ./internal/jobs/... ./internal/integration/...`. Expected: PASS (record counts).
- [ ] **Step 2: Rename.** `git mv` the directory and package; rewrite imports and strings with `rg -l "img_tool|img-tool" --glob '!docs/**' --glob '!.venv/**' --glob '!node_modules/**'`; delete the generated `*.egg-info` and reinstall with `.venv/bin/pip install -e services/media_factory`.
- [ ] **Step 3: Verify.** Re-run both commands from Step 1 plus `ruff check services/media_factory` and `.venv/bin/python -m media_factory.interfaces.machine_job --help`. Expected: same pass counts; `rg "img_tool" --glob '!docs/**' --glob '!.venv/**'` returns nothing.
- [ ] **Step 4: Commit** `refactor: rename img_tool to media_factory`.

---

### Task 2: Docstrings for existing `media_factory` functions and the checker

**Files:**
- Create: `services/media_factory/scripts/check_docstrings.py`
- Test: `services/media_factory/tests/test_check_docstrings.py`
- Modify: every function in `infrastructure/{cutout,layout,outline,webp_export}.py`, `application/{process_job,process_sheet}.py`, `domain/{pack,sticker}.py`, `interfaces/{cli,machine_job}.py`

**Interfaces:**
- Produces: `check_functions(source: str) -> list[str]` returns `"<name>: missing <Section>"` for each public or private function lacking `Algorithm:`, `Techniques:`, `Keywords:`, `References:`; `check_links(urls: Iterable[str]) -> dict[str, int]` returns HTTP status per URL; CLI `python scripts/check_docstrings.py src/media_factory` exits 1 on any missing section or non-200 link. Later tasks call this on their new files.

- [ ] **Step 1: Write failing tests** `test_flags_function_without_sections` (a function with a one-line docstring yields four "missing" entries) and `test_accepts_complete_docstring` (returns `[]`).
- [ ] **Step 2: Run** `python -m unittest tests.test_check_docstrings -v`. Expected: FAIL (module missing).
- [ ] **Step 3: Implement** `check_functions` with `ast` (walk `FunctionDef`/`AsyncFunctionDef`, `ast.get_docstring`) and `check_links` with `urllib.request` HEAD falling back to GET, 15 s timeout, browser User-Agent.
- [ ] **Step 4: Document each existing function** (spec "Quy ước docstring"). Content the implementer should use: cutout (exterior flood fill on near-white, connected components, ink-halo fallback, rembg engine switch), layout (aspect-preserving fit, thumbnail resize, outlier box filter, row-major box assignment), outline (morphological dilation with elliptical structuring element, alpha "over" compositing), webp_export (lossless VP8L, `method` effort), process_job (protocol with atomic result write). Links: only ones that return 200 from `check_links`; known good: `https://en.wikipedia.org/wiki/Dilation_(morphology)`, `https://en.wikipedia.org/wiki/Alpha_compositing`, `https://pillow.readthedocs.io/en/stable/handbook/image-file-formats.html`, `https://developers.google.com/speed/webp/docs/riff_container`. OpenCV docs returned 403 to curl: leave out unless verified in a browser.
- [ ] **Step 5: Verify.** `python scripts/check_docstrings.py src/media_factory` exits 0; full unittest suite and ruff pass.
- [ ] **Step 6: Commit** `docs(media_factory): document algorithms and references per function`.

---

### Task 3: Frame reader and animated WebP encoder

**Files:**
- Create: `services/media_factory/src/media_factory/infrastructure/video.py`, `services/media_factory/tests/test_video.py`
- Modify: `infrastructure/webp_export.py` (add encoder), `tests/` for the encoder in `test_video.py`

**Interfaces:**
- Consumes: `ProcessingError` from `media_factory.application.process_job` (codes are strings).
- Produces:
  - `read_frames(path: Path, fps: int, max_frames: int, max_side: int = 1920, max_seconds: float = 4.0) -> list[Image.Image]` (RGBA), samples by timestamp (`cv2.CAP_PROP_POS_MSEC`) so variable-frame-rate clips work; raises `ProcessingError("clip_too_long", ...)` above `max_seconds`, `("decode_failed", ...)` when unreadable or zero frames, `("clip_too_large", ...)` when a side exceeds `max_side`.
  - `encode_animated_webp(frames: list[Image.Image], fps: int, quality: int) -> bytes` using `save_all=True, method=4, alpha_quality=100, lossless=False, loop=0, duration=round(1000 / fps)`.

- [ ] **Step 1: Write failing tests** using a helper that writes a synthetic clip with `cv2.VideoWriter` (`mp4v`, 24 fps, 6 frames of a moving dark square on white, in a temp dir; if `mp4v` cannot be opened the test calls `self.skipTest`). Names and assertions: `test_reads_expected_frame_count_at_12_fps` (1 s clip -> 12 frames), `test_rejects_clip_over_four_seconds` (code `clip_too_long`), `test_rejects_unreadable_file` (`decode_failed`), `test_samples_by_timestamp_for_variable_rate` (write frames with irregular `POS_MSEC` is not possible in `VideoWriter`; instead monkeypatch the capture in the test to return uneven timestamps and assert frame count still equals `round(duration*fps)`), `test_animated_webp_has_all_frames_and_alpha` (`Image.open(BytesIO(data)).n_frames == len(frames)`, first-frame corner alpha 0).
- [ ] **Step 2: Run** `python -m unittest tests.test_video -v`. Expected: FAIL.
- [ ] **Step 3: Implement** both functions with docstrings (timestamp sampling, nearest-frame selection, animated WebP, `alpha_quality`, libwebp `method`); keep them in their existing modules' style.
- [ ] **Step 4: Verify** `python -m unittest discover -s tests -v`, `ruff check`, `python scripts/check_docstrings.py src/media_factory`.
- [ ] **Step 5: Commit** `feat(media_factory): read video frames and encode animated webp`.

---

### Task 4: `process_clip` and the `clip_job` command

**Files:**
- Create: `application/process_clip.py`, `interfaces/clip_job.py`, `tests/test_clip_job.py` (all under `services/media_factory/`)

**Interfaces:**
- Consumes: Task 3 `read_frames`, `encode_animated_webp`; existing `execute_cutout`, `compose_sticker`, `ProcessingError`, `COLOR_PATTERN`, `MIN_FOREGROUND_RATIO`, `MAX_FOREGROUND_RATIO` (from `process_job`).
- Produces:
  - `process_clip(request_path: Path, result_path: Path) -> None`, protocol `restick.clip-request.v1` (`job_id`, `pack_id`, `sticker_index` int>=0, `clip_path`, `output_dir`, `fps` 6-24 default 12, `quality` 30-95 default 75, `outline_width` 0-16, `outline_color` `#RRGGBB`) and result `restick.clip-result.v1` (`animation: {filename, frames, fps, duration_ms, byte_size, quality, width, height}`, `diagnostics: {foreground_ratio_min, mask_iou_min}`) written atomically, or `error: {code, message}`.
  - `shared_box(alphas: list[np.ndarray]) -> tuple[int, int, int, int]` union of `alpha > 32` boxes.
  - `fit_under_ceiling(frames: list[Image.Image], fps: int, quality: int, ceiling: int = 512_000) -> tuple[bytes, int, int]` returns `(data, fps_used, quality_used)` following the ladder q75, q60, fps 10, fps 8 (subsampling frames), raising `ProcessingError("output_too_large", ...)`.
  - CLI `python -m media_factory.interfaces.clip_job --request R --result S` (exit 1 on any failure, stderr `code: message`).
  - Output file `animated/NNN.webp` where `NNN = f"{sticker_index:03d}"` inside `output_dir`.

- [ ] **Step 1: Write failing tests** with synthetic clips (skip if `VideoWriter` unavailable): `test_clip_job_writes_animated_webp_and_result` (512x512, `n_frames > 1`, result schema and echoed identity), `test_shared_box_is_union` (two masks at different positions give the union box), `test_blank_middle_frame_fails_no_foreground` (Review Focus 1), `test_portrait_clip_is_composed_on_512_canvas` (Review Focus 2, 384x848 clip), `test_ladder_reduces_to_fit_ceiling` (call `fit_under_ceiling` with `ceiling` small enough to force q60 and assert `quality_used == 60`), `test_output_too_large_when_ladder_exhausted`, `test_relative_clip_path_rejected` (`invalid_path`), `test_static_machine_job_unchanged` (run the existing `process_job` test fixture path again).
- [ ] **Step 2: Run** `python -m unittest tests.test_clip_job -v`. Expected: FAIL.
- [ ] **Step 3: Implement** `process_clip`: validate request (reuse `_text`/`_integer`/`_absolute_path` patterns from `process_job.py`), read frames, cut every frame with `execute_cutout`, check per-frame foreground ratio, compute `shared_box`, crop all frames by it, `compose_sticker`, `fit_under_ceiling`, write `animated/NNN.webp` and result. `mask_iou_min` is the min IoU of consecutive cropped masks.
- [ ] **Step 4: Verify** unittest suite, ruff, `check_docstrings.py`, and run the real spike clip: `python -m media_factory.interfaces.clip_job` with a request pointing at `/Users/doduongtamdang/Downloads/Lotus_mascot_waving_animation_3f81f3b1.mp4` limited to its first 4 s; expected result `byte_size <= 512000`, `mask_iou_min >= 0.95`.
- [ ] **Step 5: Commit** `feat(media_factory): clip job producing animated webp`.

---

### Task 5: Migration and `animations` service (drafts)

**Files:**
- Create: `services/api/internal/storage/migrations/0004_animated_stickers.sql`, `services/api/internal/animations/{types.go,service.go,service_test.go}`
- Modify: `services/api/internal/assets/store.go` (clip storage), `services/api/internal/storage/db_test.go` (schema version expectation)

**Interfaces:**
- Produces (package `animations`):
  - `type Status string` with `StatusQueued, StatusRunning, StatusReady, StatusFailed, StatusInterrupted`.
  - `type Animation struct { ID, PackID string; StickerIndex int; StaticChecksum, MotionText string; ClipByteSize int64; Status Status; UseAnimated bool; AnimatedPath, AnimatedChecksum string; Frames, FPS, DurationMS, Quality, Width, Height int; AnimatedByteSize int64; ErrorCode, ErrorMessage string; Stale bool }`.
  - `func NewService(db *sql.DB, files *assets.Store) *Service`.
  - `func (s *Service) Upload(ctx context.Context, packID string, index int, clip io.Reader, filename, motionText string) (Animation, error)` (replaces an existing row, removes old files after commit; validates pack exists, not revoked, `index < sticker_count` of the current version, no running sheet/publish job -> `ErrJobActive`).
  - `func (s *Service) List(ctx context.Context, packID string) ([]Animation, error)` (`Stale` computed by comparing `static_checksum_sha256` with the current version's sticker).
  - `func (s *Service) SetUse(ctx context.Context, packID string, index int, use bool) (Animation, error)` (`ErrNotReady` unless `ready`).
  - `func (s *Service) Delete(ctx context.Context, packID string, index int) error`.
  - `func (s *Service) Busy(ctx context.Context, packID string) (bool, error)` true when any row is `queued`/`running`.
  - sentinel errors `ErrNotFound, ErrInvalid, ErrJobActive, ErrNotReady`.
- Produces (assets): `func (s *Store) SaveClip(ctx context.Context, r io.Reader, filename string) (Clip, error)` (limit 20 MiB, extension and container sniff for mp4/mov/webm, stored under `clips/<uuid>.<ext>`), `func (s *Store) ClipPath(id string) (string, error)`, `func (s *Store) RemoveClip(id string) error`, plus an `animations/` data-root directory for outputs; `NewStore` creates both.

- [ ] **Step 1: Write failing tests** in `service_test.go` following the harness style of `internal/studio/studio_test.go`: `TestUploadCreatesQueuedRow`, `TestReuploadReplacesRowAndRemovesOldClip`, `TestUploadRejectsIndexOutOfRange`, `TestUploadRejectedWhileSheetJobActive` (`ErrJobActive`), `TestSetUseRequiresReady`, `TestListMarksStaleWhenStaticChecksumChanged` (Review Focus 3), `TestDeleteRemovesFiles`, `TestMigration0004AppliesOnFreshDatabase` in `db_test.go`.
- [ ] **Step 2: Run** `cd services/api && go test ./internal/animations/... ./internal/storage/...`. Expected: FAIL.
- [ ] **Step 3: Write the migration** per spec "Dữ liệu" (table `sticker_animations` with `UNIQUE (pack_id, sticker_index)`, `CHECK` on `status`; `ALTER TABLE stickers ADD COLUMN animated_path TEXT NOT NULL DEFAULT ''`, `animated_checksum_sha256 TEXT NOT NULL DEFAULT ''`, `animated_byte_size INTEGER NOT NULL DEFAULT 0`, `animated_frames INTEGER NOT NULL DEFAULT 0`, `animated_duration_ms INTEGER NOT NULL DEFAULT 0`); implement the service and store methods.
- [ ] **Step 4: Verify** `go vet ./... && go test -race ./internal/animations/... ./internal/assets/... ./internal/storage/...`. Expected: PASS.
- [ ] **Step 5: Commit** `feat(api): animation drafts table and service`.

---

### Task 6: Clip worker (queue loop, Python processor, recovery)

**Files:**
- Create: `services/api/internal/animations/{worker.go,processor.go,worker_test.go}`
- Modify: `services/api/internal/jobs/runner.go` (export the child lock), `services/api/cmd/restick/commands.go` (start the worker next to `Runner.Run`)

**Interfaces:**
- Consumes: Task 5 `Service`; `jobs.Runner.ChildLock() *sync.Mutex` (added here, returns the existing `childMu`).
- Produces:
  - `type ClipRequest struct { AnimationID, PackID string; StickerIndex int; ClipPath, OutputDir string; FPS, Quality, OutlineWidth int; OutlineColor string }`, `type ClipResult struct { Filename string; Frames, FPS, DurationMS, ByteSize, Quality, Width, Height int; Diagnostics string }`.
  - `type ClipProcessor interface { ProcessClip(ctx context.Context, r ClipRequest) (ClipResult, error) }`; `type PythonClipProcessor struct{}` runs `python -m media_factory.interfaces.clip_job` with fixed argv like `PythonProcessor`.
  - `func (s *Service) RunWorker(ctx context.Context, p ClipProcessor, childLock *sync.Mutex) error` claims the oldest `queued` row, takes `childLock` around the child, writes `ready` with result (checksum, size computed by Go) or `failed` with the machine error code; `Recover` marks stale `running` rows `interrupted` then re-queues them on startup.

- [ ] **Step 1: Write failing tests** with a fake `ClipProcessor`: `TestWorkerMarksReadyAndStoresResult`, `TestWorkerMarksFailedWithCode`, `TestWorkerNeverRunsConcurrentlyWithSheetJob` (holds `childLock`, asserts the fake is not called until released), `TestStartupRequeuesRunningRows`.
- [ ] **Step 2: Run** `go test ./internal/animations/...`. Expected: FAIL.
- [ ] **Step 3: Implement** worker, processor (output dir under `work/` per animation, copy result file to `animations/<id>.webp` after verifying `DecodeConfig` 512x512), recovery, and wire into `commands.go`.
- [ ] **Step 4: Verify** `go vet ./... && go test -race ./internal/animations/... ./internal/jobs/...`; then an integration check: add `TestClipRoundTrip` in `internal/integration/` that runs the real `PythonClipProcessor` on a synthetic clip (skip when `python` or the package is unavailable).
- [ ] **Step 5: Commit** `feat(api): clip worker and python clip processor`.

---

### Task 7: `animated` in manifest, bundle, stickers table and `pack.json`

**Files:**
- Modify: `services/api/internal/assets/bundle.go`, `services/api/internal/packs/{types.go,publication.go,catalog.go}`
- Test: `services/api/internal/assets/bundle_test.go`, `services/api/internal/packs/publication_test.go`

**Interfaces:**
- Produces:
  - `type ManifestAnimated struct { File string; Frames, DurationMS, Width, Height int }`; `ManifestSticker` gains `Animated *ManifestAnimated` (file is a flat name in `work/<job>/output/animated/`).
  - `packDocumentSticker` gains `Animated *packDocumentAnimated \`json:"animated,omitempty"\`` with fields `file` (`animated/NNN.webp`), `frames`, `duration_ms`, `width`, `height`.
  - `CommittedSticker` and `packs.Sticker` gain `AnimatedPath, AnimatedChecksumSHA256 string; AnimatedByteSize int64; AnimatedFrames, AnimatedDurationMS int`.
  - `StageBundle` copies animated files into `pack/animated/`, verifies each with a new `verifyAnimatedWebP(path string, w, h int) error` (uses `webp.DecodeConfig` only), appends animated bytes to the aggregate hash after that sticker's static bytes, and adds them to `writeBundle`.

- [ ] **Step 1: Write failing tests:** `TestStaticPackChecksumUnchanged` (golden: build a static manifest and assert the aggregate equals the checksum computed by the previous algorithm, hard-coded value generated from `main` before the change), `TestStageBundleIncludesAnimatedAndAppendsToChecksum`, `TestStageBundleRejectsMissingAnimatedFile`, `TestStageBundleRejectsAnimatedWrongSize` (Review Focus 6), `TestPackJSONOmitsAnimatedWhenAbsent`, `TestCommitProcessedStoresAnimatedColumns`.
- [ ] **Step 2: Run** `go test ./internal/assets/... ./internal/packs/...`. Expected: FAIL.
- [ ] **Step 3: Implement** per Interfaces; extend `validateProcessed` so a sticker with `AnimatedPath != ""` must also have a valid checksum and frames >= 2.
- [ ] **Step 4: Verify** `go vet ./... && go test -race ./...`. Expected: PASS, including all pre-existing tests.
- [ ] **Step 5: Commit** `feat(api): animated variant in manifest, bundle and stickers`.

---

### Task 8: `publish` job

**Files:**
- Modify: `services/api/internal/jobs/{types.go,runner.go}`; create `services/api/internal/jobs/publish.go`, `publish_test.go`
- Modify: `services/api/internal/animations/service.go` (add `Publishable`)

**Interfaces:**
- Consumes: Task 5 `Service`, Task 7 manifest types.
- Produces:
  - `JobRequest.Kind string \`json:"kind,omitempty"\`` (empty = `sheet`); `const KindPublish = "publish"`.
  - `func (r *Runner) EnqueuePublish(ctx context.Context, packID string) (Job, error)` returns `ErrAnimationBusy`, `ErrNoChanges`, `ErrJobActive`, `ErrPackRevoked`, `ErrNotFound`.
  - `type AnimationSource interface { Busy(ctx, packID string) (bool, error); Publishable(ctx, packID string) ([]PublishableAnimation, error) }`, `type PublishableAnimation struct { StickerIndex int; Path, Checksum string; Frames, DurationMS, Width, Height int }`; `Runner` gets it via `NewRunner` option `WithAnimations(src AnimationSource)`. `Publishable` returns only `ready`, `use_animated`, non-stale rows.
  - `execute` branches on `Kind == KindPublish` into `executePublish`: no Python; builds `work/<job>/output` by copying the current version's static files, thumbs and the publishable animated files, builds the manifest from the current version plus `Animated`, then the existing stage/commit path.
  - "No changes" means the set of `(index, animated checksum)` pairs equals the current version's animated stickers.

- [ ] **Step 1: Write failing tests** (harness from `runner_test.go`): `TestPublishCreatesNextVersionWithAnimatedStickers`, `TestPublishKeepsStaticFilesAndThumbsIdentical`, `TestPublishWithNoChangesReturnsErrNoChanges` (Review Focus 4), `TestPublishAllTogglesOffAfterAnimatedVersionCreatesVersionWithoutAnimated`, `TestPublishBlockedWhileClipRunning` (`ErrAnimationBusy`), `TestPublishBlockedWhileSheetJobActive`, `TestPublishExcludesStaleAnimation`, `TestPublishDoesNotInvokeProcessor` (fake processor fails the test if called), `TestPublishFailureKeepsOldVersionReady`.
- [ ] **Step 2: Run** `go test ./internal/jobs/...`. Expected: FAIL.
- [ ] **Step 3: Implement** per Interfaces; keep the sheet path untouched (guard `prepare` so a publish job skips `request.Source`).
- [ ] **Step 4: Verify** `go vet ./... && go test -race ./...`. Expected: PASS, existing runner tests unchanged.
- [ ] **Step 5: Commit** `feat(api): publish job creating a version with animated stickers`.

---

### Task 9: Admin API, animation prompt and `on-white.png`

**Files:**
- Create: `services/api/internal/interfaces/http/admin_animations.go`, `admin_animations_test.go`
- Modify: `services/api/internal/interfaces/http/{admin_auth.go (routes),admin_packs.go (adminPackSticker/animation summary),server.go (wiring)}`, `services/api/internal/studio/prompt.go`, `services/api/internal/interfaces/http/docs/` (API docs if the folder lists endpoints)

**Interfaces:**
- Produces routes (all `ownerOnly` except reads which use `sessionOnly`): `POST /api/v1/admin/packs/{id}/stickers/{index}/animation` (multipart `clip`, `motion_text`; 202 + animation JSON), `GET /api/v1/admin/packs/{id}/animations`, `PATCH .../animation` body `{"use_animated": bool}`, `DELETE .../animation`, `GET .../animation/preview` (animated WebP, owner only, `Cache-Control: private, no-store`), `GET /api/v1/admin/packs/{id}/stickers/{index}/on-white.png`, `POST /api/v1/admin/animations/compile` body `{pack_id, sticker_index, motion_text}` returning `{prompt}`, `POST /api/v1/admin/packs/{id}/publish-animations` (202 + job JSON).
- Error mapping: `ErrAnimationBusy` -> 409 `animation_busy`, `ErrNoChanges` -> 409 `no_changes`, `ErrJobActive` -> 409 `job_active`, `ErrNotReady` -> 409 `not_ready`, `ErrNotFound` -> 404, `ErrInvalid` -> 422.
- `studio`: `func (s *Service) CompileAnimation(ctx context.Context, designID, motionText string) (string, error)` builds the prompt from the design's character description, the spec rules (flat white background, fixed camera, seamless 3 s loop, no shadow/watermark) and `motion_text`; `on-white.png` is composed in Go (decode the committed static WebP with `webp.Decode`, draw over white, encode PNG).

- [ ] **Step 1: Write failing tests** in the style of `admin_test.go`: `TestUploadAnimationAccepts202`, `TestUploadRejectedForReaderSession` (non-owner 403), `TestPublishNoChanges409`, `TestPreviewRequiresSession`, `TestOnWhiteIsOpaqueWhiteBackground` (corner pixel `255,255,255,255`, size 512), `TestCompilePromptContainsMotionAndBackgroundRules`, `TestAnimationErrorMapping` (table over the sentinels).
- [ ] **Step 2: Run** `go test ./internal/interfaces/http/... ./internal/studio/...`. Expected: FAIL.
- [ ] **Step 3: Implement** handlers and wiring; extend `adminPackSticker` with `animation` (status, `use_animated`, `stale`, `preview_url`) so the web page can render badges from one call.
- [ ] **Step 4: Verify** `go vet ./... && go test -race ./...`. Expected: PASS.
- [ ] **Step 5: Commit** `feat(api): animation admin endpoints and prompt`.

---

### Task 10: Web factory UI

**Files:**
- Create: `clients/web-next/app/packs/animate/page.tsx`
- Modify: `clients/web-next/app/packs/detail/page.tsx`, `clients/web-next/lib/types.ts`, `clients/web-next/lib/ai-providers.ts` (video providers), `clients/web-next/lib/vi.ts` (+ English file if `I18nProvider` has one), `clients/web-next/qa/mock-server.mjs`, `clients/web-next/qa/interact.mjs`

**Interfaces:**
- Consumes: Task 9 routes. Produces: `AdminPackSticker.animation?: { status: "queued"|"running"|"ready"|"failed"|"interrupted"; use_animated: boolean; stale: boolean; preview_url: string; error_code?: string }`; page `/packs/animate?id=&index=` with `data-testid`s `animate-prompt-copy`, `animate-upload`, `animate-toggle`; detail-page elements `publish-animations`, per-sticker `animate-button-{index}`.

- [ ] **Step 1: Read** `clients/web-next/node_modules/next/dist/docs/` for the static-export/query-param page conventions (per `clients/web-next/AGENTS.md`) and copy the pattern of `app/packs/detail/page.tsx` (`Suspense` + `useSearchParams`).
- [ ] **Step 2: Extend the mock server first** (`qa/mock-server.mjs`): seeded pack `pk-ready-1` gets animation states; endpoints mirror Task 9 including 409 `animation_busy`. Add an `interact.mjs` scenario: open pack detail, press animate, copy prompt, upload a small mp4, see the status change to `ready`, toggle off and on, press publish, confirm the dialog lists the change; also check `no_changes` surfaces as an inline message.
- [ ] **Step 3: Implement** the page and detail-page changes; strings in `vi.ts` only through `t(...)` keys.
- [ ] **Step 4: Verify.** `cd clients/web-next && npm run lint && npx next build`; start the mock (`cd qa && npm run mock`), run `node qa/interact.mjs http://localhost:4500` and confirm no console errors and no horizontal overflow at 390 px and 1440 px (the report prints both). Expected: lint clean, build succeeds, scenario passes.
- [ ] **Step 5: Commit** `feat(web): animation factory and publish flow`.

---

### Task 11: Flutter playback

**Files:**
- Modify: `clients/flutter/restick/lib/src/domain/entities/sticker_asset.dart`, `lib/src/data/repositories/pack_repository_impl.dart` (`getStickers`), `lib/src/presentation/widgets/sticker_grid_tile.dart`
- Test: `clients/flutter/restick/test/sticker_assets_test.dart`, `clients/flutter/restick/test/sticker_grid_tile_test.dart`

**Interfaces:**
- Produces: `StickerAsset.animatedFile` (`File?`, null when absent), `bool get isAnimated`; `file` remains the static file; `StickerGridTile` shows `Image.file(animatedFile!, cacheWidth: 128, cacheHeight: 128)` when `isAnimated` and `!MediaQuery.disableAnimationsOf(context)`, else the existing thumb.

- [ ] **Step 1: Write failing tests:** `getStickers parses animated entries and old pack.json without animated` (Review Focus 6, temp directory with two `pack.json` variants), `tile shows thumb when disableAnimations is true`, `tile uses animated file when enabled and present`.
- [ ] **Step 2: Run** the Dart MCP `run_tests` for those two files (CLI fallback: `cd clients/flutter/restick && flutter test test/sticker_assets_test.dart test/sticker_grid_tile_test.dart`). Expected: FAIL.
- [ ] **Step 3: Implement** per Interfaces; resolve `animated.file` relative to `localDir` and ignore entries whose file does not exist.
- [ ] **Step 4: Verify** `flutter test` and `dart analyze` (Dart MCP `analyze_files`). Then measure memory on a device or simulator with a 20-sticker animated pack through DevTools and record the result in the spec's risk section (not measured yet).
- [ ] **Step 5: Commit** `feat(flutter): play animated stickers with reduce-motion fallback`.

---

### Task 12: End to end check and docs

**Files:**
- Modify: `README.md` (rename, clip job, animated flow), `services/media_factory/README.md`, `docs/superpowers/specs/2026-11-13-animated-stickers-design.md` (replace "pytest" by "unittest", record measured Flutter memory and any resolved risk)

- [ ] **Step 1: Run the full local stack** per the existing rerun-local practice (build, start the mock/real API on :4500): create a pack from a sheet, upload the lotus clip for one sticker, wait for `ready`, toggle on, publish, download the bundle, confirm `pack.json` has `animated` for that sticker only, the aggregate checksum matches the documented algorithm, and the static pack's older version is untouched.
- [ ] **Step 2: Run every suite:** Python unittest + ruff + `check_docstrings.py`; `cd services/api && go vet ./... && go test -race ./...`; web lint, build and `interact.mjs`; Flutter test and analyze. Expected: all green.
- [ ] **Step 3: Run `superpowers:verification-before-completion`** and then `superpowers:requesting-code-review`.
- [ ] **Step 4: Commit** `docs: animated stickers flow`.
