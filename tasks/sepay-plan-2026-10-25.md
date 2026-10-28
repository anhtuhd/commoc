# Kế hoạch SePay webhook → cập nhật thanh toán → Zalo admin

Ngày khảo sát: 25/10/2026. Trạng thái: đang triển khai theo yêu cầu; người dùng chọn deploy trước và cấu hình dịch vụ thật sau. Tiến độ tại tasks/todo.md.

## Mục tiêu và hiện trạng

Giữ QR VietQR và worker `zca-js` hiện có. SePay báo giao dịch ngân hàng vào endpoint riêng; hệ thống đối soát, ghi nhận đã thanh toán và xếp thông báo Zalo cho số admin đã cấu hình. Không tự thay đổi trạng thái xử lý/giao hàng khi ghi nhận thanh toán.

Các điểm đã kiểm tra trong mã nguồn:
- `lib/shop/payment.ts`: QR dùng tổng tiền và `order.number`; số đơn hiện có dạng `MV-XXXXXXXX`.
- `lib/shop/commerce.ts`: đơn ngân hàng có hạn thanh toán; hủy đơn trả lại tồn kho. Thông tin tài khoản ngân hàng được chụp vào từng đơn.
- `lib/shop/repository.ts`: state JSON trong SQLite, cập nhật bằng revision CAS; `updateState` chạy hết hạn đơn trước callback.
- `lib/vps/runtime.ts`: adapter SQLite VPS hiện chỉ cung cấp prepare, chưa có giao dịch nhiều câu lệnh cho ledger + state.
- `scripts/zalo-worker.ts`: dùng cùng SQLite, kiểm tra mỗi 15 giây, gửi thông báo đơn mới. `notificationLog` chống trùng theo order ID nên chưa hỗ trợ thông báo thanh toán riêng.
- API store có kiểm tra Origin và giới hạn tra cứu 15 lượt/10 phút. Không dùng route này làm webhook hoặc tăng polling mà bỏ qua giới hạn.

## Kiến trúc đề xuất

Ngân hàng → SePay → `POST /api/webhooks/sepay` → xác thực HMAC → giao dịch SQLite (ledger + đơn + outbox) → HTTP 200 → worker Zalo gửi admin. Trang khách đọc trạng thái thanh toán từ server.

### Xác thực và nhận sự kiện

- HMAC-SHA256 với `X-SePay-Timestamp`, `X-SePay-Signature`; ký trên `timestamp + "." + raw_body`, so sánh constant-time, sai lệch tối đa 300 giây. Kiểm tra định dạng, giới hạn body và schema; không log secret/header hoặc toàn bộ dữ liệu ngân hàng.
- Secret chỉ ở env máy chủ, quản trị chỉ thấy trạng thái đã cấu hình. Route độc lập không yêu cầu cookie/Origin và không nới CSRF của API store.
- Khi đã ghi bền vững: trả HTTP 200 với `{"success":true}`. Không đợi Zalo. Lỗi DB trả 5xx để SePay thử lại; giao dịch hợp lệ nhưng không khớp đơn lưu hàng chờ rồi ACK.
- Test và Live dùng secret, endpoint/cấu hình, DB độc lập. Không cho payload test ghi vào dữ liệu production. Chỉ điều chỉnh Cloudflare cho đúng route nếu challenge chặn webhook.

### Mã thanh toán và đối soát

