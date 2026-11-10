# Restick Single-Operator Sticker Factory Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Chuyển Restick thành studio một quản trị viên, ghép prompt cho AI ngoài, xử lý sheet thành pack và phục vụ client trên một VPS/Docker với SQLite + filesystem.

**Architecture:** Một application container: Go REST/static server, SQLite local, tài nguyên trên bind mount, Python child chỉ chạy khi có job. Giữ read API/pack schema của SDK; bỏ multi-user/review và các service network không còn cần. Static export Next.js được Go phục vụ, không Node runtime.

**Tech Stack:** Go 1.26.0, `modernc.org/sqlite v1.62.0`, Python >=3.12 với Pillow/NumPy/OpenCV-headless/Typer hiện có, Next.js/React/Tailwind/React Query hiện có; Docker Compose v2 và TLS proxy bên ngoài application.

**Spec:** [`docs/specs/2026-11-10-single-operator-sticker-factory-design.md`](../specs/2026-11-10-single-operator-sticker-factory-design.md).

## Global Constraints

- SQLite + filesystem, một VPS/một instance ghi; target data dir mới, không tự migrate/reset/xóa dữ liệu PostgreSQL/MinIO cũ.
- Go 1.26.0; modernc.org/sqlite v1.62.0; không ORM, GraphQL, gRPC, LLM/GPU server, Node runtime hoặc worker HTTP riêng.
- WAL, foreign_keys=ON, synchronous=FULL, busy_timeout=5000 ms; một DB connection, transaction ngắn, không giữ trong processing/file streaming.
- Seed 425 visual +329 context fragments và 24 palettes; `restick.seed.v1`; runtime không parse Markdown, init không ghi đè user edits.
- Một Python child active; job timeout 300s mặc định; không retry tự động; version cấp tăng, không tái dùng; revoke thắng completion.
- Input PNG/JPEG/WebP tối đa 8 MiB/file, cạnh tối đa 4096 px; multipart body 8 MiB +64 KiB; JSON request/result tối đa 1 MiB; ZIP output tối đa 32 MiB.
- Rows/columns 1–8, tổng không quá 64; outline mặc định 4px ở canvas 512, range 0–16, màu #FFFFFF; vùng trống thêm 4px; thumbnail 128, WebP lossless.
- API key 32 random bytes chỉ đọc, SHA256 at rest; owner session 32 random bytes, SHA256 at rest, TTL24h, cookie __Host-restick_admin HttpOnly/Secure/SameSite=Strict/Path=/.
- Login 5 lần sai/15 phút/source; admin unsafe requests bắt buộc Origin đúng public base URL; forwarded client IP chỉ khi peer nằm trong RESTICK_TRUSTED_PROXY_CIDRS (mặc định rỗng).
- `/v1/objects` và `/v1/bundles` cho client key hoặc admin session; auth/visibility trước range/304. Không có API key trong URL hoặc static file root /data.
- `restick.pack.v1`; checksum là SHA256 nối sticker bytes theo thứ tự manifest, không phải ZIP. Manifest version immutable là snapshot; catalog mutable có thể khác metadata trong bundle cũ.
- Tombstone RFC3339 UTC precision giây, since inclusive >=, không pagination trong cutover; revoked terminal, slug giữ unique cả sau revoke.
- Docker UID/GID10001, bind /data local, rootfs read-only, tmpfs /tmp, init=true, stop_grace_period=30s, log10m×3; port127.0.0.1:8080:8080.
- Backup/restore offline với app lock; restore chỉ vào data dir trống, bắt buộc password mới, clear sessions và disable toàn bộ API keys được restore.
- Mọi implementation chỉ bắt đầu sau **một lần duyệt spec + plan + phương pháp/workspace**; không coi tài liệu là code đã chạy.

## Review Focus

1. Crash giữa filesystem rename và SQLite commit không công khai bản nửa chừng — T6 kiểm `TestCrashWindowDoesNotExposeVersion`.
2. Revoke cạnh tranh completion không hồi sinh pack hoặc job — T6 kiểm `TestRevokeWinsLateCompletion`.
3. File/member/symlink độc hại không thoát data root hoặc SDK dest — T3 và T7 kiểm đường dẫn qua public operation, không test source text.
4. Restore backup cũ không làm sống lại session/key đã mất hiệu lực — T10 kiểm `TestRestoreInvalidatesCredentials`.
5. Anonymous/expired key vẫn bị chặn với HEAD/Range/If-None-Match, admin cookie vẫn xem được thumbnail — T7/T8 kiểm actual HTTP semantics.

## Bản đồ file và quyền sở hữu task

Đường dẫn dưới đây là target mới có chủ đích, không khẳng định file đã tồn tại. Không đổi module Go chỉ để rename repo. Các test đặt cùng package theo convention Go; Python dùng `unittest` chuẩn để không thêm test dependency.

