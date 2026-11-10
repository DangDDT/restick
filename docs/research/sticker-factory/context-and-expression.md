# Thư viện ngữ cảnh & biểu đạt — Sticker Factory

Ngày biên soạn: 2026-11-10 · Trạng thái: **tài liệu nghiên cứu đề xuất, chưa triển khai** · Phạm vi: chỉ tài liệu, không kèm mã nguồn, schema hay cấu hình.

Mục tiêu: cung cấp một kho "mảnh ngữ cảnh" (context fragments) để một người vận hành Sticker Factory — quy trình soạn prompt thủ công rồi dán sang công cụ AI tạo ảnh bên ngoài — có thể lắp nhanh một sticker mới mà vẫn giữ được giọng điệu và bản sắc của bộ. Tài liệu **không** hứa hẹn AI sẽ tuân thủ prompt, không khẳng định danh sách là đầy đủ, và không thay thế việc kiểm tra bản quyền/ngữ cảnh văn hoá trước khi phát hành.

---

## 0. Cách đọc tài liệu

Ba nhãn xuất hiện xuyên suốt:

| Nhãn | Nghĩa |
|---|---|
| `[NGUỒN]` | Nguyên tắc rút ra từ tài liệu chuẩn (Unicode/CLDR, ISO/UN, UNESCO, tài liệu nhà cung cấp AI). Có URL ở §14. |
| `[GIÁ TRỊ]` | Giá trị do nhóm biên soạn tự đặt ra cho dự án này. Không phải dữ liệu chuẩn, không phải "sự thật phổ quát", không kiểm chứng được bằng nguồn ngoài. |
| `[ĐỀ XUẤT]` | Cách dùng/đề xuất chưa được triển khai trong sản phẩm. |

Quy ước nội dung:

- Mỗi giá trị có **mã** (ví dụ `EM-VUI-2`, `GES-TAY-07`) để sau này đối chiếu khi viết prompt, báo lỗi, hoặc xây thư viện tra cứu. Mã là quy ước của tài liệu này, không phải khái niệm chuẩn.
- Cột **mảnh prompt** viết bằng tiếng Anh, dạng cụm ngắn, **mô tả hình thể nhìn thấy được** (vai, mắt, tay, tư thế) thay vì nhãn cảm xúc trừu tượng. Lý do: mô hình ảnh bám vào mô tả thị giác tốt hơn nhãn tâm lý; đây là kinh nghiệm biên soạn `[GIÁ TRỊ]`, không phải kết luận đo lường.
- Nhãn tiếng Việt dùng để người vận hành chọn nhanh, không phải để dán vào prompt.
- Các giá trị **không** phải thuộc tính của một quốc gia hay một dân tộc. Khi một mục gắn với một nền văn hoá cụ thể, đó là **ví dụ có nguồn gốc**, không phải biểu tượng đại diện cho toàn bộ con người nơi đó.

Tổng dung lượng: khoảng **330 giá trị** trong 8 nhóm lớn (đếm theo dòng có mã), cộng 5 bộ sticker mẫu và 1 khung công thức prompt. Con số này mô tả phạm vi tài liệu, không phải chỉ tiêu chất lượng hay độ phủ.

Thư viện này để **hỗ trợ sáng tạo có hiểu biết**, không phải một policy engine: không mục nào là luật, là lệnh cấm, hay là kết luận pháp lý. Các mục `A/B/C` ở §9 chỉ nói về độ tin cậy nghiên cứu và mức cần cẩn trọng.

---

## 1. Nguyên tắc từ nguồn (phần này khác phần giá trị)

### 1.1 Emoji & CLDR: dùng làm phân loại, không dùng làm "ý nghĩa cảm xúc phổ quát" `[NGUỒN]`

- Bộ dữ liệu kiểm thử emoji của Unicode chia emoji thành **10 nhóm** (Smileys & Emotion, People & Body, Component, Animals & Nature, Food & Drink, Travel & Places, Activities, Objects, Symbols, Flags) và nhiều **nhóm con** như `face-smiling`, `face-concerned`, `hand-fingers-open`, `person-gesture`, `place-building`. Đây là **cấu trúc dữ liệu**, không phải bảng ánh xạ cảm xúc.
- UTS #51 mô tả emoji có **biến thể màu da**, **chuỗi ZWJ** (ghép nhiều thành phần thành một hình), và **bộ chọn biến thể** VS15/VS16 (buộc hiển thị dạng văn bản hoặc dạng emoji). Nghĩa: cùng một "ý" có thể được biểu diễn bằng nhiều hình khác nhau, và một hình có thể mang nhiều tầng ghép.
- **Annotation của CLDR là nhãn dùng cho tìm kiếm và nhóm**, không phải định nghĩa ngữ nghĩa. Vì vậy: dùng nhóm emoji làm **khung phân loại** cho thư viện (dễ duyệt, dễ mở rộng), nhưng **không** suy ra "emoji X = cảm xúc Y ở mọi nền văn hoá". Cùng một khuôn mặt có thể đọc là thân thiện ở nơi này và khiếm nhã ở nơi khác.

Hệ quả thiết kế cho Sticker Factory: thư viện dưới đây chia nhóm **theo chức năng giao tiếp** (cảm xúc, cử chỉ, chức năng hội thoại, quan hệ, tình huống, thời gian, dịp) — bám theo cách con người *dùng* sticker, chứ không bám theo danh mục emoji.

### 1.2 Địa lý: lấy từ danh sách chuẩn, không tự phát minh `[NGUỒN]`

- **ISO 3166-1** cung cấp mã quốc gia (alpha-2/alpha-3) — dùng làm khoá ổn định khi cần mở rộng.
- **UNSD M49** cung cấp phân vùng địa lý/kinh tế (ví dụ Đông Nam Á, Nam Á, Châu Đại Dương…) — dùng làm khung nhóm cấp 1.
- **CLDR** cung cấp tên vùng lãnh thổ bản địa hoá (ví dụ tên tiếng Việt của quốc gia) — dùng làm nhãn hiển thị, tránh tự dịch sai.

Nguyên tắc: một mục địa lý mới chỉ được thêm khi (a) truy được về một mã ISO 3166 hoặc một vùng M49, và (b) có ít nhất một nguồn cụ thể cho chi tiết văn hoá đi kèm.

### 1.3 UNESCO: di sản là thực hành sống, không phải hoạ tiết trang trí `[NGUỒN]`

- Công ước 2003 về di sản văn hoá phi vật thể ghi nhận các thực hành do **cộng đồng** nắm giữ và trao truyền; danh sách gồm Representative List, Urgent Safeguarding List và Register of Good Safeguarding Practices.
- Hệ quả: khi một sticker mô phỏng trang phục, nhạc cụ, món ăn, nghề thủ công hay nghi lễ có nguồn gốc cộng đồng cụ thể, cần (1) ghi nguồn, (2) tránh mô phỏng hoa văn linh thiêng hoặc biểu tượng nghi lễ, (3) tránh biến một thực hành sống thành "đạo cụ hài".
- UNESCO cũng vận hành Danh sách Di sản Thế giới (WHC) cho công trình/di chỉ — khác hẳn ICH về bản chất. Không trộn hai loại này khi ghi nguồn.

### 1.4 Giới hạn của công cụ tạo ảnh: thiết kế prompt để chịu lỗi `[NGUỒN]`

Tài liệu Image generation của OpenAI (2026) liệt kê các giới hạn còn tồn tại: **chữ trong ảnh** vẫn có thể sai vị trí/không rõ; **độ nhất quán nhân vật** giữa nhiều lần sinh có thể trượt; **kiểm soát bố cục** trong ảnh có cấu trúc còn hạn chế. Ngoài ra mô hình có thể bị chặn bởi moderation.

Hệ quả cho Sticker Factory `[ĐỀ XUẤT]`:

1. **Một sticker = một hành vi rõ**, không nhồi 3 ý.
2. **Không dựa vào AI để viết chữ**; chữ (nếu có) là lớp hậu kỳ, hoặc xuất ảnh không chữ.
3. **Bản sắc bộ** phải nằm ở thứ AI tái tạo ổn định: độ dày nét, bảng màu, kiểu mắt, tỉ lệ đầu/thân, kiểu đổ bóng — không nằm ở "phong cách chung chung".
4. Dự trù **nhiều lần thử cho mỗi sticker**. Tài liệu này không có tỉ lệ lần sinh đạt nào được đo, và không có "chi phí chuẩn" nào để trích dẫn; số lần phải thử phụ thuộc công cụ, độ phức tạp hình và mức khó tính của người duyệt.

---

## 2. Cảm xúc × cường độ

Đây là trục quan trọng nhất, và cũng là nơi dễ hỏng nhất: người viết prompt hay ghi `happy` rồi nhận về một khuôn mặt chung chung. Cách của tài liệu này: chọn **họ cảm xúc** (12 họ), rồi chọn **cường độ** (4 mức), rồi lấy **mảnh hình thể**.

Bốn mức cường độ dùng chung:

| Mức | Mã | Nghĩa | Dấu hiệu thị giác |
|---|---|---|---|
| Nén | `0` | Có cảm xúc nhưng che giấu | Cơ mặt căng nhẹ, mắt né, tay giữ |
| Nhẹ | `1` | Thoáng qua | Một nét mặt, thân không đổi |
| Vừa | `2` | Rõ, đọc được ngay | Vai/thân tham gia |
| Mạnh | `3` | Chiếm toàn thân | Tư thế lệch, tay chân mất trật tự |

### 2.1 Vui `EM-VUI`

| Mã | Cường độ | Nhãn | Mảnh prompt |
|---|---|---|---|
| EM-VUI-0 | Nén | Vui giấu | `pressed lips, eyes bright, shoulders held still` |
| EM-VUI-1 | Nhẹ | Hài lòng | `soft closed-mouth smile, relaxed brow` |
| EM-VUI-2 | Vừa | Vui rõ | `open smile, cheeks raised, shoulders lifted` |
| EM-VUI-3 | Mạnh | Bùng nổ | `both arms up, head back, mouth wide open` |

### 2.2 Buồn `EM-BUON`

| Mã | Cường độ | Nhãn | Mảnh prompt |
|---|---|---|---|
| EM-BUON-0 | Nén | Ngậm | `flat mouth, gaze lowered, blink held` |
| EM-BUON-1 | Nhẹ | Hơi buồn | `inner brow raised, small frown` |
| EM-BUON-2 | Vừa | Buồn | `head tilted down, shoulders curved forward` |
| EM-BUON-3 | Mạnh | Đau lòng | `face in hands, body folded, knees bent` |

