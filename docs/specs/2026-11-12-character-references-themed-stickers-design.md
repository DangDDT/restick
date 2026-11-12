# Restick — ảnh tham chiếu nhân vật và bộ sticker theo chủ đề

Ngày: 2026-11-12. Trạng thái: **bản thiết kế chờ duyệt cùng implementation plan**. Người dùng đã xác nhận phạm vi nghiệp vụ và chọn Superpowers SDD; chưa duyệt hai tài liệu để bắt đầu sửa product code. Không cho phép merge, push, deploy hoặc chạy migration trên dữ liệu thật.

## 1. Ý định

Người vận hành tạo hình bằng agent bên ngoài. Restick chuẩn bị prompt có cấu trúc và ảnh tham chiếu có thể sao chép, nhận sheet trở lại, rồi dùng pipeline cắt sticker hiện có. Người dùng phải chốt ngoại hình nhân vật trước khi ghép bộ; với bộ ẩm thực, các ô có thể là những món khác nhau kèm tên món tiếng Việt.

### Người dùng đã xác nhận

- Một nhân vật có nhiều ảnh tham chiếu. Bàn ghép chọn nhân vật trước, sau đó chọn **một** ảnh của nhân vật đó.
- Đã chọn nhân vật thì bắt buộc chọn ảnh tham chiếu trước khi biên dịch.
- Trang Nhân vật có prompt riêng để sao chép, tạo thử bằng agent bên ngoài, xem và quyết định ngoại hình, sau đó tải ảnh tham chiếu về.
- Sau biên dịch, ảnh đã chọn nằm dưới khung prompt, có nút sao chép **nội dung ảnh**, không phải URL.
- Palette phải cho thấy màu thực tế khi đang lựa chọn.
- Bộ không có nhân vật cố định được phép chứa các món khác nhau. Chữ có thể là “Bún bò Huế”, “Phở Hà Nội”, “Bánh xèo miền Tây”.
- Agent bên ngoài vẽ cả hình lẫn chữ. Prompt phải giải thích rõ hình cần vẽ và chữ cần xuất hiện; Restick không ghép chữ sau.
- Quy tắc nhân vật tùy chọn, bắt buộc ảnh nếu có nhân vật, cùng phong cách toàn bộ và giữ màu nhận diện đã được người dùng chấp nhận trong bản tổng hợp.

### Lựa chọn của người thiết kế, cần duyệt trong tài liệu này

1. Prompt nhân vật được tổng hợp trực tiếp từ tên, mô tả và ghi chú nhận dạng đang nhập; sửa các trường nguồn để đổi prompt. Không thêm trường prompt lưu riêng, trình biên tập template hoặc API gọi LLM.
2. Không có trạng thái duyệt/khóa nhân vật: “chốt” là quyết định của người vận hành. Không có ảnh thì hồ sơ vẫn lưu được để tiếp tục tạo hình, nhưng chưa dùng được để biên dịch bộ có nhân vật.
3. Mỗi ô dùng `free_text` hiện có làm mô tả hình ảnh và thêm `label` cho chữ muốn agent vẽ. Không thêm loại “món ăn”, subject registry hay cờ mode; có/không có `character_id` quyết định quy tắc nhận dạng.
4. Lưu ảnh đã chọn trong draft và snapshot. Kết quả prompt và ảnh luôn thuộc cùng một lần biên dịch, không tự đổi theo lựa chọn đang chỉnh.
5. Giữ snapshot schema version `1`: chỉ thêm trường optional, không đổi nghĩa trường cũ; không đổi job request schema hay public Flutter/read API.
6. Prompt hướng dẫn tiếp tục bằng tiếng Anh như compiler hiện tại; nội dung người dùng, nhất là label tiếng Việt, giữ nguyên, không tự dịch. Restick không có dịch vụ dịch tự động.

## 2. Mục tiêu và phạm vi không làm

**Mục tiêu:** hai luồng dùng hoàn chỉnh, không gây nhầm prompt/ảnh; không mất ảnh hay thiết kế cũ; palette dễ chọn; prompt cho agent không tự mâu thuẫn về chủ thể và chữ.