| Nhóm | File chính | Task |
| --- | --- | --- |
| Storage/config/locks | `services/api/internal/storage/{db.go,lock.go,migrate.go,migrations/0001_factory.sql}`, `internal/infrastructure/config/config.go` | T1 |
| Owner/session/key | `services/api/internal/auth/{owner.go,sessions.go,keys.go,proxy.go}` | T2 |
| Assets/catalog/bundle | `services/api/internal/assets/{store.go,bundle.go}`, `internal/packs/{catalog.go,types.go}` | T3 |
| Studio/seed/prompt | `services/api/internal/studio/{types.go,library.go,designs.go,prompt.go,seed.go,seed/*.json}` | T4 |
| Python machine processor | `services/img_tool/src/img_tool/interfaces/machine_job.py`, application/outline/layout/domain modules | T5 |
| Jobs/publication/revoke | `services/api/internal/jobs/{types.go,runner.go,processor.go,recovery.go}`, `internal/packs/publication.go` | T6 |
| Public HTTP + SDK archive | `services/api/internal/interfaces/http/{public.go,assets.go}`, Flutter downloader/tests | T7 |
| Admin HTTP + router/OpenAPI | `services/api/internal/interfaces/http/{admin_*.go,server.go,docs.go}`, `contracts/openapi/v1.yaml` | T8 |
| Static studio | `clients/web-next/app/**`, `lib/**`, existing components; obsolete UI route removal | T9 |
| Offline maintenance | `services/api/internal/maintenance/{backup.go,restore.go}` | T10 |
| App/runtime cutover | `services/api/cmd/restick/*`, root Dockerfile/Compose/env, CI and obsolete runtime deletion | T11 |
| Integrated acceptance | `services/api/internal/integration/factory_test.go`, acceptance notes | T12 |

Quy tắc interface: service dùng `*sql.DB` + struct rõ nghĩa, không thêm repository wrappers một-method. HTTP sở hữu JSON transport, studio/packs/jobs sở hữu model/invariants, Python không mở DB. T2 và T3 chạy song song sau T1; T4 sau T3 vì cần assets.Store; T5 độc lập Go. Những file chia sẻ (HTTP router, root startup, pipeline core) chỉ có một task owner tại một thời điểm.

---

### T1: SQLite, cấu hình và data-root lock

**Review:** per-task — persisted format, concurrency

**Files:** Create storage files ở bảng, `services/api/internal/storage/db_test.go`; Modify `services/api/go.mod`, `go.sum`, `internal/infrastructure/config/config.go`; Create `config_test.go`.

**Interfaces:** `storage.Open(ctx context.Context, dataDir string) (*sql.DB,error)` áp dụng migrations/pragmas; `storage.AcquireAppLock(dataDir string) (release func() error,err error)` giữ exclusive lock của app/maintenance. `config.Load() (Config,error)` với DataDir, HTTPAddr, PublicBaseURL, AdminPasswordFile, ProcessTimeout, TrustedProxyCIDRs; PublicBaseURL phải HTTPS origin, không lấy từ Host request.

- [ ] Viết `TestReopenPreservesRecords`, `TestForeignKeysAndUniqueSlug`, `TestOneActiveJobPerPack`, `TestMigrationFailureRollsBack`, `TestAppLockRejectsSecondWriter`, `TestConfigRejectsMissingPublicOrigin`. Mỗi DB test dùng TempDir; kiểm persistence/invariant thật, không chỉ PRAGMA strings.
- [ ] Chạy `go test ./internal/storage ./internal/infrastructure/config` trong `services/api`; xác nhận thất bại vì chưa có implementation/contract mới, không đổi assertion để hợp code cũ.
- [ ] Pin Go1.26.0 + driver v1.62.0. Viết schema toàn bộ tables từ spec §8.2, constraints/indices; UTC timestamp dùng representation nhất quán và tombstone giây; JSON snapshots có schema_version. Open không tạo memory fallback; app lock từ chối sai quyền/khóa đang giữ.
- [ ] Chạy lại đúng hai package test. Khi pass, commit đúng storage/config/go.mod/go.sum và tests bằng path, thông điệp `feat(storage): add durable single-instance SQLite store`.

### T2: Một owner, session và API key chỉ đọc

**Review:** per-task — authentication/security

**Files:** Create `internal/auth/{owner.go,sessions.go,keys.go,proxy.go,auth_test.go}`. Không sửa HTTP router ở task này.

