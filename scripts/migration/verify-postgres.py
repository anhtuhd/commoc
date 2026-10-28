"""Read-only verification on VPS; print counts only, not customer data."""
import json, subprocess, sys, datetime
source=json.load(open(sys.argv[1]))
container=sys.argv[2]
def query(sql):
    raw=subprocess.check_output(['docker','exec',container,'psql','-U','commoc','-d','commoc','-At','-c',sql],text=True)
    return json.loads(raw.strip() or 'null')
state=query("SELECT coalesce(jsonb_object_agg(key,body),'{}') FROM settings")
for table in ['products','orders','collections','campaigns','articles','coupons','inquiries','reviews']:
    state[table]=query(f"SELECT coalesce(jsonb_agg(body ORDER BY position,id),'[]') FROM {table}")
for key,value in source['state'].items():
    if key=='rate':continue # ephemeral request throttles intentionally reset at cutover
    assert state.get(key)==value, f'State differs: {key}'
transactions=query("SELECT coalesce(jsonb_agg(to_jsonb(t)),'[]') FROM payment_transactions t")
assert len(transactions)==len(source['payment_transactions'])
by_id={r['id']:r for r in transactions}
for old in source['payment_transactions']:
    new=by_id[old['id']]
    for key in ['status','reason','order_id','conflict']:
        assert new[key]==old[key],f'Payment differs: {key}'
    assert new['payload']==(json.loads(old['payload']) if isinstance(old['payload'],str) else old['payload'])
outbox=query("SELECT coalesce(jsonb_agg(to_jsonb(t)),'[]') FROM notification_outbox t")
by_id={r['id']:r for r in outbox}
for old in source['payment_outbox']:
    new=by_id[old['id']]
    assert new['status']==('failed' if old['status']=='sending' else old['status'])
    for key in ['kind','message','attempts']:
        assert new[key]==old[key],f'Outbox differs: {key}'
metrics=query('SELECT count(*) FROM performance_samples')
assert metrics==len(source['performance_samples'])
print(json.dumps({'verified':True,'orders':len(state['orders']),'products':len(state['products']),'payments':len(transactions),'outbox':len(outbox),'metrics':metrics}))
