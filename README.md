# Commo · Mộc Vị

Website tiếng Việt để quảng bá sản phẩm và nhận đơn hàng, với thương hiệu tạm **Mộc Vị**. Chủ cửa hàng có thể thay tên, logo, màu sắc, nội dung và thông tin liên hệ trong trang quản trị.

- **Repository:** https://github.com/anhtuhd/commo
- **Website riêng tư:** https://moc-vi-com-viet-231026.spunky-dune-5060.chatgpt.site
- **Quản trị:** thêm `/quan-tri` vào URL website.

## Chức năng

- Danh mục, tìm kiếm, lọc giá, sắp xếp, chi tiết sản phẩm và bộ sưu tập.
- Giỏ hàng, checkout khách vãng lai, COD, chuyển khoản thủ công, voucher và tra cứu đơn.
- Yêu cầu tư vấn, báo giá theo sản phẩm/số lượng và hỗ trợ sau mua.
- Quản trị sản phẩm/tồn kho, đơn hàng, chiến dịch, bài viết, phí giao và nhân sự.
- Thương hiệu có bản nháp, xuất bản, khôi phục; upload ảnh PNG/JPEG/WebP vào R2.
- Giá, tồn và voucher kiểm tra tại máy chủ; chống gửi đơn trùng và xử lý cạnh tranh tồn cuối.

## Công nghệ

React · TypeScript · Vinext/Vite · Cloudflare Workers · D1 · R2 · Zod. Đăng nhập hosted qua ChatGPT/Sites. File `package-lock.json` khóa phiên bản phụ thuộc.

## Chạy trên máy cá nhân

Yêu cầu Node.js **22.13 trở lên**, npm và Git. Chạy các lệnh từ thư mục repository:

```sh
git clone https://github.com/anhtuhd/commo.git
cd commo
npm run install:ci
cp .dev.vars.example .dev.vars
npm run build
node --import ./scripts/sites-env.mjs ./node_modules/wrangler/bin/wrangler.js d1 execute DB --local --config dist/server/wrangler.json --persist-to .wrangler/state --file drizzle/0000_right_fantastic_four.sql
npm run dev -- --host 127.0.0.1 --port 5173
```

Mở http://127.0.0.1:5173. Lần đầu, truy cập `/quan-tri`, đăng nhập local rồi chọn **Thiết lập cửa hàng**. Middleware đăng nhập giả lập chỉ chạy trong môi trường phát triển; không dùng làm cơ chế xác thực production.

Dữ liệu mẫu tự khởi tạo khi DB chưa có cửa hàng. Đơn QA và ảnh local lưu trong `.wrangler/`, không được đưa lên Git hoặc bản triển khai. Không chạy migration local với tùy chọn `--remote`.

## Kiểm tra

```sh
node --test tests/*.test.ts
npx tsc --noEmit
npm run build
```

Kiểm thử tích hợp: chạy `node tests/api.mjs` khi dev server ở cổng 5173 đang hoạt động. Script tạo dữ liệu QA và sửa cấu hình trên DB local; dùng DB thử riêng. Giới hạn gửi có thể ảnh hưởng lần chạy lặp trong vòng 10 phút.

## Cấu trúc

| Thư mục | Nội dung |
|---|---|
| `app/` | Trang, metadata, API và xác thực |
| `components/shop/` | Giao diện cửa hàng và quản trị |
| `lib/shop/` | Dữ liệu mẫu, phân quyền, validation, tính đơn và lưu DB |
| `drizzle/` | Schema/migration D1 |
| `public/` | Logo tạm và ảnh WebP |
| `tests/` | Unit tests và kiểm thử API local |
| `docs/` | Hướng dẫn vận hành, kiến trúc và nghiệm thu |
| `scripts/` | Công cụ build/local runtime của Sites |

## Triển khai và quản trị

Bản hosted hiện dùng Sites. Giữ nguyên `project_id` trong `.openai/hosting.json` khi cập nhật site này. Push GitHub **không tự triển khai** website; bản triển khai Sites là quy trình riêng. Manifest chỉ chứa định danh và tên binding, không chứa khóa truy cập.

Lần đầu trên bản riêng tư, chủ đăng nhập `/quan-tri` và thiết lập cửa hàng. Thay nội dung mẫu, giá, tồn, chính sách, hotline và ngân hàng trước khi mở bán. Sau khi có chủ, tắt `OWNER_SETUP_ENABLED` ở môi trường hosted và triển khai lại trước khi mở site công khai.

