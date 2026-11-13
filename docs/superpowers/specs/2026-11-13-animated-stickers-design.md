# Sticker động (animated stickers)

Ngày: 2026-11-13. Trạng thái: đã duyệt, đang triển khai. Hệ thống chưa chạy thật, không cần di trú dữ liệu.

## Mục tiêu
1. Một sticker tĩnh trong pack có thể có thêm **biến thể động** mượt (~12 fps, vòng lặp 2 đến 4 giây) để dùng chính trong app Flutter.
2. Operator tạo chuyển động bằng công cụ **image-to-video bên ngoài**: app xuất ảnh sticker trên nền trắng và prompt, operator upload clip về, hệ thống tự cắt nền và đóng gói.
3. Mỗi sticker có **toggle** chọn bản tĩnh hay bản động sẽ phát hành, để pack không có sticker trùng.
4. Luồng sticker tĩnh hiện tại **giữ nguyên hành vi**. Mọi thay đổi chỉ cộng thêm.

## Ngoài phạm vi (hoãn)
Xuất Telegram (WEBM VP9), WhatsApp, GIF/APNG; sticker động tạo từ sheet nhiều khung; hiệu ứng JSON/Lottie; toggle phía người dùng cuối; cắt nền bằng model.

## Quyết định đã chốt
- Nguồn chuyển động: video image-to-video, **một clip cho mỗi sticker tĩnh**, khoảng 50% sticker trong pack được làm động.
- Biến thể động gắn **vào cùng vị trí** sticker tĩnh, không thêm mục mới. Số sticker trong pack không đổi.
- Phiên bản pack đã phát hành là **bất biến**, nên clip và toggle là bản nháp, một job "phát hành" tạo phiên bản v+1 cho cả lô.
- Mã hóa động: animated WebP 512 px, 12 fps, `quality=75`, `method=4`, `alpha_quality=100` (khoảng 400 KB cho 36 khung). Clip tối đa 4 giây (48 khung), trần 500 KB.
- Sticker tĩnh và thumb giữ lossless `method=6`. Bundle ZIP giữ `zip.Store` (WebP không nén thêm được).
- Đổi tên `services/img_tool` thành `services/media_factory`.
- Mỗi hàm trong `media_factory` có docstring (xem "Quy ước docstring").

## Bằng chứng (spike và nghiên cứu, chạy trên clip lotus nền trắng 1280x720, 24 fps)
- Cắt nền hiện có (`execute_cutout`) trên từng khung: IoU mặt nạ giữa khung liền kề trung bình 0.989 (thấp nhất 0.977); diện tích lệch tối đa 1.9%; luôn 1 thành phần liên thông; 0.29 giây mỗi khung.
- Animated WebP, 12 fps, 36 khung: q60 = 366 KB, q75 = 402 KB, q80 = 451 KB, lossless = 3.3 MB. q75 là điểm gập. Dung lượng tuyến tính theo số khung (khoảng 11.2 KB/khung ở q75).
- `method=6` cho animation nhỏ hơn chỉ 3% nhưng chậm gấp 100 lần. Mặc định method 0 của animation lớn hơn 50%, nên phải truyền `method` tường minh.
- `minimize_size`, `kmin`/`kmax`, `allow_mixed`, `exact` không có tác dụng. Canvas cắt sát không có lợi. `alpha_quality` dưới 100 làm viền lệch.
- Pillow 13.0.0 / libwebp 1.6.0 hỗ trợ animated WebP. `golang.org/x/image/webp` `DecodeConfig` đọc được kích thước file động, `Decode` thì lỗi.
- GIF nặng khoảng 2.5 lần và mất alpha mềm; APNG nặng khoảng 12 lần. Flutter liệt kê Animated WebP là định dạng tích hợp.
- Giới hạn: chỉ đo một clip; chưa thử phát trên Flutter thật; giới hạn WhatsApp 500 KB theo trí nhớ, chưa tra nguồn.

## Kiến trúc tổng quát
```
sticker tĩnh (pack vN) -> "Làm động" -> animation factory (ảnh nền trắng + prompt)
 -> operator tạo video ngoài -> upload clip -> hàng đợi clip (media_factory clip_job)
 -> bản động nháp (ready) -> xem trước + toggle -> "Phát hành bản mới"
 -> job publish (Go) -> pack v(N+1): tĩnh + động đã bật
```

## media_factory

### Đổi tên
`img_tool` thành `media_factory`: package Python, `pyproject.toml`, `Dockerfile`, hằng số `machineJobModule` trong `services/api/internal/jobs/processor.go`, README và test. Tài liệu lịch sử trong `docs/plans` giữ nguyên (chỉ ghi chú).

### Lệnh `clip_job` (mới, cạnh `machine_job`)
Giao thức `restick.clip-request.v1` / `restick.clip-result.v1`, cùng khuôn `machine_job`: request/result JSON, kiểm tra đường dẫn tuyệt đối (không `..`, không symlink), ghi kết quả nguyên tử, thoát mã khác 0 khi lỗi.

