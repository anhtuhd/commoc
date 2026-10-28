package shop

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
)

//go:embed validation.json
var validationJSON []byte

//go:embed seed.json
var Seed []byte
var schemas map[string]Object

func init() {
	if e := json.Unmarshal(validationJSON, &schemas); e != nil {
		panic(e)
	}
}
func match(pattern, s string) bool {
	r, e := regexp.Compile(pattern)
	return e == nil && r.MatchString(s)
}
func validate(name string, b any) (Object, error) {
	v, e := validateValue(schemas[name], b)
	if e != nil {
		return nil, e
	}
	o := obj(v)
	if e = refine(name, o); e != nil {
		return nil, e
	}
	return o, nil
}
func validateValue(spec Object, value any) (any, error) {
	if value == nil {
		if d, ok := spec["default"]; ok {
			return d, nil
		}
		if yes(spec["optional"]) {
			return nil, nil
		}
		return nil, fail("Thiếu thông tin bắt buộc.")
	}
	bad := func() (any, error) {
		return nil, fail("Thông tin chưa hợp lệ. Vui lòng kiểm tra các trường.")
	}
	switch str(spec["type"]) {
	case "object":
		input, ok := value.(map[string]any)
		if !ok {
			return bad()
		}
		out := Object{}
		for k, s := range obj(spec["fields"]) {
			v, e := validateValue(obj(s), input[k])
			if e != nil {
				return nil, fmt.Errorf("%s: %w", k, e)
			}
			if v != nil {
				out[k] = v
			}
		}
		return out, nil
	case "string":
		v, ok := value.(string)
		if !ok {
			return bad()
		}
		for _, c := range list(spec["checks"]) {
			switch str(c["kind"]) {
			case "trim":
				v = strings.TrimSpace(v)
			case "min":
				if len([]rune(v)) < int(num(c["value"])) {
					return bad()
				}
			case "max":
				if len([]rune(v)) > int(num(c["value"])) {
					return bad()
				}
			case "regex":
				pattern := str(c["regex"])
				if strings.Contains(pattern, "(?!") {
					if !strings.HasPrefix(v, "/") || strings.HasPrefix(v, "//") {
						return bad()
					}
				} else if !match(pattern, v) {
					return bad()
				}
			case "email":
				a, e := mail.ParseAddress(v)
				if e != nil || a.Address != v {
					return bad()
				}
			case "url":
				u, e := url.Parse(v)
				if e != nil || u.Scheme == "" || u.Host == "" {
					return bad()
				}
			}
		}
		return v, nil
	case "number":
		_, ok := value.(float64)
		if !ok {
			return bad()
		}
		v := num(value)
		for _, c := range list(spec["checks"]) {
			switch str(c["kind"]) {
			case "int":
				if v != math.Trunc(v) {
					return bad()
				}
			case "min":
				if v < num(c["value"]) {
					return bad()
				}
			case "max":
				if v > num(c["value"]) {
					return bad()
				}
			}
		}
		return v, nil
	case "boolean":
		if _, ok := value.(bool); !ok {
			return bad()
		}
		return value, nil
	case "literal":
		if fmt.Sprint(value) != fmt.Sprint(spec["value"]) {
			return bad()
		}
		return value, nil
	case "enum":
		for _, v := range spec["values"].([]any) {
			if v == value {
				return value, nil
			}
		}
		return bad()
	case "array":
		a, ok := value.([]any)
		if !ok {
			return bad()
		}
		if max := num(spec["max"]); max > 0 && len(a) > int(max) {
			return bad()
		}
		out := []any{}
		for _, v := range a {
			x, e := validateValue(obj(spec["item"]), v)
			if e != nil {
				return nil, e
			}
			out = append(out, x)
		}
		return out, nil
	case "union":
		for _, option := range list(spec["options"]) {
			v, e := validateValue(option, value)
			if e == nil {
				return v, nil
			}
		}
		return bad()
	}
	return bad()
}
func refine(name string, b Object) error {
	assetKeys := []string{"logo", "favicon", "shareImage", "heroImage", "storyImage", "image", "mobileImage"}
	for _, k := range assetKeys {
		v := str(b[k])
		if v != "" && !(strings.HasPrefix(v, "/") && !strings.HasPrefix(v, "//")) && !strings.HasPrefix(v, "https://") {
			return fail("Ảnh phải dùng HTTPS hoặc đường dẫn nội bộ.")
		}
	}
	if name == "product" && str(b["mode"]) == "fixed" && num(b["price"]) <= 0 {
		return fail("Sản phẩm bán trực tiếp cần giá lớn hơn 0.")
	}
	if name == "campaign" && str(b["endsAt"]) < str(b["startsAt"]) {
		return fail("Ngày kết thúc phải sau ngày bắt đầu.")
	}
	if name == "brand" {
		if v := str(b["zalo"]); v != "" && !strings.HasPrefix(v, "https://zalo.me/") {
			return fail("Đường dẫn Zalo không hợp lệ.")
		}
		if v := str(b["facebook"]); v != "" && !strings.HasPrefix(v, "https://") {
			return fail("Facebook cần HTTPS.")
		}
		c := obj(b["business"])
		for _, k := range []string{"handlingMin", "handlingMax", "transitMin", "transitMax", "returnDays"} {
			if v := str(c[k]); v != "" && !match(`^[0-9]{1,3}$`, v) {
				return fail("Số ngày phải từ 0 đến 999.")
			}
		}
		if yes(c["addressConfirmed"]) {
			for _, k := range []string{"street", "locality", "region", "postalCode", "country"} {
				if str(c[k]) == "" {
					return fail("Cần đầy đủ địa chỉ đã xác nhận.")
				}
			}
		}
		if yes(c["shippingConfirmed"]) {
			for _, k := range []string{"handlingMin", "handlingMax", "transitMin", "transitMax"} {
				if str(c[k]) == "" {
					return fail("Cần đầy đủ thời gian giao hàng.")
				}
			}
			for _, p := range [][2]string{{"handlingMin", "handlingMax"}, {"transitMin", "transitMax"}} {
				var min, max int
				fmt.Sscan(str(c[p[0]]), &min)
				fmt.Sscan(str(c[p[1]]), &max)
				if min > max {
					return fail("Thời gian tối đa nhỏ hơn tối thiểu.")
				}
			}
		}
		if yes(c["returnsConfirmed"]) {
			category := str(c["returnCategory"])
			if category == "" || (category != "none" && (str(c["returnMethod"]) == "" || str(c["returnFees"]) == "")) {
				return fail("Cần đủ chính sách đổi trả.")
			}
			if category == "finite" {
				var n int
				fmt.Sscan(str(c["returnDays"]), &n)
				if n < 1 {
					return fail("Số ngày đổi trả phải lớn hơn 0.")
				}
			}
		}
	}
	if name == "shipping" {
		seen := map[string]bool{}
		slots := list(b["slots"])
		if yes(b["scheduleEnabled"]) && len(slots) == 0 {
			return fail("Cần ít nhất một khung giao hàng.")
		}
		for _, s := range slots {
			label := str(s["label"])
			if seen[label] {
				return fail("Tên khung giờ bị trùng.")
			}
			seen[label] = true
		}
	}
	return nil
}