**Không làm:** tạo ảnh trong app, API AI provider, OCR/kiểm tra chính tả tự động, ghép chữ vào ảnh, editor font/layout, nhiều ảnh chọn đồng thời cho một lần biên dịch, ảnh tham chiếu riêng cho từng ô, thay pipeline tách nền/cắt ảnh, khóa/versioning nhân vật, dependency mới, thay auth, thay public pack contract.

**Anti-goals:** ép mọi món thành hồ sơ nhân vật; giả ảnh tham chiếu để vượt validation; âm thầm chọn ảnh đầu tiên; đổi màu nhân vật/món ăn chỉ để khớp palette; đổi ảnh của prompt cũ; hứa agent luôn viết đúng dấu; sửa/xóa dữ liệu thật để chạy seed hoặc test.

## 3. Hiện trạng và điểm phải thay

- `clients/web-next/app/characters/page.tsx`: upload đơn, một `reference_asset_id`; có mô tả, identity notes và nhập prompt từ identity fragment, chưa có prompt tạo nhân vật riêng.
- `clients/web-next/app/factory/page.tsx`: character/palette native select; swatch chỉ hiện sau khi chọn. State chỉ giữ text của kết quả compile, chưa giữ snapshot cạnh text. Desktop panel và mobile dialog đều cần cập nhật.
- `services/api/internal/studio/library.go`, `types.go`: một reference trên character; draft/snapshot chưa lưu lựa chọn ảnh.
- `prompt.go`: lời mở đầu ép cùng một chủ thể kể cả không chọn nhân vật; vừa cấm vẽ mô tả thành chữ vừa cấm mọi chữ. Phải phân biệt mô tả và label, không bỏ các quy tắc nền/lưới/cắt.
- SQLite migration hiện có `0001_factory.sql`, `0002_design_sheets.sql`; migration mới phải backfill ảnh thật hiện có trước khi bỏ cột cũ.
- `seed.go` tự biên dịch ví dụ Việt Nam có character nhưng chưa có ảnh; không được cho seed bỏ qua quy tắc bắt buộc ảnh.
- Restore kiểm tra schema trước khi migrate bản sao staging; yêu cầu bảng mới phải phụ thuộc schema version để backup version 2 vẫn restore được.
- Pipeline `process_job.py`/`layout.assign_boxes` đòi một cụm sticker cho mỗi ô. Chữ phải nằm trong một phần hình nối với sticker, không trôi thành các cụm chữ rời. Không sửa thuật toán cắt trong dự án này.

## 4. Các phương án và quyết định

### Quan hệ ảnh nhân vật

- **Chọn:** bảng liên kết có thứ tự `character_reference_assets`, danh sách ID trong character API. Tận dụng tài nguyên upload bất biến và endpoint private hiện có; membership được backend kiểm tra.
- JSON array trong cột character: ít bảng hơn nhưng mất foreign key trên từng asset và kiểm tra quan hệ bằng SQL; không chọn.
- Giữ cột ảnh cũ song song: hai nguồn sự thật và fallback chọn ảnh ngầm; không chọn.

### Prompt và chữ

- **Người dùng chọn:** agent vẽ cả hình/chữ; compiler mô tả rõ từng ô và exact label. Đổi lại, người vận hành kiểm tra và tạo lại ảnh nếu sai chữ.
- Restick ghép chữ sau: chủ động chính tả nhưng thêm font/layout/rendering và bước vận hành; không chọn.
- Prompt nhân vật sinh từ trường nguồn trong web: đủ cho quy trình copy thủ công, không lưu trùng dữ liệu. Editor prompt độc lập có thể lệch identity notes; không chọn.

## 5. Hành vi người dùng

### 5.1 Nhân vật