### 2.3 Giận `EM-GIAN`

| Mã | Cường độ | Nhãn | Mảnh prompt |
|---|---|---|---|
| EM-GIAN-0 | Nén | Bực kín | `jaw tight, nostrils flared, stare fixed` |
| EM-GIAN-1 | Nhẹ | Khó chịu | `one brow lowered, mouth pulled to one side` |
| EM-GIAN-2 | Vừa | Giận | `brows down hard, hands on hips, feet apart` |
| EM-GIAN-3 | Mạnh | Nổ | `mouth open shouting, veins at temple, fists shaking` |

### 2.4 Lo `EM-LO`

| Mã | Cường độ | Nhãn | Mảnh prompt |
|---|---|---|---|
| EM-LO-0 | Nén | Canh chừng | `eyes wide and still, head slightly forward` |
| EM-LO-1 | Nhẹ | Băn khoăn | `brows pinched, lips bitten` |
| EM-LO-2 | Vừa | Lo | `hands clasped at chest, shoulders raised` |
| EM-LO-3 | Mạnh | Hoảng | `curled in a ball, eyes squeezed shut, arms over head` |

### 2.5 Sợ `EM-SO`

| Mã | Cường độ | Nhãn | Mảnh prompt |
|---|---|---|---|
| EM-SO-0 | Nén | Rén | `body leaning away, one hand lifted` |
| EM-SO-1 | Nhẹ | Giật mình | `shoulders jumped, eyes wide` |
| EM-SO-2 | Vừa | Sợ | `stepped back, arms crossed in front` |
| EM-SO-3 | Mạnh | Kinh hãi | `frozen mid-step, mouth open, hair standing` |

### 2.6 Ngạc nhiên `EM-NGA`

| Mã | Cường độ | Nhãn | Mảnh prompt |
|---|---|---|---|
| EM-NGA-0 | Nén | Để ý | `single brow raised, head tilt` |
| EM-NGA-1 | Nhẹ | Ồ | `lips parted, eyes slightly wider` |
| EM-NGA-2 | Vừa | Bất ngờ | `both brows up, head pulled back` |
| EM-NGA-3 | Mạnh | Sốc | `hands on cheeks, body arched back, mouth round` |

### 2.7 Tự hào `EM-TUHAO`

| Mã | Cường độ | Nhãn | Mảnh prompt |
|---|---|---|---|
| EM-TUHAO-0 | Nén | Kín đáo | `small nod, chest lifted a little` |
| EM-TUHAO-1 | Nhẹ | Hài lòng với mình | `chin up, one corner of mouth raised` |
| EM-TUHAO-2 | Vừa | Tự hào | `chest out, hands behind back` |
| EM-TUHAO-3 | Mạnh | Chiến thắng | `fist raised, one knee up, wide grin` |

### 2.8 Xấu hổ `EM-XAUHO`

| Mã | Cường độ | Nhãn | Mảnh prompt |
|---|---|---|---|
| EM-XAUHO-0 | Nén | Ngượng | `eyes flicked aside, hand at back of neck` |
| EM-XAUHO-1 | Nhẹ | Ngại | `cheeks flushed, small awkward smile` |
| EM-XAUHO-2 | Vừa | Xấu hổ | `face turned away, shoulders hunched` |
| EM-XAUHO-3 | Mạnh | Muốn độn thổ | `crouched low, both hands covering face` |

### 2.9 Biết ơn `EM-BIETON`

| Mã | Cường độ | Nhãn | Mảnh prompt |
|---|---|---|---|
| EM-BIETON-0 | Nén | Ghi nhận | `slow nod, hand on chest` |
| EM-BIETON-1 | Nhẹ | Cảm kích | `soft eyes, head bowed slightly` |
| EM-BIETON-2 | Vừa | Biết ơn | `both hands together, small bow` |
| EM-BIETON-3 | Mạnh | Tràn ơn | `deep bow, hands pressed, eyes closed` |

### 2.10 Cô đơn `EM-CODon`

| Mã | Cường độ | Nhãn | Mảnh prompt |
|---|---|---|---|
| EM-CODon-0 | Nén | Lạc lõng | `standing apart, hands in pockets` |
| EM-CODon-1 | Nhẹ | Trống | `gaze at nothing, shoulders slack` |
| EM-CODon-2 | Vừa | Cô đơn | `sitting alone, knees drawn up, small silhouette` |
| EM-CODon-3 | Mạnh | Trống rỗng | `lying curled, phone face-down, dim light` |

### 2.11 Yêu thương `EM-YEU`

| Mã | Cường độ | Nhãn | Mảnh prompt |
|---|---|---|---|
| EM-YEU-0 | Nén | Để ý lặng | `watching from the side, soft eyes` |
| EM-YEU-1 | Nhẹ | Mến | `head tilted, warm half smile` |
| EM-YEU-2 | Vừa | Thương | `hands cupped near chest, eyes closed` |
| EM-YEU-3 | Mạnh | Ngập tràn | `arms open wide, face buried in a hug` |

### 2.12 Chán/Bình thản `EM-CHAN`

| Mã | Cường độ | Nhãn | Mảnh prompt |
|---|---|---|---|
| EM-CHAN-0 | Nén | Chờ | `flat stare, slow blink` |
| EM-CHAN-1 | Nhẹ | Nhạt | `half-lidded eyes, mouth a straight line` |
| EM-CHAN-2 | Vừa | Chán | `slumped over a desk, cheek on hand` |
| EM-CHAN-3 | Mạnh | Tan chảy | `melted posture, arms hanging, head thrown back` |

### 2.13 Cảm xúc trộn (bổ sung, không nhân cường độ) `EM-TRON`

| Mã | Nhãn | Mảnh prompt |
|---|---|---|
| EM-TRON-1 | Vui buồn lẫn lộn | `smiling with tears at the lash line` |
| EM-TRON-2 | Cười trong lo | `laughing, one hand gripping the other wrist` |
| EM-TRON-3 | Tức mà thương | `brows angry, mouth trying not to smile` |
| EM-TRON-4 | Nhẹ nhõm sau sợ | `exhale, hand on chest, head dropped forward` |
| EM-TRON-5 | Tự trào | `deadpan face, one eyebrow up, small shrug` |
| EM-TRON-6 | Lịch sự mà mệt | `polite smile, eyes not participating` |

**Đề xuất dùng** `[ĐỀ XUẤT]`: mỗi bộ sticker chỉ nên có **tối đa 2 cảm xúc mức 3**. Cảm xúc mạnh ở quá nhiều sticker làm bộ trông "hét" liên tục và giảm khả năng dùng trong chat công việc.

---

## 3. Cử chỉ & hành động

Sticker sống bằng hành động nhìn thấy được. Nhóm dưới đây chia theo bộ phận/tương tác; mỗi mảnh prompt cố tình ngắn để ghép được với mảnh cảm xúc.

### 3.1 Bàn tay `GES-TAY`

| Mã | Nhãn | Mảnh prompt |
|---|---|---|
| GES-TAY-01 | Xoè tay mời | `open palm up, fingers relaxed` |
| GES-TAY-02 | Giơ ngón cái | `thumb up at chest height` |
| GES-TAY-03 | Ngón cái chỉ xuống | `thumb down, flat mouth` |
| GES-TAY-04 | Vòng OK | `thumb and index forming a ring, other fingers fanned` |
| GES-TAY-05 | Hai ngón | `index and middle up, wrist loose` |
| GES-TAY-06 | Nắm tay giơ lên | `fist raised at shoulder height` |
| GES-TAY-07 | Vẫy | `palm open, arm raised, small motion lines` |
| GES-TAY-08 | Chặn lại | `palm forward at face height, body leaning back` |
| GES-TAY-09 | Chỉ tay | `index pointing to the side, head following` |
| GES-TAY-10 | Đan tay | `fingers laced, hands on table` |
| GES-TAY-11 | Trái tim ngón tay | `thumb and index crossed, tiny heart` |
| GES-TAY-12 | Bắt tay | `two hands clasped, forearms level` |
| GES-TAY-13 | Đếm bằng ngón | `three fingers up, brow raised` |
| GES-TAY-14 | Vuốt màn hình | `thumb swiping, phone tilted` |
| GES-TAY-15 | Gõ máy tính | `both hands on keyboard, shoulders forward` |
| GES-TAY-16 | Ký giấy | `pen in hand, wrist on paper` |

### 3.2 Thân & đầu `GES-THAN`

| Mã | Nhãn | Mảnh prompt |
|---|---|---|
| GES-THAN-01 | Nhún vai | `shoulders up, palms out, mouth flat` |
| GES-THAN-02 | Nghiêng đầu | `head tilted 15 degrees, eyes level` |
| GES-THAN-03 | Gật | `chin down, eyes closing` |
| GES-THAN-04 | Lắc | `head turned twice, one hand up` |
| GES-THAN-05 | Cúi nhẹ | `small bow from the waist, arms at sides` |
| GES-THAN-06 | Ngả người ra sau | `back arched, hands behind head` |
| GES-THAN-07 | Rướn tới trước | `torso forward, both hands on the table` |
| GES-THAN-08 | Quay lưng | `back turned, head over shoulder` |
| GES-THAN-09 | Ôm đầu | `both hands on the head, elbows wide` |
| GES-THAN-10 | Vỗ trán | `palm flat on forehead, eyes shut` |

### 3.3 Vi biểu `GES-MAT`

| Mã | Nhãn | Mảnh prompt |
|---|---|---|
| GES-MAT-01 | Nhướng một bên mày | `one brow up, other level` |
| GES-MAT-02 | Mím môi | `lips pressed thin, chin lifted` |
| GES-MAT-03 | Phồng má | `cheeks puffed, lips sealed` |
| GES-MAT-04 | Nheo mắt | `eyes narrowed, head slightly back` |
| GES-MAT-05 | Đảo mắt | `pupils up and to the side, lids low` |
| GES-MAT-06 | Liếc | `eyes sideways, face forward` |
| GES-MAT-07 | Cắn môi | `lower lip between teeth` |
| GES-MAT-08 | Nhăn mặt | `nose wrinkled, one eye squinted` |
| GES-MAT-09 | Há miệng không tiếng | `mouth open, brows up, no sound lines` |
| GES-MAT-10 | Thở dài | `lips parted, one long breath line` |

### 3.4 Cầm nắm đồ vật `GES-DO`

