package shop

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
)

type Object = map[string]any
type State map[string]any

func obj(v any) Object {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return Object{}
}
func list(v any) []Object {
	r := []Object{}
	if a, ok := v.([]any); ok {
		for _, v := range a {
			r = append(r, obj(v))
		}
	}
	if a, ok := v.([]Object); ok {
		return a
	}
	return r
}
func str(v any) string { s, _ := v.(string); return s }
func num(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	}
	return 0
}
func yes(v any) bool            { b, _ := v.(bool); return b }
func nowISO() string            { return time.Now().UTC().Format("2006-01-02T15:04:05.000Z") }
func parseTime(v any) time.Time { t, _ := time.Parse(time.RFC3339Nano, str(v)); return t }
func id() string {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	s := hex.EncodeToString(b)
	return s[:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:]
}
func encoded(v any) []byte {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return b
}
func find(s State, key, field, value string) Object {
	for _, v := range list(s[key]) {
		if str(v[field]) == value {
			return v
		}
	}
	return nil
}
func contains(a []string, s string) bool {
	for _, v := range a {
		if v == s {
			return true
		}
	}
	return false
}
func fail(message string) error { return errors.New(message) }
func clean(o Object, fields ...string) Object {
	v := Object{}
	for k, x := range o {
		if !contains(fields, k) {
			v[k] = x
		}
	}
	return v
}
func appendRow(s State, key string, v Object) { s[key] = append(list(s[key]), v) }
func prepend(s State, key string, v Object, limit int) {
	a := append([]Object{v}, list(s[key])...)
	if limit > 0 && len(a) > limit {
		a = a[:limit]
	}
	s[key] = a
}
func audit(s State, actor, action string) {
	prepend(s, "audit", Object{"at": nowISO(), "actor": actor, "action": action}, 2000)
}
