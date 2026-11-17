# Marketplace Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Đưa trang marketplace của Flutter SDK lên giao diện B (carousel thẻ lớn + danh sách "Vừa lên kệ") + ngăn kéo chi tiết D, có màu nền pack do server quy định (admin chọn tay hoặc tự gợi ý từ sticker), sticker xem trước thật từ server, và chuyển động tải/thành công/thất bại.

**Architecture:** Server (Go) thêm `preview_stickers` và `theme_color` vào summary công khai; `theme_color` (admin) tách khỏi `theme_color_suggested` (do media_factory tính khi xuất bản) để republish không đè lựa chọn của admin. SDK Flutter đọc hai trường mới vào `StickerPack`, UI mới dựng từ các widget nhỏ (mascot, sticker viền cắt, nút tải biến hình, thẻ, carousel, ngăn kéo) và vẫn đi qua `MarketplaceCubit` hiện có.

**Tech Stack:** Go (net/http, SQLite), Python (Pillow) media_factory, Next.js admin web, Flutter (`material_ui`, `flutter_bloc`, `equatable`), không thêm dependency.

**Spec:** `docs/superpowers/specs/2026-11-17-marketplace-redesign-design.md` (mockup tương tác: `.superpowers/brainstorm/80816-1794888666/content/store-motion.html`, `store-preview-cards.html`).

## Global Constraints

- **Chỉ kiểm thử bằng unit/widget/code test, không dựng local.** Không chạy mock server, backend Go, `flutter run` hay `next dev`; người dùng sẽ commit, push, migrate server trên VPS và chỉ thử client trên host đó. Các bước "chạy lại local", "quan sát bằng app" bị bỏ.
- **Không commit, không push** (người dùng tự làm). Mỗi task dừng ở trạng thái đã test xanh; các bước "Commit" bên dưới bị bỏ.

- Flutter: import `package:material_ui/material_ui.dart` (không `material.dart`), dấu nháy kép, Dart `>=3.7`, không thêm dependency (không Lottie/Rive/google_fonts/flutter_svg); dùng `very_good_analysis` hiện có.
- Giữ nguyên `ValueKey`: `marketplace.pack.download.<id>`, `marketplace.pack.remove.<id>`, `marketplace.tag`; hợp đồng `MarketplaceCubit` (`load`, `download`, `delete`) giữ nguyên.
- API tương thích ngược: client cũ bỏ qua trường mới; client mới chịu server cũ thiếu `preview_stickers`/`theme_color`.
- `theme_color` đúng `#RRGGBB` (chữ hoa/thường đều nhận, lưu dạng `#RRGGBB` chữ hoa), rỗng = chưa đặt; `preview_stickers` tối đa 6 phần tử `{thumb_url, file_url}` theo `sort_order` của phiên bản hiện tại; thumb 128px, file 512px.
- Màu thương hiệu: tím `#A98BF0`, tím nhạt `#D2C0FA`, hồng má `#F58BC0`, hồng đậm `#E6197F`, mực `#2B2433`, nền `#FFF3FA`; 5 pastel pack: `#EBE3FF`, `#FFDDEE`, `#FFE6CF`, `#D9F4E7`, `#FFF2C2`.
- Chuyển động: vào 300 đến 500ms, đường cong `Cubic(0.16, 1, 0.3, 1)`, thoát nhanh hơn; không animation nào chặn thao tác; hiệu ứng lặp chỉ chạy khi hiện; không thanh tiến độ giả (nút theo `downloadProgress` thật; "Đang chờ…" chỉ là chỗ giữ trước tiến độ đầu); `MediaQuery.disableAnimations` bỏ nảy/bập bềnh/thị sai/hạt/rung, giữ đổ màu tiến độ, đổi màu trạng thái, tick (hiện ngay, không vẽ dần); trạng thái công bố qua `Semantics`.
- Font: **Baloo 2** (OFL 1.1) đóng gói trong SDK, bốn tệp tĩnh `assets/fonts/Baloo2-500.ttf`, `-600`, `-700`, `-800` (đã thử tách bằng fontTools `instancer` từ `Baloo2[wght].ttf` trong kho `google/fonts` rồi cắt bằng `pyftsubset` với unicode `U+0020-007E,U+00A0-00FF,U+0102-0103,U+0110-0111,U+0128-0129,U+0168-0169,U+01A0-01A1,U+01AF-01B0,U+0300-0301,U+0303,U+0309,U+0323,U+1EA0-1EF9,U+2013-2014,U+2018-201D,U+2022,U+2026,U+20AB,U+2192,U+00B7` và `--layout-features='*'`; mỗi tệp ~60 KB, đủ tiếng Việt) kèm `assets/fonts/OFL.txt`; không dùng Fredoka (thiếu chữ có dấu tiếng Việt), không thêm `google_fonts`.
- Số nhiều: tiếng Anh `1 sticker`, `0 stickers`, `24 stickers`; tiếng Việt luôn `sticker`; mọi chỗ hiện số lượng đi qua `RestickStrings.stickerCount(int)`.
- Chuỗi giao diện qua `RestickStrings` (mặc định tiếng Việt `RestickStrings.vi`, có `RestickStrings.en`; quyết định của người lập kế hoạch, app chủ truyền `strings:` để đổi).
- media_factory: mọi hàm mới theo quy ước docstring (Algorithm, Techniques, Keywords, References với link đã kiểm chứng); `python services/media_factory/scripts/check_docstrings.py` phải qua.
- `PUT /api/v1/admin/packs/{id}` không được xoá màu admin khi body không có `theme_color` (nhiều chỗ gọi PUT không gửi trường này).
- Không sửa file sinh tự động; không commit file ngoài các task; thay đổi không liên quan (`git status` đang có nhiều thay đổi dở của người dùng) phải để nguyên, chỉ `git add` đúng các file của task.

