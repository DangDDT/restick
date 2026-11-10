# Sticker Factory — ghép prompt và chuẩn bị ảnh để cắt

Ngày nghiên cứu: 2026-11-10. Trạng thái: **tài liệu nghiên cứu và đề xuất, chưa triển khai**.

## 1. Phạm vi đã được người dùng xác nhận

- Một người quản lý; bỏ nghiệp vụ cộng đồng gửi bài và kiểm duyệt.
- Client dùng API key để lấy tài nguyên và sử dụng; không có quyền chỉnh sửa nội dung.
- Người quản lý ghép các mảnh ý tưởng thành prompt, copy sang một AI tạo ảnh bên ngoài, tải một ảnh lớn chứa nhiều sticker về rồi đưa vào Restick.
- Restick cần hỗ trợ tách sticker và tạo viền nhanh, đồng thời cho phép sáng tạo nhiều chủ đề và nhiều biến thể.
- Người dùng yêu cầu nghiên cứu, lưu lại một thư viện rộng về các mảnh ghép. **Đã chốt SQLite + filesystem cho lưu trữ; chưa duyệt thiết kế kiến trúc tổng thể hoặc triển khai.** Chưa yêu cầu tích hợp API tạo ảnh và chưa chốt AI.

Nguồn nghiệp vụ cũ: [ý tưởng ban đầu](../../ideas/2026-08-20-sticker-pack-ecosystem.md). Phần contributor/review của tài liệu cũ không phản ánh mục tiêu mới; bản nghiên cứu này không tự động sửa hoặc thay thế các kế hoạch triển khai cũ.

## 2. Kết luận nghiên cứu quan trọng

1. **Đa dạng nằm ở khả năng kết hợp, không phải số ô bắt buộc phải điền.** Một nhân vật có thể đổi nghề, bối cảnh văn hóa và chất liệu mà vẫn giữ dấu hiệu nhận diện. Không cần dùng mọi nhóm trong mỗi prompt.
2. **Một prompt cần thành mô tả có quan hệ**, không phải chuỗi từ khóa ngẫu nhiên. Hướng dẫn chính thức của Google nhấn mạnh “Describe the scene, don't just list keywords” [S1].
3. **Không thể bảo đảm đúng số lượng, bố cục và danh tính chỉ bằng prompt.** OpenAI công bố giới hạn về consistency, text rendering và composition control [S2]; Google cũng nêu khả năng cần chỉnh nhiều lượt [S1].
4. **Cần tách hình thức sáng tạo khỏi giao thức xử lý ảnh.** “Mèo phi hành gia dưới trời đêm” không bắt buộc có cả nền trời sao: có thể thể hiện bằng trang phục, đạo cụ và tư thế trên nền kỹ thuật đơn giản.
5. **Ảnh nhìn trong suốt chưa chắc có alpha.** Nền ô bàn cờ có thể đã nằm trong pixel ảnh; một câu “transparent background” không xác nhận định dạng đầu ra. Tài liệu OpenAI phân biệt tham số background và định dạng PNG/WebP [S2]; giao diện AI bên ngoài có thể không cung cấp cùng khả năng đó.
6. **Mã hiện tại không hỗ trợ mọi nền đơn sắc như nhau.** Nhánh `simple` dựa trên nền gần trắng; không được mặc định đổi sang nền xanh/magenta rồi kỳ vọng cắt nhanh vẫn hoạt động.

Các quy tắc phối bên dưới là **đề xuất biên soạn cho Restick**, không phải khả năng đã có hoặc quy định của các nhà cung cấp AI.

## 3. Ba lớp quyết định để ghép ý tưởng

| Lớp | Ví dụ | Phạm vi biến đổi đề xuất |
| --- | --- | --- |
| Nhận diện | Loài, hình dáng mắt/tai, dấu lông, phụ kiện đặc trưng, ảnh tham chiếu | Giữ xuyên bộ; có thể giữ xuyên nhiều bộ |
| Chỉ đạo nghệ thuật của bộ | Chủ đề, nghề nghiệp, chất liệu, bảng màu, thời đại, chi tiết văn hóa | Đổi khi khám phá bộ mới |
| Nội dung từng sticker | Hành động, biểu cảm, cường độ, đạo cụ phụ, thông điệp hội thoại | Đổi giữa các ô |

