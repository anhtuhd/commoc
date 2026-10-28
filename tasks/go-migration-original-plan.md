# Kế hoạch chuyển Mộc Vị sang Go + PostgreSQL

Ngày: 28/10/2026. Trạng thái: đề xuất, chưa triển khai. Phạm vi: pishare.cloud và repository anhtuhd/commo.
Kế hoạch SePay cũ được giữ tại `sepay-plan-2026-10-25.md`, checklist tại `sepay-todo-2026-10-25.md`.

## 1. Mục tiêu và quyết định kiến trúc

Chuyển nghiệp vụ, API, tác vụ nền và cuối cùng phần render website/quản trị sang Go; PostgreSQL là nguồn dữ liệu chính. Giữ toàn bộ chức năng và URL hiện có. Đây là kế hoạch chuyển cả ứng dụng, không chỉ thay driver SQLite.

Triển khai theo hai mốc:
- Mốc A: Go API + PostgreSQL, giao diện Next.js hiện tại gọi API Go qua lớp adapter, không đọc SQLite. Bước này giúp kiểm chứng nghiệp vụ trước khi viết lại giao diện.
- Mốc B (đích cuối): Go render HTML bằng html/template, dùng lại CSS và tài nguyên hình ảnh, JavaScript nhỏ cho giỏ hàng, modal, polling và biểu đồ; thay cả storefront và quản trị, gỡ Next.js khỏi runtime.
- Ngoại lệ có chủ đích: giữ Node.js + zca-js trong dịch vụ Zalo riêng. Thư viện hiện tại là JavaScript, không chuyển trực tiếp sang Go. Nếu yêu cầu loại bỏ Node.js hoàn toàn, phải khảo sát lại tích hợp Zalo thành một dự án riêng trước khi cam kết.

Đề xuất Go net/http, pgx/v5 với connection pool, migration SQL có phiên bản. Chọn và ghim phiên bản Go/PostgreSQL ổn định được hỗ trợ khi bắt đầu triển khai. Chưa cần microservices, Redis hoặc Kubernetes. Một ứng dụng Go chia module nghiệp vụ, PostgreSQL và một Zalo bridge là đủ cho giai đoạn này.

Luồng đích: Cloudflare → Caddy → Go web/API → PostgreSQL. Go worker claim outbox → Zalo bridge nội bộ → admin. Media giữ volume hiện tại; URL ảnh giữ nguyên. Database và bridge không mở cổng public.

## 2. Hiện trạng đã kiểm tra từ source local

- Next.js/React, TypeScript; route storefront tổng hợp và các API store, auth, diagnostics, SEO audit, admin payments, SePay.
- `lib/shop/repository.ts`: shop_state.body chứa gần toàn bộ dữ liệu; revision CAS chống ghi đè. updateState chạy expireOrders trước callback.
- `lib/vps/runtime.ts`: SQLite WAL và media trên disk; Sites có adapter D1/R2 riêng. Bản Go nhắm VPS; không giả định có thể deploy Go vào Sites runtime hiện tại.
- `lib/payments/sqlite.ts`: payment_transactions và payment_outbox độc lập; ledger + cập nhật đơn + outbox đang dùng transaction SQLite.
- `lib/vps/auth.ts`: tài khoản cấu hình env, hash scrypt, cookie HMAC hạn 8 giờ. Phải kiểm tra tương thích hoặc yêu cầu đăng nhập lại, không tự thay mật khẩu.
- `lib/diagnostics/storage.ts`: performance_samples có retention 30 ngày, tối đa 20.000 mẫu.
- `lib/shop/types.ts`: thương hiệu có draft/history; sản phẩm, bộ sưu tập, chiến dịch, bài viết, coupon, đơn, lịch giao, liên hệ/báo giá, review có duyệt, quyền quản trị, audit, cấu hình Zalo và SePay.
- Chưa kiểm tra lại VPS hoặc số lượng dữ liệu production trong bước lập kế hoạch này. Trạng thái liên kết MB/SePay và phiên Zalo cần xác minh trước nghiệm thu Live.

## 3. Thiết kế dữ liệu