## Review Focus

- PUT cập nhật pack không kèm `theme_color` (đổi sheet, form catalog cũ) → màu admin đã chọn vẫn còn (Task 1).
- Server cũ/pack không có `preview_stickers` hoặc `theme_color` hoặc chỉ có 1 đến 2 sticker → thẻ vẫn đẹp, không lỗi, rơi về `thumbnailUrl` và màu hash (Task 6, 9, 10).
- `theme_color` sai định dạng từ admin bị từ chối 422; `suggested_theme_color` sai định dạng từ processor bị bỏ qua chứ không làm hỏng việc xuất bản (Task 1, 3).
- Tải lỗi giữa chừng rồi thử lại; sự kiện tiến độ về lại 0 hoặc lặp trạng thái; bấm liên tiếp nhanh không tải trùng (Task 8).
- Chữ tiếng Việt có dấu thanh chồng (ặ ỡ ự ẳ ổ Ệ Ữ Đ) hiển thị đúng bằng font đóng gói, không rơi về font hệ thống (Task 7, kiểm thực tế ở Task 11); tên pack tiếng Việt dài có dấu, `textScaler` 2.0, màn hình hẹp 320dp không tràn bố cục (Task 9, 10).
- Sticker toàn trắng/xám/đen hoặc trong suốt → màu gợi ý rơi về `#EBE3FF`, không ngoại lệ (Task 4).

---

## File Structure

| Phần | Tạo / sửa |
|---|---|
| Go catalog | `services/api/internal/storage/migrations/0005_pack_theme_color.sql` (tạo); `packs/types.go`, `packs/catalog.go`, `packs/publication.go` (sửa) |
| Go HTTP | `interfaces/http/public.go`, `interfaces/http/admin_packs.go` (sửa) |
| Go jobs | `jobs/processor.go`, `jobs/types.go`, `jobs/runner.go` (sửa) |
| Contract/docs | `contracts/openapi/v1.yaml`, `docs/specs/2026-11-10-single-operator-sticker-factory-design.md` (sửa) |
| media_factory | `src/media_factory/infrastructure/theme_color.py` (tạo); `application/process_job.py` (sửa); `tests/test_theme_color.py` (tạo) |
| Admin web | `clients/web-next/lib/types.ts`, `lib/i18n-messages.ts`, `app/packs/detail/page.tsx`, `components/ThemeColorPicker.tsx` (tạo), `qa/mock-server.mjs` (sửa) |
| SDK data | `lib/src/domain/entities/sticker_pack.dart`, `sticker_preview.dart` (tạo), `data/dto/pack_dto.dart`, `data/repositories/pack_repository_impl.dart` (sửa) |
| SDK UI nền tảng | `assets/fonts/Baloo2-{500,600,700,800}.ttf`, `assets/fonts/OFL.txt` (tạo); `pubspec.yaml` (khai báo font); `presentation/theme/restick_brand.dart`, `restick_strings.dart`, `pack_tones.dart` (tạo); `widgets/restick_mascot.dart`, `die_cut_sticker.dart`, `pack_skeleton.dart`, `download_button.dart` (tạo) |
| SDK UI trang | `widgets/pack_hero_carousel.dart`, `pack_compact_card.dart`, `pack_detail_sheet.dart` (tạo); `sticker_marketplace.dart` (viết lại); `marketplace_pack_tile.dart` (xoá sau khi không còn dùng); `lib/restick.dart` (export) |
| Ví dụ | `example/lib/src/marketplace_page.dart`, `home_shell.dart` (chỉnh cho hợp) |

---

### Task 1: `theme_color` trên pack (DB, catalog, admin API)

**Files:**
- Create: `services/api/internal/storage/migrations/0005_pack_theme_color.sql`
- Modify: `services/api/internal/packs/types.go`, `packs/catalog.go` (`packColumns`, `scanPack`, `UpdateCatalog`, `normalizeMetadata`), `interfaces/http/admin_packs.go` (`adminPack`, `pack()`, `updatePack`)
- Test: `services/api/internal/packs/catalog_test.go` (hoặc tệp test catalog hiện có), `interfaces/http/admin_test.go`, `storage/db_test.go`

**Interfaces:**
- Produces: cột `packs.theme_color TEXT NOT NULL DEFAULT ''` và `packs.theme_color_suggested TEXT NOT NULL DEFAULT ''`, cả hai `CHECK (value = '' OR value GLOB '#[0-9A-F][0-9A-F][0-9A-F][0-9A-F][0-9A-F][0-9A-F]')`; `Metadata.ThemeColor *string` (nil = giữ nguyên, `""` = xoá); `Pack.ThemeColor`, `Pack.ThemeColorSuggested string`; `func (p Pack) EffectiveThemeColor() string` (màu admin, không có thì màu gợi ý); `ErrInvalid` cho hex sai; JSON admin `theme_color`, `theme_color_suggested`, `theme_color_effective`.