Không phải mọi nhóm có phạm vi cố định. “Mũ” có thể là dấu nhận diện hoặc đạo cụ dùng một lần. Người sáng tạo quyết định; hệ thống không nên tự tháo phụ kiện đã được giữ lại.

### 3.1 Những thông tin hữu ích cho một mảnh ghép

Đây là mô tả khái niệm để thảo luận, **không phải schema dữ liệu hay hợp đồng API đã chốt**:

- Tên dễ đọc bằng tiếng Việt.
- Mô tả thị giác đủ rõ để đưa vào prompt; có thể có bản tiếng Anh khi hữu ích.
- Nhóm và nhóm con; từ đồng nghĩa để không nhân đôi “mèo nhà” và “domestic cat” như hai ý tưởng khác nhau.
- Phạm vi: cả nhân vật, cả bộ hoặc từng sticker.
- Quan hệ: hợp với, cần đi cùng, xung đột với; lý do bằng lời, không cấm máy móc.
- Mức rủi ro khi cắt: phụ kiện rời, tóc mảnh, cạnh trong suốt, glow, chữ.
- Nguồn/giấy phép/quyền sử dụng nếu mảnh có ảnh hoặc văn bản của bên khác.
- Ví dụ tổ hợp và ghi chú kết quả thực tế nếu đã chạy thử.

Cho phép người quản lý bổ sung nhóm và giá trị riêng là hướng đề xuất. Không chọn cơ sở dữ liệu, định dạng lưu hoặc cơ chế nhập hàng loạt trong bản nghiên cứu này.

### 3.2 Giữ và thay đổi có chủ đích

- **Ghép tay:** người sáng tạo tự chọn từng yếu tố.
- **Biến thể quanh ý tưởng:** giữ nhân vật và phong cách, đổi nghề hoặc câu chuyện.
- **Khám phá có ràng buộc:** giữ các mảnh đã chọn, đề xuất thêm một liên kết bất ngờ nhưng giải thích được.
- **Lai ý tưởng:** chọn hai trục tương phản có quan hệ thị giác, như “thợ bánh + thám hiểm đại dương”; không mặc định trộn mọi nhóm.

Đây là các cách sử dụng để cân nhắc, không phải các chế độ sản phẩm đã được người dùng duyệt. Không cần thêm mô hình AI chạy trong Restick chỉ để ghép các mô tả có sẵn. Mức chủ động tạo ý tưởng chưa được chốt.

## 4. Xung đột cần nhận ra, không âm thầm bỏ ý tưởng

| Tổ hợp | Vì sao dễ thất bại | Cách diễn đạt lại đề xuất |
| --- | --- | --- |
| Chibi cực đơn giản + chi tiết trang phục siêu nhỏ | Thu nhỏ làm mất thông tin | Giữ họa tiết lớn và dấu hiệu đặc trưng |
| Nỉ mềm + cạnh pixel sắc tuyệt đối | Hai quy ước dựng hình khác nhau | Chọn một chủ đạo; mượn bảng màu của cái còn lại |
| Đêm tối + nền đen + nét đen | Khó tách và khó đọc ở kích thước nhỏ | Giữ cảm giác đêm qua màu nhân vật, dùng nền kỹ thuật phù hợp |
| Nhân vật trắng + nền trắng + không đường bao | Khó xác định biên | Dùng đường bao nghệ thuật có màu hoặc xem xét alpha thật; cần thử ảnh |
| Vui cực độ + buồn sâu trong cùng một nét mặt | Mục tiêu biểu cảm mơ hồ | Chỉ rõ “cười gượng, mắt rưng rưng” nếu cố ý pha trộn |
| Cả thành phố + nhân vật nhỏ + nhiều chữ | Trở thành tranh cảnh thay vì sticker | Chọn một đạo cụ hoặc mảnh kiến trúc gắn với nhân vật |
| Nhiều người đối thoại trong cùng ô | Dễ sai số lượng và dính hình | Mô tả nhóm là một sticker duy nhất; không suy ra một người = một contour |
| Lấp lánh, giọt nước hoặc chữ tách xa nhân vật | Có thể bị nhận thành sticker riêng | Giữ trong vùng cụm, kiểm tra nhóm thành phần khi nhập |
| “Giữ đúng ảnh tham chiếu” + thay toàn bộ loài/hình dáng | Mâu thuẫn mức giữ nhận diện | Nêu rõ giữ đặc điểm nào và cho phép thay phần nào |

