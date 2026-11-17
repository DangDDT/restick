# Redesign trang Marketplace của Flutter SDK (Restick)

Ngày: 2026-11-17. Trạng thái: chờ duyệt.

## Ý định và lý do

`StickerMarketplace` hiện là một `ListView` + `ListTile` phẳng, không có thương hiệu, không cho người dùng thấy sticker thật trước khi tải. Chủ dự án muốn trang cửa hàng **đẹp, nhiều màu, mang tên Restick và chú gấu tím làm nhân vật đại diện**, theo giao diện sáng và hồng của web Restick.

Mockup đã được chủ dự án chọn: **B (thẻ lớn vuốt ngang + danh sách "Vừa lên kệ") kết hợp D (bấm thẻ thì ngăn kéo chi tiết trượt lên)**. Mockup tham chiếu nằm ở `.superpowers/brainstorm/80816-1794888666/content/store-preview-cards.html` (cột B và D) và `store-layouts.html` (bố cục thanh tab).

## Mục tiêu

- Trang chủ cửa hàng theo mockup B: đầu trang có gấu + chữ "Restick" + tìm kiếm; carousel thẻ lớn (mỗi thẻ một sticker khổng lồ, thẻ kế bên ló ra, chấm trang); danh sách "Vừa lên kệ" gồm thẻ gọn có dải 3 sticker xem trước và nút Tải/Đã có/đang tải.
- Ngăn kéo chi tiết theo mockup D: lưới 6 sticker, tag, nút "Tải về · N sticker", gấu ngồi vắt trên mép ngăn kéo.
- Sticker xem trước là **ảnh thật của pack**, có viền cắt trắng như sticker dán, xếp hơi lệch.
- Mỗi pack có **màu nền riêng do server quy định** (admin chọn cho hợp sticker). Pack chưa có màu thì client rơi về bảng 5 pastel của mockup, chọn theo hash `id` (cùng pack luôn cùng màu).
- Chuyển động phong phú nhưng có kỷ luật, đặc biệt cho tải/thành công/thất bại (xem mục Chuyển động).
- Đủ trạng thái: chưa tải, đang tải (nút tự đổ màu theo tiến độ), đã có, lỗi, rỗng, đang tải danh sách.
- Chip lọc tag giữ chức năng `tag` hiện có của `StickerMarketplace`.

## Phi mục tiêu

- Không có trang Cài đặt. Thanh dưới trong mockup chỉ có "Cửa hàng" và "Của tôi" để minh hoạ; SDK là widget nhúng, app chủ quyết định điều hướng, nên **thanh tab không thuộc SDK** (xem "Để builder quyết").
- Không thêm lượt tải, đánh giá, giá, xếp hạng (server chưa có dữ liệu này).
- Không làm theme tối.
- Không đổi luồng tải/xoá pack, cache, thu hồi (tombstone).

## Chống mục tiêu (thất bại dù chạy được)

- Nhìn vẫn như danh sách phẳng, hoặc sticker xếp thành ô vuông đều tăm tắp.
- Dùng emoji làm icon, gradient trên chữ, bóng cứng không blur, thẻ lồng thẻ (nguyên tắc từ skill `impeccable` của repo).
- Ảnh xem trước nhảy loạn hoặc làm giật khi cuộn.
- Tải đủ N ảnh sticker đầy đủ (file_url) để làm xem trước thay vì thumb nhỏ.

## Ràng buộc

- Flutter SDK dùng `material_ui` (không phải `material`), `flutter_bloc`, `equatable`, Dart `>=3.7`; theo quy ước hiện có (import `package:material_ui/material_ui.dart`, double quote, `ValueKey` selector đặt tên `marketplace.pack.download.<id>`, `marketplace.pack.remove.<id>`, `marketplace.tag` đã được test/ví dụ dùng, phải giữ nguyên).
- Server Go: `GET /api/v1/packs` hiện trả `summary()` (id, slug, name, author, tags, sticker_count, version, thumbnail_url, checksum_sha256). Danh sách chạy qua `packs.Service.ListReady` (`services/api/internal/packs/catalog.go`), chi tiết `GET /api/v1/packs/{id}` đã có `stickers[].thumb_url`.
- Hợp đồng cũ phải tương thích ngược: client cũ bỏ qua trường mới; client mới chịu được server cũ thiếu trường (`preview_stickers` rỗng thì rơi về `thumbnailUrl`, `theme_color` rỗng thì rơi về màu theo hash `id`).
- Ảnh được tải bằng header `X-Api-Key` (xem `PackThumbnail`), `cacheWidth/Height` theo kích thước hiển thị.
- Không sửa file sinh tự động; không thêm dependency nếu không cần (font xem mục Font chữ).

