# Kho bảng màu đề xuất cho Sticker Factory

Ngày: 2026-11-10. Trạng thái: **nghiên cứu / ví dụ biên soạn, chưa triển khai và chưa thử trên AI tạo ảnh**.

## 1. Cách dùng

Mục tiêu là có điểm bắt đầu cụ thể, không chỉ nhãn “pastel” hoặc “rực”. Mỗi hàng là một bảng màu do người biên soạn đề xuất, không phải bảng màu truyền thống được xác thực, tiêu chuẩn văn hóa, hoặc trích xuất từ tác phẩm của bên khác.

- Mã `#RRGGBB` trong tài liệu là sRGB; xem [W3C CSS Color 4 — hexadecimal notation](https://www.w3.org/TR/css-color-4/#hex-notation).
- Màu chủ đạo, phụ, nhấn và nét là **vai trò**, không phải bốn mảnh bắt buộc trong mọi nhân vật. Có thể đổi vai trò có chủ đích.
- AI có thể không giữ đúng mã màu; mô tả tên màu + vai trò + ảnh tham chiếu khi công cụ hỗ trợ, rồi kiểm tra ảnh đầu ra. Mã hex không phải cơ chế khóa pixel của AI.
- Màu bảng này áp dụng cho nhân vật/đạo cụ, **không tự động là màu nền sheet**. Engine simple hiện tại dựa trên nền gần trắng; xem [quy tắc xử lý](studio-composition.md).
- Nét là nét nghệ thuật để nhận ra hình; không nhầm với lớp viền ngoài do Restick được đề xuất thêm sau khi tách nền.
- Chưa đo tỷ lệ tương phản từng cặp; không xem các hàng là chứng nhận WCAG hoặc bảo đảm dễ cắt.

## 2. Hai mươi bốn bảng màu khởi đầu

| Tên gợi nhớ | Chủ đạo | Phụ | Nhấn | Nét / chi tiết tối | Gợi ý ngữ cảnh, không ràng buộc |
| --- | --- | --- | --- | --- | --- |
| Cà phê phin | `#6F4E37` | `#C89B75` | `#E8B44F` | `#302521` | Quán nhỏ, nhân vật đi làm |
| Trà xanh | `#73945A` | `#B7C99A` | `#D59252` | `#27382C` | Vườn, chăm cây, nghỉ ngơi |
| Bánh dâu | `#D97A90` | `#F2C5CB` | `#A0B866` | `#51333D` | Tiệm bánh, dịu dàng |
| Cam quýt | `#E98A3F` | `#F4C66A` | `#5D9980` | `#46332B` | Năng lượng, vận động |
| Biển nông | `#438F9B` | `#A8D5D1` | `#EDAA69` | `#203F4B` | Sinh vật biển, du lịch |
| Đêm có sao | `#414C79` | `#7986AF` | `#E9C96F` | `#22283E` | Đồ ngủ, không gian, mơ mộng |
| Đồng quê | `#8C9F65` | `#D2BB8F` | `#BE6F4A` | `#3C4230` | Thu hoạch, chăm sóc thú |
| Đất nung | `#BB7151` | `#D8AA85` | `#548F8E` | `#49372F` | Gốm, đồ thủ công |
| Bạc hà và đào | `#8DC7B2` | `#E8B2A0` | `#D87562` | `#36544C` | Hoạt hình mềm, thân thiện |
| Mực xanh | `#315C80` | `#8DB1C9` | `#D7A353` | `#1F3044` | Bưu tá, sách vở, ghi chép |
| Công xưởng | `#78838B` | `#BAC0C3` | `#D9A53F` | `#2E383E` | Robot, sửa chữa |
| Cứu hộ nhỏ | `#C75A4A` | `#D1B79A` | `#D5A83E` | `#473332` | Đội giúp đỡ, hành động |
| Rừng mưa | `#39775E` | `#8FB88B` | `#D68D52` | `#213F35` | Thám hiểm, sinh vật rừng |
| Băng và san hô | `#8BBBC8` | `#CCDDE1` | `#D77468` | `#2B4855` | Băng tuyết, đồ bảo hộ |
| Mùa thu | `#B66B3D` | `#D8AE58` | `#7A8C59` | `#49362D` | Lá cây, đi dạo |
| Mận chín | `#82506F` | `#C897AE` | `#D6B46B` | `#3D293D` | Nhân vật bí ẩn, lịch thiệp |
| Lễ hội giấy | `#D55B52` | `#E2B84F` | `#478E8B` | `#453534` | Trang trí giấy, biểu cảm vui |
| Đồ chơi gỗ | `#B88557` | `#D2AF7E` | `#6C91A2` | `#4D382A` | Chất liệu gỗ, đồ vật nhân hóa |
| Vũ trụ kẹo | `#806AB4` | `#D79EC5` | `#73C1BB` | `#3A3259` | Fantasy vui nhộn |
| Kẹo chanh | `#D1BD4E` | `#E8D998` | `#8A9A4C` | `#4F492C` | Đồ ăn biết đi, năng lượng nhẹ |
| Than và hổ phách | `#494A4C` | `#98928A` | `#D59A3B` | `#22262A` | Đơn giản, tương phản bằng màu nhấn |
| Giấy mực đỏ | `#CEBEA4` | `#8E887A` | `#BD594D` | `#343B3D` | Sách, thư tín, thủ công |
| Hồ tím | `#7776AD` | `#B5B5D6` | `#E1B579` | `#353651` | Thư giãn, âm nhạc |
| Nắng trên mái | `#D9A055` | `#B9785D` | `#66979A` | `#514139` | Cuộc sống thành thị, ngày nghỉ |

## 3. Tạo biến thể mà vẫn nhận ra cùng nhân vật

Các bước đề xuất, không phải thuật toán đã có:

1. Giữ màu nhận diện chính và hình dáng; thay màu nhấn của phụ kiện theo chủ đề.
2. Giữ bảng màu nhưng đổi chất liệu: cùng mèo màu trà xanh qua giấy cắt, nỉ và đất sét có thể cho ba hướng khác nhau.
3. Giữ độ sáng tương đối của mảng nhận diện khi thay hue; không chỉ xem toàn bộ ảnh ở kích thước lớn.
4. Nếu đổi cả bảng màu và chất liệu, giữ một dấu nhận diện đủ rõ như tai lệch hoặc chiếc khăn đặc trưng.
5. Đọc hình ở thumbnail trên cả nền sáng/tối. Không dựa chỉ vào màu để truyền cảm xúc hoặc phân biệt nhân vật.

Ví dụ: “Cà phê phin” dùng màu nhấn vàng cho phụ kiện một bộ đi làm; một bộ ngày mưa có thể thay màu nhấn bằng xanh của áo mưa, nhưng giữ màu lông và kính. Đây là phép sáng tạo thủ công, không phải kết quả AI đã kiểm chứng.

## 4. Mẫu diễn đạt màu trong prompt

> Keep the character's main fur coffee brown, with lighter caramel patches, a warm golden accent on the scarf and dark brown drawn outlines. Treat the palette as visual guidance. Use a separate plain white sheet background for extraction.

- Mã màu có thể đi kèm nếu công cụ hiểu, nhưng không thay cho mô tả vai trò.
- Không yêu cầu “đúng tuyệt đối mã hex” nếu chưa có bước hậu kỳ kiểm màu.
- Không yêu cầu cả màu chủ đạo “đơn sắc” và năm màu nhấn rực trong cùng mô tả mà không giải thích chủ ý.

## 5. Nguồn và giới hạn

- [W3C CSS Color Module Level 4](https://www.w3.org/TR/css-color-4/) hỗ trợ cách ghi mã màu; không bảo chứng các bảng biên soạn ở đây.
- [W3C Use of Color](https://www.w3.org/WAI/WCAG22/Understanding/use-of-color.html) là nguồn cho nguyên tắc không chỉ dùng màu truyền thông tin; áp dụng sang nhận diện sticker là đề xuất thiết kế.
- [OpenAI image generation limitations](https://platform.openai.com/docs/guides/image-generation#limitations) nêu rủi ro consistency/composition; không có cam kết giữ màu chính xác.
- Toàn bộ 24 bảng màu là ví dụ mới do người biên soạn chọn. Chưa render, đo tương phản, chạy AI hoặc cắt ảnh. Không gán các palette thành màu đại diện chính thức cho bất kỳ quốc gia hay cộng đồng nào.