- [ ] **Step 1: Viết test thất bại** `TestUpdateCatalogThemeColor`: đặt `#ffdde` (thiếu một ký tự) → `ErrInvalid`; `#ffddee` → lưu `#FFDDEE`; `ThemeColor: nil` giữ nguyên màu cũ; `ThemeColor: ptr("")` xoá. `TestEffectiveThemeColor`: admin thắng suggested; rỗng cả hai → `""`. Test HTTP `TestAdminUpdatePackKeepsThemeColorWhenOmitted`: PUT body không có `theme_color` sau khi đã đặt → GET vẫn trả màu cũ; PUT `"theme_color": "bad"` → 422. `TestMigration0005KeepsExistingPacks` trong `storage/db_test.go`: DB đã có pack ở schema 0004 lên 0005 giữ hàng, hai cột bằng `''`.
- [ ] **Step 2: Chạy** `cd services/api && go test ./internal/packs/... ./internal/interfaces/http/... ./internal/storage/... -run 'ThemeColor|Migration0005' -v`. Dự kiến FAIL (chưa có cột/trường).
- [ ] **Step 3: Cài đặt** migration `0005_pack_theme_color.sql` (đủ hai `ALTER TABLE packs ADD COLUMN`, theo kiểu `0004`); thêm hai cột vào `packColumns` và `scanPack`; `normalizeMetadata` chuẩn hoá `ThemeColor` (trim, regex `^#[0-9a-fA-F]{6}$`, viết hoa, sai thì `fmt.Errorf("%w: theme_color must be #RRGGBB", ErrInvalid)`); `UpdateCatalog` chỉ `UPDATE theme_color` khi `Metadata.ThemeColor != nil`; `updatePack` đọc `ThemeColor *string json:"theme_color"`; `adminPack` trả ba trường màu.
- [ ] **Step 4: Chạy lại** lệnh Step 2 và `go test ./...` trong `services/api`. Dự kiến PASS.
- [ ] **Step 5: Commit** `git add` đúng các tệp trên; `git commit -m "feat(api): add admin-editable pack theme color"`.

### Task 2: `preview_stickers` và `theme_color` trong summary công khai + contract

**Files:**
- Modify: `packs/types.go` (`PreviewSticker`, `Pack.Previews`), `packs/catalog.go` (`loadPreviews`, gọi trong `ListReady` và `Get`), `interfaces/http/public.go` (`summary`), `contracts/openapi/v1.yaml`, `docs/specs/2026-11-10-single-operator-sticker-factory-design.md` dòng 201
- Test: `packs/catalog_test.go`, `interfaces/http/public_test.go`

**Interfaces:**
- Consumes: `Pack.EffectiveThemeColor()` (Task 1).
- Produces: `type PreviewSticker struct { ThumbPath, ObjectPath string }`; `Pack.Previews []PreviewSticker`; `func loadPreviews(ctx context.Context, db *sql.DB, items []Pack) error` (một truy vấn, tối đa 6 mỗi pack, `sort_order` tăng dần, chỉ phiên bản `current_version`); summary JSON thêm `"theme_color": "<effective>"` và `"preview_stickers": [{"thumb_url","file_url"}]` (luôn là mảng, rỗng thì `[]`).

- [ ] **Step 1: Viết test thất bại** `TestListReadyLoadsAtMostSixPreviewsInOrder` (pack 8 sticker → 6, đúng thứ tự `sort_order`, chỉ phiên bản hiện tại khi có v2), `TestListReadyPreviewsDoNotMultiplyQueries` (đếm truy vấn bằng wrapper hoặc kiểm bằng 3 pack → số truy vấn không tăng theo số pack), `TestPublicListPacksIncludesPreviewsAndThemeColor` (URL tuyệt đối dạng `<base>/v1/objects/...`, `theme_color` là màu hiệu lực, pack không màu → `""`), `TestPublicPackKeepsFilterByTagAndKey` (lọc tag/key không làm mất previews), `TestPublicGetPackIncludesPreviews`.
- [ ] **Step 2: Chạy** `cd services/api && go test ./internal/packs/... ./internal/interfaces/http/... -run 'Preview|ThemeColor' -v`. Dự kiến FAIL.
- [ ] **Step 3: Cài đặt** `loadPreviews` bằng một truy vấn `ROW_NUMBER() OVER (PARTITION BY s.pack_id ORDER BY s.sort_order)` ≤ 6, join `packs.current_version`, đặt kết quả vào `Pack.Previews` theo `ID` (theo kiểu `loadTags`); gọi ở `ListReady` và `Get`; `summary()` thêm hai trường qua `a.objectURL`. Cập nhật `contracts/openapi/v1.yaml` (schema summary + ví dụ) và dòng 201 của spec 2026-11-10.
- [ ] **Step 4: Chạy lại** lệnh Step 2, `go test ./...`; nếu repo có kiểm tra contract/openapi (grep `openapi` trong test) chạy kèm. Dự kiến PASS.
- [ ] **Step 5: Không commit** (người dùng tự commit); ghi danh sách tệp đã đổi vào báo cáo.

### Task 3: Nhận `suggested_theme_color` từ kết quả job

**Files:**
- Modify: `jobs/processor.go` (`machineResult`, `readSuccess`), `jobs/types.go` (`JobResult`), `jobs/runner.go` (`publish`), `packs/publication.go` (`ProcessedVersion`, `CommitProcessed`)
- Test: `jobs/processor_test.go`, `jobs/runner_test.go`, `packs/publication_test.go` (dùng tệp test hiện có của từng package)

**Interfaces:**
- Consumes: cột `theme_color_suggested` (Task 1).
- Produces: `machineResult.SuggestedThemeColor string json:"suggested_theme_color"`; `JobResult.SuggestedThemeColor string`; `ProcessedVersion.SuggestedThemeColor string`; `CommitProcessed` ghi `theme_color_suggested` trong cùng transaction; giá trị sai định dạng bị bỏ (coi như rỗng, không lỗi); không bao giờ đụng `theme_color`.