| Mã | Nhãn | Mảnh prompt |
|---|---|---|
| GES-DO-01 | Ôm điện thoại | `phone held to chest with both hands` |
| GES-DO-02 | Nâng cốc | `mug lifted with both hands, steam curling` |
| GES-DO-03 | Che ô | `umbrella tilted forward, rain streaks` |
| GES-DO-04 | Rút khăn giấy | `tissue pulled from a box, eyes wet` |
| GES-DO-05 | Bê thùng carton | `box hugged, chin over the edge` |
| GES-DO-06 | Ôm laptop | `laptop closed, hugged like a pillow` |
| GES-DO-07 | Mở bản đồ | `paper map unfolded, head down` |
| GES-DO-08 | Cầm vé | `ticket held up at eye level` |
| GES-DO-09 | Xách túi | `bag handles in one hand, arm straight` |
| GES-DO-10 | Tưới cây | `watering can tilted, one leaf trembling` |
| GES-DO-11 | Bưng bát | `bowl cupped in one palm, chopsticks in the other` |
| GES-DO-12 | Đội mũ bảo hiểm | `helmet mid-air above head, strap swinging` |

### 3.5 Trạng thái cơ thể `GES-THE`

| Mã | Nhãn | Mảnh prompt |
|---|---|---|
| GES-THE-01 | Nằm bẹp | `lying face down, limbs spread` |
| GES-THE-02 | Co tròn | `curled on the side, knees to chest` |
| GES-THE-03 | Ngồi bệt | `sitting on the floor, legs out front` |
| GES-THE-04 | Nhón chân | `on tiptoe, arms out for balance` |
| GES-THE-05 | Nhảy giữa không | `mid-air jump, feet apart, shadow below` |
| GES-THE-06 | Lê bước | `dragging feet, shoulders low` |
| GES-THE-07 | Chạy | `mid-stride run, hair and clothes trailing` |
| GES-THE-08 | Đứng hình | `frozen posture, one foot lifted, eyes wide` |
| GES-THE-09 | Ngồi xổm chờ | `squatting, elbows on knees` |
| GES-THE-10 | Ngả lưng ghế | `sprawled on a chair, legs over the armrest` |

### 3.6 Tương tác hai chủ thể `GES-CAP`

| Mã | Nhãn | Mảnh prompt |
|---|---|---|
| GES-CAP-01 | Đưa nhận | `one offering an object, other reaching` |
| GES-CAP-02 | Kéo tay | `one pulling the other's sleeve` |
| GES-CAP-03 | Ôm từ sau | `arms wrapped from behind, chins close` |
| GES-CAP-04 | Cõng | `one on the other's back, arms around neck` |
| GES-CAP-05 | Cụng đầu | `foreheads touching, eyes closed` |
| GES-CAP-06 | Chen ô | `two under one umbrella, one shoulder wet` |
| GES-CAP-07 | Đập tay | `palms meeting above head` |
| GES-CAP-08 | Khoác vai | `arm over shoulders, bodies angled in` |
| GES-CAP-09 | Chuyền đồ | `object mid-air between two hands` |
| GES-CAP-10 | Cãi nhẹ | `two faces turned away, arms crossed` |

**Ghi chú văn hoá** `[GIÁ TRỊ]`: cùng một cử chỉ có thể mang nghĩa khác nhau giữa các cộng đồng (ví dụ cử chỉ tay, khoảng cách khi đứng gần, việc chạm vào đầu người khác). Với sticker dùng nội bộ một nhóm nhỏ thì rủi ro thấp; với sticker phát hành rộng, nên tránh các cử chỉ dễ bị đọc thành xúc phạm và tránh mô phỏng cử chỉ nghi lễ.

---

## 4. Chức năng hội thoại

Một sticker thường không "diễn cảm xúc" mà **làm một việc** trong đoạn chat: chào, từ chối, nhắc, chốt. Bảng này lấp khoảng trống giữa cảm xúc và hành động.

| Mã | Chức năng | Biến thể | Mảnh prompt / ghi chú |
|---|---|---|---|
| CF-CHAO-1 | Chào | Chào thân mật | `hand raised, big smile, one foot forward` |
| CF-CHAO-2 | Chào | Chào lịch sự | `small bow, hands at sides, neutral smile` |
| CF-TAMBIET-1 | Tạm biệt | Vẫy nhanh | `waving hand, body already turning away` |
| CF-TAMBIET-2 | Tạm biệt | Vẫy lâu | `both arms up, standing still, motion lines` |
| CF-CAMON-1 | Cảm ơn | Nhẹ | `hand on chest, nod` |
| CF-CAMON-2 | Cảm ơn | Trịnh trọng | `both hands together, bow, eyes lowered` |
| CF-XINLOI-1 | Xin lỗi | Nhỏ | `hands clasped in front, shoulders up` |
| CF-XINLOI-2 | Xin lỗi | Thành khẩn | `deep bow, hands on knees` |
| CF-DONGY-1 | Đồng ý | Gật mạnh | `double nod motion lines, thumb up` |
| CF-DONGY-2 | Đồng ý | Đồng ý miễn cưỡng | `flat smile, slow nod, eyes aside` |
| CF-TUCHOI-1 | Từ chối | Mềm | `palms together, small shake of head` |
| CF-TUCHOI-2 | Từ chối | Dứt khoát | `X formed with forearms, chin down` |
| CF-KHEN-1 | Khen | Thật lòng | `hands on cheeks, eyes shining` |
| CF-KHEN-2 | Khen | Trêu nhẹ | `slow clap, one brow up` |
| CF-ANUI-1 | An ủi | Vỗ về | `hand patting the air, soft eyes` |
| CF-ANUI-2 | An ủi | Ở cạnh | `sitting close, shoulder to shoulder, no words` |
| CF-CHUCMUNG-1 | Chúc mừng | Bùng nổ | `confetti in both hands, jump` |
| CF-CHUCMUNG-2 | Chúc mừng | Ấm | `cake held forward, candle flame` |
| CF-MOI-1 | Mời | Mời ăn | `plate pushed forward, chopsticks offered` |
| CF-MOI-2 | Mời | Mời vào | `door held open, arm swept inward` |
| CF-NHAC-1 | Nhắc nhẹ | Nhắc deadline | `tapping a watch, eyebrows high` |
| CF-NHAC-2 | Nhắc nhẹ | Nhắc lần hai | `leaning over a shoulder, finger raised` |
| CF-HEN-1 | Hẹn | Chốt lịch | `calendar page, circle drawn, pen up` |
| CF-HEN-2 | Hẹn | Hẹn lại | `phone at ear, other hand waving off` |
| CF-HOITHAM-1 | Hỏi thăm | Quan tâm | `head tilted, hand reaching forward` |
| CF-HOITHAM-2 | Hỏi thăm | Hỏi dò | `magnifier raised, one eye squinted` |
| CF-XACNHAN-1 | Xác nhận | Đã hiểu | `OK ring, firm nod` |
| CF-XACNHAN-2 | Xác nhận | Đã nhận | `two hands holding an object, thumbs up` |
| CF-THUNHAN-1 | Thú nhận | Ngại | `hand behind head, eyes up` |
| CF-THUNHAN-2 | Thú nhận | Thẳng | `palms up, shoulders square, flat mouth` |
| CF-BIMAT-1 | Giữ bí mật | Rào trước | `finger to lips, eyes to the side` |
| CF-BIMAT-2 | Giữ bí mật | Đã hứa | `pinky interlocked, other hand on chest` |
| CF-DUA-1 | Đùa | Chọc nhẹ | `tongue out, one eye closed` |
| CF-DUA-2 | Đùa | Đùa dai | `poking with one finger, grin` |
| CF-KETTHUC-1 | Kết thúc | Chốt | `hands brushed together, satisfied face` |
| CF-KETTHUC-2 | Kết thúc | Rút lui | `walking away, hand waving behind` |

**Ghi chú** `[GIÁ TRỊ]`: nhóm chức năng hội thoại hữu ích khi bộ sticker phục vụ trao đổi ngắn trong công việc; nhóm cảm xúc mạnh hợp với trao đổi thân mật. Tài liệu này **không có dữ liệu** về việc nhóm nào được dùng nhiều nhất. Nếu cần một cách chia để bắt đầu, xem các ví dụ ở §11 — chúng chỉ minh hoạ cách dùng thư viện, không phải chuẩn.

---

## 5. Quan hệ, giọng điệu, hài

### 5.1 Cặp quan hệ `REL`

| Mã | Quan hệ | Mảnh prompt / lưu ý |
|---|---|---|
| REL-01 | Bạn thân | `two figures shoulder to shoulder, one leaning on the other` |
| REL-02 | Đồng nghiệp | `two figures at a desk, equal height, neutral posture` |
| REL-03 | Sếp – nhân viên | `one seated, one standing, slight forward lean` |
| REL-04 | Thầy – trò | `one pointing at a board, one taking notes` |
| REL-05 | Gia đình nhiều thế hệ | `three heights, hands linked in a chain` |
| REL-06 | Anh chị em | `two figures, one holding an object above the other's reach` |
| REL-07 | Cặp đôi | `two figures close, one hand at the other's elbow` |
| REL-08 | Hàng xóm | `two figures over a low fence, one holding a bowl` |
| REL-09 | Người lạ – giúp đỡ | `one holding a dropped item out, other reaching` |
| REL-10 | Nhóm chat đông người | `three or more small figures, overlapping speech bubbles` |
| REL-11 | Chủ – thú nuôi | `small animal at ankle height, human hand reaching down` |
| REL-12 | Khách – người phục vụ | `one seated at a table, one placing a cup, both smiling` |

**Lưu ý** `[GIÁ TRỊ]`: quan hệ trong sticker nên diễn đạt bằng **tư thế và khoảng cách**, không bằng phụ kiện định kiến (ví dụ không dùng "nữ = váy", "lãnh đạo = vest"). Tư thế đọc được ở mọi nền văn hoá và giảm rủi ro.

### 5.2 Giọng điệu `TON`

| Mã | Giọng | Mảnh prompt |
|---|---|---|
| TON-01 | Thân mật | `loose posture, asymmetric smile` |
| TON-02 | Lịch sự | `upright posture, hands visible, soft smile` |
| TON-03 | Trang trọng | `straight back, chin level, minimal gesture` |
| TON-04 | Mỉa nhẹ | `flat mouth, one brow raised, eyes forward` |
| TON-05 | Tự trào | `slumped shoulders, palm on own forehead` |
| TON-06 | Hài vô tri | `blank stare, object doing something impossible` |
| TON-07 | Dịu dàng | `rounded shoulders, slow hand movement` |
| TON-08 | Sốt ruột | `weight on one leg, fingers drumming` |
| TON-09 | Điềm tĩnh | `hands folded, level gaze` |
| TON-10 | Phấn khích | `bouncing on toes, hands flapping` |
| TON-11 | Chán nản | `head hanging, arms slack` |
| TON-12 | Bí ẩn | `half face in shadow, finger at lips` |

