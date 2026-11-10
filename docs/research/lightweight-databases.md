# Database nhẹ cho Restick Sticker Factory

Ngày nghiên cứu: 2026-11-10. **Người dùng đã chốt SQLite + filesystem** trong cuộc trao đổi cùng ngày. Thiết kế tổng thể, migration và triển khai chưa được duyệt.

## 1. Câu hỏi cần trả lời

Ngoài SQLite, có database nào gọn và tối ưu hơn cho một người quản lý Sticker Factory, nhiều client chỉ đọc/tải tài nguyên, không dùng PostgreSQL và MinIO?

**Kết luận:** có các database nhúng gọn như **bbolt** và **LMDB**, nhưng chưa có bằng chứng chúng ít RAM hoặc nhanh hơn SQLite trên workload Restick. **bbolt là phương án thay thế đáng cân nhắc nhất nếu giữ backend Go và chấp nhận lưu key–value; SQLite vẫn là mặc định hợp lý hơn cho catalog có quan hệ và nhiều điều kiện lọc.** Đây là đánh giá phù hợp kiến trúc, không phải kết quả benchmark.

## 2. Tiêu chí, không đánh đồng chữ “nhẹ”

| Tiêu chí | Câu hỏi cho Restick |
| --- | --- |
| Vận hành | Có cần một server/database daemon riêng không? |
| Bộ nhớ | RSS, Go heap, OS page cache, mmap và đỉnh khi ghi/backup là bao nhiêu? |
| CPU/I/O nền | Có compaction hoặc value-log GC cần theo dõi không? |
| Dữ liệu | Có lọc theo nhiều thuộc tính, quan hệ pack–sticker–tag, phân trang, sắp xếp không? |
| Nhất quán | Cập nhật catalog, chỉ mục và trạng thái có thể commit cùng transaction không? |
| Bảo trì | Phần tìm kiếm, index, integrity và tooling nào ứng dụng phải tự viết? |
| Triển khai | Có ổ đĩa local bền vững? Có cần nhiều process hoặc nhiều máy ghi cùng dữ liệu? |
| Khôi phục | Backup nhất quán, mở lại sau crash, kiểm tra và phục hồi có đơn giản không? |

“Không có server riêng” không có nghĩa dùng 0 RAM. “Thư viện mã nhỏ” không phải “toàn ứng dụng RSS nhỏ”. SQLite, bbolt, LMDB, Badger và DuckDB đều có cách dùng embedded; không cần chuyển sang key–value chỉ để bỏ database container.

## 3. Workload đang biết

Theo mục tiêu người dùng đã xác nhận và mã/catalog hiện tại:

- Nội dung cần lưu: mảnh ghép, nhóm, prompt/tổ hợp, bộ sticker, metadata ảnh, API key và trạng thái xử lý.
- Một người quản lý, nhưng vẫn có request từ client và kết quả tác vụ nền: không được hiểu là hoàn toàn không có concurrency.
- Nhiều phép đọc nhỏ theo ID; đồng thời có nhu cầu duyệt, lọc theo nhóm/tag/trạng thái và sắp xếp danh mục.
- Ảnh gốc, sticker, thumbnail và ZIP có thể lưu ở filesystem thay cho MinIO; chọn database metadata là vấn đề riêng.
- Go là backend hiện có, Python xử lý ảnh và Flutter là SDK client. Không chốt lại ngôn ngữ chỉ để chọn database.
- Backend hiện dùng `pgxpool` và SQL đặc thù PostgreSQL; đổi sang bất kỳ database nào cũng không chỉ là đổi connection string.

Liên quan: [nghiên cứu studio](sticker-factory/README.md), [quy tắc ghép prompt/xử lý ảnh](sticker-factory/studio-composition.md).

## 4. So sánh những ứng viên thực sự khác nhau

Các mô tả dưới đây dựa trên tài liệu chính thức; cột “phù hợp” là suy luận cho Restick, không phải đo hiệu năng.

