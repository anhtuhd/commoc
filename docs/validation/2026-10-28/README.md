# Go + PostgreSQL milestone A — production verification

Ngày 28/10/2026, triển khai release `d9b6b05` tại https://pishare.cloud. Repo mới: https://github.com/anhtuhd/commoc. Frontend Next.js giữ nguyên để render HTML, metadata, sitemap và JSON-LD; Go xử lý nghiệp vụ và PostgreSQL lưu dữ liệu; Zalo bridge Node.js chạy riêng.

## Kiểm thử

- `go test -race -count=1 ./...`, `go vet ./...`: đạt trên PostgreSQL 18 QA. Kiểm tra cạnh tranh tồn kho, idempotency, rollback, SePay chữ ký/replay/conflict, sai số tiền/tài khoản/ngân hàng/thời gian, outbox retry, import lịch sử, lịch giao và hạn mức coupon.
- Bổ sung regression cho `/api/diagnostics`: truy vấn PostgreSQL cần `AS "at"`; đã sửa trước cutover và xác nhận API trả 200.
- TypeScript: 52/52 tests; `tsc --noEmit` đạt; Next.js production Docker build đạt; npm audit production dependencies: 0 vulnerabilities tại thời điểm kiểm tra.
- HTTP QA: đăng nhập, role/CSRF, checkout, duplicate order, signed webhook, paid tracking, order workflow, review duyệt, draft/publish/restore thương hiệu và JSON-LD đều đạt. Bridge mock xác minh retry; không gửi Zalo thật.
- Browser QA đã kiểm tra đặt hàng chuyển khoản, webhook giả lập có chữ ký và UI chuyển trạng thái paid/ẩn QR. Production kiểm tra mobile 390×844: sản phẩm → mua ngay → thanh toán, giá 120.000 + giao hàng 30.000 = 150.000; scrollWidth bằng 390, không ảnh lỗi, trang thanh toán không xuất JSON-LD. Không tạo đơn giả trên production.
- Smoke trên VPS: public projection khớp nguồn; 5 API quản trị, 5 page routes, session owner cũ và health Go/PostgreSQL đạt.

## Di chuyển và vận hành

- Import rehearsal và final đều đối chiếu toàn bộ state, thứ tự sản phẩm, ledger/outbox và metrics. Snapshot cuối: 7 sản phẩm, 0 đơn, 0 payment transaction, 0 outbox và 134 performance samples.
- Đã thử restore pg_dump vào database tạm thành công trước cutover.
- Backup trước chuyển: `/opt/commoc/backups/cutover-20261028T042433Z`, gồm SQLite snapshot, media/Zalo, cấu hình và secret chỉ lưu trên VPS, quyền hạn chế.
- Caddy chuyển riêng pishare.cloud từ 3301 sang 3401, giữ các website khác. Go/Next.js chỉ bind loopback; PG/bridge không mở cổng ra ngoài. Ghi đè IP client tại proxy Cloudflare.
- Các container cũ đã dừng, source/snapshot giữ lại. Timer cũ đã tắt; timer PostgreSQL backup mới chạy 03:30 giờ Việt Nam. Service backup đã chạy thành công.
- Sau khi PostgreSQL nhận ghi mới, không quay thẳng về SQLite snapshot. Xem `docs/migration/runbook.md`.

## Structured data — skill schema

[Google Rich Results Test](https://search.google.com/test/rich-results/result?id=7-fZAIMdlkuWk6gpbIWHHA) crawl thành công lúc 11:25:15 giờ Việt Nam, 28/10/2026: **3 mục hợp lệ**, gồm Product snippet, Merchant listing và Breadcrumb; không có lỗi bắt buộc.

Các cảnh báo không bắt buộc: `review`, `aggregateRating`, `shippingDetails`, `hasMerchantReturnPolicy`. Dữ liệu thật chưa được nhập/xác nhận nên giữ vắng mặt, không tạo đánh giá hoặc chính sách giả để xóa cảnh báo.

[Schema.org Validator — sản phẩm](https://validator.schema.org/#url=https%3A%2F%2Fpishare.cloud%2Fsan-pham%2Fcom-tuoi): 0 lỗi, 0 cảnh báo, Product và BreadcrumbList. [Trang chủ](https://validator.schema.org/#url=https%3A%2F%2Fpishare.cloud%2F): WebSite với Organization/Brand tham chiếu, 0 lỗi, 0 cảnh báo.

Đối chiếu tự động HTML server-rendered trên 11 URL: trang chủ, 7 sản phẩm và 3 trang riêng tư. Tên thương hiệu, tên sản phẩm, giá, tiền tệ, tồn kho và breadcrumb khớp API quản trị công khai. Trang báo giá chỉ có Brand/Breadcrumb; không tạo Offer giá 0. Không có review/aggregateRating khi không có đánh giá duyệt. Kết quả: `schema-parity.json`. Chạy lại: `python3 scripts/validation/check-production-schema.py`.

## Giới hạn / cấu hình còn cần thực hiện

SePay vẫn chưa có secret/tài khoản ngân hàng và đang tắt; Zalo chưa có phiên đăng nhập/người nhận và đang tắt. Checkout hiển thị đúng rằng chuyển khoản chưa kích hoạt. Đã kiểm thử cơ chế bằng QA; chưa xác nhận giao dịch ngân hàng hoặc tin Zalo thực tế.

Mốc A giữ DTO trong JSONB cùng constraint và transaction PostgreSQL, khóa ghi toàn cửa hàng. Chưa phải schema chuẩn hóa hoàn toàn, chưa có PITR/đối soát SePay ngoài webhook; các hạng mục này là bước sau.