**Interfaces:** `auth.NewService(db *sql.DB) *Service`; methods `BootstrapOwner(ctx context.Context,password []byte) error`, `Login(ctx context.Context,password string,now time.Time) (string,error)`, `ValidateSession(ctx context.Context,raw string,now time.Time) error`, `Logout(ctx context.Context,raw string) error`, `SetPassword(ctx context.Context,password []byte) error`, `CreateKey(ctx context.Context,name string,expiresAt *time.Time) (CreatedKey,error)`, `RotateKey(ctx context.Context,id string) (CreatedKey,error)`, `DisableKey(ctx context.Context,id string) error`, `ValidateKey(ctx context.Context,raw string,now time.Time) error`. `CreatedKey{ID,Name,Prefix,Raw string}` chỉ create/rotate trả Raw; Login trả raw session. Bootstrap không overwrite owner; SetPassword revoke sessions. Bcrypt hiện có, token entropy crypto/rand.

- [ ] Viết `TestOwnerBootstrapDoesNotOverwritePassword`, `TestSessionExpiryAndLogout`, `TestPasswordResetRevokesSessions`, `TestRotateDisablesOldKeyAtomically`, `TestOnlyHashesPersisted`, `TestExpiryBoundary`, `TestForwardedIPRequiresTrustedPeer`. Kiểm request-source derivation với peer ngoài/ở CIDR và chuỗi XFF có địa chỉ giả phía trái.
- [ ] Chạy `go test ./internal/auth`; xác nhận fail vì API chưa triển khai.
- [ ] Implement services trên storage T1. Không plaintext log/DB; không viết last_used cho mỗi download. Phân tách validation session/key, không suy ra quyền admin từ key.
- [ ] Chạy lại package test; commit `feat(auth): add single-owner sessions and read-only client keys` với path task.

### T3: File store local, catalog và ZIP an toàn

**Review:** per-task — security/persisted format

**Files:** Create `internal/assets/{store.go,bundle.go,store_test.go,bundle_test.go}`; Create `internal/packs/{types.go,catalog.go,catalog_test.go}`. Dùng lại logic ZIP member layout từ `services/ftp/internal/application/build_bundle.go` theo cách streaming disk; không kéo dependency MinIO sang module API.

**Interfaces:** `assets.NewStore(dataDir string) (*Store,error)`; methods `SaveSource(ctx context.Context,r io.Reader,filename string) (Source,error)`, `StageBundle(ctx context.Context,jobID string,manifest PackManifest) (StagedVersion,error)`, `CommitFiles(ctx context.Context,staged StagedVersion) (CommittedFiles,error)` đóng/fsync/rename, **chưa** public. `packs.NewService(db *sql.DB) *Service`; methods `Create(ctx context.Context,draft PackDraft) (Pack,error)`, `UpdateCatalog(ctx context.Context,id string,metadata Metadata) (Pack,error)`, `ListReady(ctx context.Context,filter Filter) (Page,error)`, `Get(ctx context.Context,id string) (Pack,error)`, `GetVersion(ctx context.Context,id string,version int) (Version,error)`, `DeleteDraft(ctx context.Context,id string) error`. Source có ID/relative path/media type; Pack/Version giữ status/current pointer/immutable metadata, không expose absolute paths. PackManifest/StagedVersion/CommittedFiles thuộc package assets và có jobID, packID, version, file lists/checksum cần cho T6; assets không import jobs/packs.

- [ ] Viết `TestCatalogEditDoesNotMutateCommittedManifest`, `TestSlugRemainsReservedAfterRevocation`, `TestDraftDeleteRejectsActiveJob`, `TestPathTraversalAndSymlinkAreRejected`, `TestBundleLayoutAndStickerChecksum`, `TestOversizedBundleLeavesNoCommittedArtifact`.
- [ ] Chạy `go test ./internal/assets ./internal/packs` và xác nhận fail.
- [ ] Implement source magic/decode/size/path boundary, identity server cấp và ZIP unique members; checksum bằng streaming WebP bytes theo manifest. File/member resolution từ allowlist DB, không join trực tiếp key HTTP không validate. Metadata catalog và immutable manifest tách rõ.
- [ ] Chạy lại hai package tests; commit `feat(assets): store catalog and versioned packs on local disk`.

### T4: Thư viện đầy đủ và prompt studio có snapshot

**Review:** per-task — persisted format

**Files:** Create `internal/studio/{types.go,library.go,designs.go,prompt.go,seed.go,studio_test.go}` và `seed/{groups.json,fragments.json,palettes.json}`. Nguồn biên soạn ở `docs/research/sticker-factory/{creative-fragments,context-and-expression,palette-recipes}.md`; không sửa bài nghiên cứu chỉ để khớp parser.

