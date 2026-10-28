package shop

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"io"
	"net/http"
	"strings"
	"time"
)

func (a *App) enqueue(ctx context.Context, tx pgx.Tx, key, transaction, kind, message string) error {
	_, e := tx.Exec(ctx, `INSERT INTO notification_outbox(id,transaction_id,kind,message) VALUES($1,$2,$3,$4) ON CONFLICT(id) DO NOTHING`, key, transaction, kind, message)
	return e
}
func (a *App) paymentMessage(o, t Object, key, kind, reason string) string {
	title := "CẦN ĐỐI SOÁT CHUYỂN KHOẢN"
	if kind == "payment.received" {
		title = "ĐÃ NHẬN THANH TOÁN"
	}
	return fmt.Sprintf("%s\nĐơn: %s\nĐã nhận: %.0f VND\nTổng đơn: %.0f VND\nThời gian: %s (VN)\nGiao dịch: %s\n%s\nQuản trị: %s/quan-tri", title, str(o["number"]), num(t["transferAmount"]), num(o["total"]), str(t["transactionDate"]), key, reason, a.Site)
}
func markPaid(o Object, key string) {
	o["paymentStatus"] = "paid"
	o["paidAt"] = nowISO()
	o["paymentSource"] = "sepay"
	o["sepayTransactionId"] = key
}
func (a *App) receive(ctx context.Context, t Object) (any, error) {
	return a.DB.Update(ctx, false, func(s State, tx pgx.Tx) (any, error) {
		key := fmt.Sprintf("%s:%.0f", a.Mode, num(t["id"]))
		fingerprint := fmt.Sprintf("%x", sha256.Sum256(encoded(t)))
		var oldFingerprint, status string
		e := tx.QueryRow(ctx, "SELECT fingerprint,status FROM payment_transactions WHERE id=$1", key).Scan(&oldFingerprint, &status)
		if e == nil {
			if oldFingerprint != fingerprint {
				if _, e = tx.Exec(ctx, "UPDATE payment_transactions SET conflict=1 WHERE id=$1", key); e != nil {
					return nil, e
				}
				status = "conflict"
			}
			return Object{"status": status, "duplicate": true}, nil
		}
		if e != pgx.ErrNoRows {
			return nil, e
		}
		status = "needs_review"
		reason := "Không có mã đơn duy nhất khớp nội dung giao dịch"
		o := matchingOrder(s, t)
		var orderID any
		if o != nil {
			orderID = o["id"]
		}
		if str(t["transferType"]) == "out" {
			status = "ignored"
			reason = "Giao dịch tiền ra"
		} else if o != nil {
			reason = eligibility(s, t, o, timeNow())
			if reason == "" {
				status = "matched"
				reason = "Khớp mã, tài khoản, thời hạn và tổng tiền"
				if !a.AutoMatch || !yes(obj(s["payments"])["enabled"]) {
					status = "needs_review"
					reason = "Tự động xác nhận đang tắt"
				}
			}
		}
		if _, e = tx.Exec(ctx, `INSERT INTO payment_transactions(id,fingerprint,payload,status,reason,order_id) VALUES($1,$2,$3,$4,$5,$6)`, key, fingerprint, encoded(t), status, reason, orderID); e != nil {
			return nil, e
		}
		if status == "matched" {
			markPaid(o, key)
		}
		if status != "ignored" {
			kind := "payment.needs_review"
			if status == "matched" {
				kind = "payment.received"
			}
			if e = a.enqueue(ctx, tx, key+":"+kind, key, kind, a.paymentMessage(o, t, key, kind, reason)); e != nil {
				return nil, e
			}
		}
		audit(s, "sepay", "Giao dịch "+key+": "+status)
		return Object{"status": status, "duplicate": false}, nil
	})
}
func (a *App) webhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	if len(a.SePaySecret) < 32 {
		http.Error(w, "Webhook not configured", 503)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 65536)
	raw, e := io.ReadAll(r.Body)
	if e != nil {
		http.Error(w, "Payload too large", 413)
		return
	}
	if !verifySignature(raw, r.Header.Get("X-SePay-Timestamp"), r.Header.Get("X-SePay-Signature"), a.SePaySecret, time.Now()) {
		http.Error(w, "Invalid signature", 401)
		return
	}
	var b Object
	if json.Unmarshal(raw, &b) != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}
	t, e := parseTransaction(b)
	if e != nil {
		http.Error(w, "Invalid payload", 400)
		return
	}
	if _, e = a.receive(r.Context(), t); e != nil {
		http.Error(w, "Storage unavailable", 503)
		return
	}
	jsonReply(w, Object{"success": true}, 200)
}
func rowsJSON(ctx context.Context, q queryer, sql string, args ...any) ([]Object, error) {
	rows, e := q.Query(ctx, sql, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	result := []Object{}
	for rows.Next() {
		var raw []byte
		if e = rows.Scan(&raw); e != nil {
			return nil, e
		}
		var o Object
		if e = json.Unmarshal(raw, &o); e != nil {
			return nil, e
		}
		result = append(result, o)
	}
	return result, rows.Err()
}
func (a *App) outboxRows(ctx context.Context, failed bool) ([]Object, error) {
	where := ""
	if failed {
		where = " WHERE status='failed'"
	}
	return rowsJSON(ctx, a.DB.Pool, "SELECT row_to_json(t) FROM (SELECT id,kind,status,attempts,last_error,sent_at,created_at,id AS number FROM notification_outbox"+where+" ORDER BY created_at DESC LIMIT 100) t")
}
func (a *App) payments(w http.ResponseWriter, r *http.Request) {
	s, e := a.DB.Read(r.Context())
	user := a.user(r)
	if e != nil || requireRole(s, user, "owner") != nil {
		http.Error(w, "Forbidden", 403)
		return
	}
	if r.Method == "GET" {
		transactions, e := rowsJSON(r.Context(), a.DB.Pool, "SELECT row_to_json(t) FROM (SELECT id,payload,status,reason,order_id,created_at,conflict FROM payment_transactions ORDER BY created_at DESC LIMIT 100) t")
		if e != nil {
			http.Error(w, "Unavailable", 503)
			return
		}
		for _, t := range transactions {
			p := obj(t["payload"])
			delete(t, "payload")
			for _, k := range []string{"code", "gateway", "transactionDate"} {
				t[k] = p[k]
			}
			t["amount"] = p["transferAmount"]
			t["content"] = truncate(str(p["content"]), 500)
			acc := str(p["accountNumber"])
			if len(acc) > 4 {
				acc = acc[len(acc)-4:]
			}
			t["account"] = "••••" + acc
		}
		outbox, e := a.outboxRows(r.Context(), false)
		if e != nil {
			http.Error(w, "Unavailable", 503)
			return
		}
		counts, e := rowsJSON(r.Context(), a.DB.Pool, "SELECT row_to_json(t) FROM (SELECT status,count(*) count FROM payment_transactions GROUP BY status) t")
		if e != nil {
			http.Error(w, "Unavailable", 503)
			return
		}
		config := obj(s["payments"])
		if len(config) == 0 {
			config = Object{"enabled": false, "accountNumber": "", "gateway": "", "bankBin": ""}
		}
		jsonReply(w, Object{"config": config, "secretConfigured": len(a.SePaySecret) >= 32, "mode": a.Mode, "autoMatch": a.AutoMatch, "transactions": transactions, "outbox": outbox, "counts": counts}, 200)
		return
	}
	if r.Method != "POST" || r.Header.Get("Origin") != a.Site {
		http.Error(w, "Forbidden", 403)
		return
	}
	b, e := body(w, r, 8192)
	if e != nil {
		errorReply(w, e)
		return
	}
	v, e := a.DB.Update(r.Context(), false, func(s State, tx pgx.Tx) (any, error) {
		if e := requireRole(s, user, "owner"); e != nil {
			return nil, e
		}
		switch str(b["action"]) {
		case "settings":
			c := obj(b["config"])
			if _, ok := c["enabled"].(bool); !ok || !match(`^[0-9]{4,30}$`, str(c["accountNumber"])) || !match(`^[A-Za-z0-9 ]{2,80}$`, strings.TrimSpace(str(c["gateway"]))) || !match(`^[0-9]{6}$`, str(c["bankBin"])) {
				return nil, fail("Kiểm tra số tài khoản, gateway và BIN 6 số.")
			}
			brand := obj(s["brand"])
			if yes(c["enabled"]) && (len(a.SePaySecret) < 32 || brand["bankAccount"] != c["accountNumber"] || brand["bankBin"] != c["bankBin"]) {
				return nil, fail("Cần secret và ngân hàng khớp thương hiệu đã xuất bản.")
			}
			s["payments"] = Object{"enabled": c["enabled"], "accountNumber": c["accountNumber"], "gateway": strings.TrimSpace(str(c["gateway"])), "bankBin": c["bankBin"]}
		case "retry":
			tag, e := tx.Exec(r.Context(), "UPDATE notification_outbox SET status='pending',attempts=0,retry_at=now(),lease=NULL WHERE id=$1 AND (status='failed' OR (status='sending' AND lease_until<now()))", str(b["id"]))
			if e != nil {
				return nil, e
			}
			if tag.RowsAffected() != 1 {
				return nil, fail("Không có thông báo lỗi để thử lại.")
			}
		case "resolve":
			if e := a.resolve(r.Context(), s, tx, b, user); e != nil {
				return nil, e
			}
		default:
			return nil, fail("Thao tác không hợp lệ.")
		}
		audit(s, user, "SePay: "+str(b["action"]))
		return Object{"ok": true}, nil
	})
	if e != nil {
		errorReply(w, e)
		return
	}
	jsonReply(w, v, 200)
}
func (a *App) resolve(ctx context.Context, s State, tx pgx.Tx, b Object, user string) error {
	key, reason := str(b["id"]), strings.TrimSpace(str(b["reason"]))
	if len([]rune(reason)) < 8 || len([]rune(reason)) > 500 {
		return fail("Nhập lý do đối soát ít nhất 8 ký tự.")
	}
	var raw []byte
	var status string
	var conflict int
	e := tx.QueryRow(ctx, "SELECT payload,status,conflict FROM payment_transactions WHERE id=$1", key).Scan(&raw, &status, &conflict)
	if e != nil || status != "needs_review" || conflict != 0 {
		return fail("Giao dịch không thể gắn vào đơn.")
	}
	var t Object
	if e = json.Unmarshal(raw, &t); e != nil {
		return e
	}
	o := find(s, "orders", "number", strings.ToUpper(strings.TrimSpace(str(b["number"]))))
	if o == nil || str(t["transferType"]) != "in" {
		return fail("Không tìm thấy đơn phù hợp.")
	}
	if reason := eligibility(s, t, o, timeNow()); reason != "" {
		return fail(reason)
	}
	markPaid(o, key)
	if _, e = tx.Exec(ctx, "UPDATE payment_transactions SET status='matched',reason=$2,order_id=$3,resolved_by=$4,resolved_at=now() WHERE id=$1", key, reason, o["id"], user); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE notification_outbox SET status='cancelled',lease=NULL WHERE transaction_id=$1 AND kind='payment.needs_review' AND status IN ('pending','failed')", key); e != nil {
		return e
	}
	return a.enqueue(ctx, tx, key+":payment.received", key, "payment.received", a.paymentMessage(o, t, key, "payment.received", "Đã đối soát thủ công: "+reason))
}
