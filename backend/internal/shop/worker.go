package shop

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

func (a *App) bridge(ctx context.Context, path string, value any) (Object, error) {
	var payload io.Reader
	if value != nil {
		payload = bytes.NewReader(encoded(value))
	}
	method := "GET"
	if value != nil {
		method = "POST"
	}
	req, e := http.NewRequestWithContext(ctx, method, a.BridgeURL+path, payload)
	if e != nil {
		return nil, e
	}
	req.Header.Set("Authorization", "Bearer "+a.BridgeSecret)
	req.Header.Set("Content-Type", "application/json")
	client := http.Client{Timeout: 20 * time.Second}
	res, e := client.Do(req)
	if e != nil {
		return nil, e
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fail("Zalo bridge unavailable")
	}
	var b Object
	e = json.NewDecoder(io.LimitReader(res.Body, 300000)).Decode(&b)
	return b, e
}
func (a *App) Tick(ctx context.Context) error {
	_, e := a.DB.Update(ctx, true, func(s State, _ pgx.Tx) (any, error) { return nil, nil })
	if e != nil {
		return e
	}
	_, e = a.DB.Pool.Exec(ctx, "DELETE FROM rate_limits WHERE until_at<now(); DELETE FROM performance_samples WHERE at<now()-interval '30 days' OR id IN (SELECT id FROM performance_samples ORDER BY at DESC OFFSET 20000)")
	if e != nil {
		return e
	}
	s, e := a.DB.Read(ctx)
	if e != nil {
		return e
	}
	config := obj(s["notifications"])
	if a.BridgeURL == "" {
		return nil
	}
	command := str(config["command"])
	if command != "" {
		_, e = a.bridge(ctx, "/command", Object{"command": command, "requestId": config["requestId"]})
		if e != nil {
			return e
		}
		_, e = a.DB.Update(ctx, false, func(s State, _ pgx.Tx) (any, error) {
			c := obj(s["notifications"])
			if c["requestId"] == config["requestId"] {
				delete(c, "command")
			}
			return nil, nil
		})
		if e != nil {
			return e
		}
	}
	status, e := a.bridge(ctx, "/status", nil)
	if e != nil {
		return e
	}
	_, e = a.DB.Update(ctx, false, func(s State, _ pgx.Tx) (any, error) {
		if obj(s["notifications"])["requestId"] == config["requestId"] {
			s["zaloStatus"] = status
		}
		return nil, nil
	})
	if e != nil || !yes(config["enabled"]) || str(status["state"]) != "connected" {
		return e
	}
	for n := 0; n < 10; n++ {
		latest, e := a.DB.Read(ctx)
		if e != nil {
			return e
		}
		if !yes(obj(latest["notifications"])["enabled"]) || obj(latest["notifications"])["phone"] != config["phone"] {
			return nil
		}
		lease := id()
		var key, message string
		var attempts int
		e = a.DB.Pool.QueryRow(ctx, `UPDATE notification_outbox SET status='sending',lease=$1,lease_until=now()+interval '120 seconds',attempts=attempts+1 WHERE id=(SELECT id FROM notification_outbox WHERE attempts<6 AND retry_at<=now() AND (status IN ('pending','failed') OR (status='sending' AND lease_until<now())) ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1) RETURNING id,message,attempts`, lease).Scan(&key, &message, &attempts)
		if e == pgx.ErrNoRows {
			return nil
		}
		if e != nil {
			return e
		}
		if strings.HasPrefix(key, "order:") {
			latest, e := a.DB.Read(ctx)
			if e != nil {
				return e
			}
			o := find(latest, "orders", "id", strings.TrimPrefix(key, "order:"))
			if o == nil || str(o["status"]) == "cancelled" {
				if _, e = a.DB.Pool.Exec(ctx, "UPDATE notification_outbox SET status='cancelled',lease=NULL WHERE id=$1 AND lease=$2", key, lease); e != nil {
					return e
				}
				continue
			}
		}
		_, sendErr := a.bridge(ctx, "/send", Object{"phone": config["phone"], "message": message + "\nSự kiện: " + key, "eventId": key})
		if sendErr == nil {
			_, e = a.DB.Pool.Exec(ctx, "UPDATE notification_outbox SET status='sent',sent_at=now(),lease=NULL,last_error=NULL WHERE id=$1 AND lease=$2 AND status='sending'", key, lease)
		} else {
			delay := int(math.Min(3600, 60*math.Pow(2, float64(attempts-1))))
			_, e = a.DB.Pool.Exec(ctx, "UPDATE notification_outbox SET status='failed',lease=NULL,last_error='Gửi Zalo thất bại; kiểm tra phiên kết nối và người nhận.',retry_at=now()+make_interval(secs=>$3) WHERE id=$1 AND lease=$2 AND status='sending'", key, lease, delay)
		}
		if e != nil {
			return e
		}
	}
	return nil
}