Request: `clip_path`, `output_dir`, `sticker_index`, `fps` (mặc định 12), `quality` (mặc định 75), `outline_width`, `outline_color`.

Các bước:
1. Kiểm tra clip: mp4/webm/mov, tối đa khoảng 20 MB, tối đa 4 giây, mỗi cạnh tối đa 1920 px, có luồng video.
2. Tách khung bằng `cv2.VideoCapture` theo `fps`, tối đa 48 khung. Nếu wheel OpenCV không đọc được H.264 thì dùng PyAV (quyết định ở kế hoạch sau khi kiểm tra).
3. Cắt nền từng khung bằng `execute_cutout`; kiểm tra tỉ lệ nền trước như `no_foreground`.
4. Tính **hộp bao chung** (hợp của mọi khung) và cắt tất cả theo cùng hộp, rồi `compose_sticker` (viền theo design). Không cắt theo hộp riêng từng khung.
5. Mã hóa animated WebP 512 px với `quality`, `method=4`, `alpha_quality=100`, `duration=1000/fps`, `loop=0`.
6. Nếu vượt trần 500 KB: thang giảm cố định q75, q60, rồi fps 10, rồi fps 8. Hết thang thì `output_too_large`.
7. Kết quả: file `animated/NNN.webp`, số khung, fps, thời lượng, dung lượng, chất lượng đã dùng, `diagnostics` (tỉ lệ nền trước, IoU thấp nhất giữa khung liền kề).

Mã lỗi: `invalid_request`, `invalid_path`, `clip_too_long`, `clip_too_large`, `decode_failed`, `no_foreground`, `output_too_large`, `output_failed`, `internal_error`.

### Quy ước docstring
Mọi hàm (cũ và mới) có docstring gồm bốn mục: **Algorithm** (từng bước), **Techniques** (kỹ thuật chuẩn), **Keywords** (từ khóa tiếng Anh để tra cứu), **References** (link). Lời giải thích viết tiếng Việt, thuật ngữ giữ tiếng Anh, code và tên biến tiếng Anh. Chỉ ghi link đã kiểm tra trả HTTP 200; link không kiểm tra được thì không ghi hoặc đánh dấu rõ. Script kiểm tra chạy tay (đủ 4 mục, link sống), không đưa vào CI.

## Dữ liệu (migration `0004_animated_stickers.sql`)
`processing_jobs` có `version > 0`, `UNIQUE (pack_id, version)`, một job đang chạy mỗi pack và `schema_version = 1`, nên **clip không đi qua `processing_jobs`** để không tiêu số phiên bản.

**`sticker_animations`** (bản nháp, `UNIQUE (pack_id, sticker_index)`): `id`, `pack_id`, `sticker_index`, `static_checksum_sha256` (phát hiện nháp lỗi thời), `motion_text`, thông tin clip gốc (đường dẫn, dung lượng, thời lượng, checksum), `status` (`queued`, `running`, `ready`, `failed`, `interrupted`), kết quả (đường dẫn animated WebP nháp, checksum, số khung, fps, chất lượng đã dùng, mã lỗi và thông báo), `use_animated` (mặc định bật khi `ready`), thời gian tạo/cập nhật. Upload lại cho cùng sticker thay dòng cũ và xóa file cũ sau khi thành công.

**`stickers`** thêm cột tùy chọn: `animated_path`, `animated_checksum_sha256`, `animated_byte_size`, `animated_frames`, `animated_duration_ms`.

Hàng đợi clip dùng cột `status`, cùng kiểu `claim` và khôi phục sau crash như `Runner`, và dùng chung `childMu` để chỉ có một tiến trình Python tại một thời điểm.

## Job phát hành (`publish`)
Là một job trong `processing_jobs` với `kind` trong `request_json` (mặc định `sheet`, nên ràng buộc `schema_version` không đổi). Hoàn toàn bằng Go, không gọi Python:
1. Chuẩn bị thư mục `output` của job: sao chép file tĩnh của phiên bản hiện tại và các file động có `use_animated = 1`, còn hợp lệ.
2. `StageBundle`, `CommitFiles`, `CommitProcessed` như hiện có.
3. Chặn khi pack có job sheet đang chạy, có clip `queued`/`running` (409 `animation_busy`), hoặc không có thay đổi so với phiên bản hiện tại.
4. Nháp có `static_checksum_sha256` khác sticker hiện tại bị đánh dấu lỗi thời và không đưa vào.
5. `verifyWebP` với file động chỉ dùng `DecodeConfig` (kiểm 512x512); không gọi `Decode`. Nội dung khung do `media_factory` báo trong kết quả.

## `pack.json` và checksum
Mỗi sticker thêm `animated` tùy chọn: `{file, frames, duration_ms, width, height}` (file trong `animated/`). Checksum tổng: với mỗi sticker theo thứ tự manifest, thêm byte bản tĩnh, rồi byte bản động **chỉ khi có**. Pack tĩnh cũ giữ nguyên checksum. Client mới phải nâng cấp bộ kiểm tra ZIP/checksum trước khi tải pack động: client cũ chỉ hash ảnh tĩnh nên sẽ từ chối bundle động vì checksum không khớp, dù bỏ qua được trường JSON mới. Không được bỏ qua kiểm tra checksum để giữ tương thích. Rollout hiện tại yêu cầu nâng cấp Flutter SDK cùng tính năng; phục vụ bundle động cho client chưa nâng cấp không nằm trong phạm vi.

