# Checklist Go + PostgreSQL

Ngày 28/10/2026. Chỉ lập kế hoạch, chưa triển khai. Chi tiết phụ thuộc, file và nghiệm thu: [plan.md](plan.md).

- [x] Khảo sát mã nguồn local và lập kế hoạch.
- [ ] T01: Chốt hợp đồng API, ma trận tính năng và baseline.
- [ ] T02: Khung Go, PostgreSQL, migration và healthcheck.
- [ ] T03: Import nội dung, media và public API.
- [ ] T04: Auth, phân quyền và quản trị nội dung.
- [ ] T05: Đặt hàng, tồn kho, coupon và idempotency.
- [ ] T06: Lịch giao, hết hạn, liên hệ và review.
- [ ] T07: SePay, ledger, QR và đối soát.
- [ ] T08: Zalo bridge, outbox và bảo toàn lịch sử gửi.
- [ ] T09: Adapter UI hiện tại, diagnostics và SEO — Mốc A.
- [ ] T10: Storefront Go SSR và JSON-LD.
- [ ] T11: Quản trị Go SSR, bỏ Next runtime — Mốc B.
- [ ] T12: Rehearsal import/restore, E2E và benchmark.
- [ ] T13: Commit/push và deploy khi triển khai; nghiệm thu production.

Checkpoint A sau T03; B sau T08; Mốc A sau T09; Mốc B sau T11; gate cuối T12 trước production.

Lịch sử SePay: [checklist cũ](sepay-todo-2026-10-25.md).