| Database | Mô hình / tích hợp | Điểm mạnh có căn cứ | Phần phải đánh đổi | Đánh giá cho Restick |
| --- | --- | --- | --- | --- |
| **SQLite — mốc so sánh** | SQL embedded; nhiều lựa chọn driver | Transactions, truy vấn và chỉ mục quan hệ; không cần database server | Một writer tại một thời điểm; phải chọn cấu hình/driver/backup phù hợp | Ưu tiên khi catalog có nhiều điều kiện lọc và liên kết |
| **bbolt** | Key–value B+tree, pure Go, file dữ liệu đơn | ACID, nhiều reader/một writer, cursor/prefix/range scan, API nhỏ, hot backup | Không có SQL/JOIN; ứng dụng phải quản lý index theo thuộc tính, unique constraints và serialization | Lựa chọn thay thế thực tế nhất nếu query đơn giản và thiên về ID/prefix |
| **LMDB** | Key–value memory-mapped, thư viện C | ACID/MVCC, nhiều reader/một writer; nhiều process cùng host có thể mở cùng environment | Go cần lớp tích hợp C/binding; kiểm soát vòng đời transaction, map sizing và index riêng | Đáng chọn khi đo được lợi ích cụ thể hoặc cần tính chất multiprocess local |
| **BadgerDB** | Key–value pure Go; LSM + value log | Concurrent ACID transactions; thiết kế nhắm tới SSD và workload key–value lớn | Compaction, cache và value-log GC; ứng dụng chủ động gọi GC theo tài liệu, vẫn phải xây index ngoài mô hình key | Không ưu tiên cho studio ít người ghi; phù hợp hơn khi tải ghi KV là bài toán chính |
| **DuckDB** | SQL embedded hướng phân tích | Tốt cho truy vấn phân tích, scan/aggregate và dữ liệu cột | Nhiều query nhỏ đồng thời không phải mục tiêu thiết kế chính; không có căn cứ để chọn chỉ vì muốn nhẹ | Không ưu tiên làm database vận hành studio; cân nhắc nếu sau này thật sự cần analytics |

Không kết luận “DuckDB không có transactions” hoặc “không bao giờ dùng làm server”: tài liệu hiện tại hỗ trợ transactions và có các lựa chọn khác nhau theo phiên bản. Lý do không chọn ở đây là **mục tiêu workload**, không phải thiếu hẳn khả năng.

### 4.1 bbolt: nhỏ ở engine, không tự nhỏ ở ứng dụng

Nguồn [S1] mô tả kho key–value với các thao tác bucket/key/cursor, không phải relational query engine.

Nếu lưu pack theo ID, việc lấy một bản ghi là trực tiếp. Nhưng nếu cần tìm:

> Các bộ có tag “mèo”, đã sẵn sàng, theo tên hoặc thời gian cập nhật, có phân trang.

Ứng dụng phải chọn giữa quét và lọc dữ liệu, hoặc tạo/duy trì các index key riêng. bbolt cho phép cập nhật nhiều bucket trong một transaction, nên **có thể** giữ index và dữ liệu nhất quán; trách nhiệm viết đúng thuộc về ứng dụng.

Các caveat trong README [S1]:

- Một transaction ghi tại một thời điểm; nhiều transaction đọc.
- Transaction đọc kéo dài giữ lại các trang cũ và có thể ảnh hưởng thu hồi dung lượng.
- Mặc định mở đọc–ghi giữ file lock; không thiết kế để nhiều process cùng ghi. Chế độ chỉ đọc có cơ chế shared lock khác, không đồng nghĩa một writer và nhiều process reader hoạt động như LMDB.
- mmap có thể hiện thành mức sử dụng bộ nhớ lớn; OS cache không đồng nghĩa toàn bộ là Go heap hay bộ nhớ không thể thu hồi.
- Xóa key không tự thu nhỏ file database; cần phân biệt tái sử dụng trang với trả dung lượng về filesystem.
- Không tắt sync chỉ để có benchmark đẹp rồi gọi đó là cải thiện tương đương độ bền.

### 4.2 LMDB: rất gọn, nhưng có thêm công việc tích hợp

Nguồn Symas và tài liệu C [S2] xác nhận memory mapping, transaction, reader/writer semantics và hỗ trợ nhiều process cùng host. Thư viện được viết bằng C; một backend Go cần chọn và duy trì binding tương ứng.

Tài liệu nhắc transaction gắn với thread theo chế độ mặc định, giá trị trả về có thể trỏ vào mmap và chỉ hợp lệ trong vòng đời transaction. Vì vậy “ít mã trong engine” không tự động biến thành “ít rủi ro tích hợp nhất trong Go”.