| Nhóm | Bảng đề xuất | Quy tắc |
|---|---|---|
| Nội dung | brand_versions, settings, products, collections, product_collections, campaigns, articles, media | Tách draft/published; JSONB cho cấu hình linh hoạt, không chứa toàn bộ shop trong một row |
| Mua hàng | orders, order_items, inventory_movements, coupons, coupon_redemptions | ID/mã đơn/mã thanh toán giữ nguyên; tiền VND BIGINT; snapshot tên, giá, ngân hàng trên đơn |
| Giao hàng | delivery_slots, delivery_reservations | Khóa/cập nhật nguyên tử sức chứa theo ngày và slot; giữ lựa chọn của đơn lịch sử |
| Khách hàng | inquiries, reviews | Giữ dữ liệu người mua trong đơn; review chỉ public khi approved và đúng điều kiện người đã mua |
| Thanh toán | payment_transactions, notification_outbox | Unique provider+mode+connection+provider_id; giữ fingerprint, trạng thái xử lý và lịch sử gửi |
| Vận hành | admin_roles, audit_logs, integration_settings, integration_status, performance_samples, rate_limits | Không đưa secret/phiên Zalo vào public API; TTL cho rate limit và metric |

Dùng TIMESTAMPTZ cho thời điểm; ngày giao là DATE, giờ slot theo Asia/Ho_Chi_Minh. Giữ ID cũ dạng text nếu cần, không ép đổi sang UUID làm hỏng liên kết. Ràng buộc FK, unique payment_code và idempotency key, CHECK số tiền/tồn kho hợp lệ. Liên kết dữ liệu cũ không hợp lệ phải xuất báo cáo để xử lý, không im lặng loại bỏ.

Transaction tạo đơn tính giá ở server, khóa sản phẩm theo thứ tự cố định, giữ chỗ slot và coupon, trừ tồn, tạo order/items và outbox một lần. Hủy/hết hạn trả lại tồn và sức chứa đúng một lần. Webhook khóa đơn, ghi ledger + paid + outbox trong cùng transaction. Retry có giới hạn khi deadlock/serialization conflict; không gọi Zalo trong transaction.

## 4. Các bước thực hiện và tiêu chí nghiệm thu

Mỗi bước gồm các lát cắt nhỏ, khoảng 2–5 file mỗi lượt; đường dẫn dưới đây là vị trí dự kiến. Kiểm tra bằng Go unit/integration tests với PostgreSQL riêng, không tạo đơn thử trên production trong lúc phát triển.