### 5.3 Cơ chế hài `HUM`

| Mã | Cơ chế | Mảnh prompt | Ví dụ ý (không phải prompt) |
|---|---|---|---|
| HUM-01 | Phóng đại | `object comically oversized in both arms` | Ôm cái deadline to hơn người |
| HUM-02 | Tương phản | `tiny figure beside huge object, calm face` | Bình thản giữa đống giấy |
| HUM-03 | Đảo vai | `object acting on the human` | Laptop đang "nhìn" người |
| HUM-04 | Vật vô tri có cảm xúc | `houseplant leaning away` | Cây nghiêng tránh |
| HUM-05 | Chậm một nhịp | `blank face, delayed reaction lines` | Mất 2 giây mới hiểu |
| HUM-06 | Trang trọng cho chuyện nhỏ | `formal bow over a single peanut` | Cúi người trước hạt đậu |
| HUM-07 | Chuỗi thất bại | `three small objects falling in sequence` | Cốc → bút → lòng tự trọng |
| HUM-08 | Tự trào nghề nghiệp | `badge worn like a medal, tired eyes` | "Chuyên gia sửa lỗi lần 4" |
| HUM-09 | Chơi chữ thị giác | `literal depiction of an idiom` | Nước đến chân mới nhảy |
| HUM-10 | Bất ngờ im lặng | `hand raised, no motion lines at all` | Giơ tay xin phát biểu rồi im |

**Lưu ý** `[GIÁ TRỊ]`: hài dựa trên **chơi chữ** phụ thuộc ngôn ngữ, nên nếu sticker không có chữ thì phải dựa trên **thị giác** (HUM-01 → 04, 07, 10). Hài dựa trên ngoại hình người thật, nghề nghiệp, vùng miền hay giọng nói địa phương dễ thành định kiến — tránh.

---

## 6. Tình huống tự sự

Mỗi tình huống có 3 nhịp: **trước** (nguyên nhân), **giữa** (phản ứng), **sau** (dư âm). Một sticker chỉ nên lấy **một nhịp**; một bộ có thể lấy cả ba để kể chuyện.

| Mã | Tình huống | Nhịp | Mảnh prompt |
|---|---|---|---|
| SIT-01a | Sát deadline | Trước | `clock in hand, eyes on the minute hand` |
| SIT-01b | Sát deadline | Giữa | `four windows open, hands in hair` |
| SIT-01c | Sát deadline | Sau | `head on desk, keyboard imprint on cheek` |
| SIT-02a | Mất kết nối | Trước | `router light blinking, face close to it` |
| SIT-02b | Mất kết nối | Giữa | `laptop held up to a window, hopeful eyes` |
| SIT-02c | Mất kết nối | Sau | `sitting on the floor, phone in airplane pose` |
| SIT-03a | Hết pin | Trước | `battery icon at one bar, worried glance` |
| SIT-03b | Hết pin | Giữa | `phone tethered to a wall socket, sitting on floor` |
| SIT-03c | Hết pin | Sau | `black screen reflected in glasses` |
| SIT-04a | Quên ô | Trước | `hand on the door, rain outside` |
| SIT-04b | Quên ô | Giữa | `newspaper over head, running` |
| SIT-04c | Quên ô | Sau | `dripping at the doorway, shoes squeaking` |
| SIT-05a | Chờ tin nhắn | Trước | `phone face-up, eyes flicking to it` |
| SIT-05b | Chờ tin nhắn | Giữa | `typing dots imagined above the phone` |
| SIT-05c | Chờ tin nhắn | Sau | `phone face-down, arms crossed` |
| SIT-06a | Sửa lỗi mãi không xong | Trước | `cursor blinking, hand on chin` |
| SIT-06b | Sửa lỗi mãi không xong | Giữa | `stack of empty cups, screen glare` |
| SIT-06c | Sửa lỗi mãi không xong | Sau | `one-line change, wide empty screen, tear` |
| SIT-07a | Đi muộn | Trước | `one shoe on, hopping` |
| SIT-07b | Đi muộn | Giữa | `running with a bag open, papers flying` |
| SIT-07c | Đi muộn | Sau | `arriving breathless, hand on the doorframe` |
| SIT-08a | Mất đồ | Trước | `patting pockets, eyes wide` |
| SIT-08b | Mất đồ | Giữa | `looking under a chair, upside down` |
| SIT-08c | Mất đồ | Sau | `item in the last place checked, flat stare` |
| SIT-09a | Nấu ăn thất bại | Trước | `recipe open, confident posture` |
| SIT-09b | Nấu ăn thất bại | Giữa | `smoke curling, pan held at arm's length` |
| SIT-09c | Nấu ăn thất bại | Sau | `ordering food, apron still on` |
| SIT-10a | Dọn nhà | Trước | `box open, room tidy` |
| SIT-10b | Dọn nhà | Giữa | `sitting inside a box, sorting photos` |
| SIT-10c | Dọn nhà | Sau | `room messier, one shelf perfectly clean` |

**Đề xuất** `[ĐỀ XUẤT]`: dùng nhịp `c` (dư âm) làm sticker "kết" cho mỗi bộ — nhịp này thường hài hơn và ít phụ thuộc ngữ cảnh.

---

## 7. Mùa, thời tiết, thời gian

### 7.1 Mùa `SEA`

| Mã | Nhãn | Mảnh prompt | Lưu ý |
|---|---|---|---|
| SEA-01 | Xuân (ôn đới) | `blossoms falling, light jacket open` | Không phổ quát: nhiều nơi không có 4 mùa |
| SEA-02 | Hạ | `sun overhead, sweat at the temple, fan` | Mùa nóng khác nhau theo bán cầu |
| SEA-03 | Thu | `dry leaves, scarf, low sun` | Ở vùng nhiệt đới, "thu" là quy ước lịch, không phải thời tiết |
| SEA-04 | Đông | `padded coat, breath visible` | Tránh gắn tuyết với mọi nơi có mùa đông |
| SEA-05 | Mùa mưa (nhiệt đới) | `rain sheet, gutter overflow, sandals` | Gắn với lịch cụ thể, không gắn với quốc gia |
| SEA-06 | Mùa khô (nhiệt đới) | `cracked earth, dust on leaves, hat` | Như trên |

### 7.2 Thời tiết `WEA`

| Mã | Nhãn | Mảnh prompt |
|---|---|---|
| WEA-01 | Nắng gắt | `hard shadow, squinting, hand as visor` |
| WEA-02 | Nắng dịu | `long soft shadow, relaxed eyes` |
| WEA-03 | Mưa nhỏ | `single drops on shoulders, no umbrella` |
| WEA-04 | Mưa lớn | `rain streaks, clothes stuck, hair flat` |
| WEA-05 | Giông | `bent tree, coat flapping, one hand on hat` |
| WEA-06 | Sương mù | `soft edges, muted background, slow step` |
| WEA-07 | Gió | `clothes pulled sideways, hair across face` |
| WEA-08 | Lạnh | `shoulders up, hands in sleeves` |
| WEA-09 | Nóng | `collar open, fan in hand, damp forehead` |
| WEA-10 | Ẩm | `window fogged, finger drawing on glass` |
| WEA-11 | Tuyết rơi | `flakes on lashes, mittens` |
| WEA-12 | Bão cát/bụi | `scarf over mouth, dust haze` |
| WEA-13 | Cầu vồng sau mưa | `rainbow arc, wet ground reflecting` |
| WEA-14 | Trong nhà nhìn mưa | `forehead on window, mug in hand` |
| WEA-15 | Ngột ngạt trước giông | `still air, curtains motionless, collar tugged` |
| WEA-16 | Nắng xuyên khe cửa | `light bars across the floor, dust motes` |

### 7.3 Thời điểm trong ngày `TIM`

| Mã | Nhãn | Mảnh prompt |
|---|---|---|
| TIM-01 | Rạng sáng | `blue-grey light, blanket around shoulders` |
| TIM-02 | Sáng sớm | `low warm light, steam from a cup` |
| TIM-03 | Giữa trưa | `overhead light, short shadows` |
| TIM-04 | Chiều muộn | `amber side light, long shadow` |
| TIM-05 | Hoàng hôn | `orange sky band, silhouette edge` |
| TIM-06 | Chạng vạng | `purple-blue, streetlight just on` |
| TIM-07 | Đêm khuya | `single desk lamp, dark room` |
| TIM-08 | Quá giờ đi ngủ | `clock past midnight, wide eyes` |
| TIM-09 | Sáng thứ Hai | `alarm ringing, blanket over head` |

### 7.4 Nhịp & thời lượng `NHP`

| Mã | Nhãn | Mảnh prompt |
|---|---|---|
| NHP-01 | Chớp nhoáng | `one motion arc, speed lines behind` |
| NHP-02 | Chậm rì | `drooping posture, snail beside` |
| NHP-03 | Lặp lại | `three ghosted duplicates behind` |
| NHP-04 | Chờ lâu | `dust on shoulders, cobweb in corner` |
| NHP-05 | Vừa kịp | `hand catching an object at the edge` |
| NHP-06 | Quá muộn | `door closing, hand reaching too late` |

---

## 8. Dịp

Quy tắc: mỗi dịp có **một biểu tượng nhẹ** và **một hành vi**. Biểu tượng dễ thành sáo rỗng; hành vi thì đọc được. Với dịp trang nghiêm (tang lễ, tưởng niệm), **không** làm sticker hài và **không** dùng khuôn mặt cười.

