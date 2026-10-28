# Kiểm tra trực tuyến JSON-LD — 23/10/2026

Dùng giao diện **nhập mã** của Google Rich Results Test và Schema.org Validator. Các file HTML tại đây chứa JSON-LD trích nguyên từ HTML SSR của ứng dụng local cùng mã nguồn website, bọc trong trang HTML tối thiểu. Không gửi dữ liệu quản trị/đơn hàng, không mở quyền truy cập website riêng tư. Đây là kiểm tra markup, không phải crawl URL hoặc kiểm chứng khả năng tải ảnh hosted.

| Mẫu | Schema.org Validator | Google Rich Results Test |
|---|---|---|
| Trang chủ | 0 lỗi, 0 cảnh báo (WebSite chứa Organization/Brand) | 1 Organization hợp lệ; 4 cảnh báo địa chỉ tùy chọn |
| Cốm tươi lá sen | 0 lỗi, 0 cảnh báo (Product/BreadcrumbList) | 3 mục hợp lệ: product snippet, merchant listing, breadcrumb |
| Báo giá trước sửa | 0 lỗi, 0 cảnh báo | Product không hợp lệ: phải có offers, review hoặc aggregateRating; breadcrumb hợp lệ |
| Báo giá sau sửa | 0 lỗi, 0 cảnh báo (Brand/BreadcrumbList) | 1 breadcrumb hợp lệ, không còn mục không hợp lệ |

## Sửa lỗi

Chỉ phát Product khi sản phẩm có giá bán trực tiếp > 0 và cửa hàng nhận đơn. Sản phẩm cần tư vấn/báo giá hoặc cửa hàng ngừng nhận đơn giữ Brand và breadcrumb. Không thêm giá hay đánh giá giả chỉ để qua validator. Dữ liệu giao diện và tính năng báo giá giữ nguyên.

## Cảnh báo tùy chọn còn lại

- Product snippet: thiếu `review`, `aggregateRating` do chưa có đánh giá thật.
- Merchant listing: thiếu `shippingDetails`, `hasMerchantReturnPolicy`. Chưa có cấu hình chính sách có cấu trúc đầy đủ; phí hiện phụ thuộc vùng và tổng giỏ, không khai một mức phí cố định sai cho mọi đơn.
- Organization: thiếu `streetAddress`, `addressLocality`, `addressCountry`, `postalCode` trong địa chỉ mẫu. Quản trị hiện lưu địa chỉ tự do; không suy đoán địa chỉ đường hay mã bưu chính.

Các cảnh báo trên được công cụ đánh dấu **không bắt buộc/không nghiêm trọng**, không làm những mục này mất trạng thái hợp lệ. Bổ sung khi có dữ liệu thực và cấu hình tương ứng.

## Kết quả Google

- [Sản phẩm bán trực tiếp](https://search.google.com/test/rich-results/result?id=__3u57CBHGny6KhWg5GwnA)
- [Thương hiệu / tổ chức](https://search.google.com/test/rich-results/result?id=Gc_N4YetXDL99iWO52jGEg)
- [Báo giá trước sửa](https://search.google.com/test/rich-results/result?id=RGM4Ec-F9bjJGiUb0BrjtA)
- [Báo giá sau sửa](https://search.google.com/test/rich-results/result?id=y9xBUxlogLbOF5v-KLo5_Q)

Các liên kết kết quả có thể hết hạn. Ảnh PNG và HTML đầu vào được lưu kèm để đối chiếu. 13 unit tests, TypeScript và production build đều pass sau sửa.