### T01 — Chốt hợp đồng và baseline (M, không phụ thuộc)
- Việc: kiểm kê endpoint, action, HTTP status, quyền, URL, schema JSON-LD và màn hình; lưu fixture đã ẩn dữ liệu khách. Đo Lighthouse mobile, API p50/p95, RSS/CPU với cùng dữ liệu và tải.
- Đạt: có ma trận mọi tính năng hiện tại và test fixture cho luồng chuẩn/ngoại lệ; xác định đủ writer gồm worker, maintenance, admin và webhook.
- Xác minh: chạy bộ test hiện tại, so mẫu response và snapshot trang. File: docs/migration/contracts.md, tests/migration/*.

### T02 — Khung Go và PostgreSQL (M, sau T01)
- Việc: cmd/web, config, HTTP middleware, pgx pool, migration runner, health/readiness, log JSON có request ID.
- Đạt: start/stop sạch; DB hỏng readiness thất bại; credentials không ở source/log; timeouts và pool giới hạn.
- Xác minh: go test ./..., go vet ./..., build Docker và thử mất kết nối DB. File: backend/cmd/web, backend/internal/platform, backend/migrations, deploy/compose.go.yml.

### T03 — Import nội dung và API public (M mỗi lát cắt, sau T02)
- Việc: schema/importer cho brand, catalog, nội dung, media manifest; public API cùng hợp đồng cũ. Import từ SQLite snapshot nhất quán, chạy lặp không nhân đôi.
- Đạt: ID/URL/ảnh giữ nguyên; draft và secret không ra public; checksum dữ liệu chuẩn hóa và số bản ghi khớp.
- Xác minh: import hai lần vào DB thử; so public response cũ/mới; kiểm tra ảnh. File: backend/cmd/import, backend/internal/catalog, backend/migrations, tests/migration/catalog.*.

Checkpoint A: nền tảng build được và dữ liệu public tương đương trước khi triển khai nghiệp vụ ghi.

### T04 — Đăng nhập và quản trị nội dung (M mỗi lát cắt, sau T03)
- Việc: auth scrypt tương thích có test vector, session an toàn, phân quyền owner/staff/editor, CSRF/Origin, upload; tiếp theo CRUD, draft/publish/history.
- Đạt: quyền cũ giữ nguyên; nội dung publish cập nhật public nhất quán; đăng nhập lại có kiểm soát nếu session thay đổi.
- Xác minh: test quyền âm tính, CSRF, cookie Secure/HttpOnly/SameSite, upload sai định dạng/path traversal, thử draft/publish/restore. File: backend/internal/auth, backend/internal/admin, backend/internal/media, migrations và tests tương ứng.

### T05 — Đặt hàng, tồn kho và coupon (M mỗi lát cắt, sau T04)
- Việc: port luật commerce; importer orders/items/coupons và snapshot cũ; idempotency, tra cứu đơn.
- Đạt: gửi lại request không nhân đơn/trừ tồn; nhiều người mua sản phẩm cuối chỉ có số đơn phù hợp tồn; không lộ đơn qua tra cứu trái quyền.
- Xác minh: golden tests TS/Go cùng đầu vào; integration nhiều connection PostgreSQL, double-submit và hủy lặp. File: backend/internal/orders, backend/internal/inventory, backend/internal/coupons, tests tương ứng.

### T06 — Lịch giao, hết hạn và yêu cầu khách (M mỗi lát cắt, sau T05)
- Việc: giữ chỗ slot; maintenance Go thay timer/script cũ; chuyển liên hệ, báo giá, hậu mãi và review/moderation.
- Đạt: slot không vượt sức chứa; hết hạn không xung đột đơn vừa paid; đánh giá hợp lệ chỉ public sau duyệt.
- Xác minh: race checkout/cancel/expire/payment, múi giờ qua ngày, lead time, review không đúng người mua bị từ chối. File: backend/internal/delivery, backend/internal/reviews, backend/internal/inquiries, backend/cmd/worker.

### T07 — SePay và đối soát quản trị (M mỗi lát cắt, sau T05–T06)
- Việc: port raw-body HMAC/timestamp, mã MV, chống trùng và fingerprint, ledger, manual resolve, QR; giữ endpoint /api/webhooks/sepay.
- Đạt: không đổi luật đúng số tiền/tài khoản/gateway/thời hạn; VA/subAccount vẫn review; paid không tự đổi trạng thái giao hàng. Giữ test/live tách biệt.
- Xác minh: chữ ký sai/body sửa/replay/ID trùng khác payload; sai tiền/hết hạn; callback đồng thời maintenance/admin; DB lỗi không ACK thành công. File: backend/internal/payments, backend/internal/http/sepay, tests/payment*.

### T08 — Zalo bridge và outbox (M mỗi lát cắt, sau T07)
- Việc: Go quản lý lease/retry/outbox; bridge Node chỉ quản lý login QR/session và gửi tin, API nội bộ xác thực bằng secret riêng. Worker Node cũ ngừng đọc/ghi SQLite.
- Đạt: giữ trạng thái notificationLog và sent event khi import, không gửi lại lịch sử; mất phiên không mất event; response lease cũ không đánh dấu job mới thành công.
- Xác minh: sender giả lập lỗi/crash/retry, reconnect QR; test thật với admin sau khi cấu hình. Dùng FOR UPDATE SKIP LOCKED để claim queue, commit trước gọi mạng. Crash sau gửi trước ACK vẫn có thể gửi lặp; message có event ID để đối chiếu.
- File: backend/internal/notifications, services/zalo-bridge, backend/migrations/outbox*, tests/notification*.

Checkpoint B: luồng đặt hàng → webhook → paid → outbox hoạt động trên staging; dữ liệu imported không sinh thông báo lịch sử.

### T09 — Kết nối UI hiện tại và đo lường (M mỗi lát cắt, sau T04–T08)
- Việc: Next.js server loaders và API adapter đọc Go; chuyển cả auth/diagnostics/SEO audit, media và maintenance để không còn writer SQLite. Auth forwarding chỉ nội bộ, không tin header từ Internet.
- Đạt: Mốc A chạy hoàn chỉnh trên staging; cache invalidation sau publish/review/tồn kho đúng; log/metric có retention tương đương hiện tại.
- Xác minh: toàn bộ UI, mobile 360/390/768px, checkout và quản trị; so response/JSON-LD; scan runtime không còn truy cập DB cũ. File: lib/shop/repository.ts, route adapters, backend/internal/diagnostics, deploy configs.

### T10 — Chuyển storefront sang Go SSR (M mỗi nhóm trang, sau T09)
- Việc: html/template cho layout/home → catalog/product → bài viết/chính sách → cart/checkout/tra cứu; tái sử dụng CSS, ảnh và browser state có migration nếu đổi định dạng.
- Đạt: URL, canonical, sitemap, robots, metadata và JSON-LD tương đương; Brand/Product/Breadcrumb/review chỉ lấy published dữ liệu thật. Giữ focus, back/forward, validation và giỏ hàng qua tải lại.
- Xác minh: visual regression desktop/mobile, keyboard, no-JS nội dung SEO; rich results + Schema.org Validator, test script injection trong JSON-LD. File: backend/internal/web, backend/templates, backend/static, tests/browser.

### T11 — Chuyển quản trị sang Go SSR (M mỗi module, sau T10)
- Việc: login/dashboard → catalog/brand → orders/reviews → SePay/Zalo → diagnostics/SEO; gỡ Next runtime khi ma trận tính năng đạt.
- Đạt: đủ thao tác quản trị và phân quyền hiện tại, form chống mất dữ liệu, upload/QR hoạt động mobile; Node chỉ còn ở Zalo bridge và build tooling nếu cần.
- Xác minh: E2E theo role, lỗi API/form, publish đến storefront/JSON-LD, build Go độc lập Next. File: backend/templates/admin, backend/internal/web/admin, backend/static/admin, deploy configs.

### T12 — Tổng kiểm thử và diễn tập chuyển dữ liệu (M mỗi lượt, sau T11)
- Việc: restore bản sao SQLite nhất quán + media; import đủ ledger/outbox/audit/settings/review/metrics; báo cáo tổng tiền theo trạng thái, tồn kho, coupon và slot. Thiết kế importer không tự kích hoạt auto-match hay gửi tin.
- Đạt: không mất dữ liệu; số liệu đối chiếu bằng nhau; rehearsal đo được thời gian import, backup/restore, cutover và rollback; đạt mọi gate bên dưới.
- Xác minh: go test -race ./..., integration DB thật, browser E2E, Lighthouse trước/sau 3 lượt cùng điều kiện lấy median; kiểm tra Schema.org và Google. File: scripts/migration, docs/migration/runbook.md, docs/validation/go-postgres.md.

### T13 — Commit, push và cutover production (M, sau T12; thực hiện khi chuyển sang triển khai)
- Việc: release tag/image bất biến; backup và restore thử; maintenance ngắn cho mọi writer; chuyển routing và kiểm tra rồi mở ghi.
- Đạt: PostgreSQL là nguồn dữ liệu duy nhất; URL SePay không đổi; đơn, payment và outbox bảo toàn; monitoring và backup PostgreSQL hoạt động.
- Xác minh: checklist cutover/rollback bên dưới, read-only smoke trước mở ghi; giao dịch thật có kiểm soát do người dùng thực hiện sau cấu hình.

## 5. Cutover và rollback an toàn

1. Kiểm tra dung lượng/RAM VPS thực tế; chạy Go + PostgreSQL staging trong mạng riêng, tắt gửi Zalo và auto-match trên bản sao. Không dùng snapshot staging ghi đè production.
2. Chọn cửa sổ bảo trì theo thời gian diễn tập. Chặn tạo/sửa đơn, admin writes; dừng maintenance và worker cũ. Webhook trả 503 tạm thời để nhà cung cấp retry, tuyệt đối không ACK rồi bỏ sự kiện. Xác minh cửa sổ retry nhà cung cấp trước khi chọn thời gian dừng.
3. Chờ request đang chạy kết thúc; chốt snapshot SQLite bằng backup API hoặc phương pháp nhất quán WAL, sao lưu media và cấu hình. Import final snapshot vào PostgreSQL; đối chiếu các invariant. Giữ toàn bộ ID, timestamps, paymentCode, idempotency keys và sent status.
4. Đổi Caddy sang bản mới; health/readiness và kiểm tra chỉ đọc khi vẫn khóa ghi. Nếu lỗi lúc này: trả routing về bản cũ và mở writer cũ, vì chưa phát sinh ghi mới.
5. Nếu đạt: mở ghi chỉ trên Go, chạy worker Go + bridge; xác minh retry webhook, hàng đợi và các đơn còn hạn. Các lease cũ cần xử lý có kiểm soát, không đánh dấu sent thành pending hàng loạt.
6. Sau khi PostgreSQL đã nhận ghi mới, không được rollback về SQLite snapshot cũ. Ưu tiên sửa tiến hoặc rollback image Go tương thích schema PostgreSQL. Nếu bắt buộc về app SQLite, cần bảo trì, reverse export đã diễn tập và đối chiếu toàn bộ delta đơn/payment/outbox rồi mới mở lại. Nếu chưa có reverse exporter đạt test thì không coi đây là đường rollback khả dụng.
7. Backup PostgreSQL tự động, mã hóa/quyền truy cập thích hợp, retention và thử restore; chọn PITR/WAL nếu yêu cầu mất dữ liệu tối đa dưới khoảng cách backup. Chốt RPO/RTO bằng kết quả diễn tập, không hứa zero downtime hay zero data loss khi chưa có cơ chế tương ứng.

## 6. Gate nghiệm thu

- Tất cả tính năng trong ma trận T01 có test đạt; không bỏ draft/history, review, shipping capacity, coupon, QR, SePay, Zalo, diagnostics hoặc SEO.
- Import không chênh số đơn, tổng tiền theo trạng thái, tồn/coupon/slot, ledger và outbox sent/pending; ảnh/link cũ còn đúng.
- Test cạnh tranh không oversell, không trả tồn hai lần, không mất paid, không tạo event thanh toán thứ hai khi replay.
- JSON-LD khớp quản trị; không bịa review/rating/policy để hết warning. Trang riêng tư không lộ schema dữ liệu khách.
- Mục tiêu thử nghiệm mobile: LCP ≤2,5 giây, CLS ≤0,1; theo dõi INP thực tế ≤200ms khi đủ mẫu. API p95 không thoái lui quá 10% ở cùng tải/dataset; chốt tải và baseline ở T01. Không coi đổi Go tự động làm frontend nhanh hơn.
- Không secret trong Git/log; role/CSRF/rate limit/upload kiểm tra đầy đủ; bridge/DB chỉ nội bộ.
- Khôi phục backup và rollback trước mở ghi đã diễn tập; sau mở ghi chỉ quay về release tương thích PostgreSQL hoặc thực hiện reverse migration đã kiểm chứng.

## 7. Ước lượng và điểm cần chốt

Ước lượng sơ bộ 15–25 ngày công kỹ thuật gồm cả Go SSR và admin; sẽ hiệu chỉnh sau T01 và thử nghiệm importer/checkout. Mốc A khoảng 8–13 ngày, Mốc B và tổng kiểm thử khoảng 7–12 ngày. Đây là ước lượng phạm vi, không phải cam kết thời gian hoàn thành tự động.

Giả định hiện tại: đích cuối Go SSR + PostgreSQL, giữ Zalo Node bridge; giữ thiết kế và URL, không redesign cùng lúc; một cửa hàng và ngân hàng nhận tiền đã cấu hình. Nếu chỉ muốn backend Go thì dừng ở Mốc A, giữ React/Next. Cần chốt cửa sổ bảo trì, mức RPO/RTO và cấu hình dịch vụ thật trước production, không cần cung cấp secret trong chat.

## Nguồn kỹ thuật

- https://pkg.go.dev/html/template — render HTML và escaping theo ngữ cảnh.
- https://github.com/jackc/pgx — driver/toolkit PostgreSQL cho Go.
- https://www.postgresql.org/docs/current/sql-select.html — row locking/SKIP LOCKED cho hàng đợi; không dùng SKIP LOCKED cho phép tính tồn hoặc số liệu đối soát cần đầy đủ.
- https://github.com/RFS-ADRENO/zca-js — SDK JavaScript cho tài khoản Zalo cá nhân, lý do giữ bridge riêng.
