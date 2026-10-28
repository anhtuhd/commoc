package shop

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/scrypt"
	"net"
	"net/http"
	"strings"
	"time"
)

const cookieName = "__Host-commo_session"

func (a *App) user(r *http.Request) string {
	c, e := r.Cookie(cookieName)
	if e != nil {
		return ""
	}
	parts := strings.Split(c.Value, ".")
	if len(parts) != 2 {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(a.SessionSecret))
	mac.Write([]byte(parts[0]))
	sig, e := base64.RawURLEncoding.DecodeString(parts[1])
	if e != nil || !hmac.Equal(mac.Sum(nil), sig) {
		return ""
	}
	raw, e := base64.RawURLEncoding.DecodeString(parts[0])
	if e != nil {
		return ""
	}
	var b Object
	if json.Unmarshal(raw, &b) != nil || num(b["expires"]) <= float64(time.Now().UnixMilli()) || str(b["user"]) != a.AdminUser {
		return ""
	}
	return str(b["user"])
}
func (a *App) session(user string) string {
	body := base64.RawURLEncoding.EncodeToString(encoded(Object{"user": user, "expires": time.Now().Add(8 * time.Hour).UnixMilli()}))
	mac := hmac.New(sha256.New, []byte(a.SessionSecret))
	mac.Write([]byte(body))
	return body + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func (a *App) password(user, password string) bool {
	parts := strings.Split(a.PasswordHash, ":")
	if len(parts) != 2 || len(password) > 256 {
		return false
	}
	hash, e := hex.DecodeString(parts[1])
	if e != nil || len(hash) != 64 {
		return false
	}
	actual, e := scrypt.Key([]byte(password), []byte(parts[0]), 16384, 8, 1, 64)
	return e == nil && subtle.ConstantTimeCompare(hash, actual) == 1 && user == a.AdminUser
}
func (a *App) rate(ctx context.Context, key string, limit int) error {
	var count int
	e := a.DB.Pool.QueryRow(ctx, `INSERT INTO rate_limits(key,count,until_at) VALUES($1,1,now()+interval '10 minutes') ON CONFLICT(key) DO UPDATE SET count=CASE WHEN rate_limits.until_at<now() THEN 1 ELSE rate_limits.count+1 END,until_at=CASE WHEN rate_limits.until_at<now() THEN now()+interval '10 minutes' ELSE rate_limits.until_at END RETURNING count`, key).Scan(&count)
	if e != nil {
		return e
	}
	if count > limit {
		return fail("Bạn đã gửi nhiều yêu cầu. Vui lòng thử lại sau ít phút.")
	}
	return nil
}
func clientKey(r *http.Request) string {
	// Direct API ports are private. Only a local/container presentation proxy may provide this header.
	host, _, e := net.SplitHostPort(r.RemoteAddr)
	if e != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	v := host
	if ip != nil && (ip.IsLoopback() || ip.IsPrivate()) {
		if forwarded := r.Header.Get("X-Commo-Client-IP"); net.ParseIP(forwarded) != nil {
			v = forwarded
		}
	}
	sum := sha256.Sum256([]byte(v))
	return fmt.Sprintf("%x", sum[:12])
}
func (a *App) auth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" || r.Header.Get("Origin") != a.Site {
		http.Error(w, "Forbidden", 403)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if strings.HasSuffix(r.URL.Path, "logout") {
		http.SetCookie(w, &http.Cookie{Name: cookieName, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: -1})
		http.Redirect(w, r, a.Site+"/", 303)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	e := a.rate(r.Context(), "login:"+clientKey(r), 10)
	if e != nil || r.ParseForm() != nil || !a.password(r.Form.Get("username"), r.Form.Get("password")) {
		http.Redirect(w, r, a.Site+"/dang-nhap?error=1", 303)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: a.session(a.AdminUser), Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: 28800})
	http.Redirect(w, r, a.Site+"/quan-tri", 303)
}
func role(s State, user string) string {
	if user == "" {
		return ""
	}
	for _, v := range list(s["admins"]) {
		if str(v["id"]) == user {
			return str(v["role"])
		}
	}
	return ""
}
func requireRole(s State, user string, roles ...string) error {
	if !contains(roles, role(s, user)) {
		return fail("Bạn không có quyền thực hiện thao tác này.")
	}
	return nil
}