Không lấy con số kích thước object code quảng bá của LMDB để kết luận RSS hoặc kích thước Docker image của Restick. Chưa khảo sát binding/phiên bản đủ để chốt nó.

### 4.3 Badger: mạnh ở bài toán khác, không mặc định nhẹ hơn

Badger có transactions và pure Go [S3]. Thiết kế LSM/value-log có các cơ chế vận hành khác với B+tree.

Quickstart [S4] nêu database gồm nhiều file trong thư mục, các process không mở chung database đồng thời theo cách thông thường, và việc thu hồi value log được ứng dụng gọi bằng `RunValueLogGC`. Khi dữ liệu nằm inline trong LSM, cơ chế thu hồi lại khác. Không gọi tất cả các loại thu hồi dung lượng bằng một quy tắc duy nhất.

Điều này không chứng minh Badger luôn tốn RAM hơn SQLite. Nó cho thấy có thêm các yếu tố cache/compaction/GC phải đánh giá, trong khi chưa có nhu cầu ghi lớn để bù lại độ phức tạp đó.

### 4.4 DuckDB: giữ đúng mục tiêu tối ưu

Tài liệu workload [S5] viết rõ việc chạy nhiều truy vấn nhỏ đồng thời không phải mục tiêu chính; engine tối ưu cho truy vấn lớn hơn, ít thường xuyên hơn. Có hướng dẫn tái sử dụng connection và các cơ chế quản lý bộ nhớ/spill cho phân tích.

Đó là lý do không chọn DuckDB làm mặc định cho thao tác chỉnh sửa fragment, cập nhật job và đọc từng pack. Không dùng benchmark scan/aggregate của DuckDB để suy ra độ trễ API catalog tốt hơn.

## 5. Về việc tránh CGO

Không cần bỏ SQLite chỉ vì muốn build Go không phụ thuộc CGO. README chính thức của `modernc.org/sqlite` mô tả **Pure-Go SQLite, no cgo** [S6].

Đây là lựa chọn driver, không phải chứng minh driver nào nhanh/nhẹ hơn. Cần đánh giá binary size, startup, bộ nhớ, phạm vi nền tảng hỗ trợ và dependency theo bản dùng thực tế. Không cài hay chốt driver trong nghiên cứu này.

Python không nên được mặc định cùng mở database chỉ vì engine có binding Python. Giữ quyền cập nhật metadata trong backend là một cách giảm số bên ghi, nhưng đó là quyết định kiến trúc cần duyệt, không phải yêu cầu của engine.

## 6. Hai lựa chọn đề xuất để đi tiếp

### A. SQLite + filesystem — đã được người dùng chọn

Phù hợp nếu studio cần thư viện nhiều nhóm/thuộc tính, tìm kiếm/lọc/sắp xếp và quan hệ giữa nhân vật, prompt, pack, tag và tác vụ.

Lợi ích chính là **ít logic dữ liệu tự viết**, không phải lời hứa luôn ít RAM nhất. Với backend đơn trên ổ đĩa local bền vững và workload đã mô tả, đây là lựa chọn bảo thủ có cơ sở [S7].

### B. bbolt + filesystem — phương án thay thế đáng thử

Phù hợp nếu người dùng ưu tiên kho dữ liệu Go đơn giản, phần lớn truy cập theo ID/prefix và bộ lọc đã được giới hạn rõ. Có thể truy vấn bằng scan ở dữ liệu nhỏ hoặc tự xây index nếu cần, nhưng phải chấp nhận chi phí đó thay vì coi là miễn phí.

Không dùng cả SQLite lẫn bbolt chỉ để “mỗi cái mạnh một phần” khi chưa có bài toán yêu cầu hai nguồn lưu metadata.

**Quyết định đã chốt: dùng SQLite cho metadata và filesystem cho tài nguyên.** Không thêm bbolt, LMDB, Badger hoặc DuckDB vào kiến trúc mặc định. Phần so sánh trên được giữ làm căn cứ nghiên cứu; chưa cài hoặc migrate database.

## 7. Những điều không phụ thuộc database

