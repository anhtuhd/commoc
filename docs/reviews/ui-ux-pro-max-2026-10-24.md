# Review UI/UX Mộc Vị — UI UX Pro Max — 24/10/2026

## Kết luận

Giao diện có nhận diện nhất quán, mobile đã đọc được, không thấy tràn ngang toàn trang trong các trạng thái đã kiểm tra. Vấn đề ưu tiên nằm ở hành vi mua hàng, thông báo trạng thái và khả năng quản trị an toàn, không cần đổi toàn bộ phong cách. Nên giữ xanh lá/kem, Playfair Display và Be Vietnam Pro, hệ icon Lucide; chuẩn hóa các token và trạng thái thay vì thêm hiệu ứng trang trí.

Skill đã cài tại `/Users/tubean/.codex/skills/ui-ux-pro-max`, nguồn chính thức: https://github.com/nextlevelbuilder/ui-ux-pro-max-skill (được https://uupm.cc/ liên kết). Có SKILL.md, data, references và scripts; `search.py` chạy được bằng Python 3, không cần dependency bổ sung. Dùng trực tiếp skill trong lượt review; có thể tự nhận diện trong lượt tiếp theo.

## Phạm vi và phương pháp

- Đọc checklist web `references/quick-reference.md`; dùng các truy vấn đã có kết quả phù hợp: `checkout form`, `error summary validation`, `keyboard focus modal`, `back behavior deep linking`, `live badge count screen reader`, và stack Next.js `navigation focus modal`. Không áp quy tắc native iOS/Android như một yêu cầu WCAG cho website.
- Truy vấn ban đầu `food ecommerce checkout` không có kết quả; đã đổi một lần sang `checkout form`, có kết quả về nhãn và phản hồi submit. Các gợi ý không phù hợp (ví dụ autoplay video trong kết quả tìm table) không được dùng làm phát hiện.
- Chrome profile riêng: production `https://pishare.cloud` cho trang công khai; quản trị/dữ liệu đơn thử trên QA `http://127.0.0.1:3308` chạy cùng production build của mã ứng dụng hiện tại. Không đăng nhập quản trị production, không tạo đơn/gửi review/liên hệ trên production.
- Rà soát 18 tuyến public gồm homepage, catalog, 2 chi tiết sản phẩm (thường/báo giá), collections, story, journal, 2 articles, contact, cart, checkout, lookup, 3 policies, success và login. Mobile 375×812; trang chi tiết sản phẩm đại diện ở 768/1024/1440px. 13 tab quản trị trên QA (gồm Tổng quan) ở 375px.
- Tái hiện filter URL, menu Escape, accessible name dialog, coupon một ký tự, draft chưa lưu và mở success trực tiếp. Có đọc mã cho trạng thái chưa kích hoạt hoặc không có dữ liệu; ghi rõ bên dưới. Không phải kiểm định WCAG đầy đủ hay nghiên cứu với người mua thực.

## Phát hiện ưu tiên P1 — sửa trước khi mở rộng quảng cáo

### UX-01 — Mã ưu đãi được áp dụng khi còn đang gõ, khóa đặt hàng

**Bằng chứng tái hiện QA:** thêm sản phẩm bằng Mua ngay, tại checkout nhập `M` vào Mã ưu đãi. Lỗi “Mã ưu đãi không hợp lệ, đã hết hạn hoặc chưa đạt giá trị tối thiểu.” xuất hiện, nút Xác nhận đặt hàng bị disabled. `Checkout` gọi quote mỗi lần coupon thay đổi và đặt quote=null khi lỗi.

**Ảnh hưởng:** người mua đang nhập bị báo sai quá sớm; mã không dùng được cũng ngăn mua với giá thường. Dễ làm bỏ thanh toán.

**Đề xuất:** tách couponDraft/appliedCoupon; nút Áp dụng/Bỏ mã; chỉ áp dụng khi người mua xác nhận. Khi mã lỗi giữ đơn giá gốc hợp lệ, lỗi đặt cạnh ô mã, không khóa đặt hàng vì coupon chưa áp dụng. Tổng tiền cần trạng thái đang cập nhật riêng.

**Nơi sửa:** `components/shop/checkout.tsx:5–8`.

### UX-02 — Thành công được hiển thị khi chưa có đơn

**Bằng chứng production:** mở trực tiếp `/dat-hang-thanh-cong` trong context sạch, vẫn có dấu check, dòng Cảm ơn và H1 “Đã nhận lời gửi thương.”; bên dưới lại đề nghị tra cứu. `OrderSuccess` luôn render phần thành công trước khi đọc session.