| Mã | Dịp | Mảnh biểu tượng | Mảnh hành vi |
|---|---|---|---|
| OCC-01 | Tết Dương lịch | `calendar page turning` | `hands clasped, hopeful look up` |
| OCC-02 | Tết Nguyên đán | `red envelope held with both hands` | `small bow, both hands offering` |
| OCC-03 | Trung thu | `round lantern on a stick` | `looking up, holding a mooncake box` |
| OCC-04 | Giáng sinh | `small pine sprig` | `hanging one ornament with care` |
| OCC-05 | Sinh nhật | `single candle flame` | `leaning in to blow, hands behind back` |
| OCC-06 | Tốt nghiệp | `rolled certificate tied with a ribbon` | `cap thrown, arms up` |
| OCC-07 | Đám cưới | `two rings on a small tray` | `two hands held, walking forward` |
| OCC-08 | Tân gia | `house key on a tag` | `carrying a box through a door` |
| OCC-09 | Khai trương | `ribbon across a doorway` | `cutting the ribbon, both hands` |
| OCC-10 | Chia tay công việc | `empty desk drawer` | `one last wave from the doorway` |
| OCC-11 | Nghỉ lễ | `suitcase with a tag` | `stretching at the gate` |
| OCC-12 | Quay lại sau nghỉ | `inbox number high` | `sighing into a mug` |
| OCC-13 | Lễ tình nhân (nếu bộ cần) | `folded paper note` | `handing a note, eyes down` |
| OCC-14 | Ngày của mẹ/cha (nếu bộ cần) | `small bouquet wrapped in paper` | `hug from the side` |
| OCC-15 | Tưởng niệm | `single white flower` | `standing still, hands together` |
| OCC-16 | Tang lễ | `folded cloth, no ornament` | `head bowed, hands at sides` |
| OCC-17 | Tốt nghiệp lớp nhỏ / hoàn thành khoá | `small badge pinned` | `two hands clapping quietly` |
| OCC-18 | Ngày đầu đi làm | `lanyard around neck` | `adjusting collar at a mirror` |

**Lưu ý** `[GIÁ TRỊ]`: các dịp như Tết Nguyên đán, Trung thu, Giáng sinh… không thuộc về một quốc gia duy nhất và được nhiều cộng đồng đón theo cách khác nhau. Sticker nên mô tả **hành vi cụ thể** (đưa phong bao, treo đèn) thay vì tuyên bố "đây là phong tục của X". Với dịp tôn giáo, tránh dùng hình ảnh linh thiêng làm trò đùa.

---

## 9. Địa lý & văn hoá (phần riêng, đọc kỹ trước khi dùng)

Phần này hỗ trợ **sáng tạo có hiểu biết**, không phải một bộ quy tắc cho phép/cấm. Bộ sticker theo quốc gia, theo thành phố, dùng cờ, hay dùng giọng hài đều là lựa chọn hợp lệ; điều cần tránh là **tuyên bố thay mặt** một cộng đồng hoặc biến con người thành một dấu hiệu duy nhất.

### 9.1 Quy tắc

1. **Bộ theo quốc gia là hợp lệ; tuyên bố đại diện thì không.** "Bộ sticker Hà Nội" hay "bộ sticker Việt Nam" là chủ đề bình thường. Vấn đề chỉ xuất hiện khi một hình được trình bày như thể nó nói thay cho tất cả người Việt, hoặc khi tính cách, trí tuệ, thứ bậc của một người bị suy ra từ quốc tịch.
2. **Thành phố ≠ cả nước.** Một chi tiết của Hà Nội hay Kyoto là chi tiết của nơi đó; đừng biến nó thành đặc điểm chung của cả quốc gia.
3. **Bối cảnh ≠ phông nền đầy đủ.** Sticker chỉ cần **một mảnh bối cảnh** (một ô cửa, một mặt bàn, một bậc thềm). Vẽ cả phố làm sticker rối và khó cắt.
4. **Ghi nguồn cho chi tiết văn hoá cụ thể.** Nếu không ghi được, hạ nó xuống thành chi tiết chung (chất liệu, kiểu dáng) thay vì tên riêng.
5. **Linh thiêng/nghi lễ: vấn đề là ngữ cảnh và quyền, không phải lệnh cấm cả loại hình.** Mặt nạ nghi lễ, nhạc cụ, trang phục lễ, hoa văn do cộng đồng nắm giữ vẫn có thể xuất hiện trong hình tôn trọng và có nguồn; điều cần tránh là dùng chúng làm đạo cụ gây cười, cắt xén biểu tượng, hoặc tái tạo thiết kế hạn chế khi chưa có phép. Nhã nhạc, haka hay bất kỳ loại hình trình diễn nào **không** bị cấm làm sticker — chỉ cần phân biệt mô tả đời thường với mô phỏng nghi lễ.
6. **Trang phục truyền thống không phải đồng phục dân tộc.** Nếu không chắc, dùng trang phục thường ngày của bối cảnh hiện đại.
7. **Cờ là biểu tượng chính trị, không phải nhãn dán cho con người.** Dùng cờ trong bộ theo quốc gia là hợp lệ; cần vẽ đúng cờ hiện hành, không dùng cờ để gán quốc tịch cho một nhân vật cụ thể, và không dùng cờ sai/thời cũ.
8. **Pha trộn văn hoá có chủ đích được khuyến khích** — xem §13 để phân biệt pha trộn có nguồn với hình gộp chung chung.

### 9.2 Bảng theo vùng (ví dụ có nguồn gốc, không đại diện)

**Về cột "mức dùng"**: đây là mức độ **tin cậy nghiên cứu và mức cần cẩn trọng**, không phải giấy phép sử dụng, không phải kết luận pháp lý, và không phải lệnh cấm.

| Mức | Nghĩa |
|---|---|
| `A` | Nhiều tham chiếu đời sống dễ kiểm chứng; ít rủi ro ngữ cảnh. |
| `B` | Cần kiểm nguồn và ngữ cảnh cụ thể vì chi tiết dễ sai hoặc dễ bị hiểu thành biểu tượng chung. |
| `C` | Cần tham vấn cộng đồng/người giữ thực hành, hoặc có vấn đề quyền/sở hữu chưa rõ. Vẫn có thể làm nếu có thông tin và sự cho phép phù hợp. |

| Mã | Vùng nhóm biên soạn (không phải mã M49) | Quốc gia/khu vực | Giá trị cụ thể | Mức dùng | Nguồn / ghi chú |
|---|---|---|---|---|---|
| GEO-01 | Đông Nam Á | Indonesia | Batik (vải vẽ sáp, hoa văn) | B | UNESCO ICH 2009 — dùng như **chất liệu/hoạ tiết chung**, không sao chép mẫu cụ thể |
| GEO-02 | Đông Nam Á | Việt Nam | Phở (món nước, bát, đũa, hành) | A | Di sản văn hoá phi vật thể quốc gia 2024 (Hà Nội, Nam Định) — không phải danh hiệu UNESCO |
| GEO-03 | Đông Nam Á | Việt Nam | Gốm Bàu Trúc của người Chăm (nặn tay, nung lộ thiên) | B | UNESCO Urgent Safeguarding List 2022 — tránh biến kỹ thuật nghi lễ thành trò |
| GEO-04 | Đông Nam Á | Việt Nam | Nón lá, áo dài, cà phê phin | A | Chi tiết đời sống; tránh dùng như "trang phục mặc định" cho mọi phụ nữ |
| GEO-05 | Đông Nam Á | Việt Nam | Nhã nhạc (nhạc cung đình Huế) | C | UNESCO ICH 2008 — có quy ước biểu diễn; mô tả trong ngữ cảnh nghi lễ cần giữ trang nghiêm, nhưng nhạc cụ/không khí âm nhạc trong một hình đời thường thì không bị cấm |
| GEO-06 | Đông Á | Nhật Bản | Washoku (văn hoá ăn uống, đặc biệt dịp năm mới) | A | UNESCO ICH 2013 — mô tả **bữa ăn và cách bày**, không "geisha/samurai" |
| GEO-07 | Đông Á | Hàn Quốc | Kimjang (làm và chia kimchi) | A | UNESCO ICH 2013 — chọn cảnh **chia sẻ**, tránh nhại giọng/ngoại hình |
| GEO-08 | Đông Á | Trung Quốc | Trà và ấm tử sa; giấy dán cửa | A | Chi tiết thủ công/đời sống; tránh hoa văn rồng–phượng như "biểu tượng quốc gia" |
| GEO-09 | Đông Á | Đài Loan | Trà sữa trân châu, chợ đêm | A | Đời sống đương đại; ghi rõ là ví dụ, không phải đại diện |
| GEO-10 | Nam Á | Ấn Độ | Rangoli/kolam trước cửa (bột màu, hoa) | B | Thực hành sống gắn với nghi lễ/ngưỡng cửa; không dùng làm hoa văn trang trí ngẫu nhiên |
| GEO-11 | Nam Á | Ấn Độ | Trà masala, bình trà, bánh quy | A | Đời sống; tránh gộp "Nam Á" thành một hình |
| GEO-12 | Nam Á | Nepal | Cờ cầu nguyện (dây cờ nhiều màu) | B | Thuộc thực hành tôn giáo; dùng nhẹ, không cắt xén biểu tượng |
| GEO-13 | Nam Á | Sri Lanka | Trà Ceylon, hái trà | A | Nghề nông; tránh hình ảnh "người nghèo đáng thương" |
| GEO-14 | Tây Á | Ả Rập Xê Út, Oman, Qatar, UAE | Cà phê Ả Rập (qahwa) và nghi thức rót mời | B | UNESCO ICH 2015 — **nghi thức hiếu khách**, không phải "đồ uống vùng sa mạc" |
| GEO-15 | Tây Á | Thổ Nhĩ Kỳ | Trà trong ly cong, bánh mì vòng | A | Đời sống; tránh dùng "thảm bay/thầy tu" làm đại diện |
| GEO-16 | Tây Á | Iran | Khatam (khảm), vườn có trục nước | B | Thủ công/kiến trúc cảnh quan; ghi nguồn nếu mô tả kỹ thuật |
| GEO-17 | Châu Phi | Ghana | Kente (dệt dải, màu và hoa văn có tên) | B | UNESCO ICH 2024 — **không** cắt/dán hoa văn như pattern ngẫu nhiên |
| GEO-18 | Châu Phi | Ethiopia | Nghi lễ cà phê (rang, giã, rót) | B | Thực hành tiếp đãi; mô tả hành vi, tránh trang phục "bộ lạc" hư cấu |
| GEO-19 | Châu Phi | Nigeria | Chợ vải, thợ may đo | A | Đời sống đô thị; tránh "động vật hoang dã" làm đại diện châu lục |
| GEO-20 | Châu Phi | Maroc | Zellige (gạch khảm), sân trong (riad) | B | Kiến trúc/thủ công; dùng một mảng nhỏ, không dựng cả cảnh |
| GEO-21 | Châu Âu | Bồ Đào Nha | Fado (hát, đàn guitar Bồ Đào Nha) | B | UNESCO ICH 2011 — không nhại giọng, không bi kịch hoá |
| GEO-22 | Châu Âu / Mỹ Latinh | Tây Ban Nha và México | Talavera (gốm men, hoa văn xanh–trắng) | B | UNESCO ICH 2019 (Puebla–Tlaxcala và Talavera de la Reina–El Puente del Arzobispo) |
| GEO-23 | Châu Âu | Ý | Quán cà phê đứng, tách nhỏ | A | Đời sống; tránh "đấu sĩ/gondola" làm biểu tượng |
| GEO-24 | Châu Âu | Hy Lạp | Cà phê trong ấm briki nhỏ, bàn ghế vỉa hè | A | Đời sống; không dùng "phin" (dụng cụ Việt Nam) cho bối cảnh Hy Lạp |
| GEO-25 | Mỹ Latinh | Argentina / Uruguay | Tango (ôm, bước) | B | UNESCO ICH 2009 — mô tả tư thế khiêu vũ, tránh "quyến rũ" như định kiến |
| GEO-26 | Mỹ Latinh | México | Bếp ngô, comal, bánh tortilla | A | Đời sống ẩm thực; tránh "sombrero + ria" |
| GEO-27 | Mỹ Latinh | Peru | Chợ khoai, dệt màu | A | Đời sống; tránh "Machu Picchu" như sticker mặc định |
| GEO-28 | Mỹ Latinh | Brasil | Quầy nước ép, trái cây nhiệt đới | A | Đời sống đô thị |
| GEO-29 | Châu Đại Dương | Tonga, Samoa | Ngatu/tapa (vải vỏ cây, hoa văn) | C | Thực hành gắn với gia tộc/nghi lễ; nhiều hoa văn thuộc quyền của gia đình hoặc cộng đồng — cần tham vấn trước khi tái tạo |
| GEO-30 | Châu Đại Dương | Aotearoa New Zealand | Haka | C | Loại hình trình diễn có chủ thể cộng đồng; mô tả đời thường thì được, nhưng mô phỏng nghi lễ hoặc dùng làm trò hài cần tham vấn và cân nhắc |
| GEO-31 | Châu Đại Dương | Hawaiʻi | Lei (vòng hoa) | B | Tặng/đón tiếp; tránh dùng "hula" như hoạ tiết trang trí không liên quan |
| GEO-32 | Bắc Mỹ | Hoa Kỳ, Canada | Quán ăn sáng, cà phê mang đi, mùa tuyết | A | Đời sống; tránh đồng nhất "Bắc Mỹ" với một thành phố |
| GEO-33 | Bắc Mỹ | Cộng đồng bản địa (nhiều nơi) | Quilt, giỏ đan, hoa văn gia đình | C | Một số hoa văn thuộc quyền gia đình/cộng đồng; cần biết cái nào và xin phép khi cần — không phải cấm mọi hình thủ công |
| GEO-34 | Xuyên vùng | Nhiều nước | Chợ sáng, xe đẩy hàng, ghế nhựa vỉa hè | A | Bối cảnh đô thị phổ biến; tốt để tạo cảm giác "gần gũi" mà không gắn cờ |
| GEO-35 | Xuyên vùng | Nhiều nước | Cửa hàng tiện lợi ban đêm, đèn huỳnh quang | A | Bối cảnh đêm; dùng làm nền một mảnh |
| GEO-36 | Xuyên vùng | Nhiều nước | Ban công nhỏ, dây phơi, chậu cây | A | Bối cảnh chung cư; tránh gắn với một quốc gia |
| GEO-37 | Xuyên vùng | Nhiều nước | Bến xe/bến tàu, ghế chờ, vali | A | Bối cảnh đi lại |
| GEO-38 | Xuyên vùng | Nhiều nước | Quán trà/nước nhỏ, ghế thấp, quạt trần | A | Bối cảnh trò chuyện |
| GEO-39 | Xuyên vùng | Nhiều nước | Lớp học, bảng, phấn | A | Bối cảnh học tập |
| GEO-40 | Xuyên vùng | Nhiều nước | Sân chung cư, cầu thang bộ, cửa sắt | A | Bối cảnh đời sống |