**Interfaces:** `studio.NewService(db *sql.DB,files *assets.Store) *Service`; CRUD Groups/Fragments/Characters/Designs với model trong types.go; methods `Compile(ctx context.Context,draft Draft) (CompiledPrompt,error)`, `Suggest(ctx context.Context,draft Draft,rng *rand.Rand) (Draft,error)`, `SaveDesign(ctx context.Context,draft Draft) (Design,error)`, `Seed(ctx context.Context) error`. Dùng math/rand seeded trong test; không dùng RNG sáng tạo cho auth. Draft gồm character ID tùy chọn, selected group/fragment IDs/pinned, palette ID tùy chọn, freeText, rows/columns, cells[]{action,expression,fragmentIDs,freeText}, outlineWidth/Color. CompiledPrompt gồm text hoàn chỉnh và snapshot; Seed idempotent qua seed_versions, không ghi đè user content.

- [ ] Viết `TestArchivedFragmentDoesNotChangeSavedPrompt`, `TestSeedRestartPreservesUserEdits`, `TestPinnedSelectionsAndCellTextSurviveSuggestion`, `TestCompileRejectsCellCountAndOutlineRange`, `TestCompileDoesNotMixWhiteAndTransparentRequirements`, `TestGroupDeleteRejectsMembers`, `TestPaletteAndCustomGroupsCompile`.
- [ ] Chạy `go test ./internal/studio`; xác nhận fail.
- [ ] Biên soạn seed canonical đủ 754+24, gán ID/scope và prompt English thật cho mục thiếu. Nhóm visual theo D01…D24; context theo tám họ trong dossier; palette group riêng. Import bằng embed JSON; validate ID/reference/schema. Một lần kiểm inventory/source coverage bằng script tạm, không permanent source-text/count-only tests.
- [ ] Implement CRUD/archive, private reference assets, group constraints và snapshot. Suggest chỉ đổi nhóm tham gia/unpinned; không API provider/keyword DSL/compatibility engine. Compile mô tả có quan hệ + technical template từ spec.
- [ ] Chạy lại tests; commit `feat(studio): add reusable fragment library and prompt composition`.

### T5: Python machine-job, kiểm lưới và tạo viền

**Review:** per-task — parser/security/persisted format

**Files:** Create `services/img_tool/src/img_tool/interfaces/machine_job.py`, `application/process_job.py`, `infrastructure/outline.py`; Modify `application/process_sheet.py`, `infrastructure/layout.py`, `domain/pack.py` chỉ phần shared core cần thiết; Create `services/img_tool/tests/test_machine_job.py`, `test_outline.py`, `test_sheet_layout.py`. Existing entrypoint/grpc imports chỉ xóa ở T11 để không phá local CLI giữa task.

**Interfaces:** `process_job(request_path: Path,result_path: Path) -> None` đọc `restick.job.v1`, ghi `restick.job-result.v1`; `compose_sticker(image: Image.Image,outline_width: int,outline_color: str) -> Image.Image`; request/result fields đúng spec §7.1. Python chỉ trả per-cell artifact metadata, Go T6 sở hữu manifest/ZIP publication. Không UUID tự chọn hoặc hardcode version1 trong machine path.

- [ ] Viết fixture-generated tests cho sheet 2×2 nền trắng, alpha thật, count sai, hai box cùng ô, box chạm giữa ô; test outline0/4/16, white/colored foreground, mép không clipping, alpha không nhân hai lần; malformed schema/path/animation decode cũng fail. Không gọi model AI trong test.
- [ ] Chạy `python -m unittest discover -s services/img_tool/tests -p 'test_*.py'` trong env có project installed; xác nhận fail đúng chức năng thiếu.
- [ ] Implement near-white/alpha core, mapping box centroid→cell unique và row-major, simple/không neural model mặc định; fit safe area trước mask outline; thông báo lỗi có code, không output subset. Ghi result mới atomically, stderr bounded diagnostics, exit nonzero khi fail.
- [ ] Chạy lại `python -m unittest discover -s services/img_tool/tests -p 'test_*.py'`; trong `test_machine_job_cli_roundtrip`, gọi module CLI bằng subprocess với request/result nằm trong TemporaryDirectory, assert exit0 + files đúng kích thước/thứ tự và dùng lại manifest inputs ở T6. Kiểm output thật, không chỉ mock argv hoặc assert không throw.
- [ ] Commit `feat(processor): process validated sticker sheets with configurable outlines` đúng Python files/tests.

### T6: Durable jobs, publication nguyên tử và revoke

**Review:** per-task — concurrency/persisted format

**Files:** Create `internal/jobs/{types.go,runner.go,processor.go,recovery.go,runner_test.go}`; Create `internal/packs/publication.go`, `publication_test.go`.

