package shop

import (
	"context"
	"encoding/json"
	"github.com/anhtuhd/commoc/backend/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func testDB(t *testing.T) *App {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; use QA PostgreSQL")
	}
	ctx := context.Background()
	pool, e := pgxpool.New(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(pool.Close)
	if _, e = pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"+migrations.SQL); e != nil {
		t.Fatal(e)
	}
	a := &App{DB: &Store{Pool: pool}, Site: "http://localhost:3401", AdminUser: "qa-owner", Mode: "test", AutoMatch: true}
	if e = a.Initialize(ctx); e != nil {
		t.Fatal(e)
	}
	_, e = a.DB.Update(ctx, false, func(s State, _ pgx.Tx) (any, error) {
		for k, v := range fixture() {
			s[k] = v
		}
		s["payments"] = Object{"enabled": true, "accountNumber": "123456", "gateway": "MB", "bankBin": "970422"}
		return nil, nil
	})
	if e != nil {
		t.Fatal(e)
	}
	return a
}
func paymentFor(o Object, tid int) Object {
	return Object{"id": float64(tid), "gateway": "MB", "transactionDate": time.Now().In(time.FixedZone("VN", 25200)).Format("2006-01-02 15:04:05"), "accountNumber": "123456", "subAccount": "", "code": str(o["paymentCode"]), "content": "Thanh toan " + str(o["paymentCode"]), "transferType": "in", "transferAmount": o["total"], "referenceCode": ""}
}
func createTestOrder(t *testing.T, a *App) Object {
	t.Helper()
	v, e := a.DB.Update(context.Background(), true, func(s State, _ pgx.Tx) (any, error) { return placeOrder(s, orderInput()) })
	if e != nil {
		t.Fatal(e)
	}
	return obj(v)
}
func TestPostgresConcurrentCheckoutAndRollback(t *testing.T) {
	a := testDB(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	success := make(chan bool, 12)
	for n := 0; n < 12; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			b := orderInput()
			b["key"] = "concurrent-order-" + strings.Repeat("x", n+1)
			_, e := a.DB.Update(ctx, true, func(s State, _ pgx.Tx) (any, error) { return placeOrder(s, b) })
			success <- e == nil
		}(n)
	}
	wg.Wait()
	close(success)
	count := 0
	for ok := range success {
		if ok {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("oversold or lost orders: %d", count)
	}
	s, e := a.DB.Read(ctx)
	if e != nil || len(list(s["orders"])) != 2 || num(list(s["products"])[0]["stock"]) != 0 {
		t.Fatal("inconsistent checkout", e)
	}
	_, e = a.DB.Update(ctx, false, func(s State, tx pgx.Tx) (any, error) {
		s["brand"] = Object{"name": "should rollback"}
		_, e := tx.Exec(ctx, "INSERT INTO notification_outbox(id,kind,message) VALUES('rollback','test','test')")
		if e != nil {
			return nil, e
		}
		return nil, fail("forced failure")
	})
	if e == nil {
		t.Fatal("expected error")
	}
	s, _ = a.DB.Read(ctx)
	if obj(s["brand"])["name"] == "should rollback" {
		t.Fatal("state leaked")
	}
	var n int
	if e = a.DB.Pool.QueryRow(ctx, "SELECT count(*) FROM notification_outbox WHERE id='rollback'").Scan(&n); e != nil || n != 0 {
		t.Fatal("outbox leaked", e)
	}
}
func TestPostgresSePayReplayConflictAndAtomicity(t *testing.T) {
	a := testDB(t)
	ctx := context.Background()
	o := createTestOrder(t, a)
	payment := paymentFor(o, 100)
	raw := encoded(payment)
	var normalized Object
	_ = json.Unmarshal(raw, &normalized)
	payment = normalized
	var wg sync.WaitGroup
	errs := make(chan error, 10)
	for n := 0; n < 10; n++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, e := a.receive(ctx, payment); errs <- e }()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	s, e := a.DB.Read(ctx)
	if e != nil {
		t.Fatal(e)
	}
	got := find(s, "orders", "id", str(o["id"]))
	if got["paymentStatus"] != "paid" || got["status"] != "new" {
		t.Fatalf("wrong paid state %v", got)
	}
	for _, table := range []string{"payment_transactions", "notification_outbox"} {
		var n int
		if e = a.DB.Pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&n); e != nil || n != 1 {
			t.Fatalf("duplicate %s %d %v", table, n, e)
		}
	}
	payment["transferAmount"] = 1.0
	v, e := a.receive(ctx, payment)
	if e != nil || obj(v)["status"] != "conflict" {
		t.Fatal("changed payload accepted", e)
	}
	s, _ = a.DB.Read(ctx)
	if find(s, "orders", "id", str(o["id"]))["paymentStatus"] != "paid" {
		t.Fatal("paid overwritten")
	}
}
func TestPostgresPaymentExceptions(t *testing.T) {
	for _, scenario := range []string{"amount", "account", "gateway", "expired", "out", "ambiguous", "disabled", "subaccount"} {
		t.Run(scenario, func(t *testing.T) {
			a := testDB(t)
			o := createTestOrder(t, a)
			p := paymentFor(o, 200)
			expected := "needs_review"
			switch scenario {
			case "amount":
				p["transferAmount"] = 1.0
			case "account":
				p["accountNumber"] = "999999"
			case "gateway":
				p["gateway"] = "OTHER"
			case "subaccount":
				p["subAccount"] = "VA123"
			case "out":
				p["transferType"] = "out"
				expected = "ignored"
			case "ambiguous":
				p["content"] = str(p["content"]) + " MVABCDEF123456"
			case "disabled":
				a.AutoMatch = false
			case "expired":
				_, e := a.DB.Update(context.Background(), false, func(s State, _ pgx.Tx) (any, error) {
					find(s, "orders", "id", str(o["id"]))["paymentExpiresAt"] = time.Now().Add(-time.Minute).Format(time.RFC3339)
					return nil, nil
				})
				if e != nil {
					t.Fatal(e)
				}
			}
			v, e := a.receive(context.Background(), p)
			if e != nil || obj(v)["status"] != expected {
				t.Fatalf("%v %v", v, e)
			}
			s, _ := a.DB.Read(context.Background())
			if find(s, "orders", "id", str(o["id"]))["paymentStatus"] != "unpaid" {
				t.Fatal("unexpected payment")
			}
		})
	}
}
func TestMigrationPreservesPaidAndSentHistory(t *testing.T) {
	a := testDB(t)
	ctx := context.Background()
	o := createTestOrder(t, a)
	if _, e := a.receive(ctx, paymentFor(o, 500)); e != nil {
		t.Fatal(e)
	}
	if _, e := a.DB.Pool.Exec(ctx, "UPDATE notification_outbox SET status='sent',sent_at=now()"); e != nil {
		t.Fatal(e)
	}
	s, e := a.DB.Read(ctx)
	if e != nil {
		t.Fatal(e)
	}
	transactions, e := rowsJSON(ctx, a.DB.Pool, "SELECT row_to_json(t) FROM payment_transactions t")
	if e != nil {
		t.Fatal(e)
	}
	outbox, e := rowsJSON(ctx, a.DB.Pool, "SELECT row_to_json(t) FROM (SELECT id,transaction_id,kind,message,status,attempts,extract(epoch from retry_at)*1000 retry_at,created_at,sent_at,last_error FROM notification_outbox) t")
	if e != nil {
		t.Fatal(e)
	}
	path := t.TempDir() + "/snapshot.json"
	if e = os.WriteFile(path, encoded(Export{State: s, Transactions: transactions, Outbox: outbox, Metrics: []Object{}}), 0600); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 2; i++ {
		if e = a.ImportFile(ctx, path); e != nil {
			t.Fatal(e)
		}
	}
	s, e = a.DB.Read(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if find(s, "orders", "id", str(o["id"]))["paymentStatus"] != "paid" {
		t.Fatal("paid lost")
	}
	var sent int
	if e = a.DB.Pool.QueryRow(ctx, "SELECT count(*) FROM notification_outbox WHERE status='sent'").Scan(&sent); e != nil || sent != 1 {
		t.Fatal("sent lost", e)
	}
	if _, e = a.receive(ctx, paymentFor(o, 500)); e != nil {
		t.Fatal(e)
	}
	var count int
	if e = a.DB.Pool.QueryRow(ctx, "SELECT count(*) FROM notification_outbox").Scan(&count); e != nil || count != 1 {
		t.Fatal("history replayed", e)
	}
}
func TestOutboxRetriesWithoutLosingPayment(t *testing.T) {
	a := testDB(t)
	ctx := context.Background()
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer qa-bridge-secret" {
			t.Error("missing bridge authentication")
		}
		if r.URL.Path == "/status" {
			w.Write([]byte(`{"state":"connected"}`))
			return
		}
		calls++
		if calls == 1 {
			w.WriteHeader(503)
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	a.BridgeURL = server.URL
	a.BridgeSecret = "qa-bridge-secret"
	_, e := a.DB.Update(ctx, false, func(s State, tx pgx.Tx) (any, error) {
		s["notifications"] = Object{"enabled": true, "phone": "0900000000"}
		return nil, a.enqueue(ctx, tx, "payment:test", "test", "payment.received", "QA payment")
	})
	if e != nil {
		t.Fatal(e)
	}
	if e = a.Tick(ctx); e != nil {
		t.Fatal(e)
	}
	var status string
	var attempts int
	if e = a.DB.Pool.QueryRow(ctx, "SELECT status,attempts FROM notification_outbox WHERE id='payment:test'").Scan(&status, &attempts); e != nil || status != "failed" || attempts != 1 {
		t.Fatal("failure not retained", status, attempts, e)
	}
	if _, e = a.DB.Pool.Exec(ctx, "UPDATE notification_outbox SET retry_at=now() WHERE id='payment:test'"); e != nil {
		t.Fatal(e)
	}
	if e = a.Tick(ctx); e != nil {
		t.Fatal(e)
	}
	if e = a.DB.Pool.QueryRow(ctx, "SELECT status,attempts FROM notification_outbox WHERE id='payment:test'").Scan(&status, &attempts); e != nil || status != "sent" || attempts != 2 {
		t.Fatal("retry failed", status, attempts, e)
	}
}
func TestDeliveryCapacityAndCouponAreReservedAtomically(t *testing.T) {
	a := testDB(t)
	ctx := context.Background()
	date := time.Now().In(time.FixedZone("VN", 25200)).AddDate(0, 0, 1).Format("2006-01-02")
	_, e := a.DB.Update(ctx, false, func(s State, _ pgx.Tx) (any, error) {
		shipping := obj(s["shipping"])
		shipping["scheduleEnabled"] = true
		shipping["leadHours"] = 1
		shipping["maxDays"] = 7
		shipping["slots"] = []Object{{"label": "Trưa", "start": "12:00", "capacity": 1}}
		s["coupons"] = []Object{{"code": "QA10", "percent": 10, "minimum": 0, "maxDiscount": 10000, "limit": 1, "used": 0, "endsAt": "2099-12-31", "active": true}}
		return nil, nil
	})
	if e != nil {
		t.Fatal(e)
	}
	b := orderInput()
	b["delivery"] = Object{"date": date, "slot": "Trưa"}
	b["coupon"] = "QA10"
	v, e := a.DB.Update(ctx, true, func(s State, _ pgx.Tx) (any, error) { return placeOrder(s, b) })
	if e != nil {
		t.Fatal(e)
	}
	b["key"] = "second-order-request"
	if _, e = a.DB.Update(ctx, true, func(s State, _ pgx.Tx) (any, error) { return placeOrder(s, b) }); e == nil {
		t.Fatal("slot/coupon overbooked")
	}
	_, e = a.DB.Update(ctx, false, func(s State, _ pgx.Tx) (any, error) {
		return nil, changeOrder(s, Object{"id": obj(v)["id"], "status": "cancelled", "paymentStatus": "unpaid"})
	})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = a.DB.Update(ctx, true, func(s State, _ pgx.Tx) (any, error) { return placeOrder(s, b) }); e != nil {
		t.Fatal("reservation not released", e)
	}
}

func TestDiagnosticsReportsPersistedMetrics(t *testing.T) {
	a := testDB(t)
	ctx := context.Background()
	_, err := a.DB.Pool.Exec(ctx, `INSERT INTO performance_samples(id,name,path,value,device) VALUES ('qa-lcp','LCP','/',2000,'mobile')`)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/api/diagnostics", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: a.session(a.AdminUser)})
	response := httptest.NewRecorder()
	a.diagnostics(response, req)
	if response.Code != 200 {
		t.Fatalf("diagnostics status %d: %s", response.Code, response.Body.String())
	}
	var report Object
	if err := json.Unmarshal(response.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if num(report["total"]) != 1 || len(list(report["groups"])) != 1 {
		t.Fatalf("unexpected report: %v", report)
	}
	group := list(report["groups"])[0]
	if num(group["p75"]) != 2000 || group["rating"] != "good" {
		t.Fatalf("unexpected aggregation: %v", group)
	}
}

func TestPostgresConcurrentOptionOrders(t *testing.T) {
	t.Setenv("PRODUCT_OPTIONS_ENABLED", "1")
	a := testDB(t)
	ctx := context.Background()
	_, e := a.DB.Update(ctx, false, func(s State, _ pgx.Tx) (any, error) {
		p := list(s["products"])[0]
		p["optionGroups"] = []Object{{"id": "size", "name": "Cỡ", "type": "single", "min": 1, "max": 1, "active": true, "choices": []Object{{"id": "large", "name": "Lớn", "priceDelta": 10000, "active": true}}}}
		p["noteEnabled"] = true
		return nil, nil
	})
	if e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	successes := make(chan bool, 6)
	for n := 0; n < 6; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			b := orderInput()
			b["key"] = "option-concurrent-" + strings.Repeat("x", n+1)
			b["items"] = []Object{{"id": "com", "quantity": 1, "optionIds": []any{"large"}, "note": "Ít ngọt"}}
			_, e := a.DB.Update(ctx, true, func(s State, _ pgx.Tx) (any, error) {
				qb := clean(b)
				qb["region"] = "hanoi"
				q, err := quote(s, qb)
				if err != nil {
					return nil, err
				}
				b["quoteRevision"] = q["quoteRevision"]
				return placeOrder(s, b)
			})
			successes <- e == nil
		}(n)
	}
	wg.Wait()
	close(successes)
	count := 0
	for ok := range successes {
		if ok {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("expected two reservations, got %d", count)
	}
	s, e := a.DB.Read(ctx)
	if e != nil {
		t.Fatal(e)
	}
	orders := list(s["orders"])
	if len(orders) != 2 || num(list(s["products"])[0]["stock"]) != 0 {
		t.Fatal("incorrect inventory")
	}
	for _, o := range orders {
		i := list(o["items"])[0]
		if str(i["note"]) != "Ít ngọt" || num(i["price"]) != 110000 || str(list(i["options"])[0]["name"]) != "Lớn" {
			t.Fatal("snapshot missing", i)
		}
	}
}

func TestPostgresOptionOrderSePayAndOutbox(t *testing.T) {
	t.Setenv("PRODUCT_OPTIONS_ENABLED", "1")
	a := testDB(t)
	ctx := context.Background()
	_, e := a.DB.Update(ctx, false, func(s State, _ pgx.Tx) (any, error) {
		p := list(s["products"])[0]
		p["optionGroups"] = []Object{{"id": "topping", "name": "Topping", "type": "multiple", "min": 0, "max": 1, "active": true, "choices": []Object{{"id": "dua", "name": "Dừa", "priceDelta": 5000, "active": true}}}}
		p["noteEnabled"] = true
		return nil, nil
	})
	if e != nil {
		t.Fatal(e)
	}
	b := orderInput()
	b["items"] = []Object{{"id": "com", "quantity": 1, "optionIds": []any{"dua"}, "note": "Ít ngọt"}}
	o, e := a.DB.Update(ctx, true, func(s State, _ pgx.Tx) (any, error) {
		qinput := clean(b)
		qinput["region"] = "hanoi"
		q, err := quote(s, qinput)
		if err != nil {
			return nil, err
		}
		b["quoteRevision"] = q["quoteRevision"]
		return placeOrder(s, b)
	})
	if e != nil {
		t.Fatal(e)
	}
	order := obj(o)
	if num(order["total"]) != 135000 {
		t.Fatal("wrong payable total", order)
	}
	var payment Object
	_ = json.Unmarshal(encoded(paymentFor(order, 230)), &payment)
	for n := 0; n < 2; n++ {
		if _, e = a.receive(ctx, payment); e != nil {
			t.Fatal(e)
		}
	}
	s, e := a.DB.Read(ctx)
	if e != nil {
		t.Fatal(e)
	}
	saved := find(s, "orders", "id", str(order["id"]))
	if saved["paymentStatus"] != "paid" || str(list(list(saved["items"])[0]["options"])[0]["name"]) != "Dừa" {
		t.Fatal("payment or snapshot lost", saved)
	}
	var count int
	if e = a.DB.Pool.QueryRow(ctx, "SELECT count(*) FROM notification_outbox").Scan(&count); e != nil || count != 1 {
		t.Fatalf("outbox duplicated: %d %v", count, e)
	}
	if msg := orderMessage(saved, "http://localhost"); !strings.Contains(msg, "Topping: Dừa") || !strings.Contains(msg, "Ít ngọt") {
		t.Fatal("missing option in notification")
	}
}
