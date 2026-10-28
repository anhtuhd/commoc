# Hiệu năng, log và SEO — 24/10/2026

## Mục tiêu và thay đổi

Đo được tốc độ tải/chuyển trang, phân biệt server/API và trình duyệt; tìm trang/thiết bị chậm; kiểm tra SEO/JSON-LD mà không ghi dữ liệu khách vào log.

- Web Vitals, NAV, API → SQLite table riêng → báo cáo p75 trong quản trị; giới hạn payload/rate/retention, chỉ owner đọc, không lưu query/PII. NAV chỉ kết thúc khi StoreApp trang đích mount + hai frame, không coi skeleton đã hiện là tải xong trang.
- Structured API log + Server-Timing + X-Request-ID; log rotation production 30MB cho web đã có.
- Lighthouse CLI (`npm run audit:web`), báo cáo HTML/JSON; quản trị có audit SEO 10 tiêu chí và liên kết công cụ Google/Schema.org.
- Tách bundle admin, breadcrumbs/review summary khỏi module validation dùng trên client; bỏ reload toàn tài liệu ở Mua ngay; loading skeleton và progress bar tôn trọng reduced motion.
- Font tự lưu thay Google CSS @import; Next Image phục vụ ảnh local theo kích thước viewport, cache ảnh/font; React cache chỉ trong mỗi request render để tránh đọc JSON SQLite lặp metadata/viewport/page.
- Canonical homepage, title riêng cho từng chính sách, sitemap thêm policy, robots loại trang giỏ/thành công/đăng nhập. Homepage có main landmark; cải thiện contrast và ẩn số giỏ hàng khỏi accessible name.

## Trước tối ưu

Lighthouse 13.5.0, Chrome headless profile riêng, mặc định mobile + simulated throttling, trang `https://pishare.cloud/`, cold-cache của Lighthouse. Lượt đầu: Performance 64, Accessibility 94, Best Practices 100, SEO 100; FCP 4.0s, LCP 6.8s, TBT 10ms, CLS 0. Báo cáo chỉ ra 641KiB ảnh có thể tiết kiệm, render blocking 2.17s. Các giá trị là lab, không phải CrUX/field.

DevTools trace không throttle (khác phép thử): LCP 611ms, TTFB 161ms, CLS 0; không có dữ liệu CrUX. Không trộn hai cấu hình đo để kết luận cải thiện.

## Kiểm tra trước deploy

- 37/37 unit tests, TypeScript và production build đạt. Tests kiểm tra bounds/PII path, p75, SEO HTML/schema parse; các test thương mại, quyền, policy và JSON-LD cũ vẫn đạt.
- API QA đạt: auth owner, origin sai, query có PII bị chặn, gộp update cùng metric, Server-Timing/Request-ID, public data không chứa telemetry.
- 18/18 trang QA đạt 10/10 audit kỹ thuật, robots/sitemap trả 200. Chưa xác minh Google Search Console hay gửi sitemap bằng tài khoản Google.
- Browser: chuyển home→catalog giữ nguyên document navigation, có NAV ghi vào SQLite; QA đo khoảng 138ms (chỉ minh chứng collector hoạt động, không dùng làm benchmark production). Màn hình quản trị 375px không tràn ngang. Console không có error/warning trong luồng đã kiểm tra.
- QA Lighthouse trung gian: Performance 92, Accessibility 100, Best Practices 100, SEO 100, LCP 3.3s; vì chạy localhost nên không dùng làm số before/after chính thức.

## Kiểm tra schema theo skill schema

Không tạo giá, review hay chính sách giả để xóa cảnh báo. So sánh graph production với build tối ưu ở homepage, sản phẩm, bài viết và policy: giữ nguyên dữ liệu thật (QA có review thử chỉ nằm local, loại khỏi phép so sánh).