## Các hướng đã cân nhắc (nguồn ảnh xem trước)

1. Chỉ client, gọi endpoint chi tiết cho từng thẻ hiện ra: không đổi server nhưng thêm request khi cuộn, thẻ phải chờ.
2. **Chọn: server trả thêm `preview_thumb_urls` trong danh sách.** Nhanh nhất, không nhảy ảnh, một request cho cả trang; chủ dự án chọn dù phạm vi lớn hơn.
3. Chỉ dùng `thumbnailUrl`: rẻ nhất nhưng mất dải xem trước, ngăn kéo nghèo nàn.

## Thiết kế (các phần đã quyết)

**Server (Go)**
- Thêm `preview_stickers` vào `summary()` ở `public.go`: mảng tối đa 6 phần tử `{thumb_url, file_url}` (URL tuyệt đối, theo thứ tự sticker của phiên bản hiện tại). Thumb chỉ 128px (`THUMB_SIZE` trong media_factory) nên đủ cho dải xem trước và lưới ngăn kéo, nhưng mờ ở thẻ lớn; thẻ lớn dùng `file_url` (WebP 512px) của sticker đầu, tải trễ chỉ cho các thẻ carousel. Bản chi tiết `GET /api/v1/packs/{id}` dùng chung `summary()` nên cũng có trường này.
- **Màu nền pack do server quy định:** cột `theme_color` (nullable, `#RRGGBB`) trên `packs` qua migration mới `0005_pack_theme_color.sql`; `Metadata`/`normalizeMetadata` kiểm tra đúng định dạng hex (rỗng = chưa đặt); `UpdateCatalog` và JSON admin (`admin_packs.go`) đọc/ghi; `summary()` trả `theme_color` (chuỗi rỗng nếu chưa đặt). Admin web (`clients/web-next`, form sửa pack) có bộ chọn màu: 5 màu pastel preset của thiết kế + ô nhập màu tuỳ ý, kèm xem trước nền với sticker của pack.
- **Màu gợi ý tự động (chủ dự án chọn):** `theme_color` là màu admin đặt tay; thêm cột riêng `theme_color_suggested` do pipeline ghi mỗi lần xuất bản/commit phiên bản mới. Màu hiệu lực trả ra client = `theme_color` nếu admin đã đặt, ngược lại `theme_color_suggested`, ngược lại rỗng. Nhờ tách hai cột, republish không bao giờ ghi đè màu admin đã chọn; admin có nút "Dùng gợi ý" để xoá màu tay. Admin web hiển thị cả màu gợi ý lẫn màu đang áp dụng.
- **media_factory:** thêm bước tính màu gợi ý cho pack, trả trong `result.json` ở trường cấp cao `suggested_theme_color` (`#RRGGBB`) (hiện có `stickers` và `diagnostics`). Thuật toán đề xuất: gom các pixel chromatic (alpha cao, bỏ trắng viền cắt, xám, và mực gần đen) của tất cả sticker, lượng tử hoá theo hue, chọn hue có tổng trọng số (độ bão hoà x diện tích) lớn nhất, rồi đổi sang pastel giữ hue, độ sáng cao (L ~ 0.9) và bão hoà vừa phải; nếu không có pixel chromatic thì trả lavender `#EBE3FF`. Phải bảo đảm tương phản đủ để sticker (có viền trắng) vẫn nổi trên nền. Hàm mới theo quy ước docstring của media_factory (thuật toán, từ khoá, link tham khảo đã kiểm chứng) và có test Python (ví dụ: sticker chủ đạo tím cho ra nền tím nhạt, toàn xám cho ra lavender mặc định).
- Go: `jobs` đọc `suggested_theme_color` từ kết quả job, kiểm tra hex, ghi vào `theme_color_suggested` trong cùng transaction commit phiên bản. Pack đã xuất bản trước migration chưa có màu gợi ý (rỗng) cho tới lần xuất bản kế; có thể để builder thêm lệnh backfill nếu đơn giản.
- Tránh N+1: nạp thumb/file path của nhiều pack bằng một truy vấn (kiểu `loadTags` trong `ListReady`), giới hạn 6 mỗi pack. Chọn cách tương tự `loadTags`.
- Pack ready luôn có sticker; trường rỗng thì trả `[]`.
- Cập nhật tài liệu API (`interfaces/http/docs*`) và spec liên quan tới `summary()` (`docs/specs/2026-11-10-single-operator-sticker-factory-design.md` dòng 201) cho khớp.
- Test Go: danh sách trả tối đa 6, đúng thứ tự, URL tuyệt đối, pack ready không bị mất khi lọc tag/key; `theme_color` hợp lệ lưu và trả đúng, giá trị sai (không phải `#RRGGBB`) bị từ chối, rỗng được chấp nhận; migration chạy trên DB đã có dữ liệu.

