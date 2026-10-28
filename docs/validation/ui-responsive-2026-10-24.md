# UI responsive — 24/10/2026

## Phát hiện và sửa

- Nhiều đoạn mô tả/nhãn 8–12px trên mobile: tăng nội dung chính 14–15px, nhãn 14px, input/select/textarea 16px; tiêu đề co theo viewport.
- Thẻ bài viết chia ảnh/chữ 40/60 tại 320–375px: chuyển một cột, ảnh có chiều cao giới hạn; nội dung đọc được.
- Vùng bấm thêm giỏ, tăng/giảm số lượng, đóng dialog và menu quá nhỏ: mở rộng vùng tương tác.
- Giỏ hàng có ảnh, tên, giá, số lượng, nút xóa trên một hàng hẹp: chuyển grid; tổng tiền và dòng tóm tắt được xuống dòng có chủ đích.
- Thanh toán còn hai cột ở tablet: chuyển một cột đến 900px, bỏ sticky tóm tắt.
- Quy tắc ảnh campaign nhắm vào img trực tiếp trong khi DOM dùng picture: đặt kích thước picture/img ở mobile, giữ ảnh phủ khung.
- Grid/flex child, chuỗi dài, tài khoản ngân hàng và ảnh QR: giới hạn kích thước và cho phép xuống dòng, ảnh QR giữ tỷ lệ.
- Quản trị mobile ẩn thông tin tài khoản/đăng xuất: hiện lại và cho phép xuống dòng. Cỡ chữ các tab/nút/form/stats được tăng; form hành động xếp dọc.
- Bảng sản phẩm cuộn trong region riêng có nhãn và focus bằng bàn phím. Dialog chỉnh sửa vừa màn hình 320px và cuộn dọc.

## Kiểm tra

Trình duyệt Chrome riêng, dữ liệu QA local, không tạo đơn hay đánh giá trên production.

- Rà soát trước sửa 16 tuyến production ở 320px. Không thấy tràn ngang với dữ liệu mặc định; các vấn đề chính là chữ quá nhỏ, vùng bấm, bố cục bài viết và trạng thái có dữ liệu.
- Sau sửa kiểm tra mobile: trang chủ, danh sách sản phẩm, chi tiết sản phẩm thường/báo giá, bộ sưu tập, câu chuyện, danh sách và 2 bài viết chi tiết, liên hệ, giỏ hàng, thanh toán, tra cứu, 3 chính sách, trang thành công, đăng nhập.
- Kiểm tra nội dung giỏ hàng và form thanh toán với sản phẩm QA; các input hiển thị 16px. Không thực hiện đặt đơn.
- Kiểm tra quản trị 320px: tổng quan, thương hiệu, sản phẩm, đơn hàng, đánh giá, tư vấn, bộ sưu tập, chiến dịch, bài viết, mã ưu đãi, giao hàng, nhân sự/dữ liệu; không tràn toàn trang. Mở dialog chỉnh sửa sản phẩm: rộng 296px trong màn hình 320px, không tràn trong dialog.
- Kiểm tra đại diện trang chủ/sản phẩm/quản trị ở 768, 1024, 1440px: không tràn ngang. 375px kiểm tra ảnh chụp bài viết, giỏ hàng có dữ liệu và thanh toán.
- 33/33 kiểm tra logic đạt; TypeScript đạt; production build đạt. Không đổi API, dữ liệu hay JSON-LD.

Các bảng và thanh tab quản trị được phép cuộn ngang trong khung riêng; không dùng overflow-x:hidden để che lỗi toàn trang. Chưa kiểm tra trên thiết bị iOS/Android vật lý.