- API key client phải chỉ đọc; quyền quản lý vẫn tách biệt.
- Không expose file database, ảnh nguồn hoặc thư mục backup qua đường tải công khai.
- Cần metadata bền vững và xử lý tình huống crash của tác vụ ảnh; một người quản lý không loại bỏ lỗi mất điện.
- SQLite transaction hay KV transaction không tự tạo atomic transaction chung với filesystem: phải thiết kế lúc file mới trở nên có thể tải và cách dọn phần còn dở.
- Backup phải khôi phục được **cả metadata lẫn file** trong trạng thái tương thích; không copy database đang chạy một cách tùy tiện rồi gọi đó là backup đã kiểm chứng.
- Đổi database không tự làm cắt ảnh nhanh hơn. CPU/RAM của Pillow/OpenCV/rembg là một phần khác cần đo.
- REST hay GraphQL không quyết định phải dùng engine nào.

## 8. Nếu cần chứng minh “tối ưu hơn”

Chưa thực hiện benchmark. Một phép so sánh có ý nghĩa cần dùng cùng điều kiện và nghiệp vụ:

- Cùng dữ liệu metadata đại diện; file ảnh/ZIP không trộn vào một bên để làm sai lệch.
- Các thao tác đọc ID, danh sách có filter/sort/page, thêm/sửa fragment, commit pack và cập nhật trạng thái.
- Độ bền tương đương, cùng disk/OS/hardware, tách warm/cold cache; không so một bên tắt sync với bên bật.
- Đo bộ nhớ idle/peak, CPU, I/O, độ trễ và kích thước file sau cập nhật/xóa; không đánh đồng mmap address space, RSS và Go heap.
- Tính cả mã index/query bổ sung và thao tác backup/restore, không chỉ throughput engine.

Các dòng này là tiêu chí đánh giá để dùng khi cần, không phải cam kết xây benchmark, không thêm framework hoặc tạo dependency.

## 9. Nguồn chính thức và kiểm chứng

- **[S1] bbolt README / package overview:** https://github.com/etcd-io/bbolt — đọc trực tiếp; transactions, buckets, backup, caveats, mmap, file lock và giới hạn thu nhỏ file. Phần so sánh nhà cung cấp không được dùng làm kết quả đo Restick.
- **[S2] Symas LMDB:** https://www.symas.com/lmdb (chuyển hướng trang `.php`); tài liệu gốc: https://raw.githubusercontent.com/LMDB/lmdb/mdb.master/libraries/liblmdb/intro.doc; mirror chính thức https://github.com/LMDB/lmdb — đọc trực tiếp. Trang `https://www.lmdb.tech/doc/` lỗi TLS trong lần nghiên cứu, đã dùng nguồn chính thức thay thế.
- **[S3] Badger README:** https://github.com/dgraph-io/badger — đọc trực tiếp; pure Go, transaction và mục tiêu thiết kế.
- **[S4] Badger Quickstart:** https://raw.githubusercontent.com/dgraph-io/badger/main/docs/quickstart.md — đọc trực tiếp; thư mục dữ liệu, transaction, GC/value log. Không biến nhận định tự động về code do công cụ tìm tài liệu trả về thành kết luận lỗi bảo mật/durability.
- **[S5] DuckDB Tuning Workloads:** https://duckdb.org/docs/2.0/guides/performance/how_to_tune_workloads.html — đọc trực tiếp sau khi đi theo redirect từ đường dẫn stable; nguồn cho mục tiêu truy vấn lớn/ít thường xuyên. Không cố định giới hạn multiprocess dựa trên tài liệu phiên bản cũ.
- **[S6] modernc SQLite README:** https://gitlab.com/cznic/sqlite/-/raw/master/README.md — đọc trực tiếp; xác nhận có driver SQLite pure Go/no CGO, không lấy benchmark quảng bá làm phép đo của dự án.
- **[S7] SQLite Appropriate Uses:** https://www.sqlite.org/whentouse.html và [WAL](https://www.sqlite.org/wal.html) — đã đối chiếu ở đợt nghiên cứu kiến trúc trong cùng cuộc trao đổi; một application server, nhiều reader/một writer, lưu trên filesystem local phù hợp hơn chia sẻ DB trực tiếp qua mạng.

Đã dùng Context7 để đối chiếu bbolt/Badger/DuckDB với tài liệu nguồn, sau đó đọc nguồn trực tiếp cho các kết luận chính. Không cài database, không sửa ứng dụng, không migrate, không benchmark CPU/RAM và không kiểm chứng backup/restore thực tế ở bước này.
