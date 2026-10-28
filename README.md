# Mộc Vị — Next.js + Go + PostgreSQL

Source: https://github.com/anhtuhd/commoc · Website: https://pishare.cloud

Giao diện React/Next.js được giữ nguyên lâu dài: SSR, metadata, canonical, sitemap, robots và JSON-LD vẫn do Next.js tạo từ API public của Go. Next.js không còn đọc hoặc ghi SQLite. Go quản lý API, đăng nhập, phân quyền, catalog, đơn hàng, tồn kho, coupon, lịch giao, review, SePay, outbox và số đo hiệu năng. Công cụ kiểm tra HTML SEO chạy trong Next.js với quyền quản trị do Go xác thực.

## Kiến trúc

Cloudflare → Caddy → Next.js → Go → PostgreSQL. Go worker → Zalo bridge Node.js (`zca-js`). Database và bridge chỉ nằm trong mạng Docker riêng; frontend/API chỉ bind localhost trên VPS.

PostgreSQL lưu mỗi sản phẩm/đơn/review thành một row trong bảng tương ứng; JSONB giữ cấu trúc DTO để tương thích giao diện. Các trường định danh, SKU, tồn và tiền có cột/ràng buộc; order_items, ledger, outbox, rate limits, metrics có bảng riêng. Đây là mô hình lai cho mốc A, chưa phải chuẩn hóa toàn bộ thuộc tính thành cột. Khóa transaction theo cửa hàng tuần tự hóa writer để giữ tính đúng; các lượt đọc dùng snapshot MVCC. Khi dữ liệu tăng lớn, cần phân trang và chuyển các thao tác ghi sang truy vấn từng tập bản ghi thay vì dựng state trong bộ nhớ.

## Chạy và kiểm thử

- Backend: Go 1.26+, PostgreSQL 18. `cd backend && go test ./... && go vet ./...`
- Integration/race: đặt `TEST_DATABASE_URL` trỏ **database QA riêng** rồi chạy `go test -race ./...`. Suite tích hợp xóa schema QA; không dùng URL production.
- Frontend: `npm ci --legacy-peer-deps`, `COMMO_RUNTIME=vps GO_API_URL=http://127.0.0.1:8080 NEXT_PUBLIC_SITE_URL=http://localhost:3401 npx next dev --webpack --port 3401`.
- Kiểm tra TypeScript và JSON-LD: `npx tsc --noEmit`, `node --test tests/*.test.ts`.
- `tests/go-api.mjs` kiểm thử HTTP localhost: auth, CSRF, đặt hàng, webhook ký giả lập, tra cứu, duyệt review, draft/publish, JSON-LD. Cần cấu hình QA trong `work/go/qa.env` và mật khẩu QA trong `work/go/qa-password`; không chạy trên production.
- Build: `deploy/Dockerfile.go`, `deploy/Dockerfile.frontend`, `deploy/Dockerfile.bridge`. Docker build có dependency độc lập; node_modules local không được đóng gói.

## Cấu hình và triển khai

Xem [runbook](docs/migration/runbook.md), [kế hoạch](tasks/plan.md), [checklist](tasks/todo.md). Docker Compose: `deploy/compose.yml` với `RELEASE` là commit triển khai. File env thực nằm tại `/opt/commoc/secrets/`, không trong Git. `.env.example` chỉ là mẫu.

`server import /path/export.json` là lệnh **offline thay toàn bộ dữ liệu đích** trong một transaction. Chỉ chạy khi đã dừng API/worker đích và giữ snapshot nguồn bất biến. Export bằng `scripts/migration/export-sqlite.py` từ SQLite snapshot nhất quán, không copy trực tiếp file SQLite đang bật WAL. Import kiểm tra owner hiện tại, giữ ID/mã đơn, trạng thái paid, sent và thứ tự nội dung; canonical hóa fingerprint thanh toán để phát lại webhook không tạo tác dụng mới.

SePay giữ `/api/webhooks/sepay`, raw-body HMAC, exact amount/account/gateway/code, deadline và dedup. Zalo đăng nhập QR trong quản trị, session riêng trên volume; lỗi Zalo không làm mất thanh toán. Không đảm bảo gửi tin đúng một lần tuyệt đối nếu tiến trình crash sau khi gửi và trước khi ghi trạng thái sent.

Các module TypeScript nghiệp vụ cũ và tests được giữ làm tài liệu/fixture hồi quy, không được gọi trong runtime backend mới. Không dùng Dockerfile/timer/backup SQLite cũ để vận hành bản PostgreSQL; runbook mới là tài liệu triển khai chính.
