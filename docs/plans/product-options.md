# Kế hoạch: phân loại, topping và ghi chú theo sản phẩm

## Mục tiêu và phạm vi

Admin cấu hình lựa chọn ngay trong mục Sản phẩm. Khách chọn trước khi thêm vào giỏ; đơn hàng, quản trị và thông báo phản ánh đúng lựa chọn. Giai đoạn đầu giữ Next.js, Go, PostgreSQL và Zalo bridge hiện tại. Đây là kế hoạch, chưa bật tính năng.

Giả định để bắt đầu: phân loại là các nhóm chọn một (ví dụ kích cỡ); topping là nhóm chọn nhiều, có thể bắt buộc/tùy chọn và giới hạn số lựa chọn; ghi chú là văn bản tùy chọn theo từng món. Giá mỗi lựa chọn là khoản cộng/trừ so với giá gốc, nhập bằng số nguyên VND; tổng đơn giá không được âm. Tồn kho vẫn dùng chung theo sản phẩm trong giai đoạn đầu; quản lý tồn kho từng biến thể hoặc topping là mốc sau.

## Thực trạng và điểm cần thay đổi

- `Product` hiện chỉ có giá/tồn kho chung; form quản trị dùng `EditDialog`. Go kiểm tra sản phẩm bằng schema validation đã lưu trong `backend/internal/shop/validation.json` và xử lý thêm trong `validation.go`.
- `ShopProvider` lưu giỏ theo `{id, quantity}` và gộp theo `id`; `quote` trong Go cũng gộp theo ID. Cùng món với các lựa chọn khác nhau hiện không thể tách dòng.
- `quote` tự đọc giá và tồn kho ở server; `placeOrder` tạo fingerprint chống gửi trùng và lưu snapshot món trong order. SePay xác nhận theo tổng đơn; Zalo dùng outbox và nội dung tạo từ đơn.
- JSON-LD hiện phát giá sản phẩm cố định. Cần xác định giá đại diện chính xác, khớp giá hiển thị, thay vì phát giá tùy chọn giả.

## Thiết kế đề xuất

1. **Cấu hình sản phẩm:** thêm danh sách nhóm có ID ổn định, tên, loại chọn một/chọn nhiều, bắt buộc, số lượng chọn tối thiểu/tối đa, thứ tự và các lựa chọn có ID, tên, chênh lệch giá VND, trạng thái bật/tắt. Thêm `noteEnabled` và giới hạn độ dài ghi chú theo món (mặc định tắt). Admin xem trước giá và lỗi cấu hình trước khi lưu. ID đã dùng trong đơn không tái sử dụng; ngừng bán bằng tắt thay vì xóa lịch sử.
2. **Khách hàng:** sản phẩm không có options tiếp tục thêm nhanh như cũ. Nếu có lựa chọn bắt buộc, nút thêm nhanh mở trang hoặc hộp chọn; khách xem giá cập nhật, chọn hợp lệ và nhập ghi chú trước khi thêm. Mỗi dòng giỏ được nhận diện bởi product ID, tập option ID đã chuẩn hóa và ghi chú; chỉ gộp khi các giá trị này giống nhau. Hiển thị chi tiết trong giỏ, thanh toán, xác nhận và tra cứu.
3. **Giá và tồn kho:** frontend chỉ gửi product ID, option ID, số lượng và ghi chú; Go kiểm tra danh sách lựa chọn thuộc đúng sản phẩm, trạng thái, số lượng, ghi chú và tự tính giá. Báo lỗi rõ khi cấu hình/giá thay đổi giữa lúc thêm giỏ và đặt; khách xác nhận giá mới. Tồn kho sản phẩm được giữ/trả một lần trong transaction cho tổng số lượng ở tất cả dòng cùng sản phẩm. Coupon, phí giao và QR/SePay dùng tổng do server tính.
4. **Lịch sử:** lưu snapshot tên nhóm, tên lựa chọn, chênh lệch giá, đơn giá cuối và ghi chú trong từng dòng đơn; thay đổi sản phẩm sau này không sửa đơn cũ. Tách ghi chú món khỏi ghi chú giao hàng. Hiển thị trong chi tiết đơn quản trị, tra cứu và tin Zalo theo giới hạn an toàn; tránh đưa dữ liệu khách vào log.
5. **Schema/SEO:** giá JSON-LD phải khớp giá thực tế được hiển thị và có thể mua. Nếu giá thay đổi theo lựa chọn, thiết kế hiển thị “từ …” và đánh giá `AggregateOffer`/`Offer` phù hợp theo dữ liệu thật, sau đó kiểm tra Google Rich Results Test và Schema.org Validator. Không phát giá biến thể, tồn kho hoặc đánh giá không có thật.

## Thứ tự thực hiện và tiêu chí nghiệm thu

### Mốc 1 — dữ liệu và quản trị
- Mở rộng DTO, schema Go và schema TypeScript/backup; dữ liệu sản phẩm cũ thiếu options được hiểu là không có lựa chọn. Cập nhật nhập/xuất/restore và các test tương thích dữ liệu cũ.
- Thêm editor các nhóm/lựa chọn, thêm/xóa/sắp xếp/bật tắt, xem trước giá; kiểm tra tên, ID trùng, giới hạn lựa chọn và giá không hợp lệ ở cả UI và Go.
- Nghiệm thu: lưu, tải lại và xuất/nhập sản phẩm cũ/mới không mất dữ liệu; quyền quản trị hiện có được giữ.

