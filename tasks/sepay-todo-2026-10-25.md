# Checklist tích hợp SePay — kế hoạch, chưa triển khai

- [x] Đọc tài liệu SePay chính thức bằng Firecrawl.
- [x] Kiểm tra QR, trạng thái đơn, SQLite và worker Zalo hiện có.
- [x] Ghi kiến trúc, điều kiện đối soát và ngoại lệ trong `tasks/plan.md`.
- [x] 1. Route HMAC + ledger bền vững, test replay/concurrency.
- [x] 2. Mã thanh toán riêng trong QR, tương thích đơn cũ.
- [x] Checkpoint A: schema DB, chữ ký và mã SePay đạt kiểm thử.
- [x] 3. Ghi paid + outbox nguyên tử, không tự đổi trạng thái giao hàng.
- [x] 4. Worker Zalo gửi sự kiện thanh toán, retry độc lập.
- [x] Checkpoint B: webhook ký giả lập localhost → paid → sender Zalo giả lập; replay không tạo event thứ hai. SePay Test Mode và Zalo thật chờ cấu hình.
- [x] 5. Quản trị giao dịch, duyệt đối soát có audit và phân quyền.
- [x] 6. Trạng thái thanh toán trên mobile/desktop, hồi quy JSON-LD.
- [x] 7. Cấu hình Test/Live, migration, backup, monitoring và rollback.
- [x] Checkpoint C: full tests/typecheck/build và nghiệm thu ngoại lệ bằng dữ liệu QA.
- [x] Commit/push và deploy VPS ea61573, smoke test production chỉ đọc; chưa kích hoạt thanh toán thật.
- [ ] Giai đoạn sau: khảo sát API và triển khai đối soát định kỳ; endpoint polling nhanh có token riêng.

Chi tiết tiêu chí nghiệm thu, file dự kiến và phụ thuộc: [plan.md](plan.md).

Triển khai 25/10/2026: 52 unit tests đạt; HTTP API và mobile 360px đã kiểm tra. Người dùng xác nhận chưa liên kết SePay, deploy trước để cấu hình sau. Live/Test Mode từ nhà cung cấp và tin Zalo thực chưa được kiểm thử.

Báo cáo triển khai: [SePay 25/10/2026](../docs/validation/sepay-2026-10-25.md).