**Ảnh hưởng:** có thể khiến khách nghĩ đã đặt thành công dù không có đơn, kể cả khi storage trống hoặc bị xóa.

**Đề xuất:** có loading khi kiểm tra đơn; chỉ hiện xác nhận khi có dữ liệu đơn hợp lệ; còn lại hiện “Chưa có đơn hàng để hiển thị”, CTA Tra cứu hoặc quay về gian hàng. Tách trạng thái đơn đã được lưu khỏi đã xác nhận thanh toán.

**Nơi sửa:** `components/shop/checkout.tsx:9`.

### UX-03 — Lỗi biểu mẫu không chỉ rõ trường cần sửa

**Bằng chứng mã:** contact/checkout/review dùng native constraints và một vùng lỗi chung; lỗi Zod API được thay bằng thông báo tổng quát. Nhiều trường không có id/error id/aria-describedby/aria-invalid. Native required vẫn có tác dụng; không kết luận form hoàn toàn không có validation.

**Ảnh hưởng:** dữ liệu bị server từ chối khiến khách phải đoán trường sai, đặc biệt với điện thoại, địa chỉ, số lượng và điều kiện giao hàng.

**Đề xuất:** server trả field errors; hiển thị inline cạnh từng trường, liên kết aria-describedby, tổng hợp lỗi có focus sau submit. Giữ dữ liệu đã nhập và không focus cưỡng bức khi đang gõ.

**Nơi sửa:** `components/shop/checkout.tsx`, `content.tsx`, `reviews.tsx`, validation/API.

### UX-04 — Bản chỉnh sửa quản trị chưa có bảo vệ khi rời trang

**Bằng chứng QA:** đổi Tên thương hiệu thành “NHÁP QA CHƯA LƯU”, chuyển tab rồi quay lại: giá trị còn trong memory. Reload quản trị đưa về Tổng quan, quay lại thương hiệu thì mất thay đổi và trả về giá trị đã lưu. Không có dirty/beforeunload guard trong Admin/EditDialog. Không bấm Lưu/Xuất bản trong thử nghiệm này.

**Ảnh hưởng:** form thương hiệu hơn 50 trường dễ mất công chỉnh sửa khi reload, logout hoặc đóng dialog. Không nói chuyển tab luôn làm mất dữ liệu: brand đang giữ state trong parent.

**Đề xuất:** trạng thái Chưa lưu/Đã lưu, dirty guard cho đóng/rời trang, autosave draft có nhãn thời gian nếu phù hợp; xác nhận riêng trước Xuất bản/Khôi phục. Giữ rõ ranh giới bản nháp và dữ liệu công khai.

**Nơi sửa:** `components/shop/admin.tsx:12,16–19,27`.

### UX-05 — Một số thao tác quản trị chưa có pending/khóa thao tác

**Bằng chứng mã:** OrderEditor cập nhật đơn và InquiryEditor Lưu chỉ await save trong onClick, không có busy/disabled; khôi phục thương hiệu cũng chưa có bước xác nhận riêng. EditDialog, review moderation và shipping có busy — không gộp chúng vào lỗi này.

**Ảnh hưởng:** dễ bấm liên tiếp hoặc không biết đang lưu. Trạng thái đơn/thanh toán là thao tác có hậu quả vận hành; chưa khẳng định đã gây đơn trùng vì chưa thử ghi lặp.

**Đề xuất:** pending theo bản ghi, khóa nút, thông báo thành công/lỗi ở đúng bản ghi; confirm với hủy/hoàn tiền/khôi phục, chỉ trình bày chuyển trạng thái được phép.

**Nơi sửa:** `components/shop/admin.tsx:13–14`.

## Phát hiện P2 — điều hướng, accessibility và nội dung

### UX-06 — Filter và tab không có deep link

**Tái hiện production:** chọn danh mục Cốm & xôi: còn 3 thức quà nhưng URL vẫn `/san-pham`. `Catalog` chỉ đọc query khi mount, không ghi state q/category/price/sort/page vào URL; tab admin chỉ là state `tab`.

**Đề xuất:** query URL cho filter/sort/page và tab admin; Back/Forward/refresh giữ được ngữ cảnh; hiển thị chip bộ lọc + Xóa tất cả. Kiểm tra effect khi đổi URL bằng client navigation, không chỉ lúc mount.

### UX-07 — Menu mobile thiếu hoàn thiện keyboard và ngữ cảnh

**Tái hiện production:** mở menu rồi Escape vẫn mở, aria-expanded=true. Nút menu có aria-expanded nhưng không có aria-controls; nav link không có aria-current. Active CSS chỉ so khớp path bằng tuyệt đối nên trang chi tiết sản phẩm không đánh dấu Thức quà.

