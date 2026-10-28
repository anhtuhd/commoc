# SePay và Zalo — vận hành

Bản VPS Node/SQLite. Người dùng chọn deploy trước, liên kết ngân hàng/SePay và Zalo sau. Chưa có giao dịch thật được kiểm thử.

## Bật nhận thanh toán

1. Liên kết tài khoản nhận tiền trong SePay, ưu tiên tài khoản riêng cho cửa hàng. Hiện chưa tự đối soát VA: subAccount khác rỗng chuyển vào hàng chờ.
2. Quản trị → Thương hiệu: nhập ngân hàng, số tài khoản, chủ tài khoản, BIN 6 số; lưu rồi xuất bản.
3. Bổ sung biến trong `.env.example` vào `/opt/commo/secrets/commo.env` hiện có, quyền 600, giữ nguyên biến đăng nhập. Secret HMAC ngẫu nhiên ít nhất 32 ký tự phải giống bên SePay. Nhập trực tiếp qua trình sửa file trên VPS, không gửi chat, không chụp màn hình, không commit hoặc đưa vào URL/client. Recreate web: `sudo docker compose -f /opt/commo/compose.yml up -d --no-deps --force-recreate web`.
4. Tạo webhook POST `https://pishare.cloud/api/webhooks/sepay`: tiền vào, lọc đúng tài khoản, xác thực HMAC-SHA256. Mã thanh toán: tiền tố MV, hậu tố tối thiểu/tối đa đều 12, chữ và số. Không bật “chỉ gửi khi có mã” nếu cần nhận chuyển khoản sai nội dung để xử lý.
5. Quản trị → Thanh toán SePay: lưu tài khoản, gateway đúng tên trong SePay (ví dụ Vietcombank), BIN. Bật tự xác nhận khi secret đã cấu hình và tài khoản/BIN khớp thương hiệu đã xuất bản. Tắt vẫn ghi nhận webhook hợp lệ để duyệt.
6. Nhân sự & dữ liệu → Zalo: lưu số admin, bật gửi, quét QR tài khoản gửi. Xem phiên kết nối và hàng đợi thanh toán. Worker không cần secret SePay.
7. Khi cấu hình xong, tạo một đơn thật có kiểm soát và người dùng tự thanh toán; đối chiếu ngân hàng → SePay → đơn → Zalo. Unit test không thay thế kiểm thử thực tế. Không giả giao dịch ngân hàng trên production.

## Quy tắc đối soát

- QR mới dùng MV + 12 ký tự; số đơn hiển thị vẫn MV-XXXXXXXX. Đơn cũ khớp nguyên token mã cũ, duy nhất.
- Chỉ tiền vào đúng ngân hàng/tài khoản/BIN, mã duy nhất, tổng tiền VND và đơn ngân hàng còn hạn, chưa trả, mới/xác nhận được ghi paid. Không tự đổi trạng thái giao hàng.
- Thiếu/thừa tiền, trả lần hai, sai mã, VA, COD, đơn hủy/hết hạn: chờ kiểm tra. Không tự cộng nhiều giao dịch, hoàn tiền hoặc hồi sinh tồn kho. Callback đến sau hạn và giao dịch trước tạo đơn đều cần đối soát.
- Owner có thể gắn giao dịch sai mã vào đơn bằng mã đơn/lý do; vẫn kiểm tra tài khoản, tiền, thời gian, chống dùng một giao dịch hai lần. Không đặt lại unpaid với đơn đã đối soát SePay.
- Thay ngân hàng không đổi snapshot đơn cũ. Khi đổi tài khoản nhận SePay, chờ đơn cũ kết thúc hoặc đối soát riêng.

## Độ bền và theo dõi

- HMAC trên raw body + timestamp (tối đa 300 giây), body tối đa 64 KiB. Chữ ký sai 401, dữ liệu sai 400, chưa cấu hình/DB lỗi 503. Thành công 200 với `{"success":true}` chỉ sau commit, không đợi gửi Zalo.
- Hai bảng payment_transactions/payment_outbox nằm trong shop.sqlite; tạo bảng/index lặp an toàn. Ledger, paid, outbox và revision commit nguyên tử. ID lặp không tác dụng mới; cùng ID nhưng khác dữ liệu hiện cảnh báo, không ghi đè.
- Log chỉ event ID, kết quả, độ trễ; không raw payload/secret. Owner xem lịch sử, tài khoản che một phần. Dữ liệu ngân hàng lưu riêng tư, không public API/JSON-LD.
- Zalo kiểm tra mỗi 15 giây, lease 120 giây, tối đa 6 lần thử, backoff 1–32 phút. Khi tắt/mất phiên giữ hàng đợi. Có thể gửi lặp nếu Zalo đã nhận nhưng worker chết trước ghi kết quả; mã giao dịch giúp đối chiếu. Không cam kết tức thời vì còn phụ thuộc ngân hàng/SePay/Zalo.
- Trang thành công và tra cứu cập nhật khoảng 60 giây. Tra cứu dừng khi kết thúc, bỏ lượt khi tab ẩn. Mạng chậm không phải lý do để chuyển thêm tiền.
- Backup VPS sao lưu toàn SQLite gồm ledger/outbox. Export JSON quản trị chỉ chứa state, không phải backup ledger. Chưa có đối soát định kỳ qua API SePay; hiện đối chiếu nhật ký và replay webhook khi bị mất quá thời gian retry.

## Kiểm thử và rollback

Chạy `node --test tests/*.test.ts`, `npx tsc --noEmit`. HTTP: `SEPAY_QA=1 node tests/sepay-api.mjs`, chỉ cho phép localhost. Cần server Next VPS runtime port 3100 với DB thử riêng, SEPAY_MODE=test, secret thử `local-qa-sepay-secret-only-not-production`, owner `qa-owner`/mật khẩu thử `qa-local-only`, ngân hàng giả 123456789/Vietcombank/BIN970436 và tự đối soát bật. Không dùng thông tin giả trong production. Script tạo đơn và kiểm tra chữ ký/replay/CSRF/quyền/đối soát/COD.

Test Mode của SePay cần server thử HTTPS, DB/secret riêng. SEPAY_MODE=test bị từ chối nếu domain cấu hình là pishare.cloud. Chưa liên kết SePay thì webhook được ký giả lập ở localhost, không coi là Test Mode do SePay gửi. Zalo sender giả lập kiểm tra retry/lỗi; tin thật cần phiên Zalo thực.

Trước deploy: backup bằng deploy/backup.py, lưu compose/image cũ, build cả target web và zalo-worker. Rollback giữ DB/media, có thể đặt SEPAY_AUTO_MATCH=false rồi recreate web hoặc tắt gửi Zalo trong quản trị. Quay về image trước tích hợp sẽ không còn endpoint; SePay retry, cần replay sau phục hồi. Không trả 200 để bỏ sự kiện, không restore đè DB có giao dịch mới, không xóa ledger.

Nguồn: [payload/ACK](https://developer.sepay.vn/vi/sepay-webhooks/tich-hop-webhook), [HMAC](https://developer.sepay.vn/vi/sepay-webhooks/xac-thuc), [mã thanh toán](https://developer.sepay.vn/vi/sepay-webhooks/cau-hinh-ma-thanh-toan), [Test Mode](https://docs.sepay.vn/test-mode.html).