**SDK (Dart)**
- `PackSummaryDto` đọc `preview_stickers` (mặc định `[]` nếu thiếu) và `theme_color` (mặc định rỗng); `StickerPack` thêm `previews` (danh sách `{thumbUrl, fileUrl}`) và `themeColor` (Color? hoặc chuỗi hex) vào `props` và `copyWith`, URL được `rewritePublicUrl` như `thumbnailUrl`.
- Cache cục bộ (drift) của pack đã tải: không cần lưu `previews`/`themeColor` (chỉ dùng cho pack remote); `_merge` giữ chúng từ remote.
- `MarketplaceCubit` giữ nguyên hợp đồng (`load`, `download`, `delete`, tiến độ tải).

**UI (Flutter)**
- Viết lại `sticker_marketplace.dart` theo B; thay `MarketplacePackTile` bằng thẻ gọn mới; thêm thẻ lớn carousel (`PageView` với `viewportFraction` để thẻ kế bên ló ra) cho vài pack đầu danh sách; danh sách "Vừa lên kệ" phần còn lại.
- Ngăn kéo D mở bằng `showModalBottomSheet` khi bấm thẻ, lưới 6 sticker từ `previews`, nút tải dùng chính `MarketplaceCubit.download`.
- Thành phần dùng lại: `PackThumbnail` (ảnh + header API key + cache size), `PackDownloadBanner`/`PackDownloadStatus`, `RestickEmptyState`.
- Chú gấu: nhúng SVG từ `clients/web-next/public/mascot.svg` dưới dạng widget vẽ tay (`CustomPainter` hoặc `flutter_svg`; quyết định ở "Để builder quyết").
- Viền cắt trắng quanh sticker: tạo bằng nhiều bóng/offset trắng quanh ảnh WebP trong suốt (không cần ảnh viền sẵn).
- `RestickTheme` mở rộng bảng màu thương hiệu (tím `#A98BF0`, tím nhạt `#D2C0FA`, hồng má `#F58BC0`, hồng đậm `#E6197F`, mực `#2B2433`, nền `#FFF3FA`) + 5 màu pastel pack; app chủ vẫn ghi đè được.
- Tiếng Việt cho chuỗi trong trang cửa hàng ("Vừa lên kệ", "Tải", "Đã có", "Đang tải", "Tải về · N sticker", "Hôm nay dán gì nào?"), giữ tiếng Anh là mặc định? Xem "Để builder quyết".

## Chuyển động (animation)

**Ý tưởng chủ đạo:** "bóc sticker và dán". Mọi chuyển động đều kể một việc: sticker đang được mang về và dán vào bộ sưu tập của bạn. Mockup tương tác: `.superpowers/brainstorm/80816-1794888666/content/store-motion.html`.

**Khoảnh khắc chính, tải pack (4 trạng thái chạy liền mạch):**
1. *Bấm Tải:* trong tối đa 100ms nút co thành thanh tiến độ ("Đang chờ…"), không chờ sự kiện tiến độ đầu tiên từ cubit; thanh có vệt sáng chạy qua. Phản hồi xúc giác nhẹ (`HapticFeedback.selectionClick`).
2. *Đang tải:* nút đổ màu hồng theo `downloadProgress` (đã làm tròn 2% nên mỗi lần dựng lại có giới hạn) và hiện phần trăm; các sticker xem trước trên thẻ **luôn đủ màu trước khi tải** (để người dùng thấy hàng thật). Khi bấm Tải chúng xám đi (khoảng 300ms) rồi **hiện lại màu lần lượt theo ngưỡng tiến độ** (mỗi cái nảy một nhịp), nên tiến độ vừa là số vừa là hình. Chỉ pack đang tải mới xám; pack chưa bấm và pack đã có luôn đủ màu.
3. *Thành công:* dấu tick tự vẽ (400ms), vòng sáng lan quanh nút, thẻ nhún một nhịp (1.0 → 1.035 → 1.0), một chùm tim/sao/lấp lánh bắn ra (chỉ một lần, dưới 1s), phản hồi xúc giác `lightImpact`; dòng phụ chuyển thành "Sẵn sàng trong Của tôi". Nút ổn định ở "Đã có".
4. *Thất bại (`PackDownloadStatus.failed`):* nút rung ngang một lần (450ms), đổi sang hồng nhạt "Lỗi · Thử lại", sticker chưa tới ngưỡng giữ nguyên xám, dòng phụ nêu lý do ngắn (từ `PackDownloadState.message` nếu có), phản hồi xúc giác `mediumImpact`. Bấm lại thử tải lại. Chỉ nói "không phải tải lại từ đầu" nếu repository thật sự tiếp tục từ chỗ dở; nếu không thì bỏ câu đó.