**Đề xuất:** đóng Escape/outside, aria-controls, nhãn Mở/Đóng tương ứng; active theo section và aria-current đúng ngữ cảnh. Giữ focus hợp lý khi đóng. Menu disclosure không bắt buộc trap focus như dialog modal.

**Nơi sửa:** `components/shop/chrome.tsx:7`.

### UX-08 — Dialog chưa có accessible name

**Tái hiện:** snapshot giỏ rỗng có `dialog modal` nhưng không có tên, dù bên trong có h2 Giỏ thức quà. CartDrawer/ EditDialog thiếu aria-labelledby/aria-label. Native dialog đã có modal/focus behavior; không kết luận không có focus trap.

**Đề xuất:** id cho heading và aria-labelledby, xác nhận focus được trả về trigger sau đóng, error có role alert trong editor, kiểm tra Escape và nút Hủy bằng keyboard.

**Nơi sửa:** `chrome.tsx:10`, `admin.tsx:12`.

### UX-09 — Nội dung readiness và niềm tin trước mua còn là mẫu

**Production:** sản phẩm vẫn ghi “Thông tin mẫu…” ở bảo quản/hạn dùng; footer ghi thương hiệu/nội dung mẫu; địa chỉ chung Hà Nội, chưa có hotline/email/địa chỉ thật; chính sách còn nội dung thử nghiệm.

**Đề xuất:** nhập dữ liệu thật từ quản trị trước khi chạy quảng cáo, mô tả thành phần/hạn dùng/bảo quản theo sản phẩm, kênh hỗ trợ, phạm vi và thời gian giao. Không tạo review/chứng nhận/địa chỉ giả. Với sản phẩm báo giá, bỏ diễn đạt stock và đơn vị lặp “Thiết kế theo nhu cầu / Theo yêu cầu” nếu không mang thông tin hữu ích.

### UX-10 — Tải lịch giao hàng thất bại giống trạng thái hết lịch

**Bằng chứng mã:** request delivery-options thất bại được catch→setOptions([]), UI hiển thị “Chưa có lịch còn chỗ” giống khi dữ liệu thật rỗng. Không tái hiện bằng lỗi mạng production; đây là code finding cho cấu hình scheduleEnabled.

**Đề xuất:** tách loading/error/empty, có Thử lại và liên hệ hỗ trợ. Không báo hết lịch khi chỉ mất mạng.

**Nơi sửa:** `components/shop/checkout.tsx:5,7–8`.

### UX-11 — Form thương hiệu quá dài; nhiều phần kỹ thuật chen với nội dung

**Tái hiện QA:** tab brand có 57 control (gồm upload) cùng các nhóm địa chỉ/chính sách, trên mobile phải cuộn rất dài. Chỉnh sửa chung dùng cùng tiêu đề Chỉnh sửa nội dung cho nhiều loại bản ghi; ảnh dùng chung label bao nhiều input cần kiểm tra lại naming.

**Đề xuất:** chia Nhận diện / Trang chủ / Liên hệ / Thanh toán / SEO / Chính sách; mục lục hoặc accordion, nút preview rõ, nhóm trường bắt buộc và nâng cao. Dialog có tên bản ghi/loại nội dung cụ thể. Đây là cải thiện hiệu quả, không phải lỗi layout nghiêm trọng.

### UX-12 — Dashboard hiệu năng khó đọc xu hướng

**Bằng chứng UI/mã:** có bảng p75, số mẫu và mức đánh giá; chưa có filter trang/thiết bị, biểu đồ thời gian hoặc so sánh kỳ trước. Cảnh báo dưới 20 mẫu dựa tổng toàn report, không phải từng nhóm metric.

**Đề xuất:** hiển thị sample sufficiency ở từng hàng; filter trang/device/time; card LCP/INP/CLS và xu hướng khi đủ mẫu, chú thích lab vs client metrics rõ. Chưa nên đánh giá tốt/xấu toàn website từ ít mẫu.

## P3 — polish có thể làm sau

- Body mobile nhiều khu vực 14–15px; 16px là mục tiêu đọc thoải mái của skill, không tự gọi mọi đoạn 14px là vi phạm WCAG. Hệ thống spacing/cỡ chữ đang có nhiều override; nên gom token để tránh sửa một trang ảnh hưởng trang khác.
- Review pending/hidden được owner quản lý nhưng người mua chỉ thấy đã gửi, chưa biết trạng thái duyệt. Có thể hiển thị trạng thái trong lookup mà không lộ lý do nội bộ hoặc người duyệt.
- Không cần thêm animation nhiều hoặc testimonial giả. Motion hiện có reduced-motion; giữ hiệu ứng ngắn và không che CTA.
- Trang đăng nhập cho phép password manager/autocomplete; có thể thêm Hiện mật khẩu và hướng dẫn lấy lại quyền truy cập nếu cần, không đưa credential vào giao diện công khai.

