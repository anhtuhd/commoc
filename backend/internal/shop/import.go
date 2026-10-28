package shop

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"os"
	"time"
)

type Export struct {
	State        State    `json:"state"`
	Transactions []Object `json:"payment_transactions"`
	Outbox       []Object `json:"payment_outbox"`
	Metrics      []Object `json:"performance_samples"`
}

// Import is an offline command. Stop every writer and sender before replacing a migration snapshot.
func (a *App) ImportFile(ctx context.Context, path string) error {
	raw, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	var export Export
	if e = json.Unmarshal(raw, &export); e != nil {
		return e
	}
	if role(export.State, a.AdminUser) != "owner" {
		return fail("Configured administrator is not owner in source snapshot")
	}
	if export.State["brand"] == nil {
		return fail("Missing state")
	}
	_, e = a.DB.Update(ctx, false, func(s State, tx pgx.Tx) (any, error) {
		for _, table := range []string{"notification_outbox", "payment_transactions", "performance_samples", "order_items", "reviews", "orders", "products", "collections", "campaigns", "articles", "coupons", "inquiries", "settings"} {
			if _, e := tx.Exec(ctx, "DELETE FROM "+table); e != nil {
				return nil, e
			}
		}
		for k := range s {
			delete(s, k)
		}
		for k, v := range export.State {
			s[k] = v
		}
		if e = save(ctx, tx, State{}, s); e != nil {
			return nil, e
		}
		for _, row := range export.Transactions {
			payload := row["payload"]
			if text, ok := payload.(string); ok {
				if e = json.Unmarshal([]byte(text), &payload); e != nil {
					return nil, e
				}
			}
			if _, e = tx.Exec(ctx, `INSERT INTO payment_transactions(id,fingerprint,payload,status,reason,order_id,created_at,resolved_by,resolved_at,conflict) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, row["id"], fmt.Sprintf("%x", sha256.Sum256(encoded(payload))), encoded(payload), row["status"], row["reason"], row["order_id"], row["created_at"], row["resolved_by"], row["resolved_at"], int(num(row["conflict"]))); e != nil {
				return nil, e
			}
		}
		for _, row := range export.Outbox {
			status := str(row["status"])
			if status == "sending" {
				status = "failed"
			}
			retry := time.UnixMilli(int64(num(row["retry_at"])))
			if retry.IsZero() {
				retry = time.Now()
			}
			if _, e = tx.Exec(ctx, `INSERT INTO notification_outbox(id,transaction_id,kind,message,status,attempts,retry_at,created_at,sent_at,last_error) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, row["id"], row["transaction_id"], row["kind"], row["message"], status, int(num(row["attempts"])), retry, row["created_at"], row["sent_at"], row["last_error"]); e != nil {
				return nil, e
			}
		}
		for key, value := range obj(s["notificationLog"]) {
			log := obj(value)
			o := find(s, "orders", "id", key)
			if o == nil {
				continue
			}
			status := "failed"
			if yes(log["sent"]) {
				status = "sent"
			}
			if _, e = tx.Exec(ctx, `INSERT INTO notification_outbox(id,kind,message,status,attempts,retry_at,created_at) VALUES($1,'order.created',$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`, "order:"+key, orderMessage(o, a.Site), status, int(num(log["attempts"])), time.UnixMilli(int64(num(log["retryAt"]))), o["createdAt"]); e != nil {
				return nil, e
			}
		}
		config := obj(s["notifications"])
		if yes(config["enabled"]) && str(config["enabledAt"]) != "" {
			for _, o := range list(s["orders"]) {
				if str(o["createdAt"]) >= str(config["enabledAt"]) && str(o["status"]) != "cancelled" && obj(s["notificationLog"])[str(o["id"])] == nil {
					if e := a.enqueue(ctx, tx, "order:"+str(o["id"]), "", "order.created", orderMessage(o, a.Site)); e != nil {
						return nil, e
					}
				}
			}
		}
		for _, row := range export.Metrics {
			if _, e = tx.Exec(ctx, `INSERT INTO performance_samples(id,name,path,value,device,at) VALUES($1,$2,$3,$4,$5,$6)`, row["id"], row["name"], row["path"], num(row["value"]), row["device"], time.UnixMilli(int64(num(row["at"])))); e != nil {
				return nil, e
			}
		}
		return nil, nil
	})
	if e != nil {
		return fmt.Errorf("import failed: %w", e)
	}
	return nil
}