**Interfaces:** `jobs.NewRunner(db *sql.DB,files *assets.Store,processor Processor,timeout time.Duration) *Runner`; `Enqueue(ctx context.Context,packID,sourceID,designID string) (Job,error)` snapshots transaction và reserve version; `Run(ctx context.Context) error` một worker; `Get(ctx context.Context,id string) (Job,error)`; `Revoke(ctx context.Context,packID string) error` DB tombstone + interrupted và cancel active child. `Processor.Process(ctx context.Context,request JobRequest) (JobResult,error)` là boundary để test scheduling; production exec argv fixed T5, không shell. `packs.CommitProcessed(ctx context.Context,db *sql.DB,result ProcessedVersion) error` phải CAS job running và pack non-revoked. `packs.ProcessedVersion` khai báo ở publication.go gồm JobID/PackID string, Version int, Manifest assets.PackManifest, Files assets.CommittedFiles; jobs chuyển result đã validate sang type này, không tạo vòng import packs→jobs.

- [ ] Viết `TestConcurrentEnqueueOneActiveJob`, `TestWorkerNeverRunsTwoChildren`, `TestStartupInterruptsRunningAndResumesQueued`, `TestTimeoutKeepsOldReadyVersion`, `TestCrashWindowDoesNotExposeVersion`, `TestRevokeWinsLateCompletion`, `TestFailedVersionNumberIsNotReused`.
- [ ] Chạy `go test ./internal/jobs ./internal/packs`; xác nhận fail vì runner/publication chưa có.
- [ ] Implement claim/short transactions, process bounded timeout/result, stage/validate files rồi DB publish. Revoke transaction đổi queued/running→interrupted và tombstone một lần; completion/worker failure không ghi đè trạng thái terminal. Startup cleanup chỉ orphan không referenced; không xóa committed data.
- [ ] Chạy lại targeted tests với `-race`; chạy một real Python invocation vào TempDir (không chỉ fake Processor); commit `feat(jobs): make image processing durable and revocation-safe`.

### T7: Public read/file API và an toàn SDK ZIP

**Review:** per-task — security/public contract

**Files:** Create `internal/interfaces/http/{public.go,assets.go,public_test.go,assets_test.go}`; Modify `clients/flutter/restick/lib/src/data/local/pack_downloader.dart`; Create `clients/flutter/restick/test/pack_downloader_test.dart`; chỉ cập nhật thumbnail headers nếu test thực tế chứng minh cần, không thay public Flutter API.

**Interfaces:** `NewPublicHandler(catalog *packs.Service,files *assets.Store,auth *auth.Service) http.Handler` đăng ký read routes spec §9.1; file handler cho phép valid key **hoặc** session cookie. Không đăng ký admin writes ở đây. Flutter giữ `PackDownloader.unzipAndVerify` signature, validate tất cả members và manifest paths trước khi tạo/ghi output.

- [ ] Viết HTTP tests `TestPublicCatalogVisibilityAndPagination`, `TestExistingSDKResponseFields`, `TestBundleChecksumContract`, `TestOldCommittedVersionStillDownloads`, `TestTombstoneInclusiveTimestamp`, `TestAuthPrecedesHeadRangeAnd304`, `TestAdminCookieLoadsThumbnail`, `TestQueryAPIKeyDoesNotAuthorize`.
- [ ] Viết Dart tests archive traversal (../, absolute, nested escape), duplicate/symlink member khi decoder biểu diễn được, manifest reference escape, valid nested thumbs và checksum mismatch không để file ngoài dest. Dùng local fixture bytes, không network.
- [ ] Chạy targeted Go tests; chạy Dart MCP run_tests cho `test/pack_downloader_test.dart` (server phải dùng Flutter SDK); chứng minh fail trên hành vi mới. Không thay backend data shape để che test fail SDK.
- [ ] Implement handlers stream/range/private cache với auth/pack-status trước mọi304; URLs từ PublicBaseURL; tombstone inclusive. Guard ZIP member và manifest paths trước extraction, không bỏ validation ZIP phía server.
- [ ] Chạy lại đúng tests Go/Dart; commit `feat(api): serve authenticated local packs with SDK-compatible downloads` với Go và Dart paths T7; cập nhật Flutter CHANGELOG cùng thay đổi bảo mật của package.

### T8: Admin API, session middleware và OpenAPI một nguồn

**Review:** per-task — authentication/security

**Files:** Create `internal/interfaces/http/{admin_auth.go,admin_studio.go,admin_packs.go,admin_keys.go,admin_test.go}`; Modify `server.go`, `docs.go`; Replace obsolete `api.go` sau migrate mọi handler cần thiết; Modify `contracts/openapi/v1.yaml`; remove duplicate `internal/interfaces/http/docs/openapi.yaml` bằng cách phục vụ spec được copy từ canonical ở build/package target T11.

**Interfaces:** `NewHandler(Dependencies) http.Handler` combines T7 + admin routes + docs/health/static dependencies; `Dependencies` gồm Auth, Studio, Packs, Jobs, Assets, PublicBaseURL, TrustedProxyCIDRs, WebDir, OpenAPIPath. Cookie/session middleware không nhận API key thay owner. `GET /admin/session` trả username; login JSON `{password}`, thành công204 Set-Cookie, logout204 và DB revoke. Các create resource trả201 object/ID; mutations successful update200, delete204; process202 Job.