Không commit `.env`, `.dev.vars`, token, khóa riêng, DB local, backup khách hàng hoặc build output. Chỉ các file mẫu `.env.example` / `.dev.vars.example` được phép theo dõi.

## Tài liệu và giới hạn bản đầu

- [Hướng dẫn vận hành](docs/operations.md)
- [Kiến trúc và giới hạn dữ liệu](docs/architecture.md)
- [Kết quả kiểm thử](docs/acceptance.md)

Ban đầu phục vụ Hà Nội; ngoài vùng chuyển tư vấn. Chuyển khoản đối soát thủ công, chưa tích hợp cổng thanh toán hay email/SMS/Zalo tự động. Sự kiện marketing ghi nội bộ, chưa kết nối GA4/Meta Pixel. Backup UI khôi phục cấu hình/danh mục, không rollback đơn hay tài khoản. Thiết kế lưu một trạng thái JSON trong D1 phù hợp bản đầu; cần tách bảng và phân trang phía máy chủ khi dữ liệu tăng.

Thương hiệu, logo, ảnh và giá đang là dữ liệu minh họa, có thể thay trong quản trị. Không sử dụng logo/ảnh thương hiệu Metri.

## Bản VPS

Website: https://pishare.cloud · Quản trị: https://pishare.cloud/quan-tri

Xem [hướng dẫn VPS, sao lưu và rollback](docs/vps.md). Dockerfile ở `deploy/Dockerfile`, Compose mẫu ở `deploy/compose.yml`. Bản VPS có đăng nhập chủ riêng và lưu dữ liệu bền vững trên máy chủ; không dùng đăng nhập ChatGPT. Thông tin đăng nhập bàn giao riêng ngoài Git.

### Vận hành cửa hàng (bản VPS)

Có lịch giao theo ngày/khung giờ và sức chứa; hạn giữ hàng riêng cho chuyển khoản; QR VietQR theo snapshot đơn; thông báo trình duyệt và Zalo bằng `zca-js` (cấu hình, kết nối QR trong Nhân sự & dữ liệu). Nội dung chính trang chủ chỉnh qua thương hiệu, danh mục lấy từ sản phẩm. VPS ẩn và chặn cấp tài khoản Sites.

Sao lưu DB + ảnh hằng ngày có kiểm tra phục hồi; công cụ tải bản sao ngoài VPS. Xem [hướng dẫn cấu hình và giới hạn vận hành](docs/vps.md). Chạy `node --test tests/*.test.ts` và `npx tsc --noEmit` trước triển khai.

## Đánh giá từ khách đã mua

Khách mở `/tra-cuu`, nhập mã đơn và điện thoại đã đặt hàng. Đơn phải ở trạng thái **Hoàn tất** và **Đã thanh toán**; mỗi sản phẩm trong mỗi đơn được gửi một đánh giá (1–5 điểm, tên công khai 2–60 ký tự, nội dung 10–2.000 ký tự). Tên hiển thị do khách chọn để tránh công khai thông tin thanh toán. Đây là xác minh qua thông tin đơn, không phải xác minh OTP số điện thoại.

Chủ cửa hàng vào **Quản trị → Đánh giá** để duyệt, từ chối hoặc ẩn. Từ chối/ẩn cần lý do nội bộ và được ghi nhật ký. Nội dung và điểm gốc không thể sửa qua API quản trị. Có thể duyệt lại đánh giá đã ẩn/từ chối; khách không thể gửi trùng để bỏ qua quyết định duyệt. Không loại đánh giá chỉ vì điểm thấp.

Chỉ đánh giá đã duyệt xuất hiện trên trang sản phẩm và trong JSON-LD `Product.review` / `Product.aggregateRating`. Số lượng và điểm trung bình dùng chung dữ liệu với phần hiển thị; khi chưa có đánh giá hoặc ẩn đánh giá cuối cùng, hai trường này được bỏ hoàn toàn. Dữ liệu thay đổi ở lần tải trang tiếp theo, không cần build lại. Không thêm rating cho Organization/Brand. Hoàn tiền sau khi đã gửi không tự động xóa trải nghiệm thật của người mua.

Đánh giá lưu trong SQLite cùng dữ liệu cửa hàng và được sao lưu đầy đủ bởi tác vụ backup hiện có. Khôi phục cấu hình từ trang quản trị giữ lại đơn hàng và đánh giá. Không có dữ liệu đánh giá mẫu trên production.

Kiểm tra logic: `node --test tests/*.test.ts`. Kết quả kiểm tra trình duyệt, API và validators: [báo cáo đánh giá](docs/validation/reviews-2026-10-24.md).