**Chuyển động hỗ trợ (đều có nhiệm vụ rõ):**
- *Mở trang:* khung chờ lấp lánh theo màu pastel (thay `CircularProgressIndicator`), sau đó thẻ trồi lên lần lượt, tổng thời gian stagger dưới 500ms; không phát lại khi cuộn.
- *Carousel:* sticker lớn của thẻ đang hiện nổi bập bềnh nhẹ (vòng 4 đến 5s), thẻ kế bên có hiệu ứng thị sai và co nhẹ theo vị trí trang; chấm trang giãn ra ở trang hiện tại.
- *Chạm:* thẻ và nút nhấn co lại (khoảng 0.96) trong 100 đến 150ms.
- *Ngăn kéo D:* trượt lên 350 đến 450ms, gấu nảy lên mép ngăn kéo, 6 sticker trong lưới nảy ra lần lượt (cap 300ms tổng), chạm vào sticker thì nó lắc nhẹ; chuyển giữa các nút (Tải, tiến độ, Đã có, Lỗi) dùng cùng cơ chế với thẻ.
- *Gấu ở đầu trang:* thỉnh thoảng chớp mắt; reo mừng ngắn khi có pack tải xong. Tạm dừng khi trang không hiện.
- *Kéo để làm mới:* gấu xuất hiện thay vòng quay mặc định (nếu không tốn quá nhiều công; có thể bỏ).

**Thông số chung:** vào 300 đến 500ms với đường cong giảm tốc kiểu `cubic-bezier(0.16, 1, 0.3, 1)` (Flutter: `Cubic(0.16, 1, 0.3, 1)`), thoát nhanh hơn vào; nảy chỉ dùng ở khoảnh khắc "sticker xuất hiện/dán xong", không dùng khắp nơi.

**Hiệu năng và triển khai:**
- Không thêm dependency (không Lottie/Rive): dùng `AnimatedSwitcher`, `TweenAnimationBuilder`, `AnimationController` dùng chung, `CustomPainter` cho chùm hạt và tick.
- Hiệu ứng liên tục (bập bềnh, chớp mắt) chỉ chạy cho widget đang hiện và dừng khi mở ngăn kéo, rời trang hoặc app vào nền (`TickerMode`, `VisibilityDetector`-kiểu thủ công hoặc `RouteAware`); mỗi vùng động bọc `RepaintBoundary`; không animate layout (dùng transform/opacity).
- Tránh mỗi thẻ một `AnimationController` thường trực: chỉ tạo khi thẻ có chuyển trạng thái.
- Tiến độ đi qua `MarketplaceCubit` như hiện tại (đã giới hạn 2%), không thêm luồng sự kiện mới.

**Truy cập (accessibility):** khi `MediaQuery.disableAnimations` bật: bỏ bập bềnh, thị sai, chùm hạt, rung, nảy; **giữ** đổ màu tiến độ, đổi màu trạng thái, tick xuất hiện (không vẽ dần) để phản hồi vẫn rõ. Trạng thái tải, thành công, lỗi cũng được công bố qua `Semantics` (label/liveRegion), không chỉ bằng hoạt ảnh.

**Kiểm chứng chuyển động:** widget test các chuyển trạng thái (remote → tải → đã có; remote → tải → lỗi → thử lại) với `pumpAndSettle` và nhánh `disableAnimations`; xác nhận hiệu ứng liên tục dừng khi ngăn kéo mở; chạy ở chế độ profile trên máy thật (hoặc simulator) xem đồ thị khung hình khi cuộn danh sách dài và khi tải nhiều pack cùng lúc.

**Chống mục tiêu của chuyển động:** vòng lặp chạy khi không nhìn thấy; hiệu ứng chặn thao tác (không có hoạt ảnh nào được chặn người dùng bấm tiếp); mọi thứ cùng nhún nhảy; thanh tiến độ giả (phải phản ánh `downloadProgress` thật, giai đoạn "Đang chờ…" chỉ là chỗ giữ trước khi có tiến độ đầu).