- [ ] **Step 1: Viết test thất bại** `TestReadSuccessParsesSuggestedThemeColor` (`#a98bf0` → `#A98BF0`), `TestReadSuccessIgnoresInvalidSuggestedThemeColor` (`"purple"` → `""`, job vẫn thành công), `TestCommitProcessedStoresSuggestedColorAndKeepsAdminColor` (đặt `theme_color` admin trước, commit v2 với màu gợi ý khác → `theme_color` giữ nguyên, `theme_color_suggested` mới, `EffectiveThemeColor` là màu admin), `TestCommitProcessedClearsStaleSuggestion` (commit mới không có màu → `theme_color_suggested` rỗng).
- [ ] **Step 2: Chạy** `cd services/api && go test ./internal/jobs/... ./internal/packs/... -run 'SuggestedThemeColor|SuggestedColor' -v`. Dự kiến FAIL.
- [ ] **Step 3: Cài đặt** đúng chuỗi truyền trường qua bốn struct; chuẩn hoá hex bằng hàm dùng chung trong package `packs` (xuất `NormalizeThemeColor(string) (string, bool)` để `jobs` gọi, và `normalizeMetadata` dùng lại); `CommitProcessed` thêm `theme_color_suggested = ?` vào câu `UPDATE packs` đang có.
- [ ] **Step 4: Chạy lại** Step 2 và `go test ./...`. Dự kiến PASS.
- [ ] **Step 5: Không commit** (người dùng tự commit); ghi danh sách tệp đã đổi vào báo cáo.

### Task 4: media_factory tính màu gợi ý

**Files:**
- Create: `services/media_factory/src/media_factory/infrastructure/theme_color.py`, `services/media_factory/tests/test_theme_color.py`
- Modify: `services/media_factory/src/media_factory/application/process_job.py` (`_render`, `process_job`)

**Interfaces:**
- Produces: `def suggest_theme_color(images: list[Image.Image]) -> str` trong `theme_color.py` (trả `#RRGGBB` chữ hoa, luôn hợp lệ; ảnh đầu vào RGBA); hằng `DEFAULT_THEME_COLOR = "#EBE3FF"`; `_render` trả `tuple[list[dict], dict, str]` (thêm màu gợi ý) và `process_job` ghi `"suggested_theme_color"` cấp cao trong `result.json` (cạnh `stickers`, `diagnostics`).

- [ ] **Step 1: Viết test thất bại** trong `test_theme_color.py`: `test_dominant_purple_gives_light_purple` (ảnh RGBA hình tròn tím `#A98BF0` trên nền trong suốt → màu kết quả có hue trong ±15° của tím, độ sáng HSL ≥ 0.85, hợp lệ `^#[0-9A-F]{6}$`); `test_orange_cat_gives_light_orange`; `test_all_gray_returns_default`; `test_white_and_black_only_returns_default`; `test_fully_transparent_returns_default`; `test_empty_list_returns_default`; `test_white_outline_is_ignored` (sticker đỏ viền trắng dày → màu hồng/đỏ nhạt, không phải trắng); `test_contrast_keeps_sticker_visible` (kết quả có độ sáng ≤ 0.95 để viền trắng của sticker vẫn tách khỏi nền). Thêm vào `test_machine_job.py`: `test_result_contains_suggested_theme_color` (kết quả job thành công có trường hợp lệ).
- [ ] **Step 2: Chạy** `cd services/media_factory && python -m pytest tests/test_theme_color.py -v`. Dự kiến FAIL (module chưa có).
- [ ] **Step 3: Cài đặt** `suggest_theme_color(images)`: thu nhỏ mỗi ảnh về tối đa 64px; chỉ lấy pixel alpha ≥ 200; bỏ pixel có bão hoà HSV < 0.2, hoặc giá trị V > 0.92 (trắng/viền), hoặc V < 0.2 (mực); gom theo hue vào 36 thùng, trọng số = bão hoà x diện tích; chọn thùng lớn nhất, lấy hue trung bình có trọng số; trả pastel HSL với L = 0.90, S = 0.85 (S tối đa 0.9); nếu không còn pixel nào hoặc tổng trọng số dưới 1% diện tích thì `DEFAULT_THEME_COLOR`. Trong `_render` thu thập bản thu nhỏ của mỗi `canvas` (không giữ ảnh 512px) rồi gọi hàm; docstring đủ Algorithm/Techniques/Keywords/References (link Wikipedia HSL and HSV, Color quantization đã kiểm chứng bằng `scripts/check_links` nếu có).
- [ ] **Step 4: Chạy lại** `python -m pytest tests -v` trong `services/media_factory` và `python services/media_factory/scripts/check_docstrings.py`. Dự kiến PASS.
- [ ] **Step 5: Không commit** (người dùng tự commit); ghi danh sách tệp đã đổi vào báo cáo.

### Task 5: Admin web: chọn màu, hiện gợi ý, mock server

**Files:**
- Create: `clients/web-next/components/ThemeColorPicker.tsx`
- Modify: `clients/web-next/lib/types.ts` (`AdminPack`, `PackUpdate`), `lib/i18n-messages.ts` (vi và en), `app/packs/detail/page.tsx` (`CatalogForm`), `clients/web-next/qa/mock-server.mjs` (pack dựng sẵn và `PUT`)

