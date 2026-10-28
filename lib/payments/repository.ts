import type {DatabaseSync} from 'node:sqlite';
import {createHash} from 'node:crypto';
import type {ShopState,Order} from '../shop/types.ts';
import {eligibility,matchOrder,type Transaction} from './sepay.ts';

import {atomic} from './sqlite.ts';
export {initializePayments} from './sqlite.ts';
function state(db:DatabaseSync):ShopState {const row=db.prepare("SELECT body FROM shop_state WHERE id='shop'").get() as {body:string}|undefined;if(!row)throw Error('Shop state unavailable');return JSON.parse(row.body)}
function save(db:DatabaseSync,s:ShopState){db.prepare("UPDATE shop_state SET body=?,revision=revision+1 WHERE id='shop'").run(JSON.stringify(s))}
function audit(s:ShopState,actor:string,message:string,now:number){s.audit.unshift({at:new Date(now).toISOString(),actor,action:message});s.audit=s.audit.slice(0,2000)}
function markPaid(o:Order,key:string,now:number){o.paymentStatus='paid';o.paidAt=new Date(now).toISOString();o.paymentSource='sepay';o.sepayTransactionId=key}
function enqueue(db:DatabaseSync,key:string,t:Transaction,kind:string,reason:string,o:Order|undefined,now:number){
 const message=[kind==='payment.received'?'ĐÃ NHẬN THANH TOÁN':'CẦN ĐỐI SOÁT CHUYỂN KHOẢN',o?`Đơn ${o.number} · ${o.brand.name}`:'Chưa khớp đơn hàng',`Đã nhận: ${t.transferAmount.toLocaleString('vi-VN')}đ`,o?`Tổng đơn: ${o.total.toLocaleString('vi-VN')}đ`:'',`Thời gian: ${t.transactionDate} (VN)`,`Giao dịch: ${key}`,reason,'Quản trị: https://pishare.cloud/quan-tri'].filter(Boolean).join('\n');
 db.prepare('INSERT OR IGNORE INTO payment_outbox(id,transaction_id,kind,message,created_at) VALUES(?,?,?,?,?)').run(key+':'+kind,key,kind,message,new Date(now).toISOString());
}
export function receivePayment(db:DatabaseSync,t:Transaction,mode:'live'|'test',autoMatch:boolean,now=Date.now()){
 const key=mode+':'+t.id,payload=JSON.stringify(t),fingerprint=createHash('sha256').update(payload).digest('hex');
 return atomic(db,()=>{
  const previous=db.prepare('SELECT fingerprint,status FROM payment_transactions WHERE id=?').get(key) as {fingerprint:string;status:string}|undefined;
  if(previous){if(previous.fingerprint!==fingerprint){db.prepare('UPDATE payment_transactions SET conflict=1 WHERE id=?').run(key);return {status:'conflict',duplicate:true}}return {status:previous.status,duplicate:true}}
  const s=state(db),result=matchOrder(s,t,now);
  if(result.status==='matched'&&(!autoMatch||!s.payments?.enabled)){result.status='needs_review';result.reason='Tự động xác nhận đang tắt'}
  db.prepare('INSERT INTO payment_transactions(id,fingerprint,payload,status,reason,order_id,created_at) VALUES(?,?,?,?,?,?,?)').run(key,fingerprint,payload,result.status,result.reason,result.order?.id||null,new Date(now).toISOString());
  if(result.status==='matched')markPaid(result.order!,key,now);
  if(result.status!=='ignored')enqueue(db,key,t,result.status==='matched'?'payment.received':'payment.needs_review',result.reason,result.order,now);
  audit(s,'sepay',`Giao dịch ${key}: ${result.status}${result.order?' / '+result.order.number:''}`,now);save(db,s);
  return {status:result.status,duplicate:false};
 });
}
export function resolvePayment(db:DatabaseSync,key:string,number:string,reason:string,actor:string,now=Date.now()){
 if(typeof reason!=='string'||reason.trim().length<8||reason.length>500||typeof number!=='string'||number.length>50)throw Error('Nhập mã đơn và lý do đối soát ít nhất 8 ký tự.');
 return atomic(db,()=>{
  const row=db.prepare('SELECT payload,status,conflict FROM payment_transactions WHERE id=?').get(key) as {payload:string;status:string;conflict:number}|undefined;
  if(!row||row.status!=='needs_review'||row.conflict)throw Error('Giao dịch không thể gắn vào đơn.');
  const t=JSON.parse(row.payload) as Transaction,s=state(db),matches=s.orders.filter(o=>o.number===number.trim().toUpperCase());
  if(matches.length!==1||t.transferType!=='in')throw Error('Không tìm thấy đơn phù hợp.');
  const o=matches[0],error=eligibility(s,t,o,now);if(error)throw Error(error);
  if(db.prepare("SELECT id FROM payment_transactions WHERE order_id=? AND status='matched'").get(o.id))throw Error('Đơn đã được gắn giao dịch khác.');
  markPaid(o,key,now);db.prepare("UPDATE payment_transactions SET status='matched',reason=?,order_id=?,resolved_by=?,resolved_at=? WHERE id=?").run(reason.trim(),o.id,actor,new Date(now).toISOString(),key);
  // Suppress unsent review messages now that the issue has been resolved.
  db.prepare("UPDATE payment_outbox SET status='cancelled',lease=NULL WHERE transaction_id=? AND kind='payment.needs_review' AND status IN ('pending','failed')").run(key);
  enqueue(db,key,t,'payment.received','Đã đối soát thủ công',o,now);audit(s,actor,`Đối soát ${key} / ${o.number}: ${reason.trim()}`,now);save(db,s);return {ok:true};
 });
}
export function paymentDashboard(db:DatabaseSync){
 const rows=db.prepare('SELECT id,payload,status,reason,order_id,created_at,conflict FROM payment_transactions ORDER BY created_at DESC LIMIT 100').all() as unknown as {id:string;payload:string;status:string;reason:string;order_id:string|null;created_at:string;conflict:number}[];
 return {transactions:rows.map(({payload,...r})=>{const t=JSON.parse(payload) as Transaction;return {...r,amount:t.transferAmount,code:t.code,content:t.content.slice(0,500),account:'••••'+t.accountNumber.slice(-4),gateway:t.gateway,transactionDate:t.transactionDate}}),
  outbox:db.prepare('SELECT id,kind,status,attempts,last_error,sent_at,created_at FROM payment_outbox ORDER BY created_at DESC LIMIT 100').all(),
  counts:db.prepare('SELECT status,COUNT(*) count FROM payment_transactions GROUP BY status').all()};
}
