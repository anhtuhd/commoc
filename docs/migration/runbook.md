# Vận hành Go + PostgreSQL

Đích cuối theo yêu cầu 28/10/2026: **pishare.cloud**, frontend Next.js giữ nguyên. Không triển khai demo.pishare.cloud trong đợt này.

## Release và secret

- Dự án mới `/opt/commoc`; compose riêng, cổng web 3401 và API 3402 chỉ trên 127.0.0.1; DB/bridge không publish port.
- Image web/api/zalo ghim theo commit. `/opt/commoc/secrets/{api,database,bridge}.env` mode 600.
- Giữ ADMIN_USER, ADMIN_PASSWORD_HASH và SESSION_SECRET cũ trên VPS; không đổi mật khẩu/phiên đăng nhập. Tài khoản phải có role owner trong snapshot; import từ chối nếu không khớp.
- Chỉ sao chép các biến SePay đang có, giữ live/test và trạng thái enabled trong dữ liệu. Thiếu secret thì webhook trả 503; không tự bật thanh toán.
- Zalo session cũ nếu có được copy trên VPS sang volume bridge, không đưa vào Git; dừng worker cũ trước chạy bridge mới.
- Caddy ghi đè X-Commo-Client-IP bằng IP do Cloudflare cung cấp; UFW origin đã giới hạn 80/443 cho dải Cloudflare. Nếu mở origin cho nguồn khác phải cập nhật logic trusted proxy, không tin header client tự gửi. Go chỉ tin header này từ proxy private/loopback.

## Rehearsal và cutover

1. Sao lưu SQLite bằng sqlite3 backup API (bao gồm WAL nhất quán), media, Zalo và env; quyền backup hạn chế. Xuất JSON bằng script từ snapshot. Đọc/so sánh chỉ số lượng/hash, không in PII/secret.
2. Build mọi image trước khi gián đoạn; chạy database, import bản thử, kiểm tra restore và đối chiếu toàn bộ state, ledger, outbox, metrics. API/bridge chưa chạy với cấu hình thật.
3. Trước chuyển chính thức, lưu Caddyfile và trạng thái timer. Bật bảo trì 503 trên riêng pishare.cloud; dừng web/worker SQLite và timer maintenance. Webhook nhận 503 để retry; không trả 200 rồi bỏ giao dịch.
4. Tạo snapshot cuối, export và import offline vào DB mới. Copy media/session. Đối chiếu số lượng và toàn bộ dữ liệu chuẩn hóa, owner, trạng thái thanh toán/notification. Nếu có lỗi thì hủy cutover, phục hồi Caddy và chạy lại stack cũ.
5. Khởi động Go, Next.js và bridge; smoke test nội bộ. Đổi Caddy proxy sang 3401, validate rồi reload. Mở website; kiểm tra public/admin session/SEO/health, log, webhook cấu hình.
6. Sau khi mở ghi PostgreSQL: không quay lại SQLite cũ. Rollback image tương thích PostgreSQL hoặc sửa tiến. Nếu bắt buộc quay SQLite, phải có reverse migration được kiểm chứng; chưa cung cấp đường này trong mốc A.
7. Giữ backup trước chuyển và stack cũ dừng. Timer backup mới dùng pg_dump và tar media; thử restore PostgreSQL vào database kiểm thử trên VPS. Backup hiện theo ngày: chưa cấu hình PITR, không hứa RPO=0.

## Theo dõi và sao lưu

`docker compose --env-file /opt/commoc/release.env -f /opt/commoc/compose.yml ps`

`curl -f http://127.0.0.1:3402/healthz`

Log JSON có request ID, route, method, status, duration; không log raw webhook/password/cookie. PostgreSQL pool 8 connection, timeout SQL 15 giây. Worker chạy mỗi 15 giây, outbox lease 120 giây và retry tối đa 6 lần; frontend chờ thanh toán poll theo cơ chế hiện tại.

Chạy `/opt/commoc/source/deploy/backup-postgres.sh` bằng root. Bản dump, media và SHA256 nằm trong `/opt/commoc/backups`; quyền 700/600. Thư mục secret cần bản sao bảo vệ riêng khi thay đổi cấu hình. Kiểm tra dung lượng/retention định kỳ; không tự xóa backup nguồn migration.

## Giới hạn có chủ đích

- Mốc A giữ entity JSONB và khóa ghi toàn cửa hàng; phù hợp quy mô hiện tại, chưa tối ưu tải lớn. Không thay UI để chuẩn hóa dữ liệu cùng lúc.
- Dịch vụ zca-js là tích hợp Zalo cá nhân. Không tự đăng nhập hay gửi tin thật trong kiểm thử; chỉ bật theo cấu hình quản trị đã được chủ cửa hàng nhập.
- API định kỳ đối soát SePay ngoài webhook vẫn là hạng mục tiếp theo, như phiên bản cũ.