**Interfaces:**
- Consumes: JSON admin `theme_color`, `theme_color_suggested`, `theme_color_effective` (Task 1).
- Produces: `ThemeColorPicker({ value, suggested, previews, onChange })` (value `""` = dùng gợi ý); `PackUpdate.theme_color?: string` (bỏ qua = giữ nguyên).

- [ ] **Step 1:** Đọc `node_modules/next/dist/docs/` mục liên quan trước khi viết (AGENTS.md của web-next yêu cầu). Cập nhật `types.ts`: `AdminPack` thêm `theme_color`, `theme_color_suggested`, `theme_color_effective: string`; `PackUpdate` thêm `theme_color?: string`.
- [ ] **Step 2:** Cài `ThemeColorPicker`: hàng 5 ô pastel preset + ô nhập `#RRGGBB` (kiểm định dạng phía client, nút "Dùng gợi ý" đặt `""`), nhãn nêu rõ màu gợi ý hiện tại và màu đang áp dụng, khung xem trước nền chứa tối đa 6 sticker đầu của pack. Thêm khoá i18n vi và en (`themeColor`, `themeColorSuggested`, `themeColorUseSuggestion`, `themeColorInvalid`).
- [ ] **Step 3:** `CatalogForm` thêm state `themeColor` (khởi tạo từ `pack.theme_color`), gửi `theme_color` trong `PUT`; `changeSheet` (dòng ~285) **không** gửi `theme_color` (giữ nguyên nhờ pointer ở server). `mock-server.mjs`: pack dựng sẵn có `theme_color`, `theme_color_suggested`, `theme_color_effective` và `preview_stickers` trong các route công khai `/api/v1/packs`, `PUT` chấp nhận `theme_color` (sai định dạng → 422) và giữ nguyên khi vắng.
- [ ] **Step 4: Kiểm chứng** `cd clients/web-next && npx next build && npm run lint`. Dự kiến không lỗi (không chạy mock server).
- [ ] **Step 5: Không commit** (người dùng tự commit); ghi danh sách tệp đã đổi vào báo cáo.

### Task 6: SDK: dữ liệu `previews` và `themeColor`

**Files:**
- Create: `lib/src/domain/entities/sticker_preview.dart`
- Modify: `domain/entities/sticker_pack.dart`, `data/dto/pack_dto.dart` (`PackSummaryDto`), `data/repositories/pack_repository_impl.dart` (`_merge`, `_fromRow`)
- Test: `test/pack_dto_test.dart` (tạo), `test/marketplace_cubit_test.dart` (giữ xanh)

**Interfaces:**
- Produces: `class StickerPreview extends Equatable { const StickerPreview({required String thumbUrl, required String fileUrl}); }`; `StickerPack` thêm `final List<StickerPreview> previews` (mặc định `const []`) và `final String themeColor` (hex `#RRGGBB` hoặc `""`, mặc định `""`), cả hai trong `props` và `copyWith`; `PackSummaryDto.fromJson` đọc `preview_stickers` (thiếu/null → `[]`, phần tử hỏng bị bỏ) và `theme_color` (thiếu/null → `""`); URL qua `rewritePublicUrl`.

- [ ] **Step 1: Viết test thất bại** `test/pack_dto_test.dart`: `parses preview_stickers and theme_color`; `server without new fields yields empty previews and empty color`; `malformed preview entries are skipped`; `preview urls are rewritten against apiBase`; `copyWith keeps previews and themeColor`; `merge keeps remote previews for a locally downloaded pack` (kiểm qua `PackRepositoryImpl.cachedMarketplace`/hàm merge hiện có theo cách `sticker_assets_test.dart` đang dựng repository).
- [ ] **Step 2: Chạy** `cd clients/flutter/restick && flutter test test/pack_dto_test.dart` (hoặc Dart MCP `run_tests`). Dự kiến FAIL.
- [ ] **Step 3: Cài đặt** như phần Interfaces; `_merge` giữ `previews` và `themeColor` từ pack remote khi trộn với bản local; cache drift không đổi.
- [ ] **Step 4: Chạy** `flutter test` toàn bộ và `flutter analyze` (hoặc `analyze_files`). Dự kiến PASS, không cảnh báo mới.
- [ ] **Step 5: Không commit** (người dùng tự commit); ghi danh sách tệp đã đổi vào báo cáo.

### Task 7: SDK: nền tảng giao diện (thương hiệu, chuỗi, màu pack, gấu, sticker viền, khung chờ)

**Files:**
- Create: `lib/src/presentation/theme/restick_brand.dart`, `restick_strings.dart`, `pack_tones.dart`; `widgets/restick_mascot.dart`, `die_cut_sticker.dart`, `pack_skeleton.dart`
- Modify: `theme/restick_theme.dart` (thêm token thương hiệu vào `RestickTheme`, vẫn ghi đè được)
- Test: `test/pack_tones_test.dart`, `test/restick_widgets_test.dart`