- Thêm `paymentCode` bất biến, ví dụ `MV` + 12 ký tự chữ/số ngẫu nhiên, kiểm tra duy nhất; SePay cấu hình prefix MV, hậu tố đúng 12, loại chữ/số. Giữ số đơn hiển thị cũ; QR dùng paymentCode mới.
- Đơn cũ còn hạn dùng bộ khớp tương thích có ranh giới mã rõ ràng; không dùng contains hoặc bỏ mọi dấu câu tùy tiện. Có nhiều mã/mâu thuẫn giữa code và nội dung thì chuyển đối soát.
- Chỉ tự trả tiền thành công khi giao dịch tiền vào, đúng ngân hàng/tài khoản nhận đã lưu trong đơn và cấu hình kết nối được cho phép, đúng mã duy nhất, đúng tổng tiền VND nguyên, đơn chuyển khoản chưa trả và còn hiệu lực.
- Dùng `transferAmount`, không dùng `accumulated`. Parse transactionDate theo UTC+07:00; không chấp nhận giao dịch trước thời điểm tạo đơn.
- Thiếu/thừa tiền, chuyển lần hai với ID giao dịch khác, sai mã, COD, đơn đã hủy/hết hạn: `needs_review`, không tự cộng nhiều giao dịch, không tự hoàn tiền hoặc mở lại đơn. Callback đến sau hạn dù thời gian giao dịch trước hạn vẫn cần đối soát để tránh tranh chấp tồn kho.
- Không bật bộ lọc “chỉ gửi khi có mã” ở SePay nếu muốn nhìn thấy chuyển khoản sai nội dung. Ưu tiên tài khoản nhận tiền riêng cho cửa hàng; tài khoản dùng chung cần giới hạn lưu trữ và thông báo để tránh lộ giao dịch không liên quan.

### Tính nhất quán và thông báo

- Bảng `payment_transactions`: khóa UNIQUE theo provider + môi trường/kết nối + transaction ID; giữ số tiền, tài khoản được bảo vệ, mã, thời gian, trạng thái đối soát, order ID và fingerprint. Cùng ID nhận lại không tạo tác dụng mới; payload khác với ID cũ phải cảnh báo, không ghi đè.
- Bảng `notification_outbox`: event key duy nhất, loại `payment.received`/`payment.needs_review`, order/transaction ID, trạng thái, số lần thử, lần thử tiếp, lease. Giữ đường gửi `order.created` hiện có, không phát lại lịch sử.
- Ledger, paymentStatus/paidAt/paymentSource và outbox phải commit cùng một transaction `BEGIN IMMEDIATE`, tăng revision state để các writer CAS thử lại. Không gọi mạng trong transaction; không gọi lồng `updateState` vốn chạy hết hạn trước callback.
- Kiểm thử cạnh tranh webhook, maintenance, admin và Zalo. Chỉ cập nhật `paymentStatus=paid`; `status` xử lý đơn giữ nguyên.
- Worker đọc outbox, retry có backoff và hiển thị lỗi cuối/cho thử lại. Zalo mất phiên không làm mất thanh toán. Không hứa gửi đúng một lần tuyệt đối: gửi thành công rồi crash trước khi ghi DB vẫn có thể gây gửi lặp.
- Nội dung Zalo: loại sự kiện, mã đơn, tiền nhận/tổng đơn, thời gian, mã giao dịch và link quản trị; hạn chế dữ liệu khách. Gửi đến số admin đã cấu hình, không tự gửi khách.
- MVP giữ cơ chế cập nhật trang hiện có tối đa khoảng 60 giây; hiển thị rõ đang chờ xác nhận. Nếu cần 5–10 giây, bổ sung endpoint trạng thái hẹp với token ngẫu nhiên, giới hạn riêng, pause khi tab ẩn/backoff; không tái sử dụng giới hạn tra cứu hiện tại.

## Các bước triển khai

Mỗi bước chạy test tập trung; checkpoint chạy `node --test tests/*.test.ts`, `npx tsc --noEmit` và build VPS bằng Dockerfile hiện có. Test tích hợp dùng DB riêng, không tạo đơn/thông báo thử trong production.

