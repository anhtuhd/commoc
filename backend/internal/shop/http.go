package shop

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type App struct {
	DB                                                                                                                     *Store
	Site, AdminUser, AdminEmail, PasswordHash, SessionSecret, SePaySecret, Mode, BridgeURL, BridgeSecret, MediaDir, WebURL string
	AutoMatch                                                                                                              bool
}

func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if e := a.DB.Pool.Ping(r.Context()); e != nil {
			http.Error(w, "unavailable", 503)
			return
		}
		jsonReply(w, Object{"ok": true, "backend": "go", "database": "postgresql"}, 200)
	})
	mux.HandleFunc("/api/auth/", a.auth)
	mux.HandleFunc("/api/store/", a.store)
	mux.HandleFunc("/api/admin/payments", a.payments)
	mux.HandleFunc("/api/webhooks/sepay", a.webhook)
	mux.HandleFunc("/api/diagnostics", a.diagnostics)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusWriter{ResponseWriter: w, status: 200}
		w = recorder
		requestID := id()
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Cache-Control", "no-store")
		defer func() {
			if v := recover(); v != nil {
				slog.Error("request_panic", "requestId", requestID)
				http.Error(w, "Internal error", 500)
			}
			slog.Info("http_request", "requestId", requestID, "method", r.Method, "path", r.URL.Path, "durationMs", time.Since(start).Milliseconds(), "status", recorder.status)
		}()
		mux.ServeHTTP(w, r)
	})
}
func jsonReply(w http.ResponseWriter, v any, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func errorReply(w http.ResponseWriter, e error) {
	var pgerr *pgconn.PgError
	if errors.As(e, &pgerr) || errors.Is(e, context.DeadlineExceeded) {
		jsonReply(w, Object{"error": "Không thể lưu dữ liệu. Vui lòng thử lại."}, 503)
		return
	}
	jsonReply(w, Object{"error": e.Error()}, 400)
}
func body(w http.ResponseWriter, r *http.Request, limit int64) (Object, error) {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	raw, e := io.ReadAll(r.Body)
	if e != nil {
		return nil, e
	}
	var b Object
	e = json.Unmarshal(raw, &b)
	if b == nil && e == nil {
		e = fail("Thông tin không hợp lệ.")
	}
	return b, e
}
func publicData(s State) Object {
	v := Object{"brand": clean(obj(s["brand"]), "draft"), "shipping": s["shipping"], "collections": list(s["collections"])}
	for _, key := range []string{"products", "campaigns", "articles", "reviews"} {
		rows := []Object{}
		for _, r := range list(s[key]) {
			show := yes(r["active"])
			switch key {
			case "articles":
				show = yes(r["published"])
			case "campaigns":
				start, _ := time.Parse(time.RFC3339, str(r["startsAt"])+"T00:00:00+07:00")
				end, _ := time.Parse(time.RFC3339, str(r["endsAt"])+"T23:59:59+07:00")
				show = show && !timeNow().Before(start) && !timeNow().After(end)
			case "reviews":
				show = str(r["status"]) == "approved" && yes(find(s, "products", "id", str(r["productId"]))["active"])
				r = clean(r, "orderId", "status", "reason", "moderatedAt", "moderatedBy")
			}
			if show {
				rows = append(rows, r)
			}
		}
		v[key] = rows
	}
	collections := list(v["collections"])
	sort.SliceStable(collections, func(i, j int) bool { return num(collections[i]["sortOrder"]) < num(collections[j]["sortOrder"]) })
	return v
}
func (a *App) store(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/store/")
	if strings.HasPrefix(path, "media/") {
		a.media(w, r, strings.TrimPrefix(path, "media/"))
		return
	}
	if r.Method == "GET" {
		a.storeGet(w, r, path)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	if r.Header.Get("Origin") != a.Site {
		http.Error(w, "Forbidden", 403)
		return
	}
	if path == "upload" {
		a.upload(w, r)
		return
	}
	b, e := body(w, r, 1000000)
	if e != nil {
		errorReply(w, fail("Dữ liệu quá lớn hoặc không hợp lệ."))
		return
	}
	if limit := map[string]int{"orders": 20, "inquiries": 8, "track-order": 15, "reviews": 8, "event": 100}[path]; limit > 0 {
		if e = a.rate(r.Context(), path+":"+clientKey(r), limit); e != nil {
			jsonReply(w, Object{"error": e.Error()}, 429)
			return
		}
	}
	user := a.user(r)
	v, e := a.DB.Update(r.Context(), true, func(s State, tx pgx.Tx) (any, error) {
		switch path {
		case "notification-retry":
			if e := requireRole(s, user, "owner"); e != nil {
				return nil, e
			}
			tag, e := tx.Exec(r.Context(), "UPDATE notification_outbox SET status='pending',attempts=0,retry_at=now(),lease=NULL WHERE id=$1 AND (status='failed' OR (status='sending' AND lease_until<now()))", str(b["id"]))
			if e != nil {
				return nil, e
			}
			if tag.RowsAffected() != 1 {
				return nil, fail("Không có thông báo lỗi để thử lại.")
			}
			audit(s, user, "Thử lại thông báo "+str(b["id"]))
			return Object{"ok": true}, nil
		case "quote":
			return quote(s, b)
		case "orders":
			existing := find(s, "orders", "key", str(b["key"]))
			o, e := placeOrder(s, b)
			if e != nil {
				return nil, e
			}
			if existing == nil {
				audit(s, "customer", "Tạo đơn "+str(o["number"]))
				appendEvent(s, "order_created", str(o["utm"]))
				if str(obj(s["notifications"])["enabledAt"]) != "" {
					if e = a.enqueue(r.Context(), tx, "order:"+str(o["id"]), "", "order.created", orderMessage(o, a.Site)); e != nil {
						return nil, e
					}
				}
			}
			return clean(o, "key", "fingerprint", "customer", "note"), nil
		case "track-order":
			o := find(s, "orders", "number", strings.ToUpper(strings.TrimSpace(str(b["number"]))))
			if o == nil || str(obj(o["customer"])["phone"]) != str(b["phone"]) {
				return nil, fail("Không tìm thấy đơn phù hợp.")
			}
			v := Object{}
			for _, k := range []string{"number", "status", "paymentStatus", "total", "createdAt", "items", "delivery", "paymentExpiresAt", "payment", "paymentCode", "paidAt", "brand"} {
				if x, ok := o[k]; ok {
					v[k] = x
				}
			}
			reviewed := []string{}
			for _, rv := range list(s["reviews"]) {
				if rv["orderId"] == o["id"] {
					reviewed = append(reviewed, str(rv["productId"]))
				}
			}
			v["reviewedProductIds"] = reviewed
			return v, nil
		case "event":
			name := str(b["name"])
			if !contains([]string{"view_product", "add_to_cart", "begin_checkout"}, name) {
				return nil, fail("Sự kiện không hợp lệ.")
			}
			appendEvent(s, name, truncate(str(b["source"]), 100))
			return Object{"ok": true}, nil
		case "inquiries":
			return inquiry(s, b)
		case "reviews":
			return submitReview(s, b)
		default:
			if user == "" {
				return nil, fail("Vui lòng đăng nhập.")
			}
			return a.adminAction(s, path, b, user)
		}
	})
	if e != nil {
		errorReply(w, e)
		return
	}
	status := 200
	if contains([]string{"orders", "inquiries", "reviews"}, path) {
		status = 201
	}
	jsonReply(w, v, status)
}
func appendEvent(s State, name, source string) {
	appendRow(s, "events", Object{"name": name, "source": source, "at": nowISO()})
	a := list(s["events"])
	if len(a) > 2000 {
		s["events"] = a[len(a)-2000:]
	}
}
func (a *App) storeGet(w http.ResponseWriter, r *http.Request, path string) {
	if path == "delivery-options" {
		v, e := a.DB.Update(r.Context(), true, func(s State, _ pgx.Tx) (any, error) { return deliveryOptions(s, timeNow()), nil })
		if e != nil {
			errorReply(w, e)
			return
		}
		jsonReply(w, v, 200)
		return
	}
	s, e := a.DB.Read(r.Context())
	if e != nil {
		http.Error(w, "Database unavailable", 503)
		return
	}
	user := a.user(r)
	if path == "public" {
		jsonReply(w, publicData(s), 200)
		return
	}
	if path == "session" {
		var u any
		var roleValue any
		if user != "" {
			u = Object{"id": user, "name": "Chủ cửa hàng", "email": a.AdminEmail}
			roleValue = role(s, user)
		}
		jsonReply(w, Object{"user": u, "role": roleValue, "setup": false}, 200)
		return
	}
	if user == "" {
		http.Error(w, "Unauthorized", 401)
		return
	}
	if e = requireRole(s, user, "owner", "staff", "editor"); e != nil {
		http.Error(w, "Forbidden", 403)
		return
	}
	switch path {
	case "admin":
		v := Object{}
		for _, k := range []string{"brand", "products", "collections", "campaigns", "articles", "coupons", "shipping", "orders", "inquiries", "reviews", "audit", "admins", "events"} {
			v[k] = s[k]
		}
		orders := []Object{}
		for _, o := range list(s["orders"]) {
			orders = append(orders, clean(o, "key", "fingerprint"))
		}
		sort.Slice(orders, func(i, j int) bool { return str(orders[i]["createdAt"]) > str(orders[j]["createdAt"]) })
		v["orders"] = orders
		if role(s, user) == "editor" {
			v["orders"] = []Object{}
			v["inquiries"] = []Object{}
		}
		if role(s, user) != "owner" {
			for _, k := range []string{"reviews", "audit", "admins", "events"} {
				v[k] = []Object{}
			}
		}
		jsonReply(w, v, 200)
	case "notifications":
		if role(s, user) != "owner" {
			http.Error(w, "Forbidden", 403)
			return
		}
		rows, e := a.outboxRows(r.Context(), true)
		if e != nil {
			http.Error(w, "Unavailable", 503)
			return
		}
		jsonReply(w, Object{"config": s["notifications"], "status": s["zaloStatus"], "failed": rows}, 200)
	case "backup":
		if role(s, user) != "owner" {
			http.Error(w, "Forbidden", 403)
			return
		}
		jsonReply(w, Object{"version": 1, "exportedAt": nowISO(), "state": s}, 200)
	default:
		http.NotFound(w, r)
	}
}
func (a *App) media(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != "GET" || !match(`^[a-f0-9-]+\.(png|jpg|webp)$`, key) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeFile(w, r, filepath.Join(a.MediaDir, key))
}
func (a *App) upload(w http.ResponseWriter, r *http.Request) {
	s, e := a.DB.Read(r.Context())
	if e != nil || requireRole(s, a.user(r), "owner", "editor") != nil {
		http.Error(w, "Forbidden", 403)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 5500000)
	if e = r.ParseMultipartForm(5500000); e != nil {
		errorReply(w, fail("Ảnh tối đa 5MB."))
		return
	}
	defer r.MultipartForm.RemoveAll()
	f, _, e := r.FormFile("file")
	if e != nil {
		errorReply(w, e)
		return
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, 5000001))
	if e != nil || len(b) > 5000000 {
		errorReply(w, fail("Ảnh tối đa 5MB."))
		return
	}
	ext := map[string]string{"image/png": "png", "image/jpeg": "jpg", "image/webp": "webp"}[http.DetectContentType(b)]
	if ext == "" {
		errorReply(w, fail("Chỉ nhận ảnh PNG, JPEG hoặc WebP."))
		return
	}
	key := id() + "." + ext
	if e = os.MkdirAll(a.MediaDir, 0700); e == nil {
		e = os.WriteFile(filepath.Join(a.MediaDir, key), b, 0600)
	}
	if e != nil {
		http.Error(w, "Upload failed", 503)
		return
	}
	jsonReply(w, Object{"url": "/api/store/media/" + key}, 200)
}
func (a *App) Initialize(ctx context.Context) error {
	_, e := a.DB.Update(ctx, false, func(s State, _ pgx.Tx) (any, error) {
		if s["brand"] != nil {
			return nil, nil
		}
		var seed State
		if e := json.Unmarshal(Seed, &seed); e != nil {
			return nil, e
		}
		for k, v := range seed {
			s[k] = v
		}
		s["admins"] = []Object{{"id": a.AdminUser, "email": a.AdminEmail, "role": "owner"}}
		s["notifications"] = Object{"enabled": false, "phone": ""}
		s["zaloStatus"] = Object{"state": "disconnected", "updatedAt": nowISO()}
		s["reviews"] = []Object{}
		return nil, nil
	})
	return e
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