**Interfaces:**
- Produces: `abstract final class RestickBrand` với `static const String fontFamily = "Baloo2"`, `static const String fontPackage = "restick"`, `static TextStyle text({double size, FontWeight weight = FontWeight.w600, Color? color, double? height})` (luôn gắn `fontFamily` + `package`) và các `Color` (tím, tím nhạt, hồng má, hồng đậm, mực, nền, và `static const List<Color> packPastels` 5 màu); `class RestickStrings` (const, các trường: `title`, `tagline`, `download`, `downloading`, `waiting`, `retry`, `owned`, `remove`, `newShelf`, `featured`, `searchHint`, `bearGreeting`, `emptyTitle`, `errorTitle`, `stickerCount(int count)` (en: `count == 1 ? "1 sticker" : "$count stickers"`, vi: `"$count sticker"`), `downloadAllLabel(int count)` và `packMeta(int count, String author)` đều gọi `stickerCount`, `a11yDownloading(String name, int percent)`, `a11yDone(String name)`, `a11yFailed(String name)`; `static const RestickStrings vi`, `en`); `@immutable class PackTones { final Color background; final Color accent; }` và `PackTones.of(StickerPack pack)` (dùng `pack.themeColor` nếu hợp lệ, không thì `RestickBrand.packPastels[stableHash(pack.id) % 5]`, hash ổn định giữa các lần chạy, không dùng `hashCode`); `RestickMascot({double size, bool blink = true, RestickMood mood = RestickMood.happy})` (CustomPainter vẽ lại SVG `clients/web-next/public/mascot.svg`, `RestickMood { happy, cheer, sad }`); `DieCutSticker({required String url, required double size, double rotation = 0})` vẽ ảnh bằng cùng cơ chế header `X-Api-Key` và `cacheWidth/Height` như `PackThumbnail` (tái dùng, không nhân đôi logic) + viền trắng bằng bóng offset 4 hướng; `PackSkeleton` (khối shimmer theo màu pastel).

- [ ] **Step 1: Viết test thất bại** `restick_strings_test.dart`: `en pluralizes sticker count` (`0 stickers`, `1 sticker`, `2 stickers`, `24 stickers`), `vi never inflects` (`1 sticker`, `24 sticker`), `packMeta and download label use the same plural rule`. `restick_font_test.dart`: `Baloo2 weights 500 600 700 800 load from the package bundle` (`rootBundle.load('packages/restick/assets/fonts/Baloo2-700.ttf')` không ném lỗi, kích thước dưới 100 KB), `RestickBrand.text uses the SDK font family and package`, `Vietnamese sample renders with no missing glyph fallback in a widget test` (dựng `Text('Mèo Cam Ngố Thỏ Bông ặ ỡ ự ẳ Đ đ')` bằng `RestickBrand.text` ở 4 độ đậm, không ném lỗi). `pack_tones_test.dart`: `uses server theme color when valid`; `falls back to stable pastel by id` (cùng id luôn cùng màu, id khác có thể khác, kết quả thuộc `packPastels`); `rejects malformed theme color and falls back`; `lowercase hex parses`. `restick_widgets_test.dart`: `RestickMascot paints without error at size 36 and 96`; `RestickMascot does not schedule frames when disableAnimations` (không tick blink); `DieCutSticker shows placeholder when url empty or load fails`; `PackSkeleton stops animating when TickerMode disabled`.
- [ ] **Step 2: Chạy** `flutter test test/pack_tones_test.dart test/restick_widgets_test.dart`. Dự kiến FAIL.
- [ ] **Step 3: Cài đặt** theo Interfaces; tạo bốn tệp font bằng đúng quy trình ở Global Constraints (tải `Baloo2[wght].ttf` và `OFL.txt` từ `https://raw.githubusercontent.com/google/fonts/main/ofl/baloo2/`, dùng một môi trường ảo tạm cho fontTools, không cài vào repo) và kiểm lại cmap đủ ký tự Việt trước khi commit; khai báo trong `pubspec.yaml`: `fonts: - family: Baloo2` với bốn `asset:` kèm `weight: 500/600/700/800` và liệt kê `assets/fonts/OFL.txt` trong `flutter: assets:`; hằng thời gian/đường cong chuyển động đặt chung trong `restick_brand.dart` (`RestickMotion.ease = Cubic(0.16, 1, 0.3, 1)`, `fast = 150ms`, `routine = 300ms`, `enter = 450ms`); mascot vẽ đúng hình trong `mascot.svg`; tôn trọng `MediaQuery.disableAnimationsOf` và `TickerMode`; bọc phần động trong `RepaintBoundary`.
- [ ] **Step 4: Chạy** `flutter test` và `flutter analyze`. Dự kiến PASS.
- [ ] **Step 5: Không commit** (người dùng tự commit); ghi danh sách tệp đã đổi vào báo cáo.

### Task 8: SDK: nút tải biến hình (tiến độ, thành công, thất bại)

**Files:**
- Create: `lib/src/presentation/widgets/download_button.dart`
- Test: `test/download_button_test.dart`

**Interfaces:**
- Consumes: `PackDownloadStatus`, `RestickStrings`, `RestickMotion` (Task 7).
- Produces: `class PackDownloadButton extends StatefulWidget { const PackDownloadButton({required StickerPack pack, required VoidCallback onDownload, required VoidCallback onRemove, RestickStrings strings = RestickStrings.vi, bool compact = false, super.key}); }`. Nội bộ: giữ `ValueKey('marketplace.pack.download.${pack.id}')` khi hành động là tải/thử lại và `ValueKey('marketplace.pack.remove.${pack.id}')` khi pack đã có và nút là "Xoá" (dùng ở ngăn kéo, Task 10; ở thẻ, trạng thái đã có hiển thị "Đã có" không bấm được). Callback `onDownloadCompleted` không cần; thành công suy ra từ chuyển `downloading → downloaded` của `pack` giữa hai lần dựng.