### 1. Tiếp nhận webhook có xác thực và lưu ledger — Medium
Phụ thuộc: không. File dự kiến: `lib/vps/runtime.ts`, `lib/payments/sepay.ts` (mới), `lib/payments/repository.ts` (mới), `app/api/webhooks/sepay/route.ts` (mới), `tests/sepay.test.ts` (mới).
- Chữ ký sai/cũ, body bị sửa, JSON sai bị từ chối; secret không xuất ra client/log.
- Sự kiện hợp lệ được lưu duy nhất và ACK đúng; DB lỗi không ACK thành công.
- Mặc định chỉ ghi nhận, chưa tự sửa đơn; migration chạy lại an toàn.
Xác minh: test HMAC trên raw bytes, replay/concurrency, lỗi DB và kiểm tra SePay Test Mode.

### 2. Sinh mã thanh toán tương thích QR — Medium
Phụ thuộc: 1. File: `lib/shop/types.ts`, `lib/shop/commerce.ts`, `lib/shop/payment.ts`, `lib/payments/sepay.ts`, `tests/sepay.test.ts`.
- Mã mới ổn định qua yêu cầu tạo đơn lặp, duy nhất, đúng cấu trúc SePay.
- QR dùng tổng tiền server và tài khoản snapshot; thay ngân hàng trong quản trị không sửa QR đơn cũ.
- Mã cũ còn hạn khớp có kiểm soát; mã nhập nhằng không tự đối soát.
Xác minh: quét QR test, kiểm tra trích xuất code ở SePay Test Mode; unit tests mã cũ/mới/collision.

Checkpoint A: kiểm tra migration, chữ ký và mã thanh toán trước khi bật thay đổi paymentStatus.

### 3. Tự xác nhận thanh toán nguyên tử — Medium
Phụ thuộc: 1–2. File: `lib/payments/repository.ts`, `lib/payments/sepay.ts`, `app/api/webhooks/sepay/route.ts`, `lib/shop/types.ts`, `tests/sepay.test.ts`.
- Đúng điều kiện: ghi paid và một outbox event; status giao hàng không đổi.
- Thiếu/thừa/sai tài khoản/hết hạn/trùng ID/mã nhập nhằng đều đúng quy tắc; không thay tồn kho hai lần.
- Crash/rollback không để ledger đã ghi mà đơn/outbox chưa ghi; writer khác không mất cập nhật.
Xác minh: test ma trận và nhiều kết nối SQLite cạnh tranh, đồng thời chạy maintenance hết hạn.

### 4. Zalo báo kết quả thanh toán — Medium
Phụ thuộc: 3. File: `scripts/zalo-worker.ts`, `lib/shop/notifications.ts`, `lib/payments/outbox.ts` (mới), `tests/notifications.test.ts`, `deploy/zalo/package.json` (nếu cần).
- Thông báo đơn mới và thanh toán tách biệt, không phát lại đơn cũ.
- Mất phiên/API lỗi vẫn giữ event, retry/lease an toàn và không cản webhook.
- Nội dung đúng admin, đúng tiền/trạng thái; giữ mã event để nhận biết khi retry gây gửi lặp.
Xác minh: giả lập send thành công/lỗi/crash; kiểm tra thật trên người nhận test được cấu hình riêng.

Checkpoint B: luồng Test Mode → paid → outbox → Zalo test hoạt động; phát lại webhook không tạo event thứ hai.

### 5. Quản trị đối soát — Medium
Phụ thuộc: 3–4. File: `components/shop/payment-settings.tsx` (mới), `components/shop/admin.tsx`, `app/api/store/[...path]/route.ts`, `lib/payments/repository.ts`, `tests/sepay-admin.test.ts` (mới).
- Chủ sở hữu thấy kết nối, lần nhận gần nhất, giao dịch cần xử lý, lỗi Zalo; secret không trả về browser.
- Duyệt gắn giao dịch vào đơn có lý do/audit, chống dùng một giao dịch cho hai đơn và kiểm tra lại trạng thái dưới transaction.
- Đơn đã hủy chỉ xử lý qua quy trình nghiệp vụ hiện có; không tự phục hồi tồn kho hoặc thực hiện chuyển tiền hoàn trả.
Xác minh: quyền staff/owner, CSRF, lọc dữ liệu nhạy cảm, thao tác lặp và mobile.