### 9.3 Thành phố & địa danh (dùng như "một mảnh", không như đại diện)

| Mã | Địa danh | Mảnh bối cảnh dùng được | Ghi chú |
|---|---|---|---|
| CIT-01 | Hà Nội | Hàng cây, ghế nhựa vỉa hè, biển hiệu hẹp | Tránh "phố cổ" như bối cảnh duy nhất của Việt Nam |
| CIT-02 | TP. Hồ Chí Minh | Hẻm, chung cư cũ, xe máy đỗ dày | Bối cảnh đô thị; không cần vẽ toàn cảnh |
| CIT-03 | Huế | Cửa vòm, tường gạch rêu, sông | Bối cảnh kiến trúc; chỉ giữ trang nghiêm khi mô tả một nghi lễ cụ thể, không áp cho mọi sticker Huế |
| CIT-04 | Đà Nẵng | Cầu, biển, quán hải sản | Tránh chỉ dùng "biển" làm đại diện miền Trung |
| CIT-05 | Đà Lạt | Đồi thông, kính nhà, hoa | Bối cảnh khí hậu mát |
| CIT-06 | Kyoto | Ngõ hẹp, cửa gỗ, vườn nhỏ | Tránh "geisha" |
| CIT-07 | Seoul | Quán cà phê tầng 2, biển hiệu dọc | Bối cảnh đương đại |
| CIT-08 | Istanbul | Phà, trà trong ly, mòng biển | Tránh "harem/thảm bay" |
| CIT-09 | Lisbon | Dốc, tàu điện vàng, gạch ốp mặt tiền | Bối cảnh đường phố |
| CIT-10 | Mexico City | Quán ăn đường phố, tranh tường | Tránh "ngày lễ người chết" như hoá trang |
| CIT-11 | Cape Town | Bàn núi, quán cà phê, gió | Tránh đồng nhất châu Phi với "hoang dã" |
| CIT-12 | Taipei | Chợ đêm, xe tay ga, biển hiệu chồng | Bối cảnh ban đêm |

### 9.4 Mở rộng địa lý từ danh sách chuẩn `[ĐỀ XUẤT]`

Quy trình thêm một mục (6 bước):

1. Chọn **mã ISO 3166-1 alpha-2** và vùng **M49** tương ứng trước khi viết mô tả. Cột "Vùng" trong bảng §9.2 là nhãn nhóm do người biên soạn đặt cho dễ đọc, **không phải** giá trị M49 nguyên văn; khi cần dùng cho dữ liệu, ánh xạ sang M49.
2. Tra tên hiển thị tiếng Việt trong dữ liệu CLDR (`cldr-localenames-full`) thay vì tự dịch.
3. Chọn **một** chi tiết cụ thể (món ăn, nghề, vật liệu, kiểu cửa) — không chọn "văn hoá".
4. Tìm nguồn: ưu tiên UNESCO ICH/WHC cho thực hành/di sản; báo chí địa phương hoặc bảo tàng cho chi tiết đời sống; nếu không có nguồn, ghi rõ `[GIÁ TRỊ]` và hạ mức xuống `B`.
5. Xác định mục có gắn với thực hành nghi lễ, hoa văn hạn chế, hoặc quyền của cộng đồng không; nếu có → mức `C` và ghi rõ cần tham vấn điều gì (không mặc định loại bỏ ý tưởng).
6. Ghi vào bảng kèm mã, mức, nguồn, và một dòng "điều gì dễ sai nếu vẽ theo trí nhớ".

---

## 10. Công thức scenario cho một sticker

### 10.1 Khung 8 ô

```
[1 chủ thể] + [2 hành vi] + [3 cảm xúc × cường độ] + [4 trang phục]
+ [5 một mảnh bối cảnh] + [6 góc nhìn] + [7 nét & màu] + [8 nền tách]
```

| Ô | Lấy từ | Ghi chú |
|---|---|---|
| 1 Chủ thể | một người / hai người / một vật có hành vi | Ít chủ thể thường dễ kiểm soát bố cục và tách nền hơn, nhưng đây không phải quy định; tương tác hai người là nội dung hợp lệ |
| 2 Hành vi | §3 hoặc §4 | **Một** động từ chính |
| 3 Cảm xúc | §2, kèm mã cường độ | Không lặp lại hành vi |
| 4 Trang phục | trung tính + 1 chi tiết | Tránh đồng phục nghề nghiệp gây định kiến |
| 5 Bối cảnh | §9.3 hoặc §7, **một mảnh** | Không vẽ cả phòng |
| 6 Góc nhìn | ngang tầm mắt / hơi cao | Giữ cố định cả bộ |
| 7 Nét & màu | hằng số của bộ | Xem §11 |
| 8 Nền | `uniform pure-white background` (engine `simple`) hoặc alpha thật đã kiểm tra | Không mặc định nền màu khác cho đường cắt nhanh; xem `studio-composition.md` §5.1–5.2 |

### 10.2 Ba ví dụ hoàn chỉnh (không chữ)

**A. Sticker "chốt việc" — dùng trong chat công việc**

> `single office worker, both hands brushing together as if dusting off, satisfied closed-mouth smile, shoulders relaxed, plain shirt with rolled sleeves, one corner of a desk with a closed laptop, eye-level wide shot, thick even black outline, two flat colors plus one accent, uniform pure-white background`

**B. Sticker "mất kết nối" — hài nhẹ**

> `one person sitting cross-legged on the floor holding a laptop up toward a window, hopeful eyes, mouth slightly open, oversized hoodie, single window frame and a potted plant, slight low angle, thick even black outline, muted palette with one warm accent, uniform pure-white background`

**C. Sticker "cảm ơn trịnh trọng" — không hài**

> `one person bowing from the waist with both hands together, eyes lowered, calm expression, simple long-sleeved top, wooden floor edge and a paper bag set aside, eye-level shot, thin-consistent line weight, warm neutral palette, uniform pure-white background`

Ba prompt trên là **giá trị gốc** của tài liệu; chúng minh hoạ cách ghép ô, không phải mẫu tối ưu.

### 10.3 Quy tắc viết prompt

