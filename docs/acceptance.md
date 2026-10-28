# Kiểm thử bản đầu — 23/10/2026

Đã kiểm tra trên dữ liệu local, không đặt đơn lên website Metri và không sử dụng tài khoản/ngân hàng thật.

- 8 unit tests pass: giá máy chủ; giỏ gộp SKU; vùng giao; sản phẩm báo giá; hạn/ngưỡng voucher; khóa chống trùng; hoàn tồn một lần; ngân hàng cần cấu hình; snapshot thương hiệu; ranh giới vai trò (một số assertion cùng một test).
- API integration pass: quản trị ẩn với khách; nháp/xuất bản/quay lại thương hiệu; hai request tranh tồn cuối; idempotency; giá không theo client; hủy sai luồng; tra cứu không lộ khách; giới hạn cả tra cứu thất bại; CSRF; file giả PNG bị chặn; hai request tranh lượt voucher cuối; khôi phục cấu hình giữ đơn.
- Upload WebP thực vào R2 local và đọc lại đúng số byte/content-type; trang sản phẩm không tồn tại trả HTTP 404.
- Chrome: mua ngay → checkout → gửi COD → trang mã đơn → đơn xuất hiện trong quản trị. Chỉ đơn local, không giao hàng. Admin hiển thị các trường thương hiệu, quyền chủ và đơn chờ.
- Trang chủ đo ở 360/390/768/1440px không tràn ngang, không ảnh hỏng. Ảnh chụp kiểm tra desktop, checkout mobile390 và admin mobile360. Chưa kiểm tra trên thiết bị iOS/Android vật lý hoặc mọi trình duyệt.
- TypeScript và git diff --check pass; production build pass với các thư viện đã cập nhật. npm audit sau vá báo 0 vulnerabilities tại thời điểm kiểm tra; đây không phải kiểm toán an ninh toàn diện.
- 5 ảnh WebP tổng dưới 1 MB; ảnh PNG gốc giữ ngoài thư mục deploy tại `../assets-originals`.

Giới hạn: tích hợp email/SMS/Zalo/GA4/Pixel, online payment gateway, danh sách địa giới/API vận chuyển chưa cấu hình. Kênh nhận đơn hiện là bảng quản trị tự cập nhật; chuyển khoản thủ công. Kiểm tra ngân hàng có cấu hình qua unit test, chưa giao dịch ngân hàng thật. Backup UI chỉ khôi phục cấu hình/danh mục, không rollback giao dịch hay ảnh R2.
