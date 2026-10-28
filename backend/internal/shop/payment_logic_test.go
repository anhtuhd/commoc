package shop

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"
)

func TestSignatureRawBytesAndTimestamp(t *testing.T) {
	now := time.Now()
	ts := fmt.Sprint(now.Unix())
	raw := []byte(`{"id":1}`)
	secret := "01234567890123456789012345678901"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "."))
	mac.Write(raw)
	sig := fmt.Sprintf("sha256=%x", mac.Sum(nil))
	if !verifySignature(raw, ts, sig, secret, now) {
		t.Fatal("valid signature rejected")
	}
	if verifySignature(append(raw, ' '), ts, sig, secret, now) || verifySignature(raw, ts, sig, secret, now.Add(301*time.Second)) {
		t.Fatal("tampered/stale accepted")
	}
}
func TestMemoBoundaries(t *testing.T) {
	s := fixture()
	o, e := placeOrder(s, orderInput())
	if e != nil {
		t.Fatal(e)
	}
	code := str(o["paymentCode"])
	for _, memo := range []string{"X" + code, code + "X", code + " MVABCDEF123456"} {
		if matchingOrder(s, Object{"content": memo}) != nil {
			t.Fatalf("ambiguous code matched: %s", memo)
		}
	}
	if matchingOrder(s, Object{"content": "payment " + code}) == nil {
		t.Fatal("valid memo missed")
	}
}
