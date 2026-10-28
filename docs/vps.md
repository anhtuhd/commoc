# Triển khai VPS pishare.cloud

Bản VPS dùng Next.js standalone/Node 24, SQLite WAL và thư mục media bền vững. Bản Sites tiếp tục dùng Vinext, D1/R2; webpack chỉ thay module Cloudflare bằng adapter Node khi COMMO_RUNTIME=vps. NEXT_PUBLIC_SITE_URL phải được đặt tại lúc build để metadata, sitemap và JSON-LD cùng domain.

## Vận hành

- SSH: `ssh -F /Users/tubean/.codex/vps/managed-vps/config managed-vps`
- Dịch vụ: `/opt/commo/compose.yml`, bind `127.0.0.1:3301`, container chỉ truy cập qua Caddy/Cloudflare.
- Dữ liệu: `/opt/commo/data/shop.sqlite`, `/opt/commo/data/media` (không nằm trong image).
- Bí mật: `/opt/commo/secrets/commo.env`, quyền 600. Không commit, in log hoặc đưa vào image.
- Đăng nhập VPS: `/dang-nhap`, tài khoản chủ riêng; tài khoản ChatGPT/Sites không dùng trên VPS. Hỗ trợ một tài khoản đăng nhập chủ ở bản này; chức năng cấp quyền Sites không cấp được mật khẩu VPS.
- Hash mật khẩu scrypt; cookie Secure, HttpOnly, SameSite=Strict; phiên ký HMAC hết hạn sau 8 giờ, kiểm tra origin và giới hạn lần đăng nhập. Muốn thu hồi tất cả phiên: thay SESSION_SECRET rồi recreate container.
- Log: `sudo docker compose -f /opt/commo/compose.yml logs --tail 100 web`
- Tình trạng: `sudo docker compose -f /opt/commo/compose.yml ps`

## Cập nhật

Kiểm tra `node --test tests/*.test.ts`, `npx tsc --noEmit`, rồi commit/push. Chuyển đúng source commit (không có .env, node_modules, DB, backup) lên `/opt/commo/source`, build `sudo docker build -f /opt/commo/source/deploy/Dockerfile -t commo:<commit> /opt/commo/source`. Sao lưu SQLite bằng sqlite backup API và media trước khi đổi schema. Đổi image trong compose, chạy `sudo docker compose -f /opt/commo/compose.yml up -d`; kiểm tra trang chủ, `/api/store/public`, admin chưa đăng nhập 401, đăng nhập, quote, upload và restart persistence. Không copy DB đang chạy bằng cp.

## Quay lại

Giữ image trước khi cập nhật. Chuyển image về tag cũ rồi `docker compose up -d`; giữ volume dữ liệu. Cấu hình Caddy trước lần triển khai đầu ở `/opt/commo/backups/Caddyfile.before-commo`; khi khôi phục chỉ bỏ khối pishare.cloud nếu cấu hình chung có thay đổi về sau, validate trước reload. Không sửa khối pishare.site/dev.pishare.site và không đổi SSH/firewall.

DNS A gốc trỏ VPS qua proxy Cloudflare; Caddy tự quản chứng chỉ ACME bằng HTTP challenge. Cloudflare dùng Full (strict). Bản sao lưu Sites ban đầu được lưu riêng, nhập dữ liệu thực, đổi định danh chủ sang tài khoản VPS và làm mới bộ đếm giới hạn yêu cầu; không nhập dữ liệu QA local.

## Vận hành đơn, lịch giao và Zalo