- [ ] **Step 1: Viết test thất bại** `download_button_test.dart` (widget test, dựng bằng `StickerPack` thủ công, đổi `pack` qua `pumpWidget`): `remote shows download label and fires onDownload once`; `rapid double tap fires onDownload once until status changes` (khoá ngắn sau bấm, hết khoá khi status đổi); `tapping shows waiting state within 100ms before any progress`; `progress fill follows downloadProgress and shows percent`; `downloading to downloaded plays success then settles on owned` (sau `pumpAndSettle` hiện nhãn "Đã có", dấu tick có mặt); `failed shows retry label and tap fires onDownload`; `failed after downloading shakes once` (dùng `tester.pump` theo bước kiểm `Transform` rồi về 0); `progress dropping back to 0 after failure does not show waiting spinner forever` (failed → remote → bấm lại hoạt động); `disableAnimations skips shake and burst but still shows state colors and tick`; `semantics announce downloading percent, done and failed` (kiểm `Semantics` label từ `RestickStrings`); `haptics called on tap, success, failure` (mock `SystemChannels.platform`).
- [ ] **Step 2: Chạy** `flutter test test/download_button_test.dart`. Dự kiến FAIL.
- [ ] **Step 3: Cài đặt** `PackDownloadButton`: `AnimatedSwitcher` giữa các trạng thái với kích thước nút giữ ổn định (không nhảy layout); lớp đổ màu hồng bằng `Transform.scale` ngang theo `downloadProgress` (`Alignment.centerLeft`) kèm vệt sáng chạy qua khi `downloading`; "Đang chờ…" từ lúc bấm tới khi nhận trạng thái `downloading` đầu tiên; tick vẽ bằng `CustomPainter` (dùng `PathMetric.extractPath` theo giá trị animation; `disableAnimations` thì vẽ đủ ngay); vòng sáng lan bằng `TweenAnimationBuilder` quanh nút; chùm 8 hạt tim/sao/lấp lánh bằng một `CustomPainter` một lần trong `OverlayPortal` hoặc `Stack` không cắt (một `AnimationController` tạo khi chuyển sang thành công, huỷ sau khi xong); rung ngang 450ms bằng `Transform.translate`; haptics qua `HapticFeedback.selectionClick/lightImpact/mediumImpact`; `Semantics(liveRegion: true, label: ...)`.
- [ ] **Step 4: Chạy** `flutter test` và `flutter analyze`. Dự kiến PASS. Kích thước nút ổn định giữa các trạng thái được khẳng định bằng widget test (so `tester.getSize` qua các trạng thái).
- [ ] **Step 5: Không commit** (người dùng tự commit); ghi danh sách tệp đã đổi vào báo cáo.

### Task 9: SDK: trang chủ cửa hàng B (carousel, thẻ gọn, tìm kiếm, khung chờ)

**Files:**
- Create: `widgets/pack_hero_carousel.dart`, `widgets/pack_compact_card.dart`
- Modify: `widgets/sticker_marketplace.dart` (viết lại `_MarketplaceView`; giữ `StickerMarketplace({client, tag, key})`, thêm `strings`), `lib/restick.dart` (export widget và `RestickStrings`, `RestickBrand`, `RestickMascot`), xoá `widgets/marketplace_pack_tile.dart` khi không còn tham chiếu (kiểm `rg MarketplacePackTile`)
- Test: `test/sticker_marketplace_test.dart`

**Interfaces:**
- Consumes: `PackTones.of`, `DieCutSticker`, `RestickMascot`, `PackSkeleton`, `PackDownloadButton`, `PackThumbnail` (Task 7, 8); `MarketplaceCubit`/`MarketplaceState` không đổi.
- Produces: `PackHeroCarousel({required List<StickerPack> packs, required void Function(StickerPack) onOpen, required void Function(String) onDownload, RestickStrings strings})` (`PageView` với `viewportFraction` ~0.78, tối đa 3 thẻ đầu, thị sai sticker lớn theo `page`, chấm trang giãn ở trang hiện tại; sticker lớn dùng `previews.first.fileUrl`, rơi về `thumbnailUrl`; bập bềnh chỉ chạy ở trang hiện tại và dừng khi `TickerMode` tắt); `PackCompactCard({required StickerPack pack, required VoidCallback onOpen, required VoidCallback onDownload, RestickStrings strings, int entranceIndex = 0})` (dải 3 sticker thumb; xám đi khi `downloading` rồi hiện lại màu theo ngưỡng `i / n <= downloadProgress`, nảy một nhịp; hiện màu đủ khi chưa tải hoặc đã có; pack không có `previews` thì dùng `thumbnailUrl` một sticker; entrance trồi lên stagger tổng ≤ 500ms, chỉ lần đầu cho mỗi `id`); Tìm kiếm: biểu tượng ở header mở ô nhập, lọc phía client theo tên/tác giả/tag trên `state.packs`; chip lọc tag lấy từ tập tag của `state.packs`, giữ `ValueKey('marketplace.tag')` cho ô lọc tag đang có ở example.

