package shop

import (
	"strings"
	"time"
)

func (a *App) adminAction(s State, path string, b Object, user string) (any, error) {
	owner := []string{"review-moderate", "brand", "brand-publish", "brand-restore", "shipping", "coupon", "admins", "restore", "notifications", "zalo-connect", "zalo-disconnect", "notification-retry"}
	roles := []string{"owner", "editor"}
	if contains(owner, path) {
		roles = []string{"owner"}
	}
	if path == "order" || path == "inquiry" {
		roles = []string{"owner", "staff"}
	}
	if e := requireRole(s, user, roles...); e != nil {
		return nil, e
	}
	switch path {
	case "product", "collection", "campaign", "article", "coupon":
		v, e := validate(path, b)
		if e != nil {
			return nil, e
		}
		table := map[string]string{"product": "products", "collection": "collections", "campaign": "campaigns", "article": "articles", "coupon": "coupons"}[path]
		field := "id"
		if path == "coupon" {
			field = "code"
		}
		old := find(s, table, field, str(v[field]))
		if path == "product" {
			for _, p := range list(s[table]) {
				if p["sku"] == v["sku"] && p["id"] != v["id"] {
					return nil, fail("SKU đã tồn tại.")
				}
			}
		}
		if path == "coupon" {
			v["used"] = num(old["used"])
		}
		if old == nil {
			appendRow(s, table, v)
		} else {
			for k := range old {
				delete(old, k)
			}
			for k, x := range v {
				old[k] = x
			}
		}
	case "brand":
		v, e := validate("brand", b)
		if e != nil {
			return nil, e
		}
		obj(s["brand"])["draft"] = v
	case "brand-publish":
		draft := obj(s["brand"])["draft"]
		if draft == nil {
			return nil, fail("Chưa có bản nháp.")
		}
		v, e := validate("brand", draft)
		if e != nil {
			return nil, e
		}
		prepend(s, "brandHistory", clean(obj(s["brand"]), "draft"), 10)
		s["brand"] = v
	case "brand-restore":
		history := list(s["brandHistory"])
		if len(history) == 0 {
			return nil, fail("Chưa có phiên bản trước.")
		}
		previous := history[0]
		s["brandHistory"] = append([]Object{clean(obj(s["brand"]), "draft")}, history[1:]...)
		s["brand"] = previous
	case "shipping":
		v, e := validate("shipping", b)
		if e != nil {
			return nil, e
		}
		s["shipping"] = v
	case "restore":
		v, e := validate("backup", b)
		if e != nil {
			return nil, e
		}
		input := obj(v["state"])
		seen := map[string]bool{}
		skus := map[string]bool{}
		for _, p := range list(input["products"]) {
			if seen[str(p["id"])] || skus[str(p["sku"])] {
				return nil, fail("Sản phẩm trùng mã.")
			}
			seen[str(p["id"])] = true
			skus[str(p["sku"])] = true
			if e = refine("product", p); e != nil {
				return nil, e
			}
		}
		if e = refine("brand", obj(input["brand"])); e != nil {
			return nil, e
		}
		if e = refine("shipping", obj(input["shipping"])); e != nil {
			return nil, e
		}
		prepend(s, "brandHistory", clean(obj(s["brand"]), "draft"), 10)
		for _, key := range []string{"brand", "products", "collections", "campaigns", "articles", "shipping"} {
			s[key] = input[key]
		}
	case "order":
		if role(s, user) != "owner" {
			o := find(s, "orders", "id", str(b["id"]))
			if o == nil || o["paymentStatus"] != b["paymentStatus"] {
				return nil, fail("Chỉ chủ cửa hàng được đối soát thanh toán.")
			}
		}
		if e := changeOrder(s, b); e != nil {
			return nil, e
		}
	case "inquiry":
		o := find(s, "inquiries", "id", str(b["id"]))
		if o == nil || !contains([]string{"new", "processing", "closed"}, str(b["status"])) {
			return nil, fail("Yêu cầu không hợp lệ.")
		}
		o["status"] = b["status"]
		o["assignee"] = truncate(str(b["assignee"]), 100)
	case "review-moderate":
		r := find(s, "reviews", "id", str(b["id"]))
		status := str(b["status"])
		reason := strings.TrimSpace(str(b["reason"]))
		if r == nil || !contains([]string{"approved", "rejected", "hidden"}, status) || len([]rune(reason)) > 500 || (status != "approved" && reason == "") {
			return nil, fail("Kiểm tra đánh giá, trạng thái và lý do duyệt.")
		}
		r["status"] = status
		r["reason"] = reason
		r["moderatedAt"] = nowISO()
		r["moderatedBy"] = user
	case "notifications":
		if _, ok := b["enabled"].(bool); !ok || !match(`^0[0-9]{9}$`, str(b["phone"])) {
			return nil, fail("Nhập số Zalo Việt Nam 10 số.")
		}
		config := obj(s["notifications"])
		config["enabled"] = b["enabled"]
		config["phone"] = b["phone"]
		if str(config["enabledAt"]) == "" && yes(b["enabled"]) {
			config["enabledAt"] = nowISO()
		}
		s["notifications"] = config
	case "zalo-connect", "zalo-disconnect":
		config := obj(s["notifications"])
		config["requestId"] = id()
		config["command"] = strings.TrimPrefix(path, "zalo-")
		if path == "zalo-disconnect" {
			config["enabled"] = false
		}
		s["notifications"] = config
		s["zaloStatus"] = Object{"state": "pending", "updatedAt": nowISO()}
	case "admins":
		return nil, fail("VPS hiện chỉ hỗ trợ tài khoản chủ cửa hàng.")
	default:
		return nil, fail("Thao tác không tồn tại.")
	}
	audit(s, user, path)
	return Object{"ok": true}, nil
}
func inquiry(s State, b Object) (any, error) {
	v, e := validate("inquiry", b)
	if e != nil {
		return nil, e
	}
	if str(v["type"]) == "quote" && !yes(find(s, "products", "id", str(v["productId"]))["active"]) {
		return nil, fail("Sản phẩm không tồn tại.")
	}
	if str(v["type"]) == "after_sales" {
		o := find(s, "orders", "number", strings.ToUpper(strings.TrimSpace(str(v["orderId"]))))
		if o == nil || obj(o["customer"])["phone"] != v["phone"] {
			return nil, fail("Mã đơn và điện thoại không khớp.")
		}
	}
	delete(v, "website")
	v["id"] = id()
	v["createdAt"] = nowISO()
	v["status"] = "new"
	v["assignee"] = ""
	prepend(s, "inquiries", v, 0)
	return Object{"id": v["id"]}, nil
}
func submitReview(s State, b Object) (any, error) {
	author, body := strings.TrimSpace(str(b["author"])), strings.TrimSpace(str(b["body"]))
	rating := num(b["rating"])
	if len([]rune(author)) < 2 || len([]rune(author)) > 60 || len([]rune(body)) < 10 || len([]rune(body)) > 2000 || rating < 1 || rating > 5 || rating != float64(int(rating)) || !match(`^0[0-9]{9}$`, str(b["phone"])) {
		return nil, fail("Kiểm tra tên, số điện thoại, số sao và nội dung đánh giá.")
	}
	o := find(s, "orders", "number", strings.ToUpper(strings.TrimSpace(str(b["number"]))))
	if o == nil || obj(o["customer"])["phone"] != b["phone"] {
		return nil, fail("Mã đơn và điện thoại không khớp.")
	}
	if str(o["status"]) != "completed" || str(o["paymentStatus"]) != "paid" {
		return nil, fail("Chỉ đánh giá đơn đã hoàn tất và thanh toán.")
	}
	found := false
	for _, i := range list(o["items"]) {
		if i["id"] == b["productId"] {
			found = true
		}
	}
	if !found || !yes(find(s, "products", "id", str(b["productId"]))["active"]) {
		return nil, fail("Sản phẩm không thuộc đơn hàng hoặc đã ngừng hiển thị.")
	}
	for _, rv := range list(s["reviews"]) {
		if rv["orderId"] == o["id"] && rv["productId"] == b["productId"] {
			return nil, fail("Bạn đã đánh giá sản phẩm này trong đơn hàng.")
		}
	}
	review := Object{"id": id(), "orderId": o["id"], "productId": b["productId"], "author": author, "rating": rating, "body": body, "status": "pending", "createdAt": time.Now().UTC().Format(time.RFC3339Nano)}
	prepend(s, "reviews", review, 0)
	return Object{"id": review["id"], "status": "pending"}, nil
}