“Không hợp” là tín hiệu giải thích và chỉnh sửa, không phải danh sách loại bỏ sáng tạo. Một ý tưởng khó vẫn có thể làm được bằng chỉnh ảnh hoặc phương pháp tách khác; chưa có benchmark để xếp hạng chắc chắn.

## 5. Giao thức ảnh cho đường xử lý hiện tại

### 5.1 Những gì đã kiểm tra trong mã nguồn

| Thành phần | Hành vi quan sát được | Tác động đến prompt/studio |
| --- | --- | --- |
| `execute_cutout` | Chọn `simple` mặc định hoặc `rembg` theo `IMG_TOOL_ENGINE` | Không phải mọi ảnh đều qua cùng phương pháp |
| `_exterior_flood_alpha` | Tìm vùng gần trắng, so RGB với 255, ngưỡng 10, giữ phần không nối nền ngoài | Trắng phẳng phù hợp hơn nền màu bất kỳ cho nhánh này |
| `_has_existing_alpha` | Bỏ bước xóa nền nếu tỷ lệ pixel alpha dưới 250 vượt 0,02 | Đây là heuristic, không chứng minh alpha đầu vào sạch |
| `_ink_halo_alpha` | Tạo vùng halo quanh cụm ink khi mask yếu | Không đồng nghĩa có công cụ viền màu/độ dày theo ý người dùng |
| `find_sticker_boxes` | Tách theo contour alpha sau morphological closing | Không đọc số hàng/cột từ prompt, không biết nội dung ô |
| `filter_outlier_boxes` | Loại box có diện tích dưới 25% median các box | Sticker cố ý rất nhỏ hoặc phần rời dễ bị bỏ |
| `sort_boxes` | Sắp theo `(top, left)` | Không phải bộ nhận diện hàng; bố cục lệch có thể sai thứ tự dự kiến |
| `load_image` | Không chấp nhận cạnh ảnh lớn hơn 4096 px | Chưa thể đưa ảnh rất lớn trực tiếp vào pipeline này |
| `fit_on_canvas`, `make_thumb` | Chuẩn hóa canvas 512 và thumbnail 128 | Cần xem ở kích thước nhỏ, không chỉ ở ảnh gốc |
| `execute_process` | Xuất WebP, ảnh xem trước, `pack.json`, checksum; `padding` 8–16 | Chưa có lựa chọn độ dày/màu viền trong tham số hàm |

Nguồn: [cutout.py](../../../services/img_tool/src/img_tool/infrastructure/cutout.py), [layout.py](../../../services/img_tool/src/img_tool/infrastructure/layout.py), [process_sheet.py](../../../services/img_tool/src/img_tool/application/process_sheet.py). Đây là phân tích tĩnh, không phải kết quả thử ảnh.

**Điều chỉnh từ phác thảo đầu:** yêu cầu “nền đơn sắc tương phản bất kỳ” quá rộng với engine hiện tại. Tài liệu này giới hạn đường nhanh hiện tại ở nền trắng gần đồng nhất hoặc alpha thật đã kiểm tra; hỗ trợ nền chroma màu khác là thay đổi chưa được triển khai.

### 5.2 Chỉ dẫn tạo sheet đề xuất