## API quản trị (chỉ owner)
- `POST /admin/packs/{id}/stickers/{index}/animation`: multipart clip + `motion_text`, trả 202.
- `GET /admin/packs/{id}/animations`; `PATCH` và `DELETE .../animation`; `GET .../animation/preview`.
- `GET .../stickers/{index}/on-white.png`: sticker đặt trên nền trắng (dựng bằng Go; cần xác nhận `webp.Decode` trên WebP tĩnh lossless).
- `POST /admin/animations/compile`: ghép prompt từ mô tả nhân vật của design và `motion_text`.
- `POST /admin/packs/{id}/publish-animations`: tạo job `publish`, trả 202.
API công khai giữ nguyên hình dạng; bundle chỉ lớn hơn. Quyền theo key không đổi.

## Web (`clients/web-next`)
- Trang chi tiết pack: huy hiệu trạng thái từng sticker, nút "Làm động", dòng tóm tắt "N sticker động, có thay đổi chưa phát hành", nút "Phát hành bản mới" (hộp xác nhận liệt kê thay đổi).
- Trang `/packs/animate?id=&index=`: ảnh nền trắng (copy/tải), `motion_text` có gợi ý theo `emoji_hint`, prompt có nút copy, danh sách công cụ video (thêm vào `ai-providers.ts`), upload clip với tiến độ và lỗi, xem trước bản động cạnh bản tĩnh, toggle "Dùng bản động".
- Chuỗi giao diện thêm vào bộ dịch hiện có; mock server QA thêm các endpoint mới.

## Flutter (`clients/flutter/restick`)
- `StickerAsset` thêm `animatedFile` tùy chọn; `file` vẫn là bản tĩnh để app chủ gửi sticker như cũ.
- `getStickers` đọc `animated` trong `pack.json`; pack cũ chạy bình thường.
- Tile phát bản động bằng `Image.file`; dùng thumb tĩnh khi hệ điều hành bật giảm chuyển động (`MediaQuery.disableAnimationsOf`). `GridView.builder` chỉ dựng ô đang hiện.
- Đã thử native macOS debug với 20 đường dẫn animated WebP riêng (clip lotus 36 khung, q80, 512 px), lưới 5 cột, tile `cacheWidth/cacheHeight: 128`: ảnh chụp cách nhau 250 ms khác pixel; giảm chuyển động hiển thị ảnh tĩnh. RSS từ 241.5 MiB lên 314.4 MiB (+72.9 MiB); image cache 20 MiB; sau bật giảm chuyển động RSS 310.5 MiB. Đây là phép đo cả tiến trình debug, không phải bộ nhớ riêng mỗi sticker hay benchmark mobile/release; chưa kết luận FPS/độ mượt khi máy chủ đang chịu tải cao.

## Kiểm thử
- media_factory (unittest, như các test hiện có): clip tổng hợp từ `cv2.VideoWriter`; hộp bao chung; thang giảm chất lượng; các mã lỗi; hồi quy toàn bộ test tĩnh.
- Go: migration; hàng đợi clip và khôi phục sau crash; job `publish`; hồi quy checksum pack tĩnh; nháp lỗi thời; 409 `animation_busy`; `verifyWebP` với file động.
- Web: kịch bản tương tác trên mock server (upload, toggle, phát hành).
- Flutter: parse `pack.json` cũ và mới; widget test giảm chuyển động.
- Docstring: script kiểm tra đủ 4 mục và link trả 200.

## Thứ tự thực hiện
0. Đổi tên `img_tool` thành `media_factory` và viết docstring cho hàm cũ.
1. `clip_job` trong `media_factory`.
2. Migration và hàng đợi clip trong Go.
3. Job `publish` và `pack.json`.
4. API quản trị.
5. Web factory.
6. Flutter.

## Rủi ro và điều chưa kiểm chứng
- Chỉ đo trên một clip lotus nền trắng; công cụ video khác có thể cho nền bẩn hoặc chuyển động mạnh hơn nên dung lượng và độ sạch viền khác.
- `cv2.VideoCapture` đọc H.264 trong môi trường chạy chưa xác nhận (phương án dự phòng PyAV).
- `webp.Decode` trên WebP tĩnh cho `on-white.png` chưa thử.
- Đã xác nhận phát/giảm chuyển động trên macOS debug với 20 sticker; iOS/Android, profile/release và cuộn lưới trên thiết bị di động chưa đo.
- Giới hạn WhatsApp 500 KB theo trí nhớ, chưa tra nguồn (chỉ ảnh hưởng hướng mở rộng).
- Điều khoản thương mại và watermark của gói miễn phí các công cụ video do operator tự kiểm tra.
- Mô hình video có thể làm méo nhân vật; bước xem trước và toggle là chốt chặn.