- Schema.org Validator URL `https://pishare.cloud/san-pham/com-tuoi`: 0 lỗi, 0 cảnh báo; Product/BreadcrumbList, Brand liên kết hợp lệ.
- Google Rich Results Test: thử code mode với JSON-LD sản phẩm lấy từ production ở hai browser context riêng; cả hai trả `Something went wrong — Log in and try again`. Không coi đây là kết quả đạt/lỗi schema, không đăng nhập Google bằng thông tin không được cấp. Có link công cụ trong quản trị để chủ cửa hàng chạy lại.
- Bộ tests structured-data/business/reviews xác nhận offer/stock/brand, policy đã xác nhận, rating chỉ từ review đã duyệt và escape JSON-LD vẫn đúng.

## Giới hạn / dependency

Mẫu RUM do client gửi có thể bị thiếu hoặc bị giả; chỉ dùng chẩn đoán. Báo cáo giữ tối đa 10.000 mẫu gần nhất trong 7 ngày. Collector tạo bảng tự động, rollback web cũ vẫn đọc được shop_state; worker Zalo giữ nguyên. Sao lưu DB/media trước deploy.

`npm audit --omit=dev` ghi nhận 1 high đã có trên dependency Zod 3: GHSA-mrhx-c7xc-m3w5; bản sửa được npm đề xuất là major 4.6.7. Không nâng major validation của toàn bộ commerce trong đợt tối ưu này; request cửa hàng có giới hạn 1MB, telemetry 2KB và chỉ nhận scalar, nên không đưa constrained collections không giới hạn vào collector. Cần đợt migration Zod riêng để xử lý advisory đầy đủ.

## Kết quả production sau deploy

Release web `d7f7112` healthy, worker Zalo giữ nguyên. Kiểm tra read-only production: 7 sản phẩm, 0 đơn và 0 review; JSON-LD khớp dữ liệu quản trị, không tạo review/policy giả. Báo cáo telemetry và SEO yêu cầu đăng nhập; Server-Timing/Request-ID có trong public API; font không gọi Google bên ngoài, ảnh tối ưu trả 200. DOM đã render có một H1 và một main hiển thị (raw streaming HTML còn skeleton tạm nên không đếm main bằng regex).

Ba lượt trước và ba lượt sau cùng `https://pishare.cloud/`, Lighthouse 13.5.0, mobile, simulate 150ms RTT / 1.6Mbps / CPU 4×, Chrome profile riêng và browser storage reset của Lighthouse:

| Chỉ số (trung vị 3 lượt) | Trước | Sau |
| --- | ---: | ---: |
| Performance | 64 | 76 |
| Accessibility | 94 | 100 |
| Best Practices | 100 | 100 |
| SEO | 100 | 100 |
| FCP | 4,03 giây | 2,98 giây |
| LCP | 6,75 giây | 4,49 giây |
| Total Blocking Time | 13 ms | 0 ms |
| CLS | 0 | 0,0015 |
| Speed Index | 5,53 giây | 4,71 giây |

Các lượt sau: Performance 92 / 76 / 76; LCP 3,06 / 4,56 / 4,49 giây. Dùng trung vị, không chọn riêng lượt 92 để kết luận. LCP giảm khoảng 33,5% nhưng vẫn vượt ngưỡng tốt 2,5 giây; cần theo dõi RUM có đủ mẫu và tiếp tục xử lý chuỗi font/render delay nếu vẫn chậm trên thiết bị thật. CLS vẫn ở mức tốt ≤0,1. Tổng bytes lượt đầu giảm từ khoảng 1,14MB còn 0,69MB; JS transfer khoảng 189KB→191KB do thêm đo lường, nên không tuyên bố tổng JS đã giảm. Không coi SEO 100 là bảo đảm thứ hạng hoặc nội dung đạt chất lượng.

Báo cáo đầy đủ (gitignored local): `work/performance/{before,before-2,before-3,after,after-2,after-3}/report.html` và `report.json`; số liệu thô `work/performance/comparison.json`. Các lần chạy cách nhau trong ngày nên mạng/edge cache có thể thay đổi; đây là so sánh lab có kiểm soát cấu hình, chưa phải xác nhận CrUX.

Audit SEO sau deploy: 18/18 trang đạt 10/10 kiểm tra; robots.txt và sitemap.xml trả 200. Lưu kết quả tại work/performance/production-seo.json.
