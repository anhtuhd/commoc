package shop

import (
	"encoding/json"
	"testing"
)

func TestProductOptionsValidationAndBackupPreserveData(t *testing.T) {
	var s State
	if err := json.Unmarshal(Seed, &s); err != nil {
		t.Fatal(err)
	}
	p := list(s["products"])[0]
	p["optionGroups"] = []any{Object{"id": "size", "name": "Cỡ", "type": "single", "min": float64(1), "max": float64(1), "active": true, "choices": []any{Object{"id": "small", "name": "Nhỏ", "priceDelta": float64(-1000), "active": true}}}}
	p["noteEnabled"] = true
	validated, err := validate("product", p)
	if err != nil {
		t.Fatal(err)
	}
	if len(list(validated["optionGroups"])) != 1 || !yes(validated["noteEnabled"]) {
		t.Fatal("configuration discarded")
	}
	backup := Object{"version": float64(1), "state": Object{"brand": s["brand"], "products": s["products"], "collections": s["collections"], "campaigns": s["campaigns"], "articles": s["articles"], "shipping": s["shipping"]}}
	restored, err := validate("backup", backup)
	if err != nil {
		t.Fatal(err)
	}
	restoredProduct := list(obj(restored["state"])["products"])[0]
	if len(list(restoredProduct["optionGroups"])) != 1 {
		t.Fatal("backup dropped options")
	}
	group := list(p["optionGroups"])[0]
	group["min"] = float64(2)
	if _, err = validate("product", p); err == nil {
		t.Fatal("invalid minimum accepted")
	}
}
