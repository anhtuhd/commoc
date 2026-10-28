import type {ShopState} from '../lib/shop/types.ts';
// Run once per minute under flock. No network call occurs inside the SQLite transaction.
import {DatabaseSync} from 'node:sqlite';
import {expireOrders} from '../lib/shop/commerce.ts';
const db=new DatabaseSync((process.env.DATA_DIR||'/data')+'/shop.sqlite');
db.exec('PRAGMA busy_timeout=10000');
function mutate(fn:(s:ShopState)=>number){db.exec('BEGIN IMMEDIATE');try{const row=db.prepare("SELECT body FROM shop_state WHERE id='shop'").get() as {body:string};const s=JSON.parse(row.body);const result=fn(s);db.prepare("UPDATE shop_state SET body=?,revision=revision+1 WHERE id='shop'").run(JSON.stringify(s));db.exec('COMMIT');return result}catch(e){db.exec('ROLLBACK');throw e}}
const expired=mutate(s=>expireOrders(s));
console.log('Expired orders:',expired);
db.close();
