package shop

import (
	"net/http/httptest"
	"testing"
)

func TestRateKeyIgnoresEphemeralPortAndUntrustedHeader(t *testing.T) {
	a := httptest.NewRequest("GET", "/", nil)
	b := httptest.NewRequest("GET", "/", nil)
	a.RemoteAddr = "203.0.113.10:1234"
	b.RemoteAddr = "203.0.113.10:9876"
	a.Header.Set("X-Commo-Client-IP", "1.2.3.4")
	if clientKey(a) != clientKey(b) {
		t.Fatal("rate limit bypass")
	}
}
