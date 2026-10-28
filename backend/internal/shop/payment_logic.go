package shop

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func verifySignature(raw []byte, timestamp, signature, secret string, now time.Time) bool {
	if secret == "" || !match(`^[0-9]{10}$`, timestamp) || !match(`^sha256=[a-fA-F0-9]{64}$`, signature) {
		return false
	}
	n, e := strconv.ParseInt(timestamp, 10, 64)
	if e != nil || math.Abs(float64(now.Unix()-n)) > 300 {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "."))
	mac.Write(raw)
	supplied, _ := hex.DecodeString(strings.TrimPrefix(signature, "sha256="))
	return hmac.Equal(mac.Sum(nil), supplied)
}
func parseTransaction(b Object) (Object, error) {
	t := Object{}
	for _, key := range []string{"id", "transferAmount"} {
		v, ok := b[key].(float64)
		if !ok || v <= 0 || v > 9007199254740991 || math.Trunc(v) != v {
			return nil, fail("Invalid transaction number")
		}
		t[key] = v
	}
	for k, max := range map[string]int{"gateway": 80, "accountNumber": 50, "transactionDate": 19, "content": 4000, "subAccount": 100, "code": 100, "referenceCode": 200} {
		v, ok := b[k].(string)
		if b[k] == nil && contains([]string{"subAccount", "code", "referenceCode"}, k) {
			ok = true
			v = ""
		}
		if !ok || len(v) > max {
			return nil, fail("Invalid transaction field")
		}
		if contains([]string{"gateway", "accountNumber"}, k) && v == "" {
			return nil, fail("Missing transaction field")
		}
		t[k] = v
	}
	if !contains([]string{"in", "out"}, str(b["transferType"])) {
		return nil, fail("Invalid direction")
	}
	t["transferType"] = b["transferType"]
	date := str(t["transactionDate"])
	at, e := time.Parse("2006-01-02 15:04:05 -07:00", date+" +07:00")
	if e != nil || at.Format("2006-01-02 15:04:05") != date {
		return nil, fail("Invalid transaction date")
	}
	return t, nil
}

var memoTokens = regexp.MustCompile(`MV([A-Z0-9]{12}|-?[A-F0-9]{8})`)

func matchingOrder(s State, t Object) Object {
	codes := map[string]bool{}
	content := strings.ToUpper(str(t["content"]))
	isAlphaNum := func(b byte) bool { return b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' }
	for _, loc := range memoTokens.FindAllStringIndex(content, -1) {
		if loc[0] > 0 && isAlphaNum(content[loc[0]-1]) {
			continue
		}
		if loc[1] < len(content) && isAlphaNum(content[loc[1]]) {
			continue
		}
		codes[content[loc[0]:loc[1]]] = true
	}

	if len(codes) != 1 {
		return nil
	}
	code := ""
	for k := range codes {
		code = k
	}
	if explicit := strings.ToUpper(str(t["code"])); explicit != "" && explicit != code {
		return nil
	}
	var found Object
	for _, o := range list(s["orders"]) {
		p := str(o["paymentCode"])
		valid := p == code
		if p == "" {
			valid = str(o["number"]) == code || strings.ReplaceAll(str(o["number"]), "-", "") == code
		}
		if valid {
			if found != nil {
				return nil
			}
			found = o
		}
	}
	return found
}
func gateway(s string) string { return strings.ToUpper(strings.ReplaceAll(s, " ", "")) }
func eligibility(s State, t, o Object, now time.Time) string {
	c := obj(s["payments"])
	if str(c["accountNumber"]) == "" || str(t["accountNumber"]) != str(c["accountNumber"]) || gateway(str(t["gateway"])) != gateway(str(c["gateway"])) || str(t["subAccount"]) != "" {
		return "Tài khoản/ngân hàng nhận hoặc tài khoản ảo chưa được cấu hình"
	}
	bank := obj(o["brand"])
	if bank["bankAccount"] != t["accountNumber"] || bank["bankBin"] != c["bankBin"] {
		return "Tài khoản nhận không khớp thông tin ngân hàng của đơn"
	}
	if str(o["payment"]) != "bank" || str(o["paymentStatus"]) != "unpaid" {
		return "Đơn không chờ thanh toán chuyển khoản"
	}
	if !contains([]string{"new", "confirmed"}, str(o["status"])) || !parseTime(o["paymentExpiresAt"]).After(now) {
		return "Đơn đã hủy, hết hạn hoặc không còn chờ thanh toán"
	}
	at, _ := time.Parse("2006-01-02 15:04:05 -07:00", str(t["transactionDate"])+" +07:00")
	if at.Before(parseTime(o["createdAt"]).Truncate(time.Second)) || at.After(now.Add(5*time.Minute)) || at.After(parseTime(o["paymentExpiresAt"])) {
		return "Thời gian giao dịch không phù hợp với đơn"
	}
	if num(t["transferAmount"]) != num(o["total"]) {
		return "Số tiền nhận không bằng tổng đơn"
	}
	return ""
}