### 6. Trải nghiệm khách và hồi quy — Small
Phụ thuộc: 3. File: `components/shop/checkout.tsx`, `components/shop/operations.tsx`, `tests/payment-ui.test.ts` (mới).
- Trang thành công/tra cứu hiển thị paid từ server, ẩn QR khi đã thanh toán/hết hạn.
- Không tăng polling vượt giới hạn; lỗi mạng không tự kết luận chưa trả/đã trả tiền.
- Kiểm tra mobile và JSON-LD hiện có không bị ảnh hưởng; không đưa trạng thái đơn/thông tin ngân hàng vào schema công khai.
Xác minh: browser 360px và desktop, webhook test cập nhật trang đang mở, regression structured-data.

### 7. Vận hành và triển khai có kiểm soát — Medium
Phụ thuộc: 1–6. File: `deploy/compose.yml`, `deploy/Dockerfile`, `.env.example`, `docs/vps.md`, `tests/sepay-integration.test.ts` (mới).
- Cấu hình env/feature flag; sao lưu và thử restore, migration bổ sung không phá state cũ; giữ ledger khi rollback.
- Chạy Test Mode riêng, kiểm tra HTTPS/Cloudflare/retry, sau đó cấu hình Live và smoke test giao dịch thật có kiểm soát khi triển khai được yêu cầu.
- Log có event ID, kết quả, độ trễ; theo dõi lỗi chữ ký/5xx/outbox, có công cụ replay. Khi rollback tắt auto-match, vẫn bảo toàn dữ liệu; nếu endpoint dừng trả 5xx để SePay retry, không trả 200 bỏ sự kiện.
Xác minh: full tests/typecheck/build, healthcheck, replay, khôi phục backup và xác nhận đơn/Zalo sau triển khai. Chỉ commit/push/deploy khi chuyển sang yêu cầu triển khai.

Checkpoint C: nghiệm thu luồng chuẩn và tất cả ngoại lệ trước bật Live tự xác nhận.

## Phần tiếp theo sau MVP

Đối soát định kỳ qua SePay API để bắt giao dịch mất webhook quá thời gian retry: cần khảo sát endpoint, phân trang, hạn mức và token chính thức trước khi viết. Dùng khoảng thời gian chồng lấn/cursor và cùng pipeline dedup; không đánh dấu paid từ nguồn không xác thực. Trong MVP dùng nhật ký SePay và replay thủ công, ghi rõ khoảng trống này trong vận hành. Endpoint trạng thái 5–10 giây có token riêng là cải tiến độc lập sau MVP.

## Điều kiện cấu hình còn thiếu

- Ngân hàng/tài khoản nhận tiền thực tế đã liên kết SePay; xác nhận hỗ trợ ngân hàng và gói dịch vụ phù hợp. Chưa xác minh tài khoản SePay của người dùng, không giả định đã kết nối.
- Secret Test/Live nhập trực tiếp vào cấu hình server, không gửi chat hoặc commit.
- Phiên Zalo đang kết nối và số admin/test recipient đã cấu hình; chưa gửi tin nhắn thử trong bước lập kế hoạch.
- Mặc định kế hoạch: một giao dịch đúng tổng tiền cho một đơn; các trường hợp còn lại duyệt thủ công.

## Nguồn chính thức đã đọc qua Firecrawl

- [Tích hợp webhook: payload, ID, ACK](https://developer.sepay.vn/vi/sepay-webhooks/tich-hop-webhook)
- [Xác thực HMAC và timestamp](https://developer.sepay.vn/vi/sepay-webhooks/xac-thuc)
- [Cấu trúc mã thanh toán và bộ lọc](https://developer.sepay.vn/vi/sepay-webhooks/cau-hinh-ma-thanh-toan)
- [Webhook và cơ chế retry](https://docs.sepay.vn/tich-hop-webhooks.html)
- [Test Mode tách biệt Live](https://docs.sepay.vn/test-mode.html)