- Một ảnh lớn, không phải ảnh chụp màn hình giao diện chat; giữ bản gốc tải từ AI.
- Mỗi sticker đủ hình, không bị cắt đầu/tai/đạo cụ bởi mép sheet.
- Khoảng trống tách rõ giữa các cụm; không chạm, không chồng, không dùng chung bóng đổ.
- Nền trắng phẳng cho engine simple hiện tại; không gradient, ô bàn cờ giả, giấy có vân hoặc đường kẻ lưới.
- Giữ cỡ cụm tương đối đồng đều nếu dùng bộ lọc box hiện tại.
- Phụ kiện cần gần nhân vật; hạt bụi, tia sáng, chữ hoặc giọt nước rời phải được kiểm tra như thành phần của cụm, không mặc định là sticker mới.
- Với tóc/lông rất mảnh, voan, kính trong hoặc glow: giữ ý tưởng nhưng đánh dấu có thể cần xử lý kỹ hơn; không hứa đường cắt nhanh cho mọi trường hợp.
- Các hướng dẫn về hàng/cột giúp AI dựng ảnh nhưng chưa biến pipeline contour thành bộ cắt theo ô. Ảnh có lưới chuẩn chỉ là mục tiêu thị giác ở giai đoạn này.

### 5.3 Chọn số ô theo độ phân giải, không chỉ theo số lượng

Ví dụ hình học, **không phải cài đặt mặc định hoặc tỷ lệ thành công đã đo**:

- Sheet vuông 2048 px với 3 cột cho khoảng 683 px mỗi ô trước khi trừ khoảng trống.
- Cùng sheet với 6 cột chỉ còn khoảng 341 px mỗi ô trước khi trừ khoảng trống. Xuất 512 px không tự khôi phục chi tiết gốc đã thiếu.
- Nội dung dày, phụ kiện nhiều hoặc nét mảnh có thể cần ít ô hơn/ảnh nguồn lớn hơn trong giới hạn pipeline.
- Kích thước AI thực sự hỗ trợ phải đối chiếu dịch vụ người dùng chọn. Một dòng kích thước trong prompt không bắt buộc công cụ xuất đúng kích thước đó.

### 5.4 Tạo viền sau khi có alpha sạch

Đề xuất tạo viền tại Restick thay vì bắt AI vẽ sẵn:

- Viền ngoài từ mask alpha; ý tưởng cơ bản là mở rộng mask rồi tô lớp viền dưới foreground. Phép dilation và morphological gradient được mô tả trong tài liệu OpenCV [S3].
- Chừa vùng an toàn quanh hình để viền không bị cắt; kiểm tra ở cả nền sáng và nền tối.
- Độ dày nên được đánh giá ở kích thước xuất cuối cùng; pixel của sheet không tương đương pixel của sticker đã chuẩn hóa.
- Viền không cứu được phần nhân vật đã bị xóa nhầm khi tách nền; cần sửa mask trước.
- Viền trắng, viền màu, viền hai lớp, hoặc không viền là các hướng sáng tạo có thể cân nhắc, **chưa phải tính năng đang có**.
- Telegram là một tham khảo ngoài: hướng dẫn import static sticker đề cập PNG/WebP có transparency và khuyến nghị stroke/shadow [S4]. Không biến quy tắc của Telegram thành yêu cầu của mọi client Restick; không thêm bóng đổ trên sheet đầu vào chỉ vì đầu ra nền tảng khác có thể cần shadow.

`rembg` là công cụ xóa nền, không phải bảo đảm chất lượng mọi ảnh. Dự án có tùy chọn model, alpha matting và yêu cầu backend riêng [S5]; chưa chốt dùng thêm model, chưa đo RAM hoặc thời gian cho sheet thực tế của người dùng.

## 6. Khung prompt có quan hệ, không phải câu thần chú

Mẫu bên dưới là **đề xuất chưa thử**, viết bằng tiếng Anh để dễ copy sang nhiều công cụ; không tuyên bố tiếng Anh luôn tốt hơn tiếng Việt. Thay phần trong ngoặc; không copy nguyên placeholder vào AI.

