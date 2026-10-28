"""Run on an offline SQLite snapshot, never a live copy of a WAL database."""
import json, sqlite3, sys, os
source, destination = sys.argv[1:]
db=sqlite3.connect(f'file:{source}?mode=ro',uri=True)
db.row_factory=sqlite3.Row
state=json.loads(db.execute("SELECT body FROM shop_state WHERE id='shop'").fetchone()['body'])
result={'state':state}
for table in ['payment_transactions','payment_outbox','performance_samples']:
    exists=db.execute('SELECT 1 FROM sqlite_master WHERE type=? AND name=?',('table',table)).fetchone()
    result[table]=[dict(row) for row in db.execute('SELECT * FROM '+table)] if exists else []
fd=os.open(destination,os.O_CREAT|os.O_TRUNC|os.O_WRONLY,0o600)
with os.fdopen(fd,'w') as f: json.dump(result,f,ensure_ascii=False)
print(json.dumps({'orders':len(state['orders']),'products':len(state['products']),'transactions':len(result['payment_transactions']),'outbox':len(result['payment_outbox'])}))