- Form giữ các trường hiện có, bổ sung prompt chỉ đọc và nút Sao chép prompt. Prompt cập nhật từ giá trị form, không đòi lưu hay có ảnh trước. Trên danh sách/detail hiện hữu cũng có thể mở form để copy lại.
- Nội dung yêu cầu **một ảnh nghiên cứu nhận dạng của một nhân vật**, toàn thân, dáng trung tính, thấy mặt/đặc điểm rõ, nền trắng phẳng, không lưới/collage/chữ/watermark, giữ tỷ lệ và màu được mô tả. Không áp kỹ thuật lưới sticker cho bước này.
- Thư viện identity fragment hiện có tiếp tục bổ sung `identity_notes`; không tạo cơ chế chọn identity thứ hai.
- Cho chọn nhiều file PNG/JPEG/WebP bằng giới hạn upload hiện có; upload lần lượt, thêm ảnh thành công vào gallery, báo file thất bại và không làm mất các ảnh đã có. Không save khi còn upload. Không tự chọn ảnh cho Bàn ghép.
- Mỗi thumbnail có chọn/xem ảnh và thao tác bỏ khỏi hồ sơ. Bỏ chỉ xóa association khi lưu, không xóa asset/file. Hủy form không thay quan hệ đã lưu; upload bất biến đã thành công không bị xóa khi hủy (theo vòng đời upload hiện có).
- Danh sách nhân vật hiển thị thumbnail đầu tiên và số ảnh; thứ tự gallery là thứ tự được thêm, không bổ sung drag/drop reorder.
- Lưu hồ sơ và thay danh sách reference là một transaction; ID không tồn tại không được lưu một phần các thay đổi khác.

### 5.2 Bàn ghép và kết quả

- Không dùng nhân vật: được compile/save mà không có reference. Có thể mô tả món/đồ vật khác nhau ở từng ô; giữ phong cách toàn bộ, không buộc cùng chủ thể.
- Có nhân vật: hiện gallery của đúng nhân vật, chọn một ảnh rõ ràng. Không tự chọn ngay cả khi chỉ có một ảnh. Không có ảnh: hướng dẫn quay lại Nhân vật để tạo/tải ảnh.
- Đổi/bỏ nhân vật luôn xóa lựa chọn reference đang chỉnh. Chọn ảnh của nhân vật khác hoặc asset không thuộc danh sách hiện tại bị backend từ chối.
- Compile và Save (vốn tự compile) đều cần ảnh hợp lệ nếu có nhân vật. Suggest chỉ đổi fragment, không compile: được chạy khi chưa chọn ảnh, nhưng phải giữ nguyên lựa chọn reference/label và mọi text.
- Kết quả là cặp `{compiled_prompt, snapshot}` lấy từ cùng response. Dùng snapshot để render ảnh dưới prompt ở cả desktop và mobile, không lấy từ gallery/state hiện tại.
- Đổi lựa chọn hoặc compile/save thất bại không thay cặp kết quả cũ. Dùng một busy guard cho compile/save/suggest/delete: trong request đang chạy, khóa thay đổi editor và switch/new/load/delete để response không áp vào bản khác. Không thêm retry hay request coordinator.
- Load design hiển thị đúng cặp đã lưu. New/delete design xóa cả prompt và ảnh. Thiết kế cũ chưa lưu reference vẫn xem/copy được prompt cũ, nhưng phải chọn ảnh trước khi save/compile lại.
- Reference đã bỏ khỏi hồ sơ vẫn xem/copy được từ snapshot cũ vì asset không bị xóa; compile mới cần association hiện tại.
- Không đổi chính sách archive: picker mới chỉ liệt kê character active; mở thiết kế cũ vẫn lấy được hồ sơ archived qua GET theo ID để hiển thị/kiểm tra reference. Chỉ thiếu reference hợp lệ mới chặn compile/save; không tự bỏ character cũ.
- Trường hợp archived không có ảnh: vẫn xem/copy kết quả cũ; để biên dịch lại, người dùng chọn một hồ sơ active đã có ảnh hoặc chủ động chọn Không dùng nhân vật và bổ sung mô tả. Giao diện giải thích hai đường này thay vì dẫn tới hồ sơ không xuất hiện ở trang Nhân vật. Không tự đổi character, không thêm unarchive hoặc editor archived; đây là lựa chọn của người thiết kế cần duyệt.

