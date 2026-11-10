# Thư viện mảnh ghép sáng tạo (creative fragments)

- **Ngày:** 2026-11-10
- **Trạng thái:** tài liệu nghiên cứu, **đề xuất chưa triển khai**. Không có mục nào ở đây là đặc tả kỹ thuật, schema, hay yêu cầu bắt buộc của sản phẩm.
- **Phạm vi:** thư viện giá trị gợi ý để soạn prompt cho một nhân vật sticker có thể lặp lại qua nhiều pack. Mọi giá trị là **ví dụ biên soạn**, không phải chuẩn mực, không phải danh sách đầy đủ, và không bảo đảm mô hình sinh ảnh sẽ tuân thủ.
- **Liên quan:** `docs/research/sticker-factory/` (các tài liệu cùng đợt); không thay thế tài liệu kỹ thuật về nền/xuất file.

---

## 1. Mục đích và cách đọc

Một "mảnh ghép" (fragment) ở đây là **một cụm từ ngắn, một ý niệm thị giác**, dùng để ghép vào prompt gửi cho AI tạo ảnh bên ngoài. Sticker Factory không vẽ nhân vật; nó giúp người dùng **nhớ, chọn, tái sử dụng** các mô tả đã hiệu quả, rồi tự copy prompt sang công cụ khác.

Vì vậy tài liệu này có ba lớp, cần đọc tách bạch:

| Lớp | Là gì | Mức độ chắc chắn |
|---|---|---|
| **Nguyên tắc có nguồn** | Cách phân loại theo facet, cách gọi tên màu, ngưỡng tương phản, các kiểu hòa phối màu | Có nguồn, trích dẫn ở §9 |
| **Danh mục biên soạn** | 24 chiều với 425 giá trị cụ thể bên dưới | Đề xuất của người biên soạn, **không** suy ra từ nguồn |
| **Suy luận vận hành** | Cách ghép mảnh, rủi ro khi cắt nền, giới hạn ảnh tham chiếu | Suy luận từ hành vi công cụ hiện tại, đánh dấu rõ |

Ba lớp này không được trộn: một giá trị như "xanh mực" không phải thuật ngữ chuẩn của Getty AAT, còn "tương phản chữ 4.5:1" là con số có trong WCAG.

Cách dùng thực tế: chọn **một** giá trị cho mỗi chiều cần thiết, không chọn hai giá trị cùng chiều (hai loài, hai nghề) trừ khi xung đột đó là chủ ý. Thư viện không có giới hạn cứng về số mảnh; các con số trong tài liệu chỉ là số lượng giá trị đang có.

---

## 2. Vì sao chia theo chiều (dimension)

