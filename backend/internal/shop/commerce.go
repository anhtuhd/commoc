package shop

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

var timeNow = time.Now

func quote(s State, b Object) (Object, error) {
	cfg := obj(s["shipping"])
	if !yes(cfg["enabled"]) || str(b["region"]) != "hanoi" {
		return nil, fail("Khu vực này cần liên hệ để báo phí giao hàng.")
	}
	input := list(b["items"])
	if len(input) == 0 || len(input) > 30 {
		return nil, fail("Giỏ hàng chưa hợp lệ.")
	}
	counts := map[string]int{}
	ids := []string{}
	for _, i := range input {
		n := num(i["quantity"])
		key := str(i["id"])
		if key == "" || n != math.Trunc(n) || n < 1 || n > 99 {
			return nil, fail("Số lượng chưa hợp lệ.")
		}
		if counts[key] == 0 {
			ids = append(ids, key)
		}
		counts[key] += int(n)
	}
	items := []Object{}
	subtotal := float64(0)
	for _, key := range ids {
		n := counts[key]
		p := find(s, "products", "id", key)
		if p == nil || !yes(p["active"]) || str(p["mode"]) == "quote" {
			return nil, fail("Sản phẩm cần báo giá hoặc không còn mở bán.")
		}
		if n > 99 || float64(n) > num(p["stock"]) {
			return nil, fail(str(p["name"]) + ": không đủ số lượng trong kho.")
		}
		items = append(items, Object{"id": key, "name": p["name"], "unit": p["unit"], "price": p["price"], "quantity": n})
		subtotal += float64(n) * num(p["price"])
	}
	shipping := num(cfg["fee"])
	if subtotal >= num(cfg["freeFrom"]) {
		shipping = 0
	}
	discount := float64(0)
	code := strings.ToUpper(strings.TrimSpace(str(b["coupon"])))
	if code != "" {
		c := find(s, "coupons", "code", code)
		end, _ := time.Parse(time.RFC3339, str(c["endsAt"])+"T23:59:59+07:00")
		if c == nil || !yes(c["active"]) || end.Before(timeNow()) || num(c["used"]) >= num(c["limit"]) || subtotal < num(c["minimum"]) {
			return nil, fail("Mã ưu đãi không hợp lệ, đã hết hạn hoặc chưa đạt giá trị tối thiểu.")
		}
		discount = math.Min(math.Floor(subtotal*num(c["percent"])/100), math.Min(num(c["maxDiscount"]), subtotal))
	}
	return Object{"items": items, "subtotal": subtotal, "shipping": shipping, "discount": discount, "total": subtotal + shipping - discount, "coupon": code}, nil
}
func deliveryOptions(s State, now time.Time) []Object {
	result := []Object{}
	c := obj(s["shipping"])
	if !yes(c["scheduleEnabled"]) {
		return result
	}
	zone := time.FixedZone("VN", 7*3600)
	today := now.In(zone).Format("2006-01-02")
	midnight, _ := time.ParseInLocation("2006-01-02", today, zone)
	days := int(num(c["maxDays"]))
	if days == 0 {
		days = 7
	}
	lead := num(c["leadHours"])
	if lead == 0 {
		lead = 4
	}
	for d := 0; d < days; d++ {
		date := midnight.AddDate(0, 0, d).Format("2006-01-02")
		for _, slot := range list(c["slots"]) {
			at, _ := time.Parse(time.RFC3339, date+"T"+str(slot["start"])+":00+07:00")
			if at.Before(now.Add(time.Duration(lead) * time.Hour)) {
				continue
			}
			used := 0
			for _, o := range list(s["orders"]) {
				v := obj(o["delivery"])
				if str(o["status"]) != "cancelled" && v["date"] == date && v["slot"] == slot["label"] {
					used++
				}
			}
			if float64(used) < num(slot["capacity"]) {
				result = append(result, Object{"date": date, "slot": slot["label"], "remaining": num(slot["capacity"]) - float64(used)})
			}
		}
	}
	return result
}
func placeOrder(s State, b Object) (Object, error) {
	c := obj(b["customer"])
	if len([]rune(strings.TrimSpace(str(c["name"])))) < 2 || len([]rune(str(c["name"]))) > 100 || !match(`^0[0-9]{9}$`, str(c["phone"])) || len([]rune(strings.TrimSpace(str(c["address"])))) < 8 || len([]rune(str(c["address"]))) > 300 || len([]rune(str(c["note"]))) > 1000 {
		return nil, fail("Vui lòng kiểm tra họ tên, số điện thoại và địa chỉ nhận hàng.")
	}
	payment := str(b["payment"])
	key := str(b["key"])
	if !contains([]string{"cod", "bank"}, payment) || len(key) < 16 || len(key) > 100 {
		return nil, fail("Phương thức thanh toán hoặc mã yêu cầu không hợp lệ.")
	}
	hash := sha256.Sum256(encoded(Object{"items": b["items"], "customer": c, "payment": payment, "coupon": str(b["coupon"]), "delivery": b["delivery"]}))
	fingerprint := hex.EncodeToString(hash[:])
	if old := find(s, "orders", "key", key); old != nil {
		oldFingerprint := str(old["fingerprint"])
		if strings.HasPrefix(oldFingerprint, "{") {
			var original Object
			if json.Unmarshal([]byte(oldFingerprint), &original) == nil {
				if _, ok := original["delivery"]; !ok {
					original["delivery"] = nil
				}
				oldSum := sha256.Sum256(encoded(original))
				oldFingerprint = hex.EncodeToString(oldSum[:])
			}
		}
		if oldFingerprint != fingerprint {
			return nil, fail("Yêu cầu đã thay đổi. Vui lòng tải lại trang thanh toán.")
		}
		return old, nil
	}
	brand := obj(s["brand"])
	if payment == "bank" && (str(brand["bankAccount"]) == "" || str(brand["bankName"]) == "" || str(brand["bankHolder"]) == "") {
		return nil, fail("Cửa hàng chưa bật chuyển khoản. Vui lòng chọn COD.")
	}
	var delivery any
	if yes(obj(s["shipping"])["scheduleEnabled"]) {
		for _, v := range deliveryOptions(s, timeNow()) {
			if v["date"] == obj(b["delivery"])["date"] && v["slot"] == obj(b["delivery"])["slot"] {
				delivery = Object{"date": v["date"], "slot": v["slot"]}
				break
			}
		}
		if delivery == nil {
			return nil, fail("Khung giao hàng đã đầy hoặc hết giờ đặt. Vui lòng chọn lại.")
		}
	}
	qb := clean(b)
	qb["region"] = c["region"]
	q, e := quote(s, qb)
	if e != nil {
		return nil, e
	}
	for _, i := range list(q["items"]) {
		p := find(s, "products", "id", str(i["id"]))
		p["stock"] = num(p["stock"]) - num(i["quantity"])
	}
	if code := str(q["coupon"]); code != "" {
		c := find(s, "coupons", "code", code)
		c["used"] = num(c["used"]) + 1
	}
	oid := id()
	q["id"] = oid
	q["number"] = "MV-" + strings.ToUpper(oid[:8])
	q["key"] = key
	q["fingerprint"] = fingerprint
	q["createdAt"] = nowISO()
	q["customer"] = Object{"name": strings.TrimSpace(str(c["name"])), "phone": c["phone"], "address": strings.TrimSpace(str(c["address"])), "region": c["region"], "note": str(c["note"])}
	q["payment"] = payment
	q["status"] = "new"
	q["paymentStatus"] = "unpaid"
	q["note"] = ""
	q["utm"] = truncate(str(b["utm"]), 200)
	q["brand"] = Object{}
	for _, k := range []string{"name", "phone", "bankName", "bankAccount", "bankHolder", "bankBin"} {
		if v, ok := brand[k]; ok {
			obj(q["brand"])[k] = v
		}
	}
	if delivery != nil {
		q["delivery"] = delivery
	}
	if payment == "bank" {
		code := "MV" + strings.ToUpper(strings.ReplaceAll(id(), "-", "")[:12])
		for find(s, "orders", "paymentCode", code) != nil {
			code = "MV" + strings.ToUpper(strings.ReplaceAll(id(), "-", "")[:12])
		}
		q["paymentCode"] = code
		minutes := num(obj(s["shipping"])["paymentMinutes"])
		if minutes == 0 {
			minutes = 60
		}
		q["paymentExpiresAt"] = timeNow().Add(time.Duration(minutes) * time.Minute).UTC().Format(time.RFC3339Nano)
	}
	prepend(s, "orders", q, 0)
	return q, nil
}

