# Vận hành cửa hàng

## Lần đầu mở quản trị

1. Mở `/quan-tri` trên URL website riêng tư, đăng nhập bằng ChatGPT.
2. Bấm **Thiết lập cửa hàng** để tài khoản hiện tại trở thành chủ. Thao tác chỉ thành công khi chưa có chủ và `OWNER_SETUP_ENABLED=true`.
3. Vào **Thương hiệu**: tải logo PNG/JPEG/WebP, đổi tên, màu, favicon/ảnh chia sẻ, banner, câu chuyện, SEO, hotline, địa chỉ, Zalo và chính sách.
4. **Lưu bản nháp** chưa làm đổi cửa hàng. **Xuất bản thương hiệu** áp dụng; tải lại trang cửa hàng để xem. **Khôi phục phiên bản trước** quay lại cấu hình trước đó.
5. Rà lại sản phẩm, ảnh bao bì, quy cách, tồn kho, giá, chiến dịch và mã MOCVI10 mẫu. Nội dung/ảnh là dữ liệu minh họa; không phải công bố nguồn gốc hay chứng nhận sản phẩm thật. Có thể tắt nhãn nội dung mẫu ở phần Thương hiệu.

Website hiện là owner-private: chỉ chủ Sites được xem. Chưa chia sẻ hay mở công khai. Trước khi đổi audience, phải hoàn tất bước thiết lập chủ; đặt biến hosted `OWNER_SETUP_ENABLED=false` và triển khai lại. Không dùng bootstrap lần đầu trên site công khai. Quyền xem site (Sites) và vai trò quản trị trong ứng dụng là hai lớp riêng.

## Đơn và thanh toán

- COD hoạt động ngay. Chuyển khoản xuất hiện khi đã điền đủ tên ngân hàng, tài khoản và chủ tài khoản. Không có cổng thanh toán hay tự động đối soát.
- Mã đơn và thông tin ngân hàng được chụp lại lúc đặt. Đổi thương hiệu sau đó không sửa đơn cũ.
- Luồng: mới → xác nhận → chuẩn bị → đang giao → hoàn tất. Hủy được trước khi đang giao. Hủy trả tồn và lượt voucher đúng một lần; không tự chuyển tiền hoàn trả.
- Nhân viên xử lý đơn/yêu cầu. Chủ mới được đổi trạng thái thanh toán. Biên tập viên chỉ sửa sản phẩm và nội dung; không xem đơn hay thông tin khách.
- Giá, tồn, voucher, phí được máy chủ tính lại. Mỗi yêu cầu đặt có khóa chống trùng. Khách ngoài Hà Nội được chuyển sang tư vấn; địa chỉ nhập tự do để tránh phụ thuộc danh sách đơn vị hành chính cũ.
- Quản trị tự cập nhật dữ liệu mỗi 30 giây, có số đơn chờ. Chưa tích hợp thông báo email/SMS/Zalo tự động; đơn vẫn lưu trong DB. Liên kết gọi/Zalo chỉ xuất hiện khi cấu hình thật.
- Tra cứu cần mã đơn + điện thoại. Form hỗ trợ sau mua cũng kiểm tra khớp hai thông tin này. Yêu cầu báo giá có sản phẩm và số lượng riêng.

## Nội dung và quảng bá

Chiến dịch có lịch theo giờ Việt Nam, ảnh desktop/mobile và đường dẫn nội bộ. Bộ sưu tập có thứ tự, mỗi sản phẩm thuộc nhiều bộ. Bài viết chưa xuất bản không xuất hiện công khai. Voucher không cộng dồn; có hạn, ngưỡng đơn, giảm tối đa và số lượt.

Sự kiện view_product/add_to_cart/begin_checkout/order_created và nguồn chiến dịch được ghi nội bộ. Chưa gắn Meta Pixel/GA4 hay tài khoản quảng cáo. Không tự gửi thông tin khách hàng sang dịch vụ quảng cáo.

## Sao lưu và khôi phục

Chủ vào **Nhân sự & dữ liệu → Tải bản sao lưu**: JSON đầy đủ có thông tin khách, đơn và phân quyền; giữ ở nơi riêng tư. File không chứa dữ liệu nhị phân ảnh R2. Khi chuyển hệ thống, cần sao lưu kho ảnh riêng.

**Khôi phục cấu hình** chỉ thay thương hiệu, danh mục/tồn, bộ sưu tập, chiến dịch, bài viết và cấu hình giao hàng từ JSON hợp lệ. Giữ đơn, yêu cầu, tài khoản, lượt voucher và lịch sử hiện tại; không dùng chức năng này để rollback giao dịch. Trước khi khôi phục, tải backup mới, ngừng nhận đơn và đối chiếu tồn kho của bản backup để tránh đưa số tồn cũ vào vận hành. UI có xác nhận rõ phạm vi. Khôi phục toàn bộ giao dịch cần bảo trì dữ liệu có kiểm soát, không có nút thay DB tùy ý.

Ảnh tải lên tối đa 5 MB, chỉ PNG/JPEG/WebP có chữ ký đúng. Logo SVG tạm đi kèm mã nguồn; upload SVG không được hỗ trợ.

## Kỹ thuật

- Source: thư mục `moc-vi`, manifest `.openai/hosting.json`; không đổi project_id hoặc tạo site mới khi cập nhật.
- Hosted `OWNER_SETUP_ENABLED=true` chỉ phục vụ bootstrap bản owner-private; `.dev.vars` không được đưa lên source.
- DB migration schema-only `drizzle/0000_right_fantastic_four.sql`; dữ liệu mẫu khởi tạo khi DB chưa có trạng thái shop. Local `.wrangler` không đóng gói nên đơn QA không lên production.
- Ví dụ apply local sau build:
  `node --import ./scripts/sites-env.mjs ./node_modules/wrangler/bin/wrangler.js d1 execute DB --local --config dist/server/wrangler.json --persist-to .wrangler/state --file drizzle/0000_right_fantastic_four.sql`
- Chạy unit/type/build sau sửa logic; kiểm thử tích hợp có tạo đơn local. Không đưa secret/backup khách lên Git.
- URL sitemap/canonical đang dùng domain Sites được cấp. Nếu đổi tên miền, cập nhật metadataBase, sitemap và robots theo domain mới.
