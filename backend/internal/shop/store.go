package shop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var tables = []string{"products", "orders", "collections", "campaigns", "articles", "coupons", "inquiries", "reviews"}

type Store struct{ Pool *pgxpool.Pool }
type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func read(ctx context.Context, q queryer) (State, error) {
	s := State{}
	rows, e := q.Query(ctx, "SELECT key,body FROM settings")
	if e != nil {
		return nil, e
	}
	for rows.Next() {
		var k string
		var raw []byte
		if e = rows.Scan(&k, &raw); e != nil {
			rows.Close()
			return nil, e
		}
		var sVal any
		if e = json.Unmarshal(raw, &sVal); e != nil {
			rows.Close()
			return nil, e
		}
		s[k] = sVal
	}
	rows.Close()
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	for _, table := range tables {
		a := []Object{}
		r, e := q.Query(ctx, "SELECT body FROM "+table+" ORDER BY position,id")
		if e != nil {
			return nil, e
		}
		for r.Next() {
			var raw []byte
			if e = r.Scan(&raw); e != nil {
				r.Close()
				return nil, e
			}
			var o Object
			if e = json.Unmarshal(raw, &o); e != nil {
				r.Close()
				return nil, e
			}
			a = append(a, o)
		}
		r.Close()
		if r.Err() != nil {
			return nil, r.Err()
		}
		s[table] = a
	}
	return s, nil
}
func (d *Store) Read(ctx context.Context) (State, error) {
	tx, e := d.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	s, e := read(ctx, tx)
	if e != nil {
		return nil, e
	}
	return s, tx.Commit(ctx)
}

// A single commerce lock deliberately serializes mutations for this one-shop migration.
// Reads remain MVCC snapshots; only changed entity rows are written, never a whole-shop blob.
func (d *Store) Update(ctx context.Context, expire bool, fn func(State, pgx.Tx) (any, error)) (any, error) {
	tx, e := d.Pool.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(771026)"); e != nil {
		return nil, e
	}
	s, e := read(ctx, tx)
	if e != nil {
		return nil, e
	}
	before := State{}
	if e = json.Unmarshal(encoded(s), &before); e != nil {
		return nil, e
	}
	if expire {
		expireOrders(s, timeNow())
	}
	v, e := fn(s, tx)
	if e != nil {
		return nil, e
	}
	if e = save(ctx, tx, before, s); e != nil {
		return nil, e
	}
	return v, tx.Commit(ctx)
}
func save(ctx context.Context, tx pgx.Tx, before, s State) error {
	for k, v := range s {
		if contains(tables, k) || k == "rate" {
			continue
		}
		if !bytes.Equal(encoded(before[k]), encoded(v)) {
			if _, e := tx.Exec(ctx, "INSERT INTO settings(key,body) VALUES($1,$2) ON CONFLICT(key) DO UPDATE SET body=excluded.body", k, encoded(v)); e != nil {
				return e
			}
		}
	}
	for _, table := range tables {
		old := map[string]Object{}
		for _, o := range list(before[table]) {
			key := str(o["id"])
			if table == "coupons" {
				key = str(o["code"])
			}
			old[key] = o
		}
		for position, o := range list(s[table]) {
			key := str(o["id"])
			if table == "coupons" {
				key = str(o["code"])
			}
			if key == "" {
				return fmt.Errorf("missing id: %s", table)
			}
			if !bytes.Equal(encoded(old[key]), encoded(o)) || oldPosition(before, table, key) != position {
				if _, e := tx.Exec(ctx, "INSERT INTO "+table+"(id,body,position) VALUES($1,$2,$3) ON CONFLICT(id) DO UPDATE SET body=excluded.body,position=excluded.position", key, encoded(o), position); e != nil {
					return e
				}
				if table == "orders" {
					if _, e := tx.Exec(ctx, "DELETE FROM order_items WHERE order_id=$1", key); e != nil {
						return e
					}
					for i, item := range list(o["items"]) {
						if _, e := tx.Exec(ctx, "INSERT INTO order_items VALUES($1,$2,$3,$4,$5,$6)", key, i, str(item["id"]), int(num(item["quantity"])), int64(num(item["price"])), encoded(item)); e != nil {
							return e
						}
					}
				}
			}
			delete(old, key)
		}
		for key := range old {
			if _, e := tx.Exec(ctx, "DELETE FROM "+table+" WHERE id=$1", key); e != nil {
				return e
			}
		}
	}
	return nil
}

func oldPosition(s State, table, key string) int {
	for i, o := range list(s[table]) {
		field := "id"
		if table == "coupons" {
			field = "code"
		}
		if str(o[field]) == key {
			return i
		}
	}
	return -1
}