- [ ] Viết `TestClientKeyCannotMutateAnyAdminRoute`, `TestMissingForeignOriginRejected`, `TestCookieFlagsAndLogout`, `TestLoginRateLimitWindow`, `TestJSONAndMultipartLimits`, `TestPrivateReferenceNotPublic`, `TestRemovedCommunityRoutesNotRegistered` (HTTP behavior, không grep code).
- [ ] Chạy `go test ./internal/interfaces/http -run 'Test(ClientKey|MissingForeign|Cookie|Login|JSONAnd|PrivateReference|RemovedCommunity)'`; xác nhận fail.
- [ ] Implement đầy đủ route list spec §9.2, transport validation/error mapping và auth order. Freeze request/response DTO trong canonical OpenAPI, không thêm endpoint stub. Resource deleted/referenced giữ invariant snapshot; HTTP400/401/403/404/409/410/413/422 theo lỗi thực tế; failures processing ở job object không biến thành reject nghiệp vụ.
- [ ] Chạy lại package tests; kiểm OpenAPI parse bằng công cụ tạm hoặc validator sẵn có, không thêm dependency runtime chỉ cho validation. Commit `feat(admin): expose single-owner factory management API`.

### T9: Studio UI và static export thật

**Files:** Modify `clients/web-next/next.config.ts`, package.json, `app/layout.tsx`, `app/page.tsx`, `app/providers.tsx`, `lib/{auth.tsx,api.ts,types.ts,i18n-messages.ts}`, `components/{app-shell.tsx,guards.tsx,ui.tsx}`; Create `app/{factory,library,characters}/page.tsx`, `app/packs/detail/page.tsx`; Modify existing `/login`, `/packs`, `/packs/new`, `/keys`, `/api-docs`; Remove `/register`, `/admin/**`, `/packs/[id]` UI and obsolete review constants/callers.

**Interfaces:** Web dùng DTO OpenAPI T8, session cookie same-origin; `useAuth` không trả token/isAdmin/register, chỉ owner/session-ready/login/logout; mọi caller được migrate. Routes cố định đọc ID client-side qua query và Suspense. Data fetch qua same-origin credentials, không localStorage bearer/API key query.

- [ ] Trước chỉnh UI, ghi acceptance scenario: login → thêm mảnh riêng → ghép/pin/suggest/save/copy → tạo pack/upload/process → thấy ready → cấp key → revoke. Bao gồm reload `/packs/detail?id=...` với ID tạo sau build.
- [ ] Implement form/empty/loading/error states đúng domain, label Việt, giữ house components. Seed hiển thị grouped/searchable; palette có màu và label; private refs dùng cookie. Không giả AI generation hay progress phần trăm không có dữ liệu.
- [ ] Đổi output export, bỏ headers/rewrites/runtime route handlers không được export; Go sẽ sở hữu dev/prod proxy phù hợp. Không dùng generateStaticParams cho pack ID phát sinh runtime. Remove `next start` production script/caller nếu còn giả Node runtime.
- [ ] Chạy `npm run lint && npm run build` trong web-next. Kiểm `out/` tồn tại và route cần thiết đều xuất. Build/lint ở đây là check UI-contract, không thay thế browser verification.
- [ ] Ghi bàn giao browser scenario cho T12: actual backend/TLS chỉ có sau T11, nên không tạo vòng phụ thuộc yêu cầu T9 chờ T11. T9 kết thúc bằng static artifact đã build; **chưa coi UI end-to-end được kiểm chứng** cho đến browser proof desktop/mobile ở T12. Không thêm UI test framework chỉ để test wording.
- [ ] Commit `feat(web): replace community console with static sticker studio` chỉ web paths.

### T10: Offline backup, restore và credential reset

**Review:** per-task — destructive operation/security/persisted format

**Files:** Create `internal/maintenance/{backup.go,restore.go,maintenance_test.go}`. CLI command wiring do T11 sở hữu.

**Interfaces:** `Backup(ctx context.Context,dataDir,outputDir string) error`; `Restore(ctx context.Context,inputDir,dataDir,passwordFile string) error`; cả hai AcquireAppLock. Format backup folder `restick.backup.v1` với manifest schema+file SHA256 và snapshot DB/sources/packs/bundles; không work/secret file. Restore data dir chỉ được có lock file do chính lệnh tạo, không file app khác; kiểm empty trước copy.