- Không có độ dài từ tối ưu được đo. Viết đủ để nói rõ chủ thể, hành vi, bối cảnh và nét; nếu kết quả mơ hồ thì thêm chi tiết cụ thể, nếu kết quả rối thì bỏ bớt chi tiết rồi thử lại. Đánh giá ở thumbnail và ở kích thước hiển thị thật.
- Động từ hành vi đặt **trước** tính từ cảm xúc.
- Tránh phủ định (`no text, no shadow`) trừ khi thật cần; mô hình xử lý phủ định kém ổn định — thay bằng mô tả khẳng định (`uniform pure-white background`).
- Không nhồi phong cách thương hiệu; giữ ô 7 là **hằng số** của bộ và lặp y hệt.
- Đường bao **nghệ thuật** trong hình (outline của nhân vật) khác với **đường cắt/viền sticker** tạo sau khi tách nền. Nếu dùng outline, nói rõ đó là outline của chủ thể, không phải cutline; viền và độ dày vẫn xử lý ở bước sau (`studio-composition.md` §5.4).

---

## 11. Năm ý tưởng bộ sticker giữ bản sắc `[ĐỀ XUẤT]`

Mỗi bộ cần **hằng số** (không đổi) và **biến số** (đổi theo sticker). Vì mô hình có thể trượt nhất quán giữa các lần sinh `[NGUỒN]`, hằng số phải là thứ mô tả được bằng từ ngữ ngắn và kiểm tra được bằng mắt.

Năm bộ dưới đây là **ví dụ minh hoạ cách ghép** thư viện — kể cả số lượng sticker và cách chia nhóm. Chúng không phải yêu cầu về số ô, không phải tỉ lệ đã đo, và không nên được đọc như đặc tả sản phẩm.

### Bộ 1 — "Văn phòng nhưng đã hết kiên nhẫn"

| Hạng mục | Nội dung |
|---|---|
| Hằng số | Nét đen cùng độ dày ở mọi sticker (đánh giá ở kích thước xuất), 2 màu phẳng + 1 màu nhấn cam, mắt chấm tròn, đầu hơi to, không đổ bóng |
| Biến số | Hành vi (SIT-01…03), cảm xúc mức 2–3, một mảnh bối cảnh bàn làm việc |
| Ví dụ 16 sticker | 6 nhịp tình huống deadline, 4 chức năng (nhắc, từ chối, xác nhận, kết thúc), 3 cảm xúc nén, 3 hài vô tri |
| Rủi ro | Dễ trượt sang "meme công sở" na ná bộ khác → giữ kiểu mắt chấm và tông cam |

### Bộ 2 — "Một ngày mưa ở đô thị nhiệt đới"

| Hạng mục | Nội dung |
|---|---|
| Hằng số | Nét xanh đậm, màu nền xám xanh, mưa vẽ bằng nét gạch chéo 2 hướng, không vẽ cảnh phố |
| Biến số | WEA-01…16, GES-DO-03, TIM-05/06 |
| Ví dụ 16 sticker | 5 thời tiết, 4 hành vi che mưa, 3 cảm xúc nhẹ, 2 chức năng (nhắc, hẹn), 2 hài |
| Rủi ro | Mưa dễ thành "u ám" — cần ít nhất 4 sticker có yếu tố ấm (cốc nóng, đèn) |

### Bộ 3 — "Bếp nhỏ, chuyện nhỏ"

| Hạng mục | Nội dung |
|---|---|
| Hằng số | Nét nâu đen, màu đất + trắng ngà, tay hơi to, không vẽ mặt bàn chi tiết |
| Biến số | §9.2 mức A (phở, tortilla, trà, bánh mì vòng), GES-DO-11, CF-MOI-1 |
| Ví dụ 16 sticker | 5 hành vi ăn/uống, 4 chức năng (mời, cảm ơn, khen, từ chối), 4 cảm xúc, 3 hài |
| Rủi ro | Món ăn dễ bị "quốc gia hoá": một hình gộp nhiều nền mà không có ý đồ sẽ đọc thành định kiến. Pha trộn có chủ đích, có nguồn cho từng phần thì hợp lệ (xem §13) |

### Bộ 4 — "Hai người, một cái ô"

| Hạng mục | Nội dung |
|---|---|
| Hằng số | Hai nhân vật cùng tỉ lệ và cùng kiểu tương tác, nét đều, nền trắng đồng nhất |
| Biến số | GES-CAP-01…10, REL-01/07/11, TON-01/07 |
| Ví dụ 12 sticker | 5 tương tác, 3 chức năng (an ủi, cảm ơn, hẹn), 2 cảm xúc trộn, 2 hài |
| Rủi ro | Hai nhân vật có thể dính thành một khối khi cắt; nếu muốn tách rời thì mô tả khoảng hở. Ôm, nép, chạm là lựa chọn nghệ thuật hợp lệ: chỉ cần tách **cụm sticker này** khỏi **cụm sticker bên cạnh**, không cần tách hai chủ thể trong cùng một sticker |

### Bộ 5 — "Đồ vật có đời sống riêng"

| Hạng mục | Nội dung |
|---|---|
| Hằng số | Không có người; mỗi vật có 2 mắt chấm; nét mảnh; nền phẳng; bóng đổ duy nhất một hướng |
| Biến số | HUM-03/04, GES-DO-01/06/10, TIM-07 |
| Ví dụ 12 sticker | 6 vật "hành động", 3 vật "phản ứng", 3 vật "tan rã" |
| Rủi ro | Dễ thành "kawaii chung chung" → giữ bóng đổ một hướng và nét mảnh làm dấu nhận |

**Kiểm tra bản sắc** `[ĐỀ XUẤT]`: đặt các sticker cạnh nhau ở hai cỡ thật — thumbnail và cỡ hiển thị trong chat — rồi tự đánh giá xem chúng có đọc như cùng một bộ. Tài liệu này không đặt ngưỡng thời gian hay khoảng cách quan sát nào; nếu thấy lệch, nguyên nhân thường là nét hoặc bảng màu khác nhau giữa các lần sinh.

---

## 12. Chữ, đa ngữ, và cắt nền

### 12.1 Chữ trong ảnh `[NGUỒN]`

- Tài liệu OpenAI ghi rõ mô hình vẫn có thể **đặt chữ sai vị trí và thiếu rõ** dù đã cải thiện. Vì vậy: **không** giao chữ quan trọng cho AI.
- Cách làm `[ĐỀ XUẤT]`: xuất sticker **không chữ**; nếu cần chữ, chèn ở lớp hậu kỳ bằng công cụ vector. Lợi ích: sửa chính tả được, đổi ngôn ngữ được, không phải sinh lại ảnh.

### 12.2 Đa ngữ

- Tiếng Việt có dấu: nhiều font không đủ glyph cho `ế ộ ữ ỹ`; nếu chèn chữ thủ công, kiểm tra **từng dấu** ở cỡ nhỏ nhất sẽ dùng.
- Bong bóng thoại dài dễ phá bố cục; thử độ dài trên thumbnail và ở cỡ hiển thị thật thay vì áp một con số cố định.
- Nếu một bộ phục vụ nhiều ngôn ngữ: giữ **cùng hình**, chỉ đổi lớp chữ. Không sinh lại ảnh theo ngôn ngữ (dễ lệch bản sắc).
- Tránh chơi chữ phụ thuộc một ngôn ngữ trong hình không chữ (xem §5.3).

### 12.3 Không chữ: prompt và kiểm tra

- Mô tả khẳng định thay vì phủ định: `uniform pure-white background, single subject, no signage` (vẫn có "no" nhưng ngắn và cụ thể).
- Hậu kiểm: soi 4 góc và vùng nền; chữ thường xuất hiện thành "chữ giả" ở biển hiệu, nhãn, màn hình.
- Nếu mô hình cứ thêm chữ trên màn hình/giấy: đổi vật thể (úp laptop, tờ giấy trắng) thay vì thêm phủ định.

### 12.4 Cắt nền & tách sticker

Engine hiện tại (`simple`) dò vùng gần trắng (so RGB với 255, ngưỡng 10) rồi giữ phần không nối với nền ngoài; vì vậy nền trắng gần đồng nhất hoặc alpha thật đã kiểm tra là hai đường phù hợp, còn nền màu khác là thay đổi chưa triển khai. Chi tiết hành vi và giới hạn nằm trong `studio-composition.md` §5.1–5.3; ảnh có cạnh lớn hơn **4096 px** chưa được nhận.

| Tình huống | Vì sao khó | Cách xử lý đề xuất |
|---|---|---|
| Nét mảnh | Có thể đứt hoặc mất khi tách, nhất là ở cỡ nhỏ | Yêu cầu nét đều; kiểm tra ở kích thước xuất cuối, không dùng một ngưỡng pixel chung cho mọi kiểu hình |
| Bóng đổ dính nền | Vùng bóng nối với nền, bị giữ lại như một mảng thay vì bị xoá | Bỏ bóng trong prompt (`no background shadows`) và thêm bóng sau khi đã có alpha nếu cần; bóng trên sheet dễ bị giữ lại cùng nền |
| Chi tiết rời (dây, khói, tóc bay) | Có thể thành một "sticker" riêng, hoặc bị bộ lọc box coi là ngoại lai (ngưỡng 25% median) | Nối chi tiết vào khối chính trong prompt; nếu vẫn rời, chỉnh mask thay vì tin vào bộ lọc |
| Nền gradient / có vân / lưới | Dò "gần trắng" không còn đúng | Yêu cầu nền trắng đồng nhất; không gradient, không ô bàn cờ, không đường kẻ |
| Chủ thể bán trong suốt (hơi nước, thuỷ tinh, voan) | Alpha mờ, dễ để lại viền hoặc mất chi tiết; không phải lý do để cấm | Giữ ý tưởng nhưng dự trù chỉnh mask thủ công và kiểm tra ở nền sáng lẫn nền tối |
| Nhiều sticker trên một tờ | Một lần sinh chứa nhiều cụm; khoảng cách và kích thước cụm ảnh hưởng bước tách | Quy định số ô và khe hở **giữa các cụm**; chưa có đo lường nào cho thấy sinh sheet hay sinh lẻ giữ bản sắc tốt hơn — cả hai đều cần xem lại |
| Viền khi cắt | AI vẽ viền sẵn dễ dính nền và khó đổi độ dày | Tạo viền **sau** khi có alpha sạch, đánh giá ở kích thước xuất |

**Đề xuất** `[ĐỀ XUẤT]`: giữ ảnh gốc từ công cụ AI, không cắt lại từ ảnh chụp màn hình; chuẩn hoá canvas theo `studio-composition.md` §5.1 (canvas 512, thumbnail 128) và kiểm tra silhouette ở thumbnail. Không đặt kích thước xuất cố định trong tài liệu này — giới hạn thật là của công cụ AI và pipeline.

---

## 13. Tránh định kiến & bản quyền