## Đo tốc độ, log và SEO

Chủ cửa hàng mở **Quản trị → Hiệu năng & SEO**:

- Web Vitals đo LCP, INP, CLS, FCP và TTFB theo vòng đời tài liệu. Chỉ số NAV đo từ bấm liên kết nội bộ đến khi nội dung trang đích đã mount và qua hai frame; đây là số đo nội bộ, không phải Soft Navigation Core Web Vitals. API đo thời gian fetch + đọc JSON từ trình duyệt.
- Báo cáo p75 theo nhóm trang và mobile/desktop trong 7 ngày, đọc tối đa 10.000 mẫu; cảnh báo thiếu mẫu khi dưới 20. Các ngưỡng NAV/API là mục tiêu vận hành nội bộ, không phải ngưỡng Google.
- Không lưu query string, nội dung form/đơn hàng, số điện thoại, cookie hoặc người dùng. Chi tiết sản phẩm/bài viết gộp thành `/san-pham/:id`, `/bai-viet/:id`. ID ngẫu nhiên chỉ dùng gộp các lần cập nhật cùng một metric. Tôn trọng Do Not Track; mẫu có thể thiếu do trình duyệt/ad blocker, nên không coi là số liệu đối soát.
- `POST /api/diagnostics`: cùng origin, payload tối đa 2KB, tên/giá trị/đường dẫn allowlist, giới hạn 120 yêu cầu/phút/IP theo process. IP chỉ tồn tại tạm trong rate limiter, không lưu DB/log. Chỉ owner đọc báo cáo. Giới hạn được áp dụng ở app; hệ thống nhiều process cần rate limiter chia sẻ hoặc giới hạn thêm tại proxy.
- SQLite tự tạo bảng riêng `performance_samples` và index; không ghi telemetry vào JSON đơn hàng. Cleanup khi có mẫu mới, tối đa mỗi 10 phút, giữ 30 ngày/20.000 mẫu sau cleanup. Backup SQLite hiện có bao gồm bảng này. Có thể xóa bảng telemetry mà không làm mất đơn hàng; ứng dụng tạo lại khi khởi động.
- API cửa hàng có `Server-Timing: app;dur=...` và `X-Request-ID`; log JSON gồm route template, method, status, durationMs và requestId. Ghi mọi API lỗi/chậm >500ms, lấy mẫu 10% phần còn lại; không ghi body, query hoặc URL media. Log lỗi telemetry chỉ có mã sự kiện và requestId. Docker production đã xoay vòng log web 3×10MB.
- Nút **Chạy kiểm tra SEO** quét tối đa 40 URL public từ dữ liệu đã xuất bản, tối đa 3 đồng thời, timeout 10s/request, nghỉ 30s giữa hai lượt. Kiểm tra HTTP/title/mô tả/canonical/H1/robots/viewport/alt/JSON-LD parse và tiêu đề trùng. Đây là kiểm tra HTML cơ bản, không thay thế validator, kiểm tra ngữ nghĩa schema hoặc đánh giá nội dung Google.
- Các liên kết PageSpeed Insights, Rich Results Test, Schema.org Validator và Search Console có sẵn. Chưa liên kết tài khoản Google/Search Console; muốn xem impressions/clicks cần chủ sở hữu xác minh trong Google.

Chạy Lighthouse bằng Chrome cài trên máy (Chrome riêng, không dùng profile đang đăng nhập):

```sh
npm run audit:web -- https://pishare.cloud work/performance/latest
```

Báo cáo HTML/JSON lưu trong thư mục chỉ định. Lighthouse chạy trên mạng/máy đo hiện tại; dùng cùng cấu hình và nhiều lượt khi so sánh. Không chạy Lighthouse trong container web production vì tăng tải CPU và RAM. Runtime chỉ thêm `web-vitals`; Lighthouse là devDependency. Font được self-host cùng giấy phép SIL OFL trong `public/fonts`; ảnh local dùng Next Image, mã admin tải riêng, request render dùng React cache theo request (không cache tồn kho xuyên request).

## Tự đối soát chuyển khoản SePay

VPS hỗ trợ webhook HMAC, mã thanh toán riêng trong QR, ledger chống ghi trùng, cập nhật paid và hàng đợi Zalo nguyên tử. Quản trị có mục Thanh toán SePay để cấu hình và xử lý giao dịch chưa khớp; chỉ owner được đối soát. Chưa cấu hình secret/ngân hàng thì tính năng chưa hoạt động với tiền thật. Xem [hướng dẫn cấu hình, kiểm thử và rollback](docs/sepay.md).