> Create one sticker sheet for use in a messaging app. Use the same [character identity] throughout: [features to preserve]. The set tells [theme / situation], with a [personality / conversational tone]. Render the character in [primary style / material], using [palette] and [line / shading treatment]. Incorporate [specific cultural or world detail] through [clothing / prop / action], not a full scenic background.
>
> Arrange [count] separate sticker compositions in [rows] rows and [columns] columns, read left to right, top to bottom. The individual compositions are: [numbered action and expression descriptions]. Keep the identity, core proportions and primary outfit consistent, unless a change is explicitly described.
>
> Use a uniform pure-white background for later background removal. Keep every composition complete, with clear empty space around it and between neighboring stickers. Keep each character and its accessories together as one visual group. Use no grid lines, captions, numbering, watermark or background shadows. Do not add a cutline or sticker border; those will be handled after extraction. Keep visible character outlines distinct from the white background.

Lưu ý:

- Chỉ dùng tham chiếu ảnh khi công cụ AI hỗ trợ upload/edit/reference; mô tả bằng chữ không thay thế được việc thật sự đính kèm ảnh.
- Nếu có nhiều ảnh tham chiếu, giải thích ảnh nào là nhân vật, ảnh nào là chất liệu, ảnh nào là đạo cụ. Không yêu cầu lấy tất cả đặc điểm từ tất cả ảnh.
- Số và thứ tự trong văn bản chỉ hướng dẫn AI, không phải dữ liệu được pipeline hiện tại tự đọc.
- Với alpha thật đã kiểm tra, thay đoạn nền trắng bằng chỉ dẫn phù hợp công cụ. Không giữ đồng thời hai yêu cầu nền trắng opaque và nền alpha trong suốt như thể chúng là một.
- Prompt văn bản khác với negative prompt/weight/seed bằng cú pháp riêng của từng AI; chưa thêm các cú pháp đó khi chưa biết công cụ sử dụng.
- Nếu muốn chữ, cần xem lại cách nhóm chữ với nhân vật, độ đọc ở thumbnail và lỗi chính tả. “Không chữ” là đường đơn giản đề xuất cho lần thử đầu, không phải lệnh cấm sáng tạo vĩnh viễn.

## 7. Vận hành như studio, không tự mở rộng thành nền tảng nhiều người

Một quy trình studio đề xuất:

1. Chọn hoặc tạo hồ sơ nhân vật; ghi đặc điểm cần giữ và quyền sử dụng ảnh tham chiếu.
2. Chọn vài mảnh tạo chủ đề; ghi vì sao chúng đi cùng nhau và điểm mới so với các bộ đã có.
3. Chọn tình huống từng sticker theo nhu cầu hội thoại, tránh cả bộ chỉ lặp lại một nét cười.
4. Đọc prompt hoàn chỉnh để tìm mâu thuẫn; giữ hướng dẫn kỹ thuật ở một chỗ thay vì rải và lặp.
5. Copy sang AI bên ngoài; đính kèm đúng ảnh tham chiếu và lưu ảnh gốc trả về.
6. Khi nhập, kiểm tra số cụm, phần bị mất/dính, thứ tự dự kiến và biên; đây là kiểm tra chất lượng của một người, không tái lập nghiệp vụ kiểm duyệt nhiều vai trò.
7. Tách nền/cắt/viền/chuẩn hóa; xem thumbnail và cả nền sáng/tối trước khi cho client lấy tài nguyên.
8. Ghi chú tổ hợp đã dùng, công cụ AI và lỗi quan sát được để lần sau chọn tốt hơn.

Lưu prompt/ảnh gốc/nhận xét có ích cho tái sử dụng, nhưng định dạng, lưu trữ và giao diện chưa được quyết định. Không thêm dịch vụ vector, semantic search, GPU thường trực hoặc LLM nội bộ chỉ vì thư viện có nhiều giá trị.

### 7.1 Đánh giá đa dạng bằng nội dung, không thổi phồng số tổ hợp