var transitions = map[string][]string{"new": {"confirmed", "cancelled"}, "confirmed": {"preparing", "cancelled"}, "preparing": {"shipping", "cancelled"}, "shipping": {"completed"}, "completed": {}, "cancelled": {}}

func changeOrder(s State, b Object) error {
	o := find(s, "orders", "id", str(b["id"]))
	if o == nil {
		return fail("Không tìm thấy đơn.")
	}
	status, payment := str(b["status"]), str(b["paymentStatus"])
	if status != str(o["status"]) && !contains(transitions[str(o["status"])], status) {
		return fail("Không thể chuyển trạng thái này.")
	}
	if !contains([]string{"unpaid", "paid", "refunded"}, payment) {
		return fail("Trạng thái thanh toán không hợp lệ.")
	}
	if str(o["sepayTransactionId"]) != "" && payment == "unpaid" && str(o["paymentStatus"]) != "unpaid" {
		return fail("Thanh toán SePay không thể đặt lại chưa thanh toán.")
	}
	if payment == "refunded" && !contains([]string{"paid", "refunded"}, str(o["paymentStatus"])) {
		return fail("Chỉ hoàn tiền đơn đã thanh toán.")
	}
	if str(o["payment"]) == "bank" && payment == "unpaid" && contains([]string{"preparing", "shipping", "completed"}, status) {
		return fail("Đối soát thanh toán trước khi chuẩn bị đơn chuyển khoản.")
	}
	if str(o["status"]) == "cancelled" && payment != str(o["paymentStatus"]) {
		return fail("Đơn đã hủy: cần đối soát riêng.")
	}
	if status == "cancelled" && str(o["status"]) != "cancelled" {
		for _, i := range list(o["items"]) {
			p := find(s, "products", "id", str(i["id"]))
			if p != nil {
				p["stock"] = num(p["stock"]) + num(i["quantity"])
			}
		}
		if c := find(s, "coupons", "code", str(o["coupon"])); c != nil {
			c["used"] = math.Max(0, num(c["used"])-1)
		}
	}
	o["status"] = status
	o["paymentStatus"] = payment
	o["note"] = truncate(str(b["note"]), 1000)
	return nil
}
func expireOrders(s State, now time.Time) {
	for _, o := range list(s["orders"]) {
		if str(o["payment"]) == "bank" && str(o["paymentStatus"]) == "unpaid" && contains([]string{"new", "confirmed"}, str(o["status"])) && !parseTime(o["paymentExpiresAt"]).IsZero() && !parseTime(o["paymentExpiresAt"]).After(now) {
			_ = changeOrder(s, Object{"id": o["id"], "status": "cancelled", "paymentStatus": "unpaid", "note": "Tự hủy do quá hạn thanh toán. Nếu đã chuyển tiền, liên hệ cửa hàng để đối soát."})
		}
	}
}
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}
func orderMessage(o Object, site string) string {
	c := obj(o["customer"])
	lines := []string{fmt.Sprintf("Đơn mới %s · %s", str(o["number"]), str(obj(o["brand"])["name"])), str(c["name"]) + " · " + str(c["phone"]), str(c["address"])}
	for _, item := range list(o["items"]) {
		lines = append(lines, fmt.Sprintf("%.0f × %s", num(item["quantity"]), str(item["name"])))
	}
	lines = append(lines, fmt.Sprintf("Tổng: %.0f VND · %s", num(o["total"]), str(o["payment"])))
	if d := obj(o["delivery"]); len(d) > 0 {
		lines = append(lines, "Giao: "+str(d["date"])+" "+str(d["slot"]))
	}
	if str(c["note"]) != "" {
		lines = append(lines, "Ghi chú: "+str(c["note"]))
	}
	lines = append(lines, "Quản trị: "+site+"/quan-tri")
	return truncate(strings.Join(lines, "\n"), 1900)
}