### 5.3 Sao chép ảnh

- Nút Sao chép ảnh độc lập với Sao chép prompt. Luôn sao chép asset của snapshot đang hiển thị.
- Dùng native `ClipboardItem` và clipboard binary write trong thao tác click. Fetch private same-origin bằng session hiện tại. PNG có thể dùng trực tiếp; JPEG/WebP được decode và chuyển PNG bằng canvas trong trình duyệt, giữ tỷ lệ/transparency.
- Khởi tạo clipboard write trong user gesture; dùng promise PNG nếu cần để không mất activation sau fetch/decode.
- Chỉ báo thành công sau khi clipboard write hoàn tất. Thiếu hỗ trợ/permission, HTTP lỗi, decode/canvas lỗi hoặc ảnh không đọc được đều báo lỗi; không âm thầm copy URL hoặc prompt thay ảnh.
- Có link mở ảnh private để người dùng sao chép/lưu thủ công khi trình duyệt không hỗ trợ. Không mở public route, không proxy URL bên ngoài.

### 5.4 Palette và nội dung ô

- Thay palette name-only select bằng danh sách lựa chọn radio/card có tên và bốn swatch `primary, secondary, accent, line` nhìn thấy **trước** khi chọn; có Không dùng bảng màu. Tái sử dụng dữ liệu `Fragment.colors`, filter/conflict rule hiện có.
- Có tên vai trò và mã hex, selected/focus state; không dùng màu làm tín hiệu duy nhất. Bàn phím và viewport nhỏ vẫn dùng được; không cần custom combobox hoặc dependency.
- Mỗi ô thêm trường “Chữ trên sticker (tùy chọn)”; mô tả hình dùng `free_text`, vẫn giữ action/expression/fragments hiện có. Đổi lưới bảo toàn label của các ô còn lại, ô mới có label rỗng.
- Mô tả là chỉ dẫn vẽ, không phải chữ xuất hiện; label rỗng nghĩa ô đó không có chữ. Chỉ có label mà không có mô tả/action/expression/fragment vẫn là ô rỗng không hợp lệ, vì agent chưa biết cần vẽ gì.
- Palette hỗ trợ độ đồng bộ, không thắng màu nhận dạng nhân vật hoặc màu tự nhiên đặc trưng của món ăn. Primary/secondary áp ở vùng hỗ trợ phù hợp, accent ở trang trí/props, line ở nét; không tự đổi dữ liệu palette hoặc thêm rule engine.

## 6. Data và API contract

| Vị trí | Contract mới |
| --- | --- |
| Character, CharacterDraft (Go/TS/OpenAPI) | `reference_asset_ids: string[]`; response luôn có array, kể cả `[]`. Request omitted/empty là không reference. Bỏ field singular trên character; không alias fallback. |
| Draft | `reference_asset_id?: string`, chỉ một ảnh được chọn cho character của thiết kế. Không có character mà có reference là invalid, không silently drop. |
| Cell, SnapshotCell | `label?: string`; omitted là rỗng. Trim đầu/cuối như các text hiện có; giữ nguyên dấu, nội dung và khoảng trắng bên trong. Không tự dịch/transliterate. |
| SnapshotCharacter | `reference_asset_id?: string`, ảnh đã xác minh lúc compile; optional để đọc snapshot cũ. |
| CompiledDesign/Design | Giữ envelope hiện tại, snapshot mang reference/label. Không trả private URL trong prompt. |

### SQLite migration `0003_character_references.sql`