- Giao hàng: cấu hình thời hạn chuyển khoản (mặc định 60 phút), thời gian chuẩn bị, số ngày mở lịch và sức chứa từng khung. Lịch mặc định tắt cho đến khi có cấu hình thật. Giờ Việt Nam; số chỗ kiểm tra trong cùng giao dịch tạo đơn. Đơn hủy trả lại chỗ.
- Chỉ đơn chuyển khoản mới, chưa trả tiền, trạng thái mới/xác nhận mới tự hủy khi hết hạn. COD và đơn cũ chưa có deadline không bị ảnh hưởng. Không chuyển đơn ngân hàng sang chuẩn bị khi chưa trả tiền. Nếu khách chuyển sát hạn hoặc sau khi hủy, chủ phải đối soát/hoàn tiền riêng, không hồi sinh đơn đã trả tồn kho.
- QR cần BIN ngân hàng 6 số, tài khoản và chủ tài khoản thực trong thương hiệu đã xuất bản. Dùng VietQR Quick Link https://www.vietqr.io/qr-dong ; số tiền và tài khoản lấy từ snapshot đơn. QR không tự xác nhận thanh toán. Kiểm tra tên người nhận trong ứng dụng ngân hàng trước khi dùng thực tế.
- Quản trị > Nhân sự & dữ liệu > Zalo: nhập số người nhận, bật gửi, lưu; tạo QR và quét bằng tài khoản gửi. zca-js 2.2.0 là API Zalo cá nhân không chính thức; upstream cảnh báo có thể khóa tài khoản. Không dùng listener đọc tin nhắn, không tự nhắn khách hàng.
- Dịch vụ `zalo` riêng, phiên ở `/opt/commo/data/zalo/session.json` quyền 600, không nằm trong Git, image hoặc bản backup DB/media. Sau khôi phục máy khác cần quét lại. Owner mới xem QR/cấu hình; staff/editor không có quyền.
- Gửi đơn mới từ thời điểm bật lần đầu, tối đa 6 lần thử với khoảng chờ tăng dần. Xem lỗi/thử lại trong quản trị. Có thể có tin trùng nếu tiến trình dừng sau khi Zalo nhận nhưng trước khi lưu kết quả; mã đơn dùng để đối chiếu. Cần số nhận có thể tìm thấy trên Zalo. Chưa quét QR thì chưa thể gửi.
- Thông báo trình duyệt chỉ hoạt động khi quản trị đang mở và người dùng đã cho phép; không thay thế Zalo chạy nền.
- Build web: `docker build --target web -f deploy/Dockerfile -t commo:<commit> .`; build bot: `docker build --target zalo-worker -f deploy/Dockerfile -t commo-zalo:<commit> .`.
- Cài các unit `deploy/commo-*.service` và `.timer` vào `/etc/systemd/system`, daemon-reload rồi enable --now hai timer. Maintenance chạy mỗi phút; backup 02:30 giờ Việt Nam, giữ 14 bản, kiểm tra integrity và phục hồi DB/media vào thư mục tạm trước khi giữ bản.
- Bản sao ngoài VPS: `python3 scripts/pull-backup.py`, lưu riêng `backups/vps/` đã gitignore. LaunchAgent trên máy chủ tài khoản macOS chạy hằng ngày và lúc đăng nhập; máy cần bật và có mạng. Mỗi bản tải được thử phục hồi SQLite. Không coi đây là sao lưu độc lập luôn hoạt động khi Mac tắt; có thể thay bằng kho riêng luôn trực tuyến sau này.
- Rollback: tắt timer maintenance, dừng container zalo nếu có vấn đề; đổi image web về `commo:6c97cc4`, giữ dữ liệu. Các trường mới là bổ sung, bản cũ bỏ qua. Không restore đè DB có đơn mới để rollback code.

## Thanh toán SePay

Xem [SePay và Zalo](sepay.md). Hai bảng payment_transactions/payment_outbox nằm trong DB hiện có và được backup cùng SQLite. Build/recreate cả web và zalo-worker khi cập nhật; không chỉ đổi image web. Chưa có secret thì webhook trả 503 và không cập nhật đơn. Biến SEPAY_AUTO_MATCH=false là công tắc dừng tự đối soát, còn dữ liệu giao dịch vẫn được giữ.