Getty AAT tổ chức từ vựng thành các **facet** — những lớp đồng nhất, mỗi lớp gom các khái niệm chia sẻ cùng đặc tính, đi từ trừu tượng đến vật thể cụ thể: Associated Concepts, Physical Attributes, Styles and Periods, Agents, Activities, Materials, Objects (và Brand Names) ([Getty AAT, About](https://www.getty.edu/research/tools/vocabularies/aat/about.html)). Đây là **mô hình để học**, không phải khuôn dán nhãn cho nhân vật hư cấu: AAT chứa thuật ngữ chung, không chứa tên riêng và không chứa chủ đề iconography.

Điều đáng lấy từ AAT là ba thói quen:

1. **Mỗi khía cạnh là một trục riêng**, không nhồi "nghề + trang phục + bối cảnh" vào một cụm từ ghép kiểu *"thợ làm bánh mặc áo trắng ở Paris"* — AAT cố ý loại các "unbound compound terms" như *Baroque cathedral* vì chúng trộn hai hierarchy.
2. **Quan hệ rộng–hẹp rõ ràng**: một giá trị là *loại của* giá trị khác. "Chó corgi" là loại của "chó"; trong thư viện này, "cáo nhiều đuôi" và "thỏ tai rủ" ngang cấp, không lồng nhau.
3. **Có căn cứ sử dụng** (literary/organizational/user warrant): thuật ngữ tồn tại vì có người dùng nó, không vì ai đó muốn đủ số lượng.

24 chiều dưới đây là ánh xạ *thoải mái* của tinh thần đó sang nhân vật sticker, không phải bản dịch facet của AAT. Một nhân vật thường cần 5–8 chiều; phần còn lại là biến thiên theo pack hoặc theo từng sticker.

---

## 3. Danh tính bất biến vs biến thiên

Đây là phần quan trọng nhất nếu muốn nhiều sticker trông như **cùng một nhân vật**.

### 3.1 Ba mức

| Mức | Nội dung | Ví dụ |
|---|---|---|
| **Bất biến (identity invariants)** | Lặp y nguyên ở mọi sticker, mọi pack. Sai một cái là mất nhân vật. | loài, dấu nhận dạng, cấu trúc màu, kiểu nét, tỉ lệ tổng thể |
| **Bán biến thiên (per-pack)** | Đổi theo chủ đề pack nhưng giữ nhất quán trong pack. | trang phục, bối cảnh, bảng màu phụ, đạo cụ chính |
| **Tự do (per-sticker)** | Đổi từng sticker, không ảnh hưởng danh tính. | dáng, biểu cảm, góc nhìn, đạo cụ phụ, thời tiết |

### 3.2 Gợi ý phân loại theo chiều

| Chiều | Mức mặc định |
|---|---|
| D01 loài/dạng cơ thể | bất biến |
| D02 nguyên mẫu | bất biến (định hình cách hành xử) |
| D03 tỉ lệ | bất biến |
| D04 khuôn mặt | bán biến thiên (giữ 2–3 nét chính, cho phép đổi biểu cảm) |
| D05 dấu nhận dạng | bất biến, tuyệt đối |
| D06 tính cách | bất biến |
| D07 bề mặt | bất biến |
| D08–D10 màu | D08/D09 bất biến, D10 bán biến thiên |
| D11–D13 nét/render/chất liệu | bất biến |
| D14–D16 trang phục/phụ kiện/đạo cụ | bán biến thiên |
| D17–D18 nghề/sở thích | bất biến hoặc bán biến thiên |
| D19–D21 thế giới/thời kỳ/ánh sáng | bán biến thiên |
| D22–D24 góc nhìn/silhouette/dáng | tự do |

Bảng này là **đề xuất**, không phải quy tắc. Một bộ sticker hài có thể cố ý đổi cả trang phục lẫn bối cảnh mỗi tấm.

### 3.3 Kiểm tra nhanh một bộ mảnh

- Có đúng **một** giá trị cho mỗi chiều bất biến?
- Dấu nhận dạng có nhìn thấy được ở góc nhìn đã chọn không? (dấu ở lưng, góc nhìn chính diện → vô nghĩa)
- Hai sticker bất kỳ có cùng "nhịp" màu không? Nếu tấm này 90% tối, tấm kia 90% sáng, chúng đọc như hai nhân vật.
- Mảnh nào sẽ khiến công cụ cắt nền ăn mất chi tiết? (xem §7)

---

## 4. Thư viện 24 chiều (425 giá trị)

Quy ước: mỗi dòng là một giá trị; phần trong ngoặc là **gợi ý từ tiếng Anh** để dán vào prompt, không phải bản dịch chuẩn. Không có thứ tự ưu tiên giữa các dòng trong cùng một chiều.

### D01. Loài & dạng cơ thể — 22 giá trị

- Người tí hon (tiny human)
- Người khổng lồ hiền (gentle giant)
- Mèo đứng hai chân (bipedal cat)
- Cáo nhiều đuôi (multi-tailed fox)
- Gấu mật ong (honey bear)
- Thỏ tai rủ (lop-eared rabbit)
- Cá voi mini bay (mini flying whale)
- Rồng con không cánh (wingless dragonling)
- Chim sẻ giấy (paper sparrow)
- Nấm biết đi (walking mushroom)
- Sen đá biết đi (walking succulent)
- Đá cuội có chân (pebble with legs)
- Mây nhỏ có mặt (small cloud being)
- Giọt mực sống (living ink drop)
- Bóng đèn có mặt (lightbulb creature)
- Ấm trà có tay (teapot creature)
- Robot hộp thiếc (tin-can robot)
- Bù nhìn rơm (straw scarecrow)
- Hồn ma vải (cloth ghost)
- Người nửa máy (cyborg)
- Người đất sét (clay figure)
- Sinh vật lai thực vật–thú (plant-beast hybrid)

### D02. Nguyên mẫu nhân vật — 20 giá trị

- Người giữ cửa (gatekeeper)
- Người sưu tầm thầm lặng (quiet collector)
- Người dẫn đường (guide)
- Người sửa chữa (fixer)
- Người kể chuyện (storyteller)
- Người canh gác ban đêm (night watch)
- Học việc vụng về (clumsy apprentice)
- Bậc thầy khó tính (grumpy master)
- Nhà thám hiểm bàn giấy (armchair explorer)
- Kẻ lạc đường tự tin (confident lost one)
- Bạn đồng hành trung thành (loyal sidekick)
- Người bạn hay do dự (hesitant friend)
- Nhà phát minh thất bại (failed inventor)
- Người gác kho ký ức (memory keeper)
- Sứ giả hòa giải (peace envoy)
- Người mộng du (sleepwalker)
- Người chăm sóc (caretaker)
- Kẻ săn tìm ánh sáng (light chaser)
- Người viết thư hộ (letter writer)
- Nhà buôn bí mật (secret merchant)

### D03. Tỉ lệ & hình khối — 18 giá trị

- Đầu to bằng thân, 1:1 (chibi 1:1)
- Cao hai đầu (two-head-tall)
- Cao ba đầu (three-head-tall)
- Cao bốn đầu (four-head-tall)
- Cao sáu đầu (six-head-tall)
- Cao tám đầu, dáng anh hùng (eight-head heroic)
- Thân dài, chân ngắn (long torso, short legs)
- Chân dài, thân ngắn (long legs, short torso)
- Vai rộng, hông hẹp (inverted triangle)
- Hông rộng, vai hẹp (pear shape)
- Thân tròn đều (round body)
- Thân thuôn giọt nước (teardrop body)
- Cổ dài (long neck)
- Bàn tay quá khổ (oversized hands)
- Bàn chân to (big feet)
- Khối vuông góc cạnh (boxy angular)
- Đầu nhỏ, thân lớn (tiny head, large body)
- Tay chân mềm như dây (noodle limbs)

### D04. Khuôn mặt & chi tiết — 20 giá trị

- Mắt chấm đen (dot eyes)
- Mắt một mí nặng (heavy-lidded eyes)
- Mắt to tròn long lanh (big sparkle eyes)
- Hai mắt không đều (asymmetric eyes)
- Lông mày rậm (thick brows)
- Mũi tam giác (triangle nose)
- Mũi tẹt tròn (button nose)
- Miệng chữ V nhỏ (small V mouth)
- Miệng rộng đến mang tai (wide grin)
- Má hồng tròn (round blush)
- Tàn nhang (freckles)
- Sẹo nhỏ trên mày (brow scar)
- Răng khểnh (snaggle tooth)
- Tai thú (animal ears)
- Râu tua tủa (whiskers)
- Tóc xoăn rối (messy curls)
- Tóc thẳng ôm mặt (sleek bob)
- Đầu hói bóng (shiny bald head)
- Kính dày (thick glasses)
- Lông mi dài (long lashes)

### D05. Dấu nhận dạng bất biến — 16 giá trị

- Đốm hình trái tim trên trán (heart-shaped forehead patch)
- Vết nứt hình tia chớp trên vai (lightning-shaped crack on shoulder)
- Một chiếc khuy lệch màu (single mismatched button)
- Đuôi buộc nơ đỏ (red ribbon on tail)
- Sẹo chéo trên mu bàn tay (diagonal hand scar)
- Đồng hồ dừng ở 3 giờ (watch stopped at 3)
- Một mắt khác màu (heterochromia)
- Khoen tai bạc (silver earring)
- Hình xăm con thuyền nhỏ (tiny boat tattoo)
- Khăn quàng luôn cùng một màu (signature scarf color)
- Một móng tay sơn khác (one painted nail)
- Túi vải vá hai mảnh (patched tote)
- Vân tay hình xoáy ốc (spiral fingerprint)
- Một lọn tóc trắng (single white streak)
- Vết mực ở ngón út (ink stain on pinky)
- Một bên dây giày không buộc (one untied lace)

### D06. Tính cách & thái độ — 20 giá trị

- Điềm tĩnh đến khó chịu (unnervingly calm)
- Hấp tấp (hasty)
- Ngại ngùng (shy)
- Kiêu ngạo nhẹ (mildly arrogant)
- Tò mò quá trớn (nosy)
- Thẳng thắn đến vụng (bluntly honest)
- Hay quên (forgetful)
- Cầu toàn (perfectionist)
- Lười nhưng khéo tay (lazy but deft)
- Hài hước kiểu bố (dad-joke humor)
- Nghiêm túc quá mức với việc nhỏ (overly earnest)
- Hoài nghi (skeptical)
- Dễ xúc động (easily moved)
- Can đảm việc nhỏ, nhút nhát việc lớn (brave in small things)
- Thích quan sát hơn nói (observer)
- Nhiệt tình quá đà (overenthusiastic)
- Bình thản khi mọi thứ sụp đổ (unflappable)
- Hay xin lỗi (habitual apologizer)
- Thích gây bất ngờ (fond of surprises)
- Trung thành với đồ vật (loyal to objects)

### D07. Bề mặt cơ thể — 16 giá trị

- Da giấy (paper skin)
- Da gốm tráng men (glazed ceramic skin)
- Bộ lông dày (plush fur)
- Lông tơ mịn (soft fuzz)
- Vảy mờ (matte scales)
- Da cao su (rubber skin)
- Thân trong suốt (translucent body)
- Thân có đường may (stitched seams)
- Bề mặt phấn (chalky surface)
- Kim loại xước (scratched metal)
- Gỗ vân rõ (visible wood grain)
- Đá phủ rêu (mossy stone)
- Thủy tinh có bọt khí (bubbled glass)
- Da sần vỏ cam (orange-peel texture)
- Nhựa bóng (glossy plastic)
- Vệt than chì (graphite smudge)

### D08. Chiến lược bảng màu — 16 giá trị

- Đơn sắc (monochrome)
- Tương đồng (analogous)
- Bổ sung (complementary)
- Bổ sung tách (split-complementary)
- Bộ ba (triadic)
- Bộ bốn (tetradic)
- Màu chủ đạo + một điểm nhấn (dominant + accent)
- Nóng/lạnh đối lập (warm/cool split)
- Giảm bão hòa, một điểm bão hòa (muted with one saturated note)
- Hai màu bù trừ + trung tính (two hues + neutrals)
- Màu đất (earth palette)
- Màu kẹo (candy palette)
- Màu bụi (dusty palette)
- Neon tiết chế (restrained neon)
- Đơn sắc ấm (warm monochrome)
- Nền đêm + một nguồn sáng (night + single light source)

### D09. Họ màu chủ đạo — 20 giá trị

- Đỏ son (vermilion)
- Đỏ rượu (oxblood)
- Hồng phấn (powder pink)
- Cam cháy (burnt orange)
- Vàng nghệ (turmeric)
- Vàng chanh (lemon)
- Xanh ô liu (olive)
- Xanh rêu (moss)
- Xanh ngọc (jade)
- Xanh mòng két (teal)
- Xanh mực (ink blue)
- Xanh băng (ice blue)
- Tím khoai (yam purple)
- Tím mận (plum)
- Nâu cacao (cocoa)
- Nâu đất son (sienna)
- Kem giấy cũ (aged paper)
- Xám bê tông (concrete grey)
- Trắng ngà (ivory)
- Đen mực (ink black)

### D10. Vai trò tương phản — 14 giá trị

- Nền sáng, chủ thể tối (light bg, dark subject)
- Nền tối, chủ thể sáng (dark bg, light subject)
- Viền sáng quanh chủ thể (light rim)
- Viền mực đậm (dark ink outline)
- Mảng tối lớn, điểm sáng nhỏ (large dark mass, small light accent)
- Tương phản độ bão hòa (saturation contrast)
- Tương phản nóng–lạnh (temperature contrast)
- Một màu bão hòa duy nhất (single saturated hue)
- Ba bậc sáng rõ ràng (three clear values)
- Ba sắc độ tách biệt, không chuyển tiếp (posterized values)
- Lõi tối, viền sáng, nền trung tính (dark core, light rim, neutral bg)
- Nền trung tính, chủ thể hai màu (neutral bg, two-hue subject)
- Điểm nhấn chỉ ở mắt (accent only at the eyes)
- Cùng độ sáng nền, tách bằng viền (same value, separated by outline)

### D11. Nét & phong cách vẽ — 18 giá trị

- Nét mực đều (uniform ink line)
- Nét lông thanh đạm (tapered brush line)
- Nét chì run (shaky pencil line)
- Nét khắc gỗ (woodcut line)
- Nét chấm (stippled line)
- Nét đứt quãng (broken line)
- Nét dày–mảnh đối lập (thick-thin line)
- Nét ngoài dày, nét trong mảnh (thick outer, thin inner)
- Không viền, mảng phẳng (no line, flat shapes)
- Viền trắng kiểu sticker (white sticker border)
- Viền kép (double outline)
- Viền màu tối hơn hình (darker tone outline)
- Nét mềm như bút lông (soft brush edge)
- Nét kỹ thuật thẳng (technical line)
- Nét phấn (chalk line)
- Nét in lưới thô (halftone edge)
- Nét cắt giấy (cut-paper edge)
- Nét mô phỏng mũi thêu (stitched line)

### D12. Render & đổ bóng — 16 giá trị

- Phẳng hoàn toàn (flat)
- Phẳng + một bóng cứng (flat with one hard shadow)
- Cel shading hai bậc (two-step cel shading)
- Cel shading ba bậc (three-step cel shading)
- Chuyển sắc mềm (soft gradient)
- Viền sáng mảnh (thin rim light)
- Ánh sáng tản (diffuse light)
- Ngược sáng (backlit)
- Sáng từ dưới (underlight)
- Bóng dạng khối cầu (spherical shading)
- Bóng gạch chéo kiểu khắc (hatched shadow)
- Chấm sáng điểm (highlight dots)
- Texture chấm lưới (dot texture)
- Bóng đổ tách khỏi nền (drop shadow)
- Viền sáng kiểu decal (decal glow)
- Chỉ viền, không đổ bóng (line only)

### D13. Chất liệu & phương tiện mô phỏng — 16 giá trị

- Màu nước (watercolor)
- Gouache
- Màu sáp (wax crayon)
- Bút dạ (marker)
- Chì màu (colored pencil)
- Mực lông (brush ink)
- Acrylic dày (thick acrylic)
- Cắt giấy (paper cut)
- Gấp giấy (origami)
- Đất sét (clay)
- Vải nỉ (felt)
- Thêu (embroidery)
- Kính màu (stained glass)
- In lưới (screen print)
- Pixel art
- 3D render nhám (matte 3D render)

### D14. Trang phục — 20 giá trị

- Áo khoác quá khổ (oversized coat)
- Yếm bò (overalls)
- Sơ mi sờn cổ (frayed collar shirt)
- Áo len chui đầu (turtleneck)
- Áo choàng có mũ (hooded cloak)
- Váy xếp ly (pleated skirt)
- Quần ống rộng (wide-leg trousers)
- Đồ liền thân (jumpsuit)
- Tạp dề (apron)
- Đồ bảo hộ (protective suit)
- Gi-lê nhiều túi (utility vest)
- Khăn choàng dài (long scarf)
- Găng hở ngón (fingerless gloves)
- Ủng cao (tall boots)
- Mũ mềm vành rộng (wide soft hat)
- Áo mưa trong (translucent raincoat)
- Đồ ngủ (pajamas)
- Vest may đo (tailored suit)
- Áo thun in hình (graphic tee)
- Đồ bơi một mảnh (one-piece swimsuit)

### D15. Phụ kiện — 18 giá trị

- Kính tròn gọng mảnh (thin round glasses)
- Kính bảo hộ (goggles)
- Tai nghe lớn (over-ear headphones)
- Khăn buộc cổ (neckerchief)
- Dây chuyền mặt nhỏ (small pendant)
- Vòng tay nhiều lớp (stacked bracelets)
- Nhẫn quá khổ (oversized ring)
- Túi đeo chéo (crossbody bag)
- Ba lô vá (patched backpack)
- Ô gấp (folding umbrella)
- Đồng hồ đeo tay (wristwatch)
- Băng dính trên má (cheek bandage)
- Huy hiệu cài ngực (lapel pin)
- Bút cài sau tai (pencil behind ear)
- Khẩu trang kéo xuống (mask pulled down)
- Mũ bảo hiểm (helmet)
- Dây giày lệch màu (mismatched laces)
- Túi giấy đựng đồ (paper bag)

### D16. Đạo cụ & vật cầm tay — 20 giá trị

- Cốc giữ nhiệt (thermos)
- Đèn bàn nhỏ (desk lamp)
- Sổ gáy xoắn (spiral notebook)
- Kéo cắt giấy (paper scissors)
- Hộp bút chì (pencil case)
- Bình tưới cây (watering can)
- Giỏ nấm (basket of mushrooms)
- Máy ảnh cũ (old camera)
- Băng cassette (cassette tape)
- Cờ lê nhỏ (small wrench)
- La bàn (compass)
- Kính lúp (magnifier)
- Bản đồ gấp (folded map)
- Chìa khóa quá khổ (oversized key)
- Hộp diêm (matchbox)
- Bóng bay cột dây (balloon on a string)
- Bánh mì kẹp (sandwich)
- Chậu cây nhỏ (potted plant)
- Quạt giấy (folding fan)
- Bảng vẽ nhỏ (small drawing tablet)

### D17. Nghề nghiệp — 20 giá trị

- Người bảo tồn bản thảo (manuscript conservator)
- Kỹ thuật viên âm thanh (audio technician)
- Người kiểm định chất lượng (quality inspector)
- Người dẫn chương trình radio đêm (late-night radio host)
- Thợ may đo (tailor)
- Người pha trà (tea blender)
- Người đóng sách (bookbinder)
- Nhân viên lưu trữ (archivist)
- Người vẽ bản đồ (cartographer)
- Kỹ thuật viên ánh sáng sân khấu (stage lighting technician)
- Người chăm vườn kính (greenhouse keeper)
- Thợ sửa đồng hồ (watchmaker)
- Thợ làm bánh (baker)
- Nhân viên trạm khí tượng (weather station observer)
- Người huấn luyện động vật (animal trainer)
- Thợ gốm (potter)
- Người soát vé (ticket inspector)
- Kỹ sư cầu đường (bridge engineer)
- Nhiếp ảnh gia động vật hoang dã (wildlife photographer)
- Giáo viên dạy bơi (swim instructor)

### D18. Sở thích & hoạt động — 18 giá trị

- Sưu tầm tem in lỗi (collecting misprinted stamps)
- Ghép hình (jigsaw puzzles)
- Trồng cây không khí (air plants)
- Nghe radio cũ (old radio)
- Chơi cờ một mình (solo chess)
- Viết nhật ký bằng hình (picture diary)
- Đan len (knitting)
- Câu cá dưới mưa (fishing in the rain)
- Làm mô hình giấy (papercraft)
- Sưu tầm đá (rock collecting)
- Nấu theo công thức cũ (cooking from old recipes)
- Học ngôn ngữ ký hiệu (sign language)
- Nhảy một mình (dancing alone)
- Sửa radio hỏng (repairing radios)
- Ký họa tốc độ (speed sketching)
- Đi bộ đêm (night walking)
- Làm vườn ban công (balcony gardening)
- Ngắm chim (birdwatching)

### D19. Thế giới & bối cảnh — 18 giá trị

- Thị trấn ven biển lúc thủy triều xuống (seaside town at low tide)
- Thành phố trên mái nhà (city on rooftops)
- Chợ đêm trong mưa (night market in the rain)
- Thư viện bỏ hoang (abandoned library)
- Trạm nghiên cứu trên băng (ice research station)
- Làng nổi trên đầm (floating village on a marsh)
- Khu vườn kính bỏ dở (unfinished greenhouse)
- Sa mạc muối (salt desert)
- Rừng trên mây (forest above the clouds)
- Ga tàu cuối tuyến (end-of-line train station)
- Xưởng in cũ (old print shop)
- Bảo tàng đang đóng cửa (museum after hours)
- Chung cư cao tầng giờ tan ca (high-rise at shift change)
- Đảo nhân tạo giữa hồ (artificial island in a lake)
- Thị trấn mô hình (model town)
- Hầm mỏ cải tạo thành nhà (mine converted into a home)
- Quán cà phê 24 giờ (24-hour café)
- Bến tàu sửa chữa (boat repair dock)

### D20. Thời kỳ & thẩm mỹ — 16 giá trị

- Art Deco
- Bauhaus
- Mid-century hiện đại (mid-century modern)
- Thập niên 1970 ấm (warm 1970s)
- Thập niên 1980 neon (1980s neon)
- Thập niên 1990 tối giản (1990s minimal)
- Y2K bóng (glossy Y2K)
- Tem bưu chính xưa (vintage postage)
- Tương lai thô (low-fi sci-fi)
- Tương lai bóng (sleek sci-fi)
- Hậu công nghiệp gỉ sét (post-industrial rust)
- Tiền cơ giới (pre-mechanical)
- Cổ điển mềm (soft classical)
- Tối giản chức năng (functional minimal)
- Thập niên 1960 biểu tượng (1960s pop)
- Đương đại đường phố (contemporary street)

### D21. Môi trường & ánh sáng — 16 giá trị

- Nắng sớm xiên (early slanted sun)
- Nắng gắt giữa trưa (harsh noon)
- Giờ vàng (golden hour)
- Giờ xanh (blue hour)
- Neon hắt (neon spill)
- Đèn huỳnh quang (fluorescent)
- Ánh nến (candlelight)
- Sương mù dày (thick fog)
- Mưa phùn (drizzle)
- Tuyết rơi nhẹ (light snow)
- Bụi lơ lửng trong nắng (dust motes in sun)
- Bóng râm tán cây (leaf shade)
- Ánh sáng cửa sổ (window light)
- Đèn xe quét qua (passing headlights)
- Trăng lạnh (cold moonlight)
- Bảng hiệu nhấp nháy (flickering sign)

### D22. Góc nhìn — 14 giá trị

- Chính diện ngang tầm mắt (eye-level front)
- Ba phần tư trái (three-quarter left)
- Ba phần tư phải (three-quarter right)
- Nghiêng mặt 3/4 (three-quarter face)
- Từ trên xuống (high angle)
- Từ dưới lên (low angle)
- Nhìn qua vai (over the shoulder)
- Toàn thân (full body)
- Nửa người (half body)
- Cận mặt (close-up face)
- Từ trên nhìn xuống đỉnh đầu (top-down)
- Nghiêng ngang (profile)
- Từ phía sau, quay đầu lại (from behind, looking back)
- Góc rộng thấy cả đạo cụ (wide enough to include the prop)

### D23. Silhouette & bố cục sticker — 13 giá trị

- Khối tròn đầy (round blob)
- Giọt nước (teardrop)
- Chữ A (A-frame)
- Chữ T (T-shape)
- Bất đối xứng, một bên nặng (asymmetric, weighted to one side)
- Viền có gai (spiky contour)
- Viền có tua (frayed contour)
- Một điểm nhô rõ (single clear protrusion)
- Có khoảng trống bên trong (negative-space hole)
- Chủ thể chiếm gần kín khung (fills the frame)
- Chủ thể nhỏ giữa khung (small in frame)
- Bố cục đường chéo (diagonal)
- Ba lớp chồng (three-layer composition)

### D24. Biểu cảm & dáng — 20 giá trị

- Đứng nghiêng, tay trong túi (standing, hands in pockets)
- Ngồi bó gối (sitting, knees hugged)
- Nằm sấp chống cằm (lying prone, chin on hand)
- Nhón chân với lấy (reaching on tiptoe)
- Ôm vật lớn (hugging a large object)
- Giơ tay chào (waving)
- Che mặt bằng tay (hiding face)
- Ngáp (yawning)
- Cười mím (tight smile)
- Cau mày nhìn kỹ (squinting closely)
- Giật mình lùi lại (startled, leaning back)
- Chống nạnh (hands on hips)
- Khoanh tay (arms crossed)
- Cúi xuống nhặt (bending to pick up)
- Quay lưng bước đi (walking away)
- Nhảy lên (jumping)
- Ngồi trên mép (perched on an edge)
- Đứng một chân (standing on one leg)
- Rụt cổ (hunched)
- Đưa tay ra trước (reaching forward)

---

## 5. Năm bộ tổ hợp

Mỗi bộ chọn **một** giá trị cho các chiều chính, kèm lý do bộ này "đứng" được: có xung đột nhẹ giữa các chiều, nhưng cấu trúc sáng và silhouette được thiết kế để đọc được ở cỡ sticker nhỏ. Đây là ví dụ biên soạn, không phải công thức tối ưu; độ đọc ở kích thước nhỏ là **chủ ý thiết kế, chưa được kiểm chứng bằng thử nghiệm**.

### Bộ 1 — Người giữ kho ký ức

| Chiều | Chọn |
|---|---|
| D01 | Giọt mực sống |
| D02 | Người gác kho ký ức |
| D03 | Cao ba đầu, thân thuôn giọt nước |
| D05 | Một lọn tóc trắng |
| D07 | Thân trong suốt |
| D08 | Nền đêm + một nguồn sáng |
| D09 | Xanh mực + kem giấy cũ |
| D11 | Viền trắng kiểu sticker |
| D12 | Cel shading hai bậc |
| D14 | Sơ mi sờn cổ |
| D16 | Sổ gáy xoắn + kính lúp |
| D19 | Thư viện bỏ hoang |
| D21 | Ánh nến |
| D22 | Ba phần tư trái |
| D23 | Một điểm nhô rõ (cuốn sổ) |

Vì sao đứng: thân trong suốt gần như cùng độ sáng nền, nên **viền trắng** làm nhiệm vụ tách hình; lọn tóc trắng là chi tiết duy nhất phá được mảng trong suốt. Đạo cụ nhô ra tạo silhouette có "móc" để mắt bám.

### Bộ 2 — Thợ sửa trên trạm băng

| Chiều | Chọn |
|---|---|
| D01 | Robot hộp thiếc |
| D02 | Người sửa chữa |
| D03 | Khối vuông góc cạnh |
| D05 | Một chiếc khuy lệch màu |
| D07 | Kim loại xước |
| D08 | Màu chủ đạo + một điểm nhấn |
| D09 | Xanh băng + xám bê tông + đỏ son |
| D11 | Nét kỹ thuật thẳng |
| D12 | Viền sáng mảnh |
| D14 | Đồ bảo hộ |
| D15 | Kính bảo hộ |
| D16 | Cờ lê nhỏ |
| D19 | Trạm nghiên cứu trên băng |
| D21 | Trăng lạnh |
| D22 | Từ dưới lên |
| D23 | Chữ A |

Vì sao đứng: hai màu lạnh gần nhau sẽ dính thành một khối, nên điểm nhấn đỏ son đặt ở **khuy áo lệch** — vừa là dấu nhận dạng, vừa là điểm bão hòa duy nhất. Góc từ dưới lên làm robot hộp thiếc bớt "đồ chơi".

### Bộ 3 — Cá voi bưu tá trên sa mạc muối

| Chiều | Chọn |
|---|---|
| D01 | Cá voi mini bay |
| D02 | Người dẫn đường |
| D03 | Thân tròn đều |
| D05 | Đuôi buộc nơ đỏ |
| D07 | Da cao su |
| D08 | Đơn sắc ấm (thêm một điểm lạnh — biến thể tự định nghĩa) |
| D09 | Kem giấy cũ + vàng nghệ + xanh mòng két |
| D11 | Không viền, mảng phẳng |
| D12 | Phẳng + một bóng cứng |
| D15 | Túi đeo chéo |
| D16 | Chìa khóa quá khổ + bản đồ gấp |
| D19 | Sa mạc muối |
| D21 | Nắng gắt giữa trưa |
| D22 | Từ trên nhìn xuống |
| D23 | Khối tròn đầy |

Vì sao đứng: bóng cứng đổ về một phía là thứ duy nhất cho thấy khối cầu trên nền phẳng; nơ đỏ là điểm bão hòa duy nhất trong bảng màu ấm. Cả bộ chỉ có ba màu, nhằm giữ độ đọc khi thu nhỏ — đây là chủ ý, chưa đo ở kích thước cụ thể nào.

### Bộ 4 — Đầu bếp lửa lạnh

| Chiều | Chọn |
|---|---|
| D01 | Người đất sét |
| D02 | Bậc thầy khó tính |
| D03 | Bàn tay quá khổ |
| D05 | Sẹo chéo trên mu bàn tay |
| D06 | Nghiêm túc quá mức với việc nhỏ |
| D07 | Bề mặt phấn |
| D08 | Nóng/lạnh đối lập |
| D09 | Cam cháy + xanh băng |
| D11 | Nét ngoài dày, nét trong mảnh |
| D12 | Bóng dạng khối cầu |
| D14 | Tạp dề |
| D16 | Cốc giữ nhiệt |
| D19 | Khu vườn kính bỏ dở |
| D21 | Ánh sáng cửa sổ |
| D22 | Nghiêng ngang |
| D23 | Chủ thể chiếm gần kín khung |

Vì sao đứng: nghịch lý "bếp lửa trong vườn kính lạnh" thể hiện bằng **cặp nhiệt độ** chứ không bằng đạo cụ; bàn tay to kéo mắt xuống chiếc cốc. Ở góc nghiêng, sẹo tay vẫn thấy, còn tạp dề tạo mảng sáng lớn; ánh sáng cửa sổ cho khối cầu của thân đất sét có một phía tối rõ.

### Bộ 5 — Nhà bản đồ học rêu

| Chiều | Chọn |
|---|---|
| D01 | Sen đá biết đi |
| D02 | Người sưu tầm thầm lặng |
| D03 | Thân dài, chân ngắn |
| D04 | Mắt chấm đen |
| D05 | Vân tay hình xoáy ốc |
| D07 | Đá phủ rêu |
| D08 | Hai màu bù trừ + trung tính |
| D09 | Xanh rêu + nâu đất son + vàng chanh |
| D11 | Nét cắt giấy |
| D13 | Cắt giấy |
| D14 | Gi-lê nhiều túi |
| D15 | Bút cài sau tai |
| D16 | Bản đồ gấp + la bàn |
| D19 | Thành phố trên mái nhà |
| D21 | Giờ vàng |
| D22 | Nhìn qua vai |
| D23 | Bất đối xứng, một bên nặng |

Vì sao đứng: chất liệu cắt giấy giải thích luôn kiểu nét — không cần viền mực. Nhìn qua vai cho phép thấy cả bản đồ và dấu vân tay; bất đối xứng vì chậu cây trên đầu nặng về một phía.

---

## 6. Mảnh ghép do người dùng tự định nghĩa

Thư viện chỉ có ích nếu người dùng thêm được thứ của họ. Gợi ý (đề xuất, chưa triển khai):

1. **Một ý niệm một mảnh.** "Áo khoác vàng có mũ" ổn; "áo khoác vàng, mũ đỏ, ủng xanh" nên tách ba mảnh, vì mỗi thứ có thể đổi độc lập.
2. **Ghi rõ tầng biến thiên** ngay khi tạo: bất biến, theo pack, hay tự do. Không ghi thì sáu tháng sau không ai nhớ.
3. **Kể chuyện được, nhưng phải dịch sang dấu hiệu nhìn thấy.** Quá khứ, quan hệ, động cơ là ngữ cảnh hữu ích để *chọn* chi tiết; chỉ phần biến thành hình — sẹo cũ, ảnh chụp chung, vật kỷ niệm, tư thế quen thuộc — mới tới được ảnh. Đừng liệt kê thuộc tính vô hình rồi kỳ vọng mô hình tự suy ra.
4. **Tránh mảnh mâu thuẫn cùng chiều** trừ khi mâu thuẫn là chủ ý: hai loài, hai chất liệu da, hai hướng sáng.
5. **Tái dùng cụm từ đã hiệu quả.** Giữ cách diễn đạt ổn định giúp so sánh các tổ hợp, nhưng đầu ra còn phụ thuộc model, tham số và tính ngẫu nhiên. Đổi một mảnh không đủ để quy toàn bộ khác biệt kết quả cho mảnh đó.
6. **Tên riêng, thương hiệu, nhân vật có bản quyền: dùng khi bạn có quyền.** Tác phẩm gốc của bạn, tài sản đã mua license, hoặc bên thứ ba đã cho phép đều hợp lệ. Cần xác định trước phạm vi được phép (thương mại hay cá nhân, có được chỉnh sửa, có được phân phối lại sticker), và **không suy ra quyền từ việc mô hình chấp nhận prompt** — mô hình không kiểm tra quyền, cũng không cấp quyền. Nếu không chắc, mô tả bằng đặc điểm thị giác thay vì tên.
7. **Mảnh do người dùng tạo không được trộn với danh mục chuẩn** khi chia sẻ, để người khác biết đâu là đề xuất biên soạn.

Câu hỏi tự kiểm: nếu bỏ mảnh này đi, nhân vật có còn là nó không? Nếu không → bất biến. Nếu có → tự do.

---

## 7. Giới hạn của ảnh tham chiếu

Ảnh tham chiếu là công cụ mạnh nhưng hữu hạn. Cần nói rõ để không hứa điều không làm được:

- **Ảnh chỉ chốt những gì nó cho thấy.** Một ảnh chính diện không nói gì về lưng, đuôi, hay mặt bên; mô hình sẽ **tự suy diễn** phần khuất. Suy diễn đó thường không nhất quán giữa các lần chạy.
- **Màu không được bảo toàn.** Ảnh tham chiếu truyền *cảm giác* về màu, không truyền giá trị điểm ảnh. Đừng kỳ vọng hex khớp; hãy chọn lại bằng mắt sau khi sinh.
- **Nét và độ dày viền trôi theo tỉ lệ.** Cùng một ảnh, xuất ở 128 px và 512 px cho cảm giác nét khác nhau. Nếu cần nhất quán, cố định kích thước xuất hoặc chấp nhận nét đổi.
- **Nhiều ảnh tham chiếu dễ mâu thuẫn.** Ảnh khác nhau về tỉ lệ hay chi tiết có thể khiến kết quả trộn đặc điểm hoặc trôi về một ảnh; không có cách chắc chắn điều khiển ảnh nào chiếm ưu thế. Chọn một ảnh "chuẩn", nói rõ nó chuẩn ở điểm nào, và coi các ảnh còn lại là tham khảo phụ.
- **Chi tiết nhỏ không sống sót.** Dấu nhận dạng nhỏ hơn vài phần trăm chiều cao nhân vật thường biến mất hoặc biến dạng. Dấu nhận dạng nên đủ lớn để đọc ở kích thước sticker thật.
- **Văn bản trong ảnh không đáng tin.** Chữ, nhãn, biển hiệu thường sai chính tả hoặc thành hình lạ. Nếu cần chữ, thêm ở bước hậu kỳ, không giao cho mô hình.
- **Không có bảo đảm tuân thủ.** Không mô hình nào hứa giữ đúng mọi mảnh prompt. Ảnh tham chiếu giảm phương sai, không xóa nó.

### 7.1 Ràng buộc riêng của công cụ hiện tại

Phần này **không** phải nguyên tắc sáng tác chung, mà là ràng buộc của engine đang có:

- Nhánh `simple` lấy alpha từ **flood fill gần trắng** (khác biệt RGB với 255 ≤ 10), không phải chroma key theo màu tùy chọn. Khi mask bị đánh giá yếu, `_ink_halo_alpha` là đường dự phòng tái tạo mask quanh ink; đây **không phải công cụ viền màu/độ dày tùy chỉnh**. Ảnh có alpha vượt heuristic hiện tại hoặc nhánh `rembg` đi đường khác; xem [phân tích pipeline](studio-composition.md).
- Hệ quả cho thư viện: **bối cảnh sáng tác trong prompt phải tách khỏi nền xuất**. Mô tả "ánh sáng, không khí, đạo cụ gắn với nhân vật" an toàn hơn mô tả "một mảng nền trắng/xanh lớn". Mảng trắng trong chính nhân vật (tạp dề trắng, mắt trắng, giấy trắng) có nguy cơ bị ăn vào.
- Đạo cụ mảnh, thứ tách rời (ly tách rời tay, glow, chữ rời, bong bóng) dễ bị cắt mất. Chọn đạo cụ **dính liền silhouette** hoặc nằm trong khối chính.
- Vùng tìm sticker dựa trên contour alpha, không dựa trên lưới ô cố định — nên số sticker trong một sheet không bị giới hạn bởi lưới, nhưng khoảng cách giữa các sticker vẫn cần đủ để contour không dính nhau.

Những điểm trên nên được kiểm chứng lại khi engine đổi; đây là mô tả hành vi tại thời điểm biên soạn.

---

## 8. Giới hạn của tài liệu này

- **Không đầy đủ và không thể đầy đủ.** 425 giá trị là điểm khởi đầu có thể duyệt, không phải không gian sáng tạo. Mọi danh mục đóng đều tạo ra nhân vật giống nhau.
- **Không phải chuẩn.** Các tên gọi như "xanh mực" là nhãn biên soạn; không có giá trị đo màu kèm theo. Nếu cần tái lập chính xác, phải chỉ định không gian màu và giá trị cụ thể — CSS Color 4 hỗ trợ sRGB, Display P3, và các không gian như OkLCh ([W3C CSS Color 4](https://www.w3.org/TR/css-color-4/)).
- **Không hứa tuân thủ.** Không mô hình sinh ảnh nào bảo đảm thực thi đúng mọi mảnh; tài liệu này cũng không đo tỉ lệ tuân thủ.
- **Không phải taxonomy chính thức.** Getty AAT là chuẩn cho tài liệu văn hóa vật chất, không dành cho nhân vật hư cấu; việc mượn cấu trúc facet là ẩn dụ có chủ đích.
- **Tránh định kiến.** Các giá trị cố ý mô tả *hình thức* (áo choàng có mũ, váy xếp ly) thay vì gán trang phục cho một quốc gia hay dân tộc. Người dùng tự thêm bối cảnh văn hóa cụ thể của họ nếu muốn, và tự chịu trách nhiệm về tính phù hợp.
- **Ngày tháng.** Tài liệu viết 2026-11-10; các liên kết có thể đổi.

---

## 9. Nguồn tham chiếu

| Nguồn | Dùng cho điều gì | Ghi chú |
|---|---|---|
| [Getty Research Institute — About the AAT](https://www.getty.edu/research/tools/vocabularies/aat/about.html) | Ý tưởng facet, quan hệ rộng–hẹp, polyhierarchy, yêu cầu có căn cứ sử dụng, phân biệt thuật ngữ chung với tên riêng | Dữ liệu Getty vocabularies phát hành theo [ODC-By 1.0](https://opendatacommons.org/licenses/by/1-0/); tài liệu này **không** sao chép bản ghi AAT, chỉ mượn cách tổ chức |
| [W3C — CSS Color Module Level 4](https://www.w3.org/TR/css-color-4/) | Không gian màu (sRGB, Display P3), OkLCh với độ sáng đồng đều về cảm nhận, các chiến lược nội suy hue (shorter/longer/increasing/decreasing), 148 màu có tên | Bản Candidate Recommendation Draft; dùng để giải thích vì sao mô tả màu bằng lời là mơ hồ |
| [Adobe Express — The color wheel explained](https://www.adobe.com/express/learn/blog/color-wheel-explained) | Các kiểu hòa phối (bổ sung, tương đồng, bộ ba, bộ bốn), shade/tint/tone, khác biệt RYB/RGB/CMYK | Tài liệu giáo dục của Adobe; không phải chuẩn kỹ thuật |
| [W3C WAI — Understanding SC 1.4.3 Contrast (Minimum)](https://www.w3.org/WAI/WCAG22/Understanding/contrast-minimum.html) | Ngưỡng 4.5:1 cho chữ thường, 3:1 cho chữ lớn; lý do dùng tương phản độ sáng thay vì hue | WCAG áp cho nội dung web; ở đây dùng như **suy luận** về độ đọc của sticker |
| [W3C WAI — Understanding SC 1.4.11 Non-text Contrast](https://www.w3.org/WAI/WCAG22/Understanding/non-text-contrast.html) | Ngưỡng 3:1 cho graphical object và ranh giới; cảnh báo nét mảnh bị mờ do khử răng cưa | Cơ sở cho gợi ý viền sticker đủ tương phản |
| [W3C WAI — Understanding SC 1.4.1 Use of Color](https://www.w3.org/WAI/WCAG22/Understanding/use-of-color.html) | Màu không được là phương tiện duy nhất truyền thông tin | Cơ sở cho việc dấu nhận dạng nên có hình khối, không chỉ đổi màu |

### 9.1 Điều gì lấy từ nguồn, điều gì không

- **Có nguồn:** cấu trúc facet và quan hệ phân cấp (Getty); sự tồn tại của các kiểu hòa phối màu và shade/tint/tone (Adobe); các ngưỡng tương phản và lý do dùng độ sáng (W3C); các không gian màu và tính phi tuyến của hue trong HSL (W3C).
- **Không có nguồn, là biên soạn:** toàn bộ 425 giá trị, cách chia 24 chiều, phân loại bất biến/bán biến thiên/tự do, năm bộ tổ hợp, các quy tắc tự định nghĩa mảnh.
- **Là suy luận:** áp ngưỡng WCAG vào sticker; cho rằng viền tương phản giúp tách hình khi cùng độ sáng; cho rằng đạo cụ dính silhouette an toàn hơn đạo cụ rời.

---

## 10. Việc chưa làm

- Chưa có bộ giá trị màu kèm mã (hex/OkLCh) cho từng họ màu — cần nếu muốn tái lập chính xác.
- Chưa đo tỉ lệ tuân thủ prompt trên bất kỳ mô hình cụ thể nào.
- Chưa có bản dịch/đối chiếu thuật ngữ sang AAT cho từng giá trị.
- Chưa kiểm chứng hành vi cắt nền trên các bộ tổ hợp ở §5; các ghi chú rủi ro là dự đoán từ mô tả engine, không phải kết quả thử.
