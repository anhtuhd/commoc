import type {DatabaseSync} from 'node:sqlite';
export function initializePayments(db:DatabaseSync){db.exec(`
 CREATE TABLE IF NOT EXISTS payment_transactions (
  id TEXT PRIMARY KEY, fingerprint TEXT NOT NULL, payload TEXT NOT NULL,
  status TEXT NOT NULL, reason TEXT NOT NULL, order_id TEXT, created_at TEXT NOT NULL,
  resolved_by TEXT, resolved_at TEXT, conflict INTEGER NOT NULL DEFAULT 0
 );
 CREATE INDEX IF NOT EXISTS payment_status ON payment_transactions(status,created_at);
 CREATE TABLE IF NOT EXISTS payment_outbox (
  id TEXT PRIMARY KEY, transaction_id TEXT NOT NULL, kind TEXT NOT NULL, message TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending', attempts INTEGER NOT NULL DEFAULT 0,
  retry_at INTEGER NOT NULL DEFAULT 0, lease TEXT, lease_until INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL, sent_at TEXT, last_error TEXT
 );
 CREATE INDEX IF NOT EXISTS payment_delivery ON payment_outbox(status,retry_at);
`);}
export function atomic<T>(db:DatabaseSync,fn:()=>T):T {db.exec('BEGIN IMMEDIATE');try{const value=fn();db.exec('COMMIT');return value}catch(e){db.exec('ROLLBACK');throw e}}