## Ma trận theo từng nhóm trang

| Trang/nhóm | Kết quả tốt | Điểm cần ưu tiên |
|---|---|---|
| Trang chủ | Nhận diện và CTA rõ, một H1/main, ảnh WebP responsive | Niềm tin/nội dung thật, LCP, chuyển đổi từ story sang mua |
| Catalog | Search, select có nhãn, empty state có CTA | UX-06; filter chips và URL; phản hồi số kết quả cho screen reader |
| Chi tiết sản phẩm | Giá/đơn vị/stock, bảo quản, related/review | UX-09; active nav, CTA mua và thêm giỏ cần phân cấp theo mục tiêu |
| Sản phẩm báo giá | Có tư vấn thay giá 0 | Câu chữ và stock generic; thông tin định lượng/thời hạn báo giá thật |
| Bộ sưu tập | Card dẫn sang catalog có query collection | Đồng bộ query khi client navigation, nhận biết filter đang áp dụng |
| Câu chuyện | Kiểu chữ/ảnh nhất quán | Nội dung thật/nguồn gốc, CTA cụ thể hơn khi có dữ liệu |
| Blog + 2 bài | Bài đọc được, có liên kết sản phẩm | Related products/articles và thông tin tác giả/ngày chỉ khi có dữ liệu |
| Liên hệ | Form có label, submit feedback, honeypot | UX-03; kênh hỗ trợ thật; lỗi field cụ thể |
| Giỏ hàng | Empty state rõ, mobile grid hợp lý | UX-08; phục hồi thao tác xóa nếu cần |
| Checkout | 16px input, tổng tiền/điều kiện, native validation | UX-01,03,10 |
| Success | Có lookup CTA | UX-02 |
| Lookup + review | Kiểm tra mã/phone, rule đơn đủ điều kiện, review chờ duyệt | UX-03; trạng thái moderation cho người mua |
| 3 policy | Tách trang/title đúng, đọc được | Nội dung thật và nhất quán với checkout |
| Login | Có label/autocomplete, không yêu cầu nhớ puzzle | Tên trang riêng, hiện mật khẩu/cơ chế khôi phục phù hợp |
| Admin overview/products/orders | Card/tab/table responsive, table cuộn trong region | UX-04,05,06,11 |
| Admin review/support/content/coupon | Trạng thái và form cơ bản có | Dirty guard, pending theo bản ghi, context dialog |
| Admin shipping/team/Zalo | Có cấu hình vận hành và hướng dẫn | Retry lỗi, phân nhóm nâng cao, confirm thao tác có hậu quả |
| Admin performance/SEO | Có số đo thực, p75, công cụ và export | UX-12 |

## Thứ tự triển khai đề nghị và tiêu chí nghiệm thu

1. **Luồng mua hàng:** UX-01/02/03/10. Mã sai không chặn mua giá thường; success không hiển thị sai; lỗi chỉ rõ field; mạng lỗi có Retry.
2. **Accessibility và điều hướng:** UX-06/07/08. Keyboard hoàn tất được luồng, Escape đúng hành vi, dialog có tên, refresh/back giữ filter.
3. **Quản trị an toàn:** UX-04/05/11. Không mất draft im lặng; nút pending không bấm lặp; confirm tác vụ phá hủy; preview/pub rõ.
4. **Hoàn thiện nội dung và đo lường:** UX-09/12 và typography token. Dữ liệu thật đủ để khách quyết định; metric có sample count theo nhóm, không hứa kết quả Google.

Kiểm tra sau sửa cần gồm mobile 320/375, tablet 768/1024, desktop 1440, keyboard và 200% zoom, các trạng thái loading/empty/error/offline. Hiện tại lượt review này không chạy mới Lighthouse; số đo gần nhất đã ghi trong `docs/validation/performance-seo-2026-10-24.md`: Performance trung vị 76 và LCP 4,49s, cần phân biệt với quan sát hiện tại.

## Giới hạn và trạng thái bàn giao

