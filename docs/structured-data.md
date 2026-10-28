# JSON-LD

JSON-LD được render trên máy chủ từ cùng `publicData(state)` dùng cho giao diện; không đọc bản nháp thương hiệu. Khi xuất bản/khôi phục thương hiệu hoặc sửa sản phẩm, request trang tiếp theo nhận dữ liệu mới, không cần build lại.

- Trang chủ: Brand, Organization, WebSite; tên, mô tả, logo, liên hệ và mạng xã hội từ quản trị. Chỉ thêm trường tùy chọn đã có dữ liệu.
- Chi tiết sản phẩm đang hiển thị: Product, Brand, BreadcrumbList. Offer dùng giá VND và tồn thực. Sản phẩm báo giá hoặc cửa hàng ngừng nhận đơn không phát khối Product; vẫn giữ Brand và BreadcrumbList. Google đánh dấu Product thiếu Offer/review/rating là không hợp lệ, nên không tạo khối này khi chưa có dữ liệu bán trực tiếp. Không tạo giá 0, đánh giá giả hay hạn giá giả.
- Breadcrumb hiển thị và JSON-LD dùng chung dữ liệu; tên sản phẩm/bài viết cập nhật theo quản trị. Trang nội bộ, quản trị, giỏ/checkout và URL không tồn tại không có schema.
- URL canonical/entity ID dùng domain Sites hiện tại tại `SITE_URL` trong `lib/shop/structured-data.ts`. Đổi thương hiệu không đổi domain; nếu thay domain cần cập nhật hằng này cùng metadata/sitemap.
- Serializer escape `<` và dấu phân cách Unicode để nội dung quản trị không đóng thẻ script.

Kiểm tra: 13 unit tests pass, TypeScript/build pass; parse JSON-LD từ HTML SSR trên trang chủ, sản phẩm trực tiếp, sản phẩm báo giá, bài viết và quản trị. Kiểm thử đổi thương hiệu, bỏ bản nháp, giá/tồn thay đổi, ẩn sản phẩm và chống script injection.

Đã kiểm tra bằng chế độ nhập mã trên Google Rich Results Test và Schema.org Validator. Xem [kết quả và bằng chứng](validation/README.md). Kiểm tra mã không xác nhận khả năng crawl URL/ảnh trên site riêng tư. Khi mở công khai, kiểm tra lại bằng URL và theo dõi Search Console; schema không bảo đảm rich result.

Tài liệu chính thức đã đối chiếu:
- https://developers.google.com/search/docs/appearance/structured-data/organization
- https://developers.google.com/search/docs/appearance/structured-data/product-snippet
- https://developers.google.com/search/docs/appearance/structured-data/breadcrumb

## Địa chỉ và chính sách có xác nhận (23/10/2026)

Trong **Quản trị → Thương hiệu → Địa chỉ & chính sách đã xác minh**, nhập địa chỉ chi tiết, thời gian chuẩn bị/vận chuyển, điều kiện và phí trả hàng. Mỗi nhóm có xác nhận riêng; mặc định trống và chưa xác nhận. Lưu nháp không tác động JSON-LD; Xuất bản mới áp dụng. Dữ liệu cũ và bản sao lưu v1 vẫn dùng được.

- Địa chỉ đầy đủ đã xác nhận tạo `PostalAddress`, đồng thời hiển thị tại liên hệ/chân trang; không còn dùng chuỗi địa chỉ mẫu làm schema.
- Phí giao hàng và ngưỡng miễn phí lấy từ cấu hình tính tiền hiện tại; giới hạn Hà Nội, Việt Nam. `OfferShippingDetails` thể hiện phí của một sản phẩm; trang giao hàng cung cấp `ShippingRateSettings.freeShippingThreshold` cho giỏ nhiều sản phẩm. Ngưỡng xét tổng hàng trước giảm giá, giống checkout.
- Chính sách trả hàng áp dụng toàn cửa hàng, xuất ở Organization và Offer; không chấp nhận trả hàng sẽ không xuất số ngày/cách trả/phí không liên quan. Không tự suy diễn hoàn tiền hay đánh giá khách hàng.
- Thông tin có cấu trúc được hiển thị trên trang chính sách. Chủ cửa hàng cần rà soát phần mô tả tự do để không mâu thuẫn.
- Server từ chối nhóm đã xác nhận nhưng thiếu dữ liệu, ngày âm/sai thứ tự. Số 0 được phân biệt với ô trống. Chưa xác nhận không xuất nhóm đó.

Kiểm tra: 17 unit tests, TypeScript, build; giao diện quản trị kiểm tra trường mới và lỗi thiếu địa chỉ. Fixture giả lập (không ghi vào dữ liệu cửa hàng) chạy chế độ nhập mã: [Google Rich Results](https://search.google.com/test/rich-results/result?id=NeSZi7Q8tRMOBvfjR31gxQ) có 6 mục hợp lệ; Schema.org 0 lỗi/0 cảnh báo. Google còn gợi ý `refundType` tùy chọn và các trường đánh giá sản phẩm; không bổ sung dữ liệu suy đoán. Ảnh: `validation/google-policies.png`, `validation/schema-policies.png`. Các kết quả này kiểm tra cấu trúc, không chứng nhận tính xác thực của dữ liệu hoặc khả năng lập chỉ mục website riêng tư.

## Hình thức hoàn tiền (refundType)

Quản trị → Thương hiệu → Đổi trả có lựa chọn **Hình thức hoàn tiền**: chưa xác định (mặc định), hoàn tiền toàn bộ, đổi sản phẩm cùng loại, hoặc tín dụng mua hàng tại cửa hàng. Trường mới tương thích dữ liệu/backup cũ, chỉ xuất khi đã chọn giá trị hợp lệ, chính sách đã xác nhận và đã xuất bản, và cửa hàng chấp nhận trả hàng. Nội dung tương ứng cũng hiển thị trên trang đổi trả. Không tự gán FullRefund cho dữ liệu cũ.

Kiểm tra 18 tests, TypeScript và build đạt. [Google Rich Results Test](https://search.google.com/test/rich-results/result?id=73pYDcN0xUlsQ1mDTftPVw) với `validation/refund-fixture.html` (dữ liệu giả lập, không lưu DB): 6 mục hợp lệ; Chính sách trả lại hàng không còn cảnh báo refundType. Product snippet vẫn có gợi ý đánh giá không bắt buộc. Bằng chứng `validation/google-refund.png`. Các lựa chọn tuân theo [tài liệu Google](https://developers.google.com/search/docs/appearance/structured-data/return-policy).
