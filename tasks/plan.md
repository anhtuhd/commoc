# Go backend + PostgreSQL, giữ Next.js

Cập nhật 28/10/2026 theo chỉ đạo mới: chỉ chuyển backend, giữ Next.js lâu dài cho UI/SSR/SEO. Repo anhtuhd/commoc. Triển khai pishare.cloud thay vì demo.pishare.cloud theo yêu cầu tiếp theo. Không thực hiện chuyển giao diện sang Go.

1. Khung Go, PostgreSQL, contract tương thích UI.
2. Auth, quản trị, catalog, đặt hàng, tồn, coupon, delivery, review và inquiries.
3. SePay raw HMAC, ledger/paid/outbox nguyên tử; Zalo bridge Node riêng.
4. Next.js gọi Go public projection/session và proxy API; JSON-LD vẫn server-rendered trong Next.
5. Test PostgreSQL concurrency, replay và rollback; HTTP/browser mobile; schema theo skill schema.
6. Offline migration từ snapshot SQLite, giữ owner/secrets, media, payment/outbox và thứ tự dữ liệu.
7. Commit/push, build image, rehearsal, backup, chuyển routing sau kiểm tra nội bộ; kiểm tra production và validator.

Thiết kế triển khai mốc A dùng bảng riêng cho mỗi loại entity, JSONB DTO và cột generated có constraints cho thuộc tính quan trọng, order_items riêng; writer khóa theo cửa hàng, reader MVCC. Chưa triển khai toàn bộ mô hình quan hệ chi tiết hoặc khóa theo từng product như hướng tối ưu trong bản kế hoạch ban đầu; cần đánh giá tải trước bước tối ưu đó. Cách này vẫn bỏ SQLite khỏi runtime và chuyển toàn bộ nghiệp vụ ghi sang Go.

Tiêu chí đạt: không bán vượt tồn, không ghi nhận paid/event hai lần, import giữ dữ liệu và lịch sử sent, UI/SEO không đổi; không bật dịch vụ thật nếu cấu hình chưa sẵn sàng. Runbook: docs/migration/runbook.md.
