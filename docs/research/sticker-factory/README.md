# Sticker Factory — bộ nghiên cứu studio một người

Ngày: 2026-11-10. **Đây là tài liệu nghiên cứu và kho ý tưởng đã lưu, chưa phải tính năng được triển khai hoặc thiết kế kỹ thuật đã duyệt.**

## Mục tiêu đã xác nhận

Một người quản lý tự sáng tạo bộ sticker bằng cách ghép các mảnh ý tưởng, copy prompt sang AI tạo ảnh bên ngoài, nhập một ảnh lớn chứa nhiều sticker về Restick để cắt/tạo viền, rồi phân phối tài nguyên cho client dùng API key chỉ đọc. Không có contributor, gửi bài hay kiểm duyệt nhiều vai trò.

Bộ nghiên cứu này tập trung vào “xưởng sáng tạo” và khả năng xử lý ảnh liên quan. **Đã chốt SQLite lưu metadata và filesystem lưu tài nguyên**, thay cho PostgreSQL/MinIO trong kiến trúc mục tiêu. Cách gộp dịch vụ, cơ chế quyền quản trị và migration chưa được chốt; ứng dụng hiện tại chưa được chuyển đổi.

## Đọc theo nhu cầu

| Tài liệu | Nội dung | Khi cần đọc |
| --- | --- | --- |
| [Thư viện sáng tạo](creative-fragments.md) | 24 chiều / 425 giá trị: nhân vật, hình dáng, tính cách, màu, nét, chất liệu, trang phục, phụ kiện, đạo cụ, nghề, thế giới, thời đại, ánh sáng, góc nhìn, tư thế; có gợi ý từ tiếng Anh | Tìm đặc điểm thị giác và giữ nhận diện nhân vật |
| [Ngữ cảnh và biểu đạt](context-and-expression.md) | Khoảng 330 mục có mã: cảm xúc, cử chỉ, chức năng hội thoại, quan hệ/giọng điệu/hài, tình huống, thời gian/dịp, địa lý và chi tiết văn hóa | Tạo nội dung từng sticker và mở rộng bối cảnh |
| [Kho bảng màu](palette-recipes.md) | 24 bảng màu sRGB có mã hex và vai trò chủ đạo/phụ/nhấn/nét | Chọn màu cụ thể thay vì chỉ ghi “pastel” |
| [Ghép prompt và xử lý ảnh](studio-composition.md) | Cách phối mảnh, phân biệt lớp nhận diện/bộ/ô, xung đột, khung prompt, giới hạn engine hiện tại, hướng tạo viền và việc cần chốt | Biến ý tưởng thành sheet có khả năng xử lý được |

Có 10 bộ ý tưởng minh họa trong hai thư viện đầu, các prompt gợi ý và ví dụ biến thể màu. Chúng do người biên soạn đề xuất; không phải ảnh/model output đã chạy hoặc công thức đã được kiểm chứng.

## 1. Không gian ý tưởng mở, không phải danh mục đóng

“Mọi giá trị có thể” không có điểm kết thúc: còn nhân vật gốc, thế giới hư cấu, chủ đề mới và các tổ hợp chưa từng có. Cách tiếp cận của tài liệu là:

- Lưu các trục độc lập đủ rộng để mở rộng mà không đổi kiến trúc sản phẩm mỗi lần thêm ý tưởng.
- Cung cấp hàng trăm giá trị cụ thể có thể chọn ngay, nhưng không coi đó là từ điển đầy đủ hoặc chuẩn văn hóa.
- Giữ chỗ cho nhóm và mảnh riêng của người quản lý, kể cả nhân vật tham chiếu có quyền sử dụng.
- Phân biệt giá trị đồng nghĩa, tổ hợp ghép sẵn và yếu tố thật sự mới; không đếm bản dịch như một giá trị độc lập.
- Dùng phân vùng/tên địa lý từ UN/ISO/CLDR làm nguồn mở rộng thay vì gán mỗi đất nước một bộ trang phục mặc định.
- Không gộp số lượng các trục rồi tuyên bố một con số “tổ hợp hợp lệ”: nhiều lựa chọn phụ thuộc hoặc xung đột.