- Không có lỗi P0 được xác nhận trong lượt này. P1/P2 ở đây là mức ưu tiên UX, không phải đánh giá lỗ hổng bảo mật.
- Không kiểm thử screen reader thật, thiết bị iOS/Android vật lý hoặc toàn bộ cấu hình thương hiệu động; các tương tác ghi dữ liệu chỉ review trên QA hoặc đọc mã, không sửa dữ liệu production.
- Chưa kiểm thử ghi order/payment/campaign hay gửi Zalo thật; không đưa những phần đó thành kết luận end-to-end đã đạt.
- Chỉ cài skill và viết báo cáo; không đổi source UI/API, không deploy trong lượt review.

## Triển khai năm nhóm P1 — 24/10/2026

- UX-01: tách mã đang nhập và mã đã áp dụng; nút Áp dụng/Bỏ mã; không gọi quote theo từng ký tự. Mã bị từ chối tự tính lại không giảm giá, khách vẫn đặt được hàng. Chặn gửi khi quote chưa khớp giỏ/khu vực/mã mới.
- UX-02: trang xác nhận đọc thông tin phiên rồi xác minh mã đơn + điện thoại qua máy chủ. Không có dữ liệu hoặc xác minh thất bại không hiển thị dấu thành công. Có trạng thái đang kiểm tra và đường dẫn tra cứu.
- UX-03: thanh toán, liên hệ, tra cứu và đánh giá có lỗi cạnh trường, aria-invalid/aria-describedby, danh sách lỗi có thể focus và liên kết tới trường; API trả lỗi từng trường cho dữ liệu khách hàng và Zod. Không xóa nội dung khi gửi lỗi.
- UX-04: biểu mẫu thương hiệu và hộp chỉnh sửa có nhãn chưa lưu; cảnh báo tải lại/rời trang/đăng xuất và xác nhận bỏ nội dung hộp thoại. Bản nháp thương hiệu vẫn giữ khi chuyển tab quản trị. Cần xác nhận xuất bản/khôi phục; chỉ cập nhật mốc đã lưu sau thành công.
- UX-05: cập nhật đơn và yêu cầu tư vấn có khóa đồng bộ chống bấm liên tiếp, vô hiệu hóa trường/nút trong lúc lưu, nhãn tiến trình, kết quả thành công/lỗi. Xác nhận hủy đơn/hoàn tiền. Hộp chỉnh sửa và thao tác thương hiệu cũng chống gửi lặp.

### Bằng chứng kiểm tra

- 39 bài kiểm thử Node đạt; TypeScript và production build đạt. ESLint các module mới đạt; các component cũ vẫn có lỗi lint tồn tại trước (any, quy tắc effect), không coi toàn bộ lint đã đạt.
- Chrome mobile 375×812: sản phẩm → Mua ngay → biểu mẫu thiếu dữ liệu (focus summary, 3 trường lỗi) → nhập thông tin QA → đặt COD sau mã không hợp lệ → trang xác nhận đã xác minh, tổng 150.000₫. Không tạo đơn thử trên production.
- Gõ NOTREAL chưa áp dụng không phát sinh quote mới; Áp dụng gọi quote và fallback giá thường, nút đặt hàng hoạt động trở lại. MOCVI10 với 2 sản phẩm: subtotal 240.000₫, giảm 24.000₫, phí 30.000₫, tổng 246.000₫.
- Trang xác nhận phiên mới không có dữ liệu: “Chưa có đơn hàng để hiển thị”, không có dấu thành công. Liên hệ/tra cứu/đánh giá kiểm tra trên 320×740, không tràn ngang; đánh giá trống trả 3 lỗi có liên kết.
- Brand QA thay tên → hủy reload: dữ liệu còn; hộp sửa sản phẩm thay tên → hủy đóng: dữ liệu còn; xác nhận bỏ mới đóng. Thử bấm cập nhật đơn hai lần trong lúc giả lập chậm chỉ gửi một request, nút disabled và hiện “Đang cập nhật…”.
- Theo skill schema: graph của 5 route (trang chủ, sản phẩm, danh sách/bài viết, chính sách) khớp dữ liệu xuất bản; khác biệt review QA được tách khỏi so sánh production. Test business/reviews xác nhận chỉ xuất chính sách được xác thực và đánh giá đã duyệt.
- Schema.org Validator, URL https://pishare.cloud/san-pham/com-tuoi: 0 lỗi, 0 cảnh báo (Product và BreadcrumbList).
- Google Rich Results Test, nhập chính JSON-LD lấy từ production bằng chế độ CODE: dịch vụ trả “Something went wrong — Log in and try again”. Chưa xác nhận đạt bằng Google; không thêm chính sách hay đánh giá giả để lấp trường tùy chọn.

Các P2 trong báo cáo vẫn là backlog; lần này tập trung năm nhóm P1 đã được yêu cầu.