- [ ] Viết `TestBackupRejectsRunningWriter`, `TestRestoreRefusesExistingData`, `TestRestoreRejectsTamperedOrEscapingFiles`, `TestRestoredFilesMatchDatabase`, `TestRestoreInvalidatesCredentials` (old session/key fail, owner login password mới được), `TestRestoreMissingPasswordFailsBeforeChanges`.
- [ ] Chạy `go test ./internal/maintenance` và xác nhận fail.
- [ ] Implement offline snapshot + backup manifest, integrity/reference validation; restore copy vào staging cùng filesystem, finalize khi đã validate, không overwrite target. Set owner bcrypt mới, clear sessions, disable keys trước nhận traffic. Document trusted-backup threat model; không hứa checksum chống forgery.
- [ ] Chạy lại package tests và restore một backup generated từ fixture pack T3; commit `feat(maintenance): add safe offline factory backup and restore`.

### T11: Một runtime Docker, entrypoint và xóa đường cũ

**Review:** per-task — deployment/security/destructive cutover

**Files:** Create root `Dockerfile`, `services/api/cmd/restick/{main.go,commands.go,main_test.go}`; Replace root `docker-compose.yml`, `docker-compose.override.yml`, `docker-compose.host.yml`, `.env.example`; Modify `.dockerignore`, `.github/workflows/{ci-backend.yml,deploy-web.yml}`, `README.md`, `services/img_tool/{pyproject.toml,Dockerfile}`, `interfaces/cli.py`; remove obsolete `cmd/api_svc`, `internal/infrastructure/postgres`, `interfaces/grpc`, old migrations/user/review code, services/ftp, services/gateway, infra/{postgres,minio}, service Compose files, web Docker/Compose, proto/gen tooling and Python grpc/serve/worker modules **after caller migration**. Keep Python core/CLI needed by T5. Không xóa unrelated research/history/assets, Flutter package hoặc thư mục host data.

**Interfaces:** `restick serve` (default), `healthcheck`, `admin set-password --password-file`, `backup --output`, `restore --input --password-file`. Serve boot order config→app lock→DB/schema→owner bootstrap if empty→seed→recovery→runner+HTTP; graceful shutdown/cancel child. `healthcheck` bypasses app lock and probes running HTTP loopback, không mở DB cạnh server chỉ để kiểm tra. Health response tối thiểu status, readiness xác minh DB và writable data root qua service state.

- [ ] Viết `TestStartupRejectsUnwritableDataAndMissingBootstrapSecret`, `TestHealthcheckUsesRunningServer`, `TestShutdownCancelsChildWithoutPublishing`, `TestUnknownAPIPathIsNotHTML`, `TestStaticRouteReload` (request actual exported directory generated T9).
- [ ] Chạy targeted command/http tests và chứng minh failure trước entrypoint/static wiring.
- [ ] Build Node24 + Go1.26 stages, runtime Python3.12-slim với dependencies thật; copy out và canonical OpenAPI; non-root10001, /data/tmpfs writable đúng spec, exec module path cố định. Pyproject bỏ grpc/protobuf khi không còn imports; không thêm rembg. Go dependencies bỏ pgx/grpc/JWT khi không còn callers.
- [ ] Compose chỉ restick, loopback port, bind data, secrets mount read-only, init/read_only/no-new-privileges/cap_drop/log rotation/restart/stop grace. Chuẩn bị owner của ./data bằng hướng dẫn rõ; không root chown lén khi runtime. Tài liệu TLS proxy prerequisite, trusted peer CIDR và body/timeouts, không dựng service ngầm.
- [ ] Migrate CI: Go1.26 chỉ API module, Python core tests, web static build, image build/contextroot và publish immutable image tag. Đổi workflow deploy-web thành publish/deploy app thống nhất, không để lệnh `compose up web`/mutable latest hoặc secret cũ. Không tự SSH/deploy VPS trong task.
- [ ] Chạy targeted tests, `docker compose config --quiet`, build image khi được approval thực thi; start trên **data dir tạm mới**, kiểm only restick service, log không lỗi thiếu shared libs/Node/PG/S3. Không `down -v`, prune hay xóa volume cũ. Cập nhật README/env/changelog liên quan như phần của cutover.
- [ ] Commit `refactor(runtime): consolidate factory into one Docker application` bằng explicit path; obsolete removals chỉ stage bằng danh sách path đã review, không `git add -A`.

### T12: Chứng minh end-to-end và bàn giao vận hành

**Files:** Create `services/api/internal/integration/factory_test.go`; bổ sung acceptance evidence vào tài liệu vận hành khi chạy thực tế, không tạo báo cáo pass giả.

**Interfaces:** Dùng production dependencies T1–T11, `httptest.NewTLSServer`, real Python process và DB/tempdata; không mock server-to-processor contract trong end-to-end test. Dùng fixture sheet sở hữu trong `services/img_tool/testdata` hoặc generated fixture được kiểm quyền.