## Font chữ và số nhiều

- **Font đóng gói luôn trong SDK: Baloo 2** (SIL OFL 1.1, tác giả Ek Type): tròn, mập, dễ thương, nhí nhảnh, và có đủ ký tự tiếng Việt. Tôi đã kiểm tra bằng công cụ: Baloo 2 có đủ mọi chữ hoa/thường có dấu (ă â ê ô ơ ư đ và 5 dấu thanh); **Fredoka (font dùng trong mockup trước đó) thiếu hầu hết chữ có dấu tiếng Việt như ả, ạ, ă, ơ, ư, đ, Đ, nên bị loại**. Nunito đủ dấu nhưng ít nhí nhảnh hơn.
- Đóng gói 4 độ đậm tĩnh 500, 600, 700, 800 (tách từ bản biến thiên, cắt chỉ giữ Latin + Latin mở rộng + Việt + dấu câu thường dùng, khoảng 60 KB mỗi tệp) trong `clients/flutter/restick/assets/fonts/`, kèm `OFL.txt`; khai báo trong `pubspec.yaml` của SDK. Không thêm dependency (`google_fonts`), không tải font lúc chạy, nên chạy offline và không phụ thuộc mạng.
- Áp dụng cho toàn bộ giao diện SDK qua một nơi duy nhất (`RestickBrand` cung cấp `TextStyle` với `fontFamily` và `package` của SDK); app chủ vẫn dùng font riêng của họ cho phần còn lại.
- **Số nhiều của "sticker":** bản tiếng Anh viết "1 sticker", "24 stickers" (0 cũng là số nhiều: "0 stickers"). Bản tiếng Việt giữ "sticker" vì tiếng Việt không biến đổi theo số lượng. Mọi chỗ hiện số lượng (thẻ, ngăn kéo "Tải về · N stickers", nhãn trợ năng) đi qua một hàm duy nhất của `RestickStrings` để không chỗ nào sót.

## Kiểm chứng

- Test Go cho trường mới; test cubit hiện có (`marketplace_cubit_test.dart`) vẫn xanh.
- Widget test: trạng thái rỗng, đang tải, danh sách có thẻ, nút Tải/Đã có/đang tải, mở ngăn kéo; giữ `ValueKey` cũ.
- Chạy app ví dụ (`example/`) với mock server trên :4500 (ghi nhớ của chủ dự án) để chủ dự án xem trực tiếp.

## Các quyết định tôi đã chốt (bạn kiểm tra khi đọc)

- **Tìm kiếm ở đầu trang:** lọc phía client theo tên/tác giả/tag trên danh sách đã tải, không thêm tham số tìm kiếm cho server. Giới hạn: chỉ phủ các pack đã tải về trang hiện tại.
- **Thứ tự "Vừa lên kệ":** giữ thứ tự server (`updated_at` giảm dần); carousel lấy 3 pack đầu. Không thêm trường ngày xuất bản.
- **Hành động trên thẻ:** "Tải" tải pack; đang tải không bấm được và không có nút huỷ; "Đã có" mở ngăn kéo, trong đó có nút "Xoá" (giữ `marketplace.pack.remove.<id>`). Cubit không đổi.
- **Thương hiệu trong SDK:** gấu và chữ "Restick" gắn cứng theo yêu cầu của chủ dự án (Restick là thương hiệu của chính SDK); không làm hook thay thế trong phiên bản này.

## Để builder quyết

- Thanh tab "Cửa hàng/Của tôi": không thuộc SDK; chỉ làm nếu app ví dụ (`example/lib/src/home_shell.dart`) đang có thanh tương tự thì chỉnh cho hợp thiết kế.
- Cách vẽ gấu (SVG package hay CustomPainter; ưu tiên `CustomPainter`, không thêm dependency).
- Ngôn ngữ chuỗi giao diện: SDK hiện dùng tiếng Anh ("Download", "Remove"); chọn giữ tiếng Anh với hook để app chủ đổi, hay chuyển hẳn sang tiếng Việt (gợi ý của tôi: chuỗi lấy từ một lớp `RestickStrings` có mặc định tiếng Anh, vì SDK dùng cho nhiều app).
- Số pack ở carousel (gợi ý: 3 pack đầu).
- Hành vi khi thiếu `previewUrls` (server cũ): dùng `thumbnailUrl` làm sticker lớn, ẩn dải xem trước.