- [ ] **Step 1: Viết test thất bại** `sticker_marketplace_test.dart` (widget test với `RestickClient` giả như `marketplace_cubit_test.dart` đã dựng): `shows skeleton while loading and no spinner`; `empty list shows empty state with mascot`; `failure with no packs shows error state and retry`; `renders carousel for first three packs and compact cards for the rest`; `search filters by name author and tag with Vietnamese diacritics` (khớp không phân biệt hoa thường; "meo" không khớp "Mèo" trừ khi chuẩn hoá dấu — cài chuẩn hoá bỏ dấu thủ công cho Latin Vietnamese); `pack without previews falls back to thumbnail and still renders`; `long Vietnamese title with textScaler 2.0 at 320dp has no overflow` (không `RenderFlex overflowed`); `compact card strip greys while downloading and recolors by progress thresholds`; `idle motion stops when TickerMode disabled`; `pull to refresh calls MarketplaceCubit.load`; `download key and remove key still present` (kiểm `find.byKey`); `en strings show 1 sticker and 24 stickers` trên thẻ.
- [ ] **Step 2: Chạy** `flutter test test/sticker_marketplace_test.dart`. Dự kiến FAIL.
- [ ] **Step 3: Cài đặt** theo Interfaces. Header: `RestickMascot(size: 38)` + chữ "Restick" (chữ "stick" màu hồng đậm) + biểu tượng tìm kiếm; thân: `CustomScrollView` (slivers: carousel, tiêu đề "Vừa lên kệ", danh sách thẻ gọn) bọc `RefreshIndicator`; `RepaintBoundary` quanh từng vùng động; không `AnimationController` thường trực trên mỗi thẻ. Giữ nền `RestickBrand.background`. Xoá `marketplace_pack_tile.dart` và export cũ nếu có.
- [ ] **Step 4: Chạy** `flutter test`, `flutter analyze`. Dự kiến PASS. Không chạy app; hiệu năng cuộn kiểm bằng cách giữ cấu trúc nhẹ (RepaintBoundary, không controller thường trực) và test widget.
- [ ] **Step 5: Không commit** (người dùng tự commit); ghi danh sách tệp đã đổi vào báo cáo.

### Task 10: SDK: ngăn kéo chi tiết D

**Files:**
- Create: `widgets/pack_detail_sheet.dart`
- Modify: `widgets/sticker_marketplace.dart` (mở ngăn kéo khi bấm thẻ), `lib/restick.dart`
- Test: `test/pack_detail_sheet_test.dart`

**Interfaces:**
- Consumes: `PackDownloadButton`, `DieCutSticker`, `RestickMascot`, `PackTones` (Task 7, 8).
- Produces: `Future<void> showPackDetailSheet(BuildContext context, {required MarketplaceCubit cubit, required String packId, RestickStrings strings})` (`showModalBottomSheet`, `isScrollControlled`, dựng lại theo `cubit.stream` để thấy tiến độ thật; trình bày: tên, `packMeta`, lưới tối đa 6 sticker trong khung `PackTones.background`, tag, nút `PackDownloadButton` đủ rộng "Tải về · N sticker", gấu vắt mép nảy lên; "Đã có" mở được ngăn kéo và nút Xoá giữ `marketplace.pack.remove.<id>`; sticker trong lưới lắc nhẹ khi chạm; nếu `previews` rỗng dùng `thumbnailUrl` một ô lớn).

- [ ] **Step 1: Viết test thất bại** `pack_detail_sheet_test.dart`: `opens from a card tap and shows name count and tags`; `grid shows up to six previews and one big thumbnail when none`; `download from sheet updates progress live then shows owned with remove button`; `remove button asks cubit.delete and returns to download state`; `sheet closes cleanly while a download continues and card still shows progress`; `idle motion on cards pauses while sheet is open` (card `TickerMode` tắt); `no overflow at textScaler 2.0 and 320dp`; `disableAnimations shows content immediately`; `en strings show Download · 1 sticker and Download · 24 stickers`.
- [ ] **Step 2: Chạy** `flutter test test/pack_detail_sheet_test.dart`. Dự kiến FAIL.
- [ ] **Step 3: Cài đặt** theo Interfaces; trượt lên 350 đến 450ms với `RestickMotion.ease` (mặc định của `showModalBottomSheet` được tuỳ biến qua `sheetAnimationStyle`), lưới nảy lần lượt tổng ≤ 300ms; dùng lại cùng `PackDownloadButton` để hành vi giống thẻ.
- [ ] **Step 4: Chạy** `flutter test`, `flutter analyze`. Dự kiến PASS.
- [ ] **Step 5: Không commit** (người dùng tự commit); ghi danh sách tệp đã đổi vào báo cáo.

### Task 11: Ví dụ, kiểm chứng toàn bộ và chạy lại local

**Files:**
- Modify: `clients/flutter/restick/example/lib/src/marketplace_page.dart`, `home_shell.dart` (chỉnh cho hợp thiết kế, giữ ô lọc tag `marketplace.tag`), `clients/flutter/restick/CHANGELOG.md`, `clients/flutter/restick/README.md` (mô tả `strings:` và trường server mới)

- [ ] **Step 1:** Cập nhật example để bọc `StickerMarketplace` bằng nền thương hiệu, bỏ thanh tab tự tạo nếu không cần; thêm mục CHANGELOG (trường server `preview_stickers`, `theme_color`; `RestickStrings`, thay đổi giao diện).
- [ ] **Step 2: Kiểm chứng toàn bộ.** `cd services/api && go test ./... && go vet ./...`; `cd services/media_factory && python -m pytest tests -v && python scripts/check_docstrings.py`; `cd clients/web-next && npx next build && npm run lint`; `cd clients/flutter/restick && flutter analyze && flutter test`. Mọi lệnh phải xanh; ghi lại kết quả thật.
- [ ] **Step 3: (bỏ)** Không dựng local; người dùng sẽ thử client trên host VPS sau khi tự commit, push và migrate.
- [ ] **Step 4: (bỏ)** Quan sát bằng mắt do người dùng làm trên host VPS; bước này chỉ còn các test: font tải được từ bundle (Task 7) và số nhiều (`restick_strings_test.dart`).
- [ ] **Step 5: Không commit** (người dùng tự commit); ghi danh sách tệp đã đổi vào báo cáo.