- Tạo `character_reference_assets(character_id, asset_id, position)` với PK `(character_id, asset_id)`, unique `(character_id, position)`, position không âm. FK character cascade, asset restrict. Một asset có thể được gắn nhiều hồ sơ; không thêm quy tắc ownership chưa được yêu cầu.
- Backfill từng `characters.reference_asset_id` không NULL thành position `0`; ảnh đã lưu không đổi ID/bytes. Sau đó bỏ index singular và cột singular. Không đổi migration cũ.
- Không viết lại `designs.draft_json`, `snapshot_json`, `compiled_prompt` hoặc ảnh sheet/version/job cũ để đoán reference lịch sử. Không tự gán ảnh hiện tại cho thiết kế cũ.
- Restore schema version 2 không cần bảng mới lúc preflight; staging mới migrate lên version 3. Schema 3 thiếu bảng mới phải bị từ chối. Chỉ gate bảng mới theo version, không mở rộng hỗ trợ backup schema 1 (vốn bị các yêu cầu bảng hiện tại từ chối). Thay fixture SQL singular trong storage/maintenance tests; giữ test upgrade dùng schema cũ có chủ đích.
- Chuẩn hóa IDs theo pattern `uniqueStrings` đang có (trim, bỏ rỗng, loại trùng, giữ thứ tự); validate từng asset tồn tại rồi ghi transaction. ListCharacters không query nested khi rows đang mở trên DB single connection.

### Seed và compatibility

- Seed character profiles vẫn có thể không có ảnh để người dùng tạo/chốt sau. Không bịa ảnh, không bypass validation trong Compile/SaveDesign.
- Với mười starter design trong `seed/vietnam.json` đang gắn character không có ảnh: bỏ `character_id` và bổ sung nguyên mô tả + identity notes của hồ sơ seed tương ứng vào `free_text` chung, để các hành động từng ô vẫn có chủ thể rõ. Giữ tên, số mẫu, lưới, nội dung ô và fragment; không viết lại thành bộ món ăn khác, không tự thêm label. Không tạo registry mới. Seed marker/version không đổi, không overwrite thiết kế đã cài; kiểm tra prompt mỗi mẫu vẫn mô tả đúng chủ thể ban đầu.
- Chỉ admin write contract đổi ở character. Public catalog/bundles/Flutter SDK không thay. Chạy lại caller/fixture search khi triển khai để migrate toàn bộ singular character consumers; draft/snapshot singular mới là ý nghĩa khác có chủ đích.

## 7. Compiler contract

- Opening: có character thì cùng nhân vật theo ảnh đính kèm và identity notes; không character thì các chủ thể được mô tả riêng ở từng ô. Cả hai giữ một phong cách/nét/tỷ lệ trình bày đồng bộ.
- Character block yêu cầu người dùng đính kèm ảnh đã chọn, dùng làm nguồn nhận dạng; chỉ đổi pose/expression/props. Không nhúng URL admin hoặc hứa tự gửi ảnh.
- Cell block có hai phần phân biệt: drawing description và exact visible label được quote/escape an toàn; label rỗng nêu không chữ cho ô đó. Giữ nguyên các tên tiếng Việt.
- Text rules: chỉ viết label được chỉ định, không dịch/đổi dấu/viết lại; không vẽ mô tả, số thứ tự, tọa độ ô, tên nhóm hoặc prompt thành chữ. Kiểu chữ đồng bộ, dễ đọc ở kích thước sticker, không che chủ thể.
- Hình và chữ phải hợp thành **một cụm nối liền** trong cùng ô: ví dụ chữ trên dải màu đặc gắn với món; dấu tiếng Việt nằm trên vùng nền hình đó. Không yêu cầu bảng chữ nhật bao cả ô, không chữ trôi rời hoặc caption dưới sheet. Đây là chỉ dẫn agent, không thay thuật toán cắt.
- Giữ nền trắng phẳng `#FFFFFF`, đúng rows × columns/ratio/số lượng, khoảng trống biên, không đường lưới/watermark/frame. Chỉ bỏ lệnh cấm mọi text để thay bằng ngoại lệ label; không cho agent tự vẽ white die-cut border vốn do phần mềm thêm.
- Nếu cả bộ không có label, kết quả vẫn cấm chữ; với bộ trộn, chỉ ô có label được vẽ chữ. Không để một câu cấm chữ toàn cục mâu thuẫn ngoại lệ.

## 8. Acceptance và kiểm chứng