### Mốc 2 — đơn hàng và thanh toán
- Cập nhật quote, fingerprint/idempotency, transaction giữ/trả tồn kho, snapshot order/order_items và UI admin/tra cứu; tương thích đơn cũ và request cũ chỉ với sản phẩm không có options bắt buộc.
- Nghiệm thu: giá client bị sửa không ảnh hưởng tổng; từ chối option sai/tắt, ghi chú quá dài, giá đổi và tồn kho thiếu; hai request đồng thời không bán vượt tồn; lặp request không tạo hai đơn; hủy/hết hạn trả đúng tồn. Kiểm tra coupon, phí giao, QR và SePay khớp tổng mới; webhook lặp không gửi trùng sự kiện thanh toán.


### Mốc 3 — chọn món và giỏ
- UI chọn phân loại/topping/ghi chú trên mobile và desktop; thêm nhanh sản phẩm không tùy chọn. Giữ định dạng mảng `localStorage`, bổ sung các trường tùy chọn theo hướng tương thích để giỏ cũ vẫn đọc được; kiểm tra rollback trước khi bật.
- Nghiệm thu: hai cấu hình của cùng món là hai dòng; cấu hình giống nhau gộp đúng; thay đổi số lượng, sửa/xóa và reload vẫn đúng; không thêm được khi thiếu lựa chọn bắt buộc.

### Mốc 4 — thông báo, SEO và triển khai
- Bổ sung snapshot lựa chọn/ghi chú vào thông báo Zalo và nội dung đơn; test với bridge giả lập, không gửi tin thật. Bổ sung JSON-LD có cơ sở dữ liệu thật và kiểm tra Google/Schema.org.
- Kiểm tra responsive 320/375/390px và desktop, bàn phím, nội dung nút và trạng thái lỗi; chạy Go race/integration, TypeScript, build, kiểm thử đầu-cuối COD/SePay giả lập và rollback. Chuẩn bị backup, triển khai có thể quay lại, không tự bật SePay/Zalo chưa cấu hình.

## Quyết định cần chốt trước khi mở rộng

- Có cần tồn kho riêng cho từng phân loại hoặc topping không? Giai đoạn đầu đề xuất dùng tồn kho sản phẩm chung.
- Topping có cần giới hạn số lượng của từng loại (ví dụ thêm 2 phần trân châu) hay chỉ chọn có/không? Giai đoạn đầu đề xuất chọn có/không.
- Giá biến thể có thể thấp hơn giá cơ bản không? Đề xuất cho phép chênh lệch âm nhưng đơn giá cuối phải không âm; nếu cửa hàng chỉ dùng giá cộng thêm có thể giới hạn về sau.

## Ghi chú triển khai và an toàn vận hành

- Cờ API `PRODUCT_OPTIONS_ENABLED` mặc định `0`; frontend cần build với `--build-arg PRODUCT_OPTIONS_ENABLED=1` để đặt `NEXT_PUBLIC_PRODUCT_OPTIONS_ENABLED=1`. Hai cờ phải được bật đồng thời sau khi kiểm thử; chưa bật trên production. Khi API tắt, sản phẩm đang có nhóm hoạt động hoặc ghi chú sẽ từ chối đặt để tránh bỏ sót lựa chọn.
- Các trường options được thêm vào JSON sản phẩm và snapshot dòng đơn, không yêu cầu xóa hoặc thay schema PostgreSQL hiện có. Giỏ cũ vẫn được đọc; dữ liệu mới bổ sung `optionIds` và `note` vào từng dòng.
- Go tính giá từ dữ liệu sản phẩm và so sánh `quoteRevision` khi đặt món có lựa chọn. Giá thay đổi yêu cầu khách xem lại và xác nhận; đơn đã tạo giữ snapshot, mã chuyển khoản và tổng cũ.
- Google yêu cầu URL cho phép chọn trực tiếp từng biến thể để đánh dấu `ProductGroup` và không khuyến nghị dùng `AggregateOffer` cho các biến thể. Khi giá các lựa chọn khác nhau và chưa có URL chọn biến thể, hiện không phát `Offer` giá biến thể; chỉ phát `Product` nếu có đánh giá thật. Không tự tạo dữ liệu biến thể hay giá. Nguồn: https://developers.google.com/search/docs/appearance/structured-data/product-variants và https://developers.google.com/search/docs/appearance/structured-data/product-snippet.
- Không rollback API về bản cũ sau khi đã bật options hoặc tạo đơn có options: bản cũ không hiểu cấu hình và có thể tính sai. Trước khi chuyển phiên bản phải có backup và thử phục hồi trên dữ liệu QA; khi sự cố, ưu tiên tắt cờ, tạm ngừng bán các món có options và sửa tiến, chỉ rollback phần tương thích sau khi xác nhận dữ liệu và đơn.