- Đã khám phá nhiều dạng nhân vật, bảng màu, vật liệu, ngữ cảnh và chức năng hội thoại chưa?
- Hai tổ hợp có khác nhau về câu chuyện và silhouette, hay chỉ đổi một màu gần giống?
- Có lặp lại định kiến văn hóa hoặc phụ kiện mặc định khi đổi quốc gia không?
- Có đọc được khi thu nhỏ, giữ được nhân vật và có chỗ dùng thật không?
- Tổ hợp khó cắt có được đánh dấu để thử trước, thay vì loại bỏ toàn bộ phong cách không?

Không nhân số lượng mọi giá trị để quảng cáo một con số tổ hợp “hợp lệ”: nhiều giá trị phụ thuộc, xung đột hoặc đồng nghĩa. Thư viện là không gian khám phá có quan hệ.

## 8. Những điều cần chốt trước khi xây

- AI/giao diện tạo ảnh thực tế, khả năng reference và kích thước đầu ra.
- Mức tự động đề xuất tổ hợp, so với người dùng ghép tay.
- Cách xử lý khi AI trả ảnh sai số lượng, dính cụm hoặc thiếu chi tiết: sửa thủ công ở đâu, hay yêu cầu tạo lại.
- Kiểu viền mong muốn và tiêu chí ảnh được coi là đủ tốt.
- Môi trường triển khai, tài nguyên máy và phạm vi giữ SDK/backend; không quyết định chỉ dựa vào số người quản lý.
- Dữ liệu hiện có cần giữ/migrate, cơ chế quyền quản trị và API key chỉ đọc.

Các mục này chưa được giải quyết bằng quyết định ngầm trong bộ tài liệu. Kiến trúc/thiết kế và kế hoạch triển khai cần một lần duyệt riêng sau nghiên cứu.

## 9. Nguồn và mức độ kiểm chứng

- **[S1] Google Developers Blog — How to prompt Gemini 2.5 Flash Image Generation for the best results (2025-08-28):** https://developers.googleblog.com/en/how-to-prompt-gemini-2-5-flash-image-generation-for-the-best-results/ — đọc trực tiếp; nguồn cho mô tả có quan hệ, sticker prompting, reference/edit và giới hạn consistency. Hướng dẫn có tính phiên bản, không bảo đảm mọi model/giao diện hiện tại giống nhau.
- **[S2] OpenAI — Image generation guide:** https://platform.openai.com/docs/guides/image-generation — đọc trực tiếp, chuyển hướng sang https://developers.openai.com/api/docs/guides/image-generation; nguồn cho alpha/output format và giới hạn text, consistency, composition. Không tích hợp API và không chốt model trong nghiên cứu này.
- **[S3] OpenCV — Morphological Transformations:** https://docs.opencv.org/4.x/d9/d61/tutorial_py_morphological_ops.html — trang docs bị HTTP 403 khi truy cập; đã đọc bản nguồn chính thức cùng bài tại https://raw.githubusercontent.com/opencv/opencv/4.x/doc/py_tutorials/py_imgproc/py_morphological_ops/py_morphological_ops.markdown. Nguồn cho erosion/dilation/closing/gradient, không phải benchmark cắt sticker.
- **[S4] Telegram — Importing Stickers, Static Stickers:** https://core.telegram.org/import-stickers — đọc trực tiếp; tham khảo đầu ra nền tảng, không phải yêu cầu sản phẩm đã chốt.
- **[S5] rembg — README chính thức:** https://github.com/danielgatis/rembg — đọc trực tiếp; nguồn cho khả năng background removal, backend/model/matting. Giấy phép thư viện không tự động cấp quyền đối với ảnh, nhân vật hay mọi model tải thêm.

Không gọi AI tạo ảnh, không benchmark CPU/RAM, không chạy ảnh mẫu và không thay đổi code ứng dụng trong nghiên cứu này. Các prompt, quy tắc ghép và quy trình studio là đề xuất cần kiểm chứng với ảnh thực tế.