Các “chiều”, mã và nhãn trong bộ nghiên cứu là quy ước biên soạn, không phải hợp đồng dữ liệu của ứng dụng.

## 2. Cấu trúc studio để thảo luận tiếp

Các khu vực này là **đề xuất**, chưa phải màn hình hoặc phạm vi triển khai được duyệt:

1. **Thư viện mảnh:** duyệt theo nhóm, đọc ý nghĩa và cách dùng; có thể bổ sung nội dung riêng.
2. **Nhận diện nhân vật:** ghi đặc điểm cần giữ và ảnh tham chiếu với phạm vi sử dụng hợp lệ.
3. **Bàn ghép ý tưởng:** kết hợp yếu tố sáng tạo theo ý nghĩa, giữ các mảnh quan trọng, phát hiện yêu cầu mâu thuẫn.
4. **Prompt hoàn chỉnh:** mô tả nhân vật + chỉ đạo nghệ thuật + tình huống từng ô + chỉ dẫn kỹ thuật, để copy sang công cụ AI người dùng chọn.
5. **Nhập sheet và xử lý:** kiểm tra ảnh, tách nền, cắt, tạo viền, xem bản nhỏ, chuẩn bị bộ tài nguyên.
6. **Ghi nhận tổ hợp thực tế:** lưu lại điều gì đã thử và lỗi quan sát được để tái sử dụng, không quảng cáo khả năng “tự học” chưa có.

Không đề xuất thêm GPU thường trực, LLM server, vector database, hệ plugin hay dịch vụ AI trả phí chỉ để duyệt và ghép thư viện văn bản. Kiến trúc triển khai sẽ được đánh giá riêng dựa trên môi trường máy và client thực tế.

## 3. Nguyên tắc tăng đa dạng mà không làm bộ sticker rời rạc

- **Giữ hạt nhân:** vài dấu nhận diện quan trọng ở cùng một nhân vật.
- **Đổi hướng kể chuyện:** thay nghề, chủ đề, môi trường và chi tiết văn hóa, không chỉ đổi màu áo.
- **Đổi cách thể hiện:** cùng một ý tưởng thử vật liệu/nét/bảng màu khác; giữ đủ dấu hiệu để vẫn nhận ra nhân vật.
- **Đổi chức năng hội thoại:** không phải ô nào cũng cười; có hỏi, từ chối, cảm ơn, động viên, chờ đợi, xin lỗi và phản ứng.
- **Lai có chủ đích:** cho phép kết hợp các nền văn hóa hoặc thời đại; giải thích chi tiết nào mượn từ đâu thay vì trộn biểu tượng ngẫu nhiên hoặc khẳng định đại diện cho cả cộng đồng.
- **Không coi khó cắt là cấm sáng tạo:** lông mảnh, kính, glow và phụ kiện rời có thể cần xử lý khác; cần thử và ghi rủi ro thay vì âm thầm xóa khỏi thư viện.

## 4. Phân biệt điều có nguồn, ví dụ và điều chưa kiểm chứng

| Lớp thông tin | Ví dụ | Cách hiểu |
| --- | --- | --- |
| Nguyên tắc có nguồn | Facet của Getty AAT; sRGB/hex của W3C; tên/keyword CLDR; phân vùng UN M49 | Dùng làm cơ sở tổ chức, không phải Getty/Unicode xác nhận danh mục riêng này |
| Giá trị biên soạn | Nhân vật, palette, tình huống, các bộ minh họa | Ý tưởng sáng tạo có thể sửa, không phải chân lý/chuẩn |
| Quan sát mã hiện tại | Flood fill gần trắng, contour alpha, giới hạn cạnh 4096, canvas 512 | Phân tích tĩnh theo code lúc nghiên cứu, không phải benchmark |
| Đề xuất chưa làm | Giữ/thay mảnh, kiểm xung đột, tạo viền có tham số, lưu công thức studio | Cần thiết kế và duyệt trước khi triển khai |

Nguồn chủ đạo:

- [Getty AAT](https://www.getty.edu/research/tools/vocabularies/aat/about.html): cách tổ chức từ vựng theo facet và quan hệ.
- [W3C CSS Color 4](https://www.w3.org/TR/css-color-4/): không gian và cách biểu diễn màu.
- [CLDR — Emoji Names and Keywords](https://cldr.unicode.org/translation/characters/short-names-and-keywords): tên và keyword theo ngôn ngữ, chú ý nghĩa phụ thuộc địa phương.
- [UNSD M49](https://unstats.un.org/unsd/methodology/m49/): mã và phân nhóm quốc gia/vùng lãnh thổ theo mục đích thống kê, không khẳng định liên hệ văn hóa/chính trị.
- [Google — image prompting](https://developers.googleblog.com/en/how-to-prompt-gemini-2-5-flash-image-generation-for-the-best-results/): mô tả có quan hệ, ảnh tham chiếu, giới hạn nhất quán.
- [OpenAI — image generation](https://platform.openai.com/docs/guides/image-generation): giới hạn bố cục, chữ và nhân vật; định dạng có transparency.

Các nguồn chi tiết, điều khoản sử dụng và giới hạn truy cập được ghi trong từng tài liệu. Một số trang UNESCO/ISO bị chặn truy cập tự động: liên kết và thông tin tìm kiếm không được coi là đã đọc đầy đủ nội dung gốc. Không tự động sao chép ảnh, nhãn hiệu hoặc bản ghi có giấy phép vào ứng dụng.

## 5. Ràng buộc quan trọng với Restick hiện tại

- **Nền đơn sắc bất kỳ chưa phải đường cắt nhanh hiện có.** Engine simple đang dựa trên nền gần trắng; phác thảo thị giác ban đầu cần được hiểu lại theo giới hạn này.
- Bố cục hàng/cột trong prompt không phải một tính năng cắt theo ô của code hiện tại; code tách theo contour alpha.
- Một chủ thể có nhiều mảnh rời không luôn tương đương một contour. Một sticker có hai nhân vật cũng không đồng nghĩa phải xuất thành hai sticker.
- Halo phục hồi mask trong nhánh cắt hiện tại không đồng nghĩa đã có công cụ tạo viền màu/độ dày theo ý người dùng.
- Thêm nhiều ô trên cùng ảnh giảm số pixel hữu ích cho từng sticker. Xuất 512 px không khôi phục được chi tiết nguồn thiếu.
- Prompt không bảo đảm AI đúng lưới, đúng số lượng, đúng danh tính hoặc alpha thật. Phải thử ảnh đầu ra của công cụ thực tế.

Chi tiết và liên kết code nằm trong [studio-composition.md](studio-composition.md).

## 6. Tiêu chí nghiên cứu đủ để chuyển sang thiết kế

Cần người quản lý phản hồi tiếp về công cụ AI sử dụng, mức tự động gợi ý tổ hợp, kiểu viền, cách sửa khi ảnh cắt lỗi và môi trường triển khai. Sau đó có thể so sánh những kiến trúc đơn giản hơn mà không bỏ phần xử lý ảnh cần thiết.

Bộ tài liệu này **không phê duyệt triển khai**. Không cài dependency, không gọi API tạo ảnh, không thay code ứng dụng, không migrate dữ liệu, không đo tốc độ/RAM hoặc chất lượng cắt ở bước nghiên cứu này.

## 7. Nghiên cứu kiến trúc liên quan

- [Database nhẹ ngoài SQLite](../lightweight-databases.md): so sánh bbolt, LMDB, Badger và DuckDB với nhu cầu catalog/studio. Người dùng đã chọn SQLite + filesystem; chưa benchmark tài nguyên.


## 8. Spec và kế hoạch để duyệt

- [Spec một người quản lý trên VPS/Docker](../../specs/2026-11-10-single-operator-sticker-factory-design.md): yêu cầu đã xác nhận, lựa chọn thiết kế, nghiệp vụ, giao thức, dữ liệu và tiêu chí nghiệm thu.
- [Kế hoạch triển khai 12 task](../../plans/2026-11-10-single-operator-sticker-factory.md): interfaces, file ownership, kiểm tra và review gates. Hai tài liệu là đề xuất cần duyệt chung, không phải bằng chứng đã triển khai.