- [ ] Viết `TestFactoryRoundTrip` thực hiện owner bootstrap/session, custom fragment/prompt snapshot, pack/source/job→ready, client key list/meta/file/ZIP, SHA256 theo manifest, catalog edit không đổi ZIP, revoke mọi version và tombstone sync. Chọn timeout test rộng hơn child timeout fixture; không sleep để đoán completion, poll job với deadline rõ.
- [ ] Viết `TestContainerRecreatePreservesPack` như smoke tạm qua actual Docker, không source-text config test. Kiểm restore vào mount mới rồi login mật khẩu mới, tạo key mới và tải/verify cùng pack. Không khôi phục vào volume đang phục vụ.
- [ ] Chạy narrow integration test rồi toàn bộ **affected Go module** một lần `go test -race ./...`; Python unittest suite mới; Dart MCP targeted config/downloader tests; web lint/build một lần tại final integration khi cần. Không tự chạy mọi module/repo, migration hoặc remote deploy ngoài scope.
- [ ] Browser proof trên actual new static studio/TLS surface: desktop/mobile, full T9 scenario, direct URL reload và cookie thumbnail. Nếu không có VPS/TLS đã cấu hình, integration TLS server và container local chứng minh phần reachable; không tuyên bố đã deploy VPS. Một lượt ảnh AI thật của owner là acceptance người dùng, không thay bằng fixture và gọi là đã đo AI.
- [ ] Kiểm từng AC01–AC13 có bằng chứng; ghi đúng giới hạn (chưa có AI sample/benchmark nếu chưa làm). Gỡ script/fixture tạm không cần giữ, giữ deterministic regression tests có ích.
- [ ] Commit `test(factory): verify studio-to-client lifecycle and recovery`; dùng whole-branch review để xem seams/persisted-state/security; không tự merge/deploy khi chỉ được duyệt triển khai local.

## Coverage và dependency order

| AC | Task chịu trách nhiệm |
| --- | --- |
| AC01 runtime/persistence | T1, T11, T12 |
| AC02 owner/client separation | T2, T8 |
| AC03 seed/extension/snapshot | T4 |
| AC04 composition/pinning/copy | T4, T9 |
| AC05 complete processing and auto-ready | T3, T5, T6, T12 |
| AC06 failures don't publish or lose old version | T5, T6 |
| AC07 active jobs/recovery/orphans | T1, T6 |
| AC08 SDK bundle/ZIP/checksum | T3, T7, T12 |
| AC09 revoke/tombstone | T6, T7, T12 |
| AC10 assets/auth/path boundary | T2, T3, T7, T8 |
| AC11 static studio/browser | T9, T11, T12 |
| AC12 offline backup/restore | T10, T12 |
| AC13 canonical docs/clean cutover | T8, T11 |

Dependencies: T1→T2 và T3; T3→T4; T5 độc lập Go; T6 sau T1/T3/T4/T5; T7 sau T2/T3/T6; T8 sau T2/T4/T6/T7; T9 sau T8; T10 sau T1/T2/T3 và có thể song song T4–T9; T11 sau T1–T10; T12 sau T11. Không tạo stub để phá dependency. Browser proof của T9 được parent tích hợp ở T12, không là điều kiện khiến T11 phải chờ chính nó.

## Kết quả self-review của plan

- Mỗi AC có task; interfaces đầu ra được dùng nhất quán, không dựa vào function chưa có owner.
- Failure modes Review Focus có test cụ thể, không chỉ prose “handle edge cases”.
- Builder check đã làm rõ manifest snapshot, revoke/CAS, admin cookie file auth, restore invalidation, cùng UID không sandbox, canonical seed và reserved slug. Các quyết định nằm trong spec, không ẩn trong task.
- Chọn **Subagent-driven** khi được duyệt vì có authentication, concurrency và persisted formats; task gắn Review có review riêng, cuối cùng một whole-branch review. Không gọi implementation skill trong giai đoạn viết tài liệu.

## Handoff và quyền thực thi

Spec và plan là một bộ để duyệt. Đề xuất thực thi trên **worktree riêng** với branch `feature/single-operator-sticker-factory`, tránh chạm các thay đổi tracked đang có trong checkout hiện tại (`.gitignore` và ba widget Flutter). Các artifact .serena/.superpowers/context/research không được stage hoặc xóa bừa. Tạo worktree chỉ sau khi người dùng duyệt workspace cùng spec/plan; nếu muốn giữ trong checkout hiện tại, cần xử lý rõ các thay đổi có sẵn trước.

Không tự làm migration production, reset dữ liệu, push, merge hoặc deploy VPS. Những bước destructive/backup thực tế phải dùng thư mục test mới hoặc có quyền vận hành riêng; chấp thuận thiết kế không phải chấp thuận xóa dữ liệu cũ.
