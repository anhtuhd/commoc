package shop

import (
	"strings"
	"testing"
	"time"
)

func fixture() State {
	return State{"brand": Object{"name": "Test", "bankName": "MB", "bankAccount": "123456", "bankHolder": "TEST", "bankBin": "970422"}, "shipping": Object{"enabled": true, "fee": 30000, "freeFrom": 500000}, "products": []Object{{"id": "com", "name": "Cốm", "unit": "gói", "stock": 2, "price": 100000, "active": true, "mode": "fixed"}}, "orders": []Object{}, "coupons": []Object{}}
}
func orderInput() Object {
	return Object{"key": "unique-request-001", "payment": "bank", "items": []Object{{"id": "com", "quantity": 1}}, "customer": Object{"name": "Khách QA", "phone": "0900000000", "address": "Địa chỉ kiểm thử", "region": "hanoi"}}
}
func TestOrderIdempotencyAndExpiry(t *testing.T) {
	s := fixture()
	b := orderInput()
	o, e := placeOrder(s, b)
	if e != nil {
		t.Fatal(e)
	}
	again, e := placeOrder(s, b)
	if e != nil || o["id"] != again["id"] || num(list(s["products"])[0]["stock"]) != 1 {
		t.Fatal("duplicate reservation", e)
	}
	b["payment"] = "cod"
	if _, e = placeOrder(s, b); e == nil {
		t.Fatal("changed idempotency payload accepted")
	}
	o["paymentExpiresAt"] = time.Now().Add(-time.Hour).Format(time.RFC3339)
	expireOrders(s, time.Now())
	expireOrders(s, time.Now())
	if num(list(s["products"])[0]["stock"]) != 2 {
		t.Fatal("stock restored twice")
	}
}
func TestQuoteRejectsInvalidQuantity(t *testing.T) {
	for _, n := range []float64{0, -1, 1.5, 100} {
		s := fixture()
		b := Object{"region": "hanoi", "items": []Object{{"id": "com", "quantity": n}}}
		if _, e := quote(s, b); e == nil {
			t.Fatalf("accepted %v", n)
		}
	}
}

func TestQuoteProductOptions(t *testing.T) {
	t.Setenv("PRODUCT_OPTIONS_ENABLED", "1")
	s := fixture()
	p := list(s["products"])[0]
	p["optionGroups"] = []Object{{"id": "size", "name": "Cỡ", "type": "single", "min": 1, "max": 1, "active": true, "choices": []Object{{"id": "large", "name": "Lớn", "priceDelta": 10000, "active": true}}}}
	p["noteEnabled"] = true
	b := Object{"region": "hanoi", "items": []Object{{"id": "com", "quantity": 1, "optionIds": []any{"large"}, "note": "Ít ngọt"}, {"id": "com", "quantity": 1, "optionIds": []any{"large"}, "note": "Không đường"}}}
	q, e := quote(s, b)
	if e != nil {
		t.Fatal(e)
	}
	items := list(q["items"])
	if len(items) != 2 || num(q["subtotal"]) != 220000 || str(items[0]["note"]) != "Ít ngọt" || len(list(items[0]["options"])) != 1 {
		t.Fatalf("wrong snapshot or totals: %#v", q)
	}
	list(b["items"])[0]["optionIds"] = []any{"bogus"}
	if _, e = quote(s, b); e == nil {
		t.Fatal("unknown option accepted")
	}
	list(b["items"])[0]["optionIds"] = []any{}
	if _, e = quote(s, b); e == nil {
		t.Fatal("missing required option accepted")
	}
}

func TestOrderOptionsRequireCurrentQuoteAndRestoreStock(t *testing.T) {
	t.Setenv("PRODUCT_OPTIONS_ENABLED", "1")
	s := fixture()
	p := list(s["products"])[0]
	p["optionGroups"] = []Object{{"id": "size", "name": "Cỡ", "type": "single", "min": 1, "max": 1, "active": true, "choices": []Object{{"id": "large", "name": "Lớn", "priceDelta": 10000, "active": true}}}}
	b := orderInput()
	b["items"] = []Object{{"id": "com", "quantity": 1, "optionIds": []any{"large"}}}
	if _, e := placeOrder(s, b); e == nil {
		t.Fatal("missing quote revision accepted")
	}
	qb := clean(b)
	qb["region"] = "hanoi"
	q, e := quote(s, qb)
	if e != nil {
		t.Fatal(e)
	}
	b["quoteRevision"] = q["quoteRevision"]
	list(list(p["optionGroups"])[0]["choices"])[0]["priceDelta"] = 20000
	if _, e = placeOrder(s, b); e == nil {
		t.Fatal("stale price accepted")
	}
	q, e = quote(s, qb)
	if e != nil {
		t.Fatal(e)
	}
	b["quoteRevision"] = q["quoteRevision"]
	o, e := placeOrder(s, b)
	if e != nil {
		t.Fatal(e)
	}
	if num(o["total"]) != 150000 || num(p["stock"]) != 1 {
		t.Fatal("incorrect order", o)
	}
	list(list(p["optionGroups"])[0]["choices"])[0]["name"] = "Đổi tên"
	if str(list(list(o["items"])[0]["options"])[0]["name"]) != "Lớn" {
		t.Fatal("snapshot changed")
	}
	o["paymentExpiresAt"] = time.Now().Add(-time.Hour).Format(time.RFC3339)
	expireOrders(s, time.Now())
	if num(p["stock"]) != 2 {
		t.Fatal("stock not restored")
	}
}

func TestOptionOrdersFailClosedWhenDisabled(t *testing.T) {
	t.Setenv("PRODUCT_OPTIONS_ENABLED", "0")
	s := fixture()
	p := list(s["products"])[0]
	p["noteEnabled"] = true
	if _, e := quote(s, Object{"region": "hanoi", "items": []Object{{"id": "com", "quantity": 1}}}); e == nil {
		t.Fatal("configurable product ordered while feature disabled")
	}
	p["noteEnabled"] = false
	if _, e := quote(s, Object{"region": "hanoi", "items": []Object{{"id": "com", "quantity": 1, "optionIds": []any{}}}}); e != nil {
		t.Fatal("legacy product blocked", e)
	}
}

func TestOptionQuoteRejectsUnavailableDuplicateAndInvalidNotes(t *testing.T) {
	t.Setenv("PRODUCT_OPTIONS_ENABLED", "1")
	s := fixture()
	p := list(s["products"])[0]
	p["noteEnabled"] = true
	p["optionGroups"] = []Object{{"id": "t", "name": "Topping", "type": "multiple", "min": 0, "max": 1, "active": true, "choices": []Object{{"id": "a", "name": "A", "priceDelta": 1000, "active": true}, {"id": "b", "name": "B", "priceDelta": 2000, "active": false}}}}
	for _, item := range []Object{{"id": "com", "quantity": 1, "optionIds": []any{"a", "a"}}, {"id": "com", "quantity": 1, "optionIds": []any{"b"}}, {"id": "com", "quantity": 1, "note": strings.Repeat("x", 501)}, {"id": "com", "quantity": 1, "optionIds": []any{"unknown"}}} {
		if _, e := quote(s, Object{"region": "hanoi", "items": []Object{item}}); e == nil {
			t.Fatalf("accepted invalid item %#v", item)
		}
	}
	b := Object{"region": "hanoi", "items": []Object{{"id": "com", "quantity": 2, "optionIds": []any{"a"}}, {"id": "com", "quantity": 1}}}
	if _, e := quote(s, b); e == nil {
		t.Fatal("accepted combined quantity greater than stock")
	}
}
