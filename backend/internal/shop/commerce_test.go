package shop

import (
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