1. Nhân vật lưu hai ảnh thật, reload giữ đúng thứ tự; bỏ association không xóa ảnh khỏi snapshot cũ; update có asset lạ thất bại nguyên khối.
2. Upgrade từ DB schema 2 giữ ảnh cũ thành danh sách một ảnh; thiết kế/prompt cũ không đổi. Restore backup 2 thành 3, backup 3 thiếu bảng quan hệ bị từ chối.
3. Compile/save có character thiếu reference, reference khác character hoặc reference không có character trả `422 unprocessable`; valid membership lưu được vào snapshot và round-trip design.
4. Đổi character xóa chọn ảnh; đang chỉnh ảnh B sau compile A vẫn hiển thị/copy A. Request lỗi/response cũ không đổi cặp kết quả; new/load/delete đúng ở desktop và mobile.
5. Copy PNG/JPEG/WebP cho ra ảnh PNG thật có kích thước/nội dung tương ứng; lỗi session/clipboard không báo thành công và vẫn có đường mở ảnh thủ công.
6. Prompt nhân vật copy được trước khi có ảnh; thay notes đổi nội dung; không lẫn grid/sticker pack instructions, không tự sinh mô tả qua LLM.
7. Bộ 1×3 không character có ba mô tả và ba tên món đã nêu: lưu/reload/suggest giữ nguyên Unicode/label, compiler tách hướng dẫn khỏi chữ thật, không ép cùng chủ thể.
8. Ô chỉ có label bị từ chối; quote/dấu xuống dòng trong label không phá ranh giới mô tả/label. Ô không label không được yêu cầu vẽ chữ, kể cả cùng bộ có ô mang label.
9. Palette hiện swatch trước khi chọn, dùng được keyboard và màn hình 390px/1440px, giữ conflict/filter rule cũ.
10. Seed khởi tạo/restart vẫn chạy sau validation mới, giữ mô tả/identity của chủ thể ban đầu trong prompt các mẫu chuyển đổi, không có fake reference và không ghi đè chỉnh sửa người dùng. Pipeline jobs vẫn đọc snapshot schema 1 mở rộng.

Verification khi triển khai: TDD cho quan hệ/migration/validation/lưu snapshot và Unicode/state invariants; chỉ chạy package/test bị tác động. UI dùng browser thật và server với data dir tạm, quan sát cả clipboard PNG và trạng thái lỗi, không chạm `.localdata*`/secrets đang có. Không thêm dependency/test runner cho web. Không chạy build/full suite/migration dữ liệu thật khi chưa được cho phép riêng.

## 9. Giới hạn và bàn giao

- Agent vẫn có thể viết sai tiếng Việt hoặc làm chữ rời; người vận hành xem và tạo lại. Không cam kết chất lượng hình chỉ bằng prompt.
- Clipboard cần secure context/hỗ trợ binary write; báo lỗi và đường mở ảnh thủ công là bắt buộc, không coi fallback URL là copy ảnh thành công.
- Không thay đổi ảnh/prompt của sheet, pack version hoặc job đã đóng băng. Ảnh reference giữ private.
- Workspace khảo sát có nhiều thay đổi người dùng chưa commit trên `feat/design-sheets-key-access`, gồm README, UI, i18n, OpenAPI, compiler và tests. Chỉ thêm hai tài liệu của dự án này; không stash/commit các thay đổi đó. SDD cần người dùng commit/stash baseline trước, hoặc thống nhất snapshot/workspace riêng; không âm thầm bỏ qua chúng bằng worktree từ HEAD.
- Builder được quyết định chi tiết chia component/helper nhỏ theo convention, nội dung prompt hướng dẫn cụ thể trong các ràng buộc trên, và bố cục thích ứng. Không được mở rộng sang AI integration, typography engine hoặc processor rewrite.

Builder check đã hoàn tất: các câu hỏi về archived character, mức chuyển đổi seed, version gating của restore và baseline workspace đã được giải quyết tại §§5.2, 6 và 9. Không phát sinh yêu cầu unarchive, viết lại nội dung seed thành món ăn, hay hỗ trợ restore schema 1.

Implementation plan đi kèm: `docs/plans/2026-11-12-character-references-themed-stickers.md`. Chỉ bắt đầu SDD sau một lần duyệt chung spec, plan và workspace.
