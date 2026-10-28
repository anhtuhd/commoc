# Kiến trúc bản đầu

Chọn Vinext/React/TypeScript với Cloudflare Worker, D1 và R2 theo nền tảng Sites để có backend lưu thực, ảnh upload và đăng nhập ChatGPT. Đây là điều chỉnh triển khai so với cấu trúc server/DB đề xuất trong kế hoạch, không kết nối VPS.

D1 lưu một bản JSON `shop_state` kèm revision. Mọi thao tác bán hàng đọc revision, tính lại điều kiện, rồi UPDATE có điều kiện revision; nếu có ghi cạnh tranh sẽ đọc và tính lại. Tồn, đơn và lượt voucher cùng một lần ghi nguyên tử; thử đồng thời đã xác nhận một lượt cuối không bị bán hai lần. Không lưu PII trong localStorage; giỏ chỉ chứa mã và lượng. Bản trả về sau đặt không chứa địa chỉ/điện thoại hay khóa chống trùng.

Cấu trúc này phù hợp bản đầu/cửa hàng nhỏ, không dành cho volume lớn: toàn trạng thái phải đọc và serialize khi ghi, DB có giới hạn kích thước mỗi giá trị, audit/events được cắt còn 2.000. Khi dữ liệu tăng, cần tách bảng orders/order_items/products/coupons và phân trang phía máy chủ, thiết lập chính sách lưu giữ/xóa PII và giám sát dung lượng. Không tuyên bố đã load-test thương mại quy mô lớn.

Quyền do backend kiểm tra, CSRF kiểm tra Origin, giới hạn gửi theo IP đã băm. Những lần tra cứu sai vẫn tính hạn mức. Hosted identity dựa trên header do Sites xác thực; mock cookie chỉ có trong dev middleware của starter, không dùng làm auth production. Bootstrap chỉ cho site riêng tư và đóng sau khi có chủ.

Không xử lý thẻ thanh toán. Chuyển khoản là đối soát thủ công. Các tích hợp thông báo/đo lường bên ngoài sẽ cần tài khoản và cấu hình thực khi vận hành.