Hai điều cần tách rõ trước khi kiểm:

- **Nhận ra một nền văn hoá từ một dấu hiệu không tự động là định kiến.** Một chi tiết cụ thể (bát phở, tấm kente, ấm trà) giúp người xem nhận ra bối cảnh; đó là mục tiêu của thư viện này.
- **Vấn đề nằm ở quy giản và gán thuộc tính.** Định kiến xuất hiện khi một dấu hiệu duy nhất *thay thế* con người (chỉ còn "người nước X" = một món ăn/cái nón), khi dấu hiệu bị dùng để gán tính cách, thứ bậc, trí tuệ, hay làm trò đùa, hoặc khi hình mô tả một cộng đồng bằng chi tiết hư cấu không ai dùng.

**Pha trộn văn hoá** `[GIÁ TRỊ]`: trộn nhiều nguồn là hướng sáng tạo hợp lệ và thường thú vị (ví dụ bàn ăn có phở và bánh mì vòng trong cùng một khung; nhân vật mặc áo dài đi giày thể thao). Điều kiện để pha trộn có chủ đích thay vì thành định kiến:

1. **Có ý đồ, không phải do thiếu hiểu biết**: người biên soạn biết mỗi chi tiết đến từ đâu và vì sao đặt cạnh nhau.
2. **Mỗi chi tiết đúng ở dạng của nó**: không "cải biên" chi tiết thiêng hoặc nghi lễ thành hoạ tiết trang trí.
3. **Ghi nguồn cho từng phần** trong ghi chú bộ, kể cả khi không hiển thị cho người dùng cuối.
4. **Không dùng pha trộn để chế giễu một nền cụ thể** hoặc để gộp nhiều cộng đồng thành "một phương Đông"/"một châu Phi" chung chung.
5. **Kiểm tra quyền**: nếu pha trộn dựa trên ảnh tham chiếu hoặc thiết kế có sẵn, cần quyền sử dụng tương ứng (xem dưới).

**Danh sách kiểm trước khi phát hành một sticker:**

1. Nếu bỏ chú thích, hình có còn là **một hành vi của một người** không? Nếu nó chỉ còn là một dấu hiệu dân tộc đứng một mình, hãy thêm hành vi hoặc bối cảnh — hoặc chấp nhận rằng đây là sticker biểu tượng, không phải chân dung cộng đồng.
2. Có chi tiết nào thuộc nghi lễ/tôn giáo/hoa văn cộng đồng không? Nếu có và chưa có phép → bỏ hoặc thay bằng chi tiết thế tục.
3. Trang phục/nghề nghiệp có đang gán giới tính, tuổi, dân tộc một cách không cần thiết không?
4. Có đang dùng ngoại hình (cân nặng, khuyết tật, màu da) làm trò đùa không? → bỏ.
5. Hình có tái tạo **nguyên bản** đang được bảo hộ (nhân vật, logo, tác phẩm, thiết kế) mà không có quyền không? Nếu có quyền, hoặc có giấy phép, hoặc có sự cho phép của chủ sở hữu → dùng bình thường và ghi nguồn; nếu không → thay bằng hình mô tả chung. Không tự động thay mọi tham chiếu có thương hiệu.
6. Nguồn của chi tiết văn hoá đã ghi chưa? Nếu chưa, hạ xuống chi tiết chung hoặc bổ sung nguồn.
7. Sticker có dùng được trong ngữ cảnh công việc không? (kiểm tra ngôn từ và mức cảm xúc)

**Giấy phép và điều khoản cần đọc** `[NGUỒN]`:

- Dữ liệu Unicode/CLDR phát hành theo giấy phép của Unicode Consortium — đọc điều khoản trước khi nhúng dữ liệu vào sản phẩm.
- Mã ISO 3166 do ISO giữ bản quyền. Tài liệu này không đưa kết luận pháp lý về việc dùng mã; hãy đọc điều khoản áp dụng cho tình huống cụ thể (hoặc hỏi người phụ trách pháp lý) trước khi phân phối lại danh mục hoặc mô tả kèm mã.
- Nội dung UNESCO (mô tả di sản, ảnh, video) có điều khoản riêng; **liên kết** khác với **sao chép** — kiểm tra điều khoản trước khi dùng lại nội dung.
- Font dùng để chèn chữ phải có giấy phép cho phép nhúng vào ảnh phát hành.

---

## 14. Nguồn

Cách xác minh trong tài liệu này: các trang Unicode, CLDR, UNSD và tài liệu OpenAI đã được đọc trực tiếp. Các trang **ich.unesco.org** chặn truy cập tự động, nên đường dẫn ICH dưới đây được **xác minh gián tiếp qua kết quả tìm kiếm** (tiêu đề trang và mã ID khớp); **nội dung đầy đủ của các trang đó chưa được đọc lại**, vì vậy phần mô tả di sản trong tài liệu này là tóm tắt biên soạn, không phải trích dẫn đã đối chiếu nguyên văn. Trước khi dùng cho mục đích phát hành, mở trang bằng trình duyệt và đối chiếu.

| Nguồn | Dùng cho | URL |
|---|---|---|
| UTS #51 — Unicode Emoji | Cấu trúc emoji, chuỗi ZWJ, biến thể màu da, VS15/VS16, vai trò của annotation | https://www.unicode.org/reports/tr51/ |
| emoji-test.txt (Unicode 16.0) | 10 nhóm và tên nhóm con dùng làm khung phân loại | https://unicode.org/Public/emoji/16.0/emoji-test.txt |
| CLDR | Nhãn annotation (dùng tìm kiếm, không phải định nghĩa), tên vùng bản địa hoá | https://cldr.unicode.org/ |
| cldr-json | Dữ liệu annotations/localenames dạng JSON | https://github.com/unicode-org/cldr-json |
| UNSD M49 | Phân vùng địa lý/kinh tế | https://unstats.un.org/unsd/methodology/m49/ |
| ISO 3166 | Mã quốc gia (trang chính thức; có thể chặn truy cập tự động) | https://www.iso.org/iso-3166-country-codes.html |
| UNESCO ICH — danh sách | Khung pháp lý và danh sách di sản phi vật thể | https://ich.unesco.org/en/lists |
| ICH — Batik Indonesia (2009) | Ví dụ chất liệu/hoa văn | https://ich.unesco.org/en/RL/indonesian-batik-00170 |
| ICH — Kente (2024) | Ví dụ dệt may, Ghana | https://ich.unesco.org/en/RL/craftsmanship-of-traditional-woven-textile-kente-02130 |
| ICH — Talavera (2019) | Ví dụ gốm men, México & Tây Ban Nha | https://ich.unesco.org/en/RL/artisanal-talavera-of-puebla-and-tlaxcala-mexico-and-ceramics-of-talavera-de-la-reina-and-el-puente-del-arzobispo-spain-making-process-01462 |
| ICH — Nhã nhạc (2008) | Ví dụ nhạc cung đình Huế (mức dùng C) | https://ich.unesco.org/en/RL/nha-nhac-vietnamese-court-music-00074 |
| ICH — Washoku (2013) | Ví dụ văn hoá ăn uống Nhật Bản | https://ich.unesco.org/en/RL/washoku-traditional-dietary-cultures-of-the-japanese-notably-for-the-celebration-of-new-year-00869 |
| ICH — Kimjang (2013) | Ví dụ làm/chia kimchi | https://ich.unesco.org/en/RL/kimjang-making-and-sharing-kimchi-in-the-republic-of-korea-00881 |
| ICH — Arabic coffee (2015) | Ví dụ nghi thức hiếu khách | https://ich.unesco.org/en/RL/arabic-coffee-a-symbol-of-generosity-01074 |
| ICH — Gốm Chăm (2022, USL) | Ví dụ kỹ thuật gốm cần bảo vệ khẩn cấp | https://ich.unesco.org/en/USL/art-of-pottery-making-of-chm-people-01574 |
| ICH — Fado (2011) | Ví dụ âm nhạc Bồ Đào Nha | https://ich.unesco.org/en/RL/fado-urban-popular-song-of-portugal-00563 |
| ICH — Tango (2009) | Ví dụ khiêu vũ Argentina/Uruguay | https://ich.unesco.org/en/RL/tango-00258 |
| UNESCO World Heritage | Di sản vật thể (khác ICH) | https://whc.unesco.org/en/list/ |
| OpenAI — Image generation | Giới hạn về chữ trong ảnh, nhất quán nhân vật, kiểm soát bố cục, moderation | https://developers.openai.com/api/docs/guides/image-generation |
| Google — Gemini image generation | Tài liệu tham khảo thêm về sinh ảnh | https://ai.google.dev/gemini-api/docs/image-generation |
| Báo Chính phủ — di sản quốc gia (phở Nam Định, mì Quảng) | Bối cảnh phở là di sản **quốc gia**, không phải UNESCO | https://en.baochinhphu.vn/local-noodle-soup-designated-as-national-intangible-cultural-heritage-111240813144900022.htm |
| VietnamPlus — phở Hà Nội, trà sen | Cùng mục đích trên | https://en.vietnamplus.vn/hanois-pho-lotus-tea-making-recognised-as-national-cultural-heritage-post291827.vnp |

---

## 15. Giới hạn của tài liệu

- Đây là **kho gợi ý**, không phải danh sách đầy đủ: cảm xúc, cử chỉ và tình huống ngoài đời nhiều hơn rất nhiều so với ~330 mục ở đây.
- Các mảnh prompt được biên soạn để **mô tả hình thể rõ ràng hơn**; tài liệu này không đo được mức cải thiện nào và không đảm bảo AI tạo ra đúng hình.
- Các nhận định về văn hoá ở §9 là **ví dụ có nguồn gốc**, không phải mô tả đầy đủ về bất kỳ cộng đồng nào. Một số mục được đánh `C` vì lý do tôn trọng và thiếu thông tin, không phải vì cấm đoán tuyệt đối.
- Mọi con số trong tài liệu (số sticker mỗi bộ, cách chia nhóm, số ô) là **ví dụ minh hoạ chưa kiểm chứng**, không phải yêu cầu hay tỉ lệ thành công.
- Nguồn ICH của UNESCO được xác minh qua tìm kiếm, chưa đọc đầy đủ nội dung trang (xem §14); mô tả di sản ở đây là tóm tắt biên soạn, cần đối chiếu trước khi phát hành.
- Tài liệu này là **nghiên cứu đề xuất**, không phải đặc tả đã được duyệt, không phải cam kết tính năng, và không thay thế quyết định của người phụ trách sản phẩm.
